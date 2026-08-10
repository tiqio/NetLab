package reconcile

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/domain"
	"github.com/netlab/netlab/internal/runtime/ownership"
)

type discoveryStore struct {
	records []ownership.Record
	owners  map[string]bool
}

func (s *discoveryStore) DeleteRuntimeOwnership(_ context.Context, resourceType string, resourceID domain.ID, objectKind, objectName string) error {
	for index := range s.records {
		value := s.records[index]
		if value.ResourceType == resourceType && value.ResourceID == resourceID && value.ObjectKind == objectKind && value.ObjectName == objectName {
			s.records = append(s.records[:index], s.records[index+1:]...)
			return nil
		}
	}
	return nil
}

func (s *discoveryStore) ListRuntimeOwnership(context.Context) ([]ownership.Record, error) {
	return append([]ownership.Record(nil), s.records...), nil
}

func (s *discoveryStore) UpsertRuntimeOwnership(_ context.Context, resourceType string, resourceID domain.ID, objectKind, objectName string, metadata map[string]string, cleanupState string) error {
	for index := range s.records {
		if s.records[index].ResourceType == resourceType && s.records[index].ResourceID == resourceID && s.records[index].ObjectKind == objectKind && s.records[index].ObjectName == objectName {
			s.records[index].Metadata = cloneMetadata(metadata)
			s.records[index].CleanupState = cleanupState
			s.records[index].OwnershipClass = ownership.Classify(resourceType, metadata)
			return nil
		}
	}
	s.records = append(s.records, ownership.Record{ResourceType: resourceType, ResourceID: resourceID, ObjectKind: objectKind, ObjectName: objectName, Metadata: cloneMetadata(metadata), CleanupState: cleanupState, OwnershipClass: ownership.Classify(resourceType, metadata)})
	return nil
}

func (s *discoveryStore) RuntimeOwnerExists(_ context.Context, resourceType string, resourceID domain.ID) (bool, error) {
	return s.owners[resourceType+":"+string(resourceID)], nil
}

type discoveryAudit struct{ actions []string }

func (a *discoveryAudit) Record(_ context.Context, _, action, _ string, _, _ domain.ID, _, _ string, _ map[string]any) (domain.AuditEvent, error) {
	a.actions = append(a.actions, action)
	return domain.AuditEvent{}, nil
}

type staticOwnershipScanner struct {
	name   string
	values []DiscoveredOwnership
	err    error
}

type blockingOwnershipScanner struct{ name string }

type nonCooperativeOwnershipScanner struct {
	name    string
	release <-chan struct{}
}

func (s blockingOwnershipScanner) Name() string { return s.name }
func (s blockingOwnershipScanner) Discover(ctx context.Context) ([]DiscoveredOwnership, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}
func (s nonCooperativeOwnershipScanner) Name() string { return s.name }
func (s nonCooperativeOwnershipScanner) Discover(context.Context) ([]DiscoveredOwnership, error) {
	<-s.release
	return nil, nil
}

func (s staticOwnershipScanner) Name() string { return s.name }
func (s staticOwnershipScanner) Discover(context.Context) ([]DiscoveredOwnership, error) {
	return s.values, s.err
}

func TestOwnershipDiscoveryAdoptsKnownQEMUDirectory(t *testing.T) {
	stateDir := t.TempDir()
	directory := filepath.Join(stateDir, "runtime", "qemu", "node-1")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "launch.json"), []byte(`{"node_id":"node-1","pid":42}`), 0o600); err != nil {
		t.Fatal(err)
	}
	store := &discoveryStore{owners: map[string]bool{"node:node-1": true}}
	if err := NewOwnershipDiscoveryReconciler(store, nil, NewQEMUOwnershipScanner(stateDir)).Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(store.records) != 1 || store.records[0].CleanupState != "active" || store.records[0].ResourceID != "node-1" {
		t.Fatalf("records=%+v", store.records)
	}
	if _, err := os.Stat(directory); err != nil {
		t.Fatalf("known directory moved: %v", err)
	}
}

func TestOwnershipDiscoveryQuarantinesMalformedQEMUDirectory(t *testing.T) {
	stateDir := t.TempDir()
	directory := filepath.Join(stateDir, "runtime", "qemu", "orphan")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	store := &discoveryStore{owners: map[string]bool{}}
	audit := &discoveryAudit{}
	if err := NewOwnershipDiscoveryReconciler(store, audit, NewQEMUOwnershipScanner(stateDir)).Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(store.records) != 1 || store.records[0].CleanupState != "quarantined" {
		t.Fatalf("records=%+v", store.records)
	}
	if _, err := os.Stat(directory); !os.IsNotExist(err) {
		t.Fatalf("orphan directory still present: %v", err)
	}
	if store.records[0].Metadata["quarantine_path"] == "" || audit.actions[0] != "ownership.resource.quarantined" {
		t.Fatalf("record=%+v audit=%v", store.records[0], audit.actions)
	}
}

func TestOwnershipDiscoveryRefusesPathOutsideOwnedRoot(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "unowned")
	if err := os.MkdirAll(outside, 0o700); err != nil {
		t.Fatal(err)
	}
	store := &discoveryStore{owners: map[string]bool{}}
	scanner := staticOwnershipScanner{name: "unsafe", values: []DiscoveredOwnership{{ObjectKind: "directory", ObjectName: "unowned", QuarantineRoot: root, QuarantinePath: outside}}}
	if err := NewOwnershipDiscoveryReconciler(store, nil, scanner).Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if store.records[0].CleanupState != "quarantine_failed" {
		t.Fatalf("records=%+v", store.records)
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatalf("unowned path modified: %v", err)
	}
}

func TestOwnershipDiscoveryRemovesDisappearedUnknownObservation(t *testing.T) {
	store := &discoveryStore{records: []ownership.Record{{ResourceType: "unknown", ResourceID: "old", ObjectKind: "capture_record", ObjectName: "old", CleanupState: "unknown_observed"}}, owners: map[string]bool{}}
	audit := &discoveryAudit{}
	if err := NewOwnershipDiscoveryReconciler(store, audit, staticOwnershipScanner{name: "empty"}).Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(store.records) != 0 {
		t.Fatalf("stale unknown record retained: %+v", store.records)
	}
}

func TestOwnershipDiscoveryCleansPreviouslyObservedSafeOrphan(t *testing.T) {
	cleaned := false
	store := &discoveryStore{records: []ownership.Record{{ResourceType: "unknown", ResourceID: "nlpc-orphan", ObjectKind: "network_namespace", ObjectName: "nlpc-orphan", CleanupState: "unknown_observed"}}, owners: map[string]bool{}}
	scanner := staticOwnershipScanner{name: "linux-network", values: []DiscoveredOwnership{{ObjectKind: "network_namespace", ObjectName: "nlpc-orphan", CleanupSafe: true, Cleanup: func(context.Context) error { cleaned = true; return nil }}}}
	if err := NewOwnershipDiscoveryReconciler(store, nil, scanner).Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !cleaned || len(store.records) != 0 {
		t.Fatalf("cleaned=%v records=%+v", cleaned, store.records)
	}
}

func TestOwnershipDiscoveryOnlyAuditsUnknownHostObject(t *testing.T) {
	store := &discoveryStore{owners: map[string]bool{}}
	audit := &discoveryAudit{}
	scanner := staticOwnershipScanner{name: "host", values: []DiscoveredOwnership{{ObjectKind: "linux_link", ObjectName: "nli-orphan"}}}
	if err := NewOwnershipDiscoveryReconciler(store, audit, scanner).Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if store.records[0].CleanupState != "unknown_observed" || audit.actions[0] != "ownership.resource.discovered" {
		t.Fatalf("records=%+v audit=%v", store.records, audit.actions)
	}
	if store.records[0].OwnershipClass != ownership.ClassForeignObserved {
		t.Fatalf("ownership class=%q", store.records[0].OwnershipClass)
	}
	if err := NewOwnershipDiscoveryReconciler(store, audit, scanner).Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if store.records[0].CleanupState != "unknown_observed" || store.records[0].ResourceType != "unknown" {
		t.Fatalf("unknown object was adopted on repeat scan: %+v", store.records[0])
	}
	if len(audit.actions) != 1 {
		t.Fatalf("repeat discovery produced duplicate audit events: %v", audit.actions)
	}
}

func TestOwnershipDiscoveryMarksMissingWithoutRemoval(t *testing.T) {
	metadata := map[string]string{"path": "/must/not/change"}
	store := &discoveryStore{records: []ownership.Record{{ResourceType: "interface", ResourceID: "if-1", ObjectKind: "tap", ObjectName: "nltif-1", Metadata: metadata, CleanupState: "active"}}, owners: map[string]bool{}}
	if err := NewOwnershipDiscoveryReconciler(store, nil).Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if store.records[0].CleanupState != "missing_validation_required" || store.records[0].Metadata["missing_since"] == "" {
		t.Fatalf("records=%+v", store.records)
	}
	if _, changed := metadata["missing_since"]; changed {
		t.Fatal("input metadata was mutated")
	}
}

func TestOwnershipDiscoveryRemovesMissingNetworkObjectLinkAfterOwnerDeletion(t *testing.T) {
	store := &discoveryStore{
		records: []ownership.Record{{
			ResourceType: "network_object_link",
			ResourceID:   "link-1",
			ObjectKind:   "network_object_link_endpoint",
			ObjectName:   "namespace-a:eth0",
			CleanupState: "missing_validation_required",
		}},
		owners: map[string]bool{},
	}
	audit := &discoveryAudit{}
	if err := NewOwnershipDiscoveryReconciler(store, audit).Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(store.records) != 0 {
		t.Fatalf("stale network object link ownership retained: %+v", store.records)
	}
	if len(audit.actions) != 1 || audit.actions[0] != "ownership.resource.resolved" {
		t.Fatalf("audit actions=%v", audit.actions)
	}
}

func TestOwnershipDiscoveryRemovesMissingNodeAfterOwnerDeletion(t *testing.T) {
	store := &discoveryStore{
		records: []ownership.Record{{
			ResourceType: "node",
			ResourceID:   "node-1",
			ObjectKind:   "docker_container",
			ObjectName:   "container-1",
			CleanupState: "missing_validation_required",
		}},
		owners: map[string]bool{},
	}
	audit := &discoveryAudit{}
	if err := NewOwnershipDiscoveryReconciler(store, audit).Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(store.records) != 0 {
		t.Fatalf("stale node ownership retained: %+v", store.records)
	}
	if len(audit.actions) != 1 || audit.actions[0] != "ownership.resource.resolved" {
		t.Fatalf("audit actions=%v", audit.actions)
	}
}

func TestOwnershipDiscoveryAuditsScannerFailureAndContinues(t *testing.T) {
	store := &discoveryStore{owners: map[string]bool{}}
	audit := &discoveryAudit{}
	failing := staticOwnershipScanner{name: "broken", err: errors.New("boom")}
	working := staticOwnershipScanner{name: "working", values: []DiscoveredOwnership{{ObjectKind: "helper", ObjectName: "netlab-helper"}}}
	if err := NewOwnershipDiscoveryReconciler(store, audit, failing, working).Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(store.records) != 1 || len(audit.actions) != 2 || audit.actions[0] != "ownership.discovery.failed" {
		t.Fatalf("records=%+v audit=%v", store.records, audit.actions)
	}
}

func TestOwnershipDiscoveryTimesOutScannerAndContinues(t *testing.T) {
	store := &discoveryStore{owners: map[string]bool{}}
	audit := &discoveryAudit{}
	reconciler := NewOwnershipDiscoveryReconciler(
		store,
		audit,
		blockingOwnershipScanner{name: "blocked"},
		staticOwnershipScanner{name: "working", values: []DiscoveredOwnership{{ObjectKind: "helper", ObjectName: "netlab-helper"}}},
	)
	reconciler.scannerTimeout = 10 * time.Millisecond
	if err := reconciler.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(store.records) != 1 || len(audit.actions) != 2 || audit.actions[0] != "ownership.discovery.failed" {
		t.Fatalf("records=%+v audit=%v", store.records, audit.actions)
	}
}

func TestOwnershipDiscoveryTimesOutNonCooperativeScanner(t *testing.T) {
	store := &discoveryStore{owners: map[string]bool{}}
	audit := &discoveryAudit{}
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	reconciler := NewOwnershipDiscoveryReconciler(
		store,
		audit,
		nonCooperativeOwnershipScanner{name: "blocked", release: release},
		staticOwnershipScanner{name: "working", values: []DiscoveredOwnership{{ObjectKind: "helper", ObjectName: "netlab-helper"}}},
	)
	reconciler.scannerTimeout = 10 * time.Millisecond
	if err := reconciler.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(store.records) != 1 || len(audit.actions) != 2 || audit.actions[0] != "ownership.discovery.failed" {
		t.Fatalf("records=%+v audit=%v", store.records, audit.actions)
	}
}

func TestOwnershipRecoveryCheckpointReportsReconnectRequired(t *testing.T) {
	store := &discoveryStore{records: []ownership.Record{{ResourceType: "node", ResourceID: "node-1", ObjectKind: "console_proxy", ObjectName: "session-1", CleanupState: "active"}}, owners: map[string]bool{}}
	scanner := staticOwnershipScanner{name: "console", values: nil}
	var outcomes []RecoveryResourceOutcome
	if err := NewOwnershipDiscoveryReconciler(store, nil, scanner).ReconcileWithCheckpoints(context.Background(), func(outcome RecoveryResourceOutcome) error { outcomes = append(outcomes, outcome); return nil }); err != nil {
		t.Fatal(err)
	}
	if len(outcomes) != 1 || outcomes[0].State != "reconnect_required" || outcomes[0].RuntimeID != "session-1" || outcomes[0].Details["cleanup_state"] != "missing_validation_required" {
		t.Fatalf("outcomes=%+v records=%+v", outcomes, store.records)
	}
}
