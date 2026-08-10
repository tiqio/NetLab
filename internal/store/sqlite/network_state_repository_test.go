package sqlite

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/domain"
)

func TestNetworkObservedStateAndEventCommitTogether(t *testing.T) {
	ctx := context.Background()
	database, err := Open(ctx, "file:network-observed-state?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	now := time.Now().UTC()
	topology := NewTopologyRepository(database)
	lab := domain.Laboratory{ID: "lab-net", Name: "net", Revision: 1, RecoveryPolicy: domain.RecoveryAutoRestore, LifecycleState: "active", CreatedAt: now, UpdatedAt: now}
	if err := topology.CreateLaboratory(ctx, lab); err != nil {
		t.Fatal(err)
	}
	repositories := NewRepositories(database)
	object := domain.NetworkObject{ID: "net-1", LaboratoryID: lab.ID, Name: "bridge", Kind: domain.NetworkBridge, Revision: 1, DesiredState: "active", ObservedState: "provisioning", Config: map[string]any{}, CreatedAt: now, UpdatedAt: now}
	if err := repositories.CreateNetworkObject(ctx, object); err != nil {
		t.Fatal(err)
	}
	if err := repositories.SetNetworkObjectState(ctx, object.ID, "active", nil); err != nil {
		t.Fatal(err)
	}
	stored, err := repositories.GetNetworkObject(ctx, object.ID)
	if err != nil || stored.ObservedState != "active" {
		t.Fatalf("stored=%#v err=%v", stored, err)
	}
	var count int
	if err := database.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM outbox_events WHERE event_type='network_object.observed_state_changed' AND resource_id=?`, object.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("events=%d", count)
	}
	if err := repositories.DeleteNetworkObject(ctx, object.ID, object.Revision); err != nil {
		t.Fatal(err)
	}
	if err := database.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM network_object_tombstones WHERE id=? AND revision=?`, object.ID, object.Revision.Next()).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("tombstones=%d", count)
	}
}

func TestCreateNetworkObjectWithPlacementCommitsAndRollsBackAtomically(t *testing.T) {
	ctx := context.Background()
	database, err := Open(ctx, "file:network-placement-atomic?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	topology := NewTopologyRepository(database)
	now := time.Now().UTC()
	laboratory := domain.Laboratory{ID: domain.NewID(), Name: "network placement", Revision: 1, RecoveryPolicy: domain.RecoveryAutoRestore, LifecycleState: "active", CreatedAt: now, UpdatedAt: now}
	if err = topology.CreateLaboratory(ctx, laboratory); err != nil {
		t.Fatal(err)
	}
	repositories := NewRepositories(database)
	object := domain.NetworkObject{ID: domain.NewID(), LaboratoryID: laboratory.ID, Name: "bridge", Kind: domain.NetworkBridge, Revision: 1, DesiredState: "active", ObservedState: "provisioning", Config: map[string]any{}, CreatedAt: now, UpdatedAt: now}
	assignment, revision, err := repositories.CreateNetworkObjectWithPlacement(ctx, object, laboratory.Revision, nil, "test")
	if err != nil || revision != 2 || assignment.Placement.ResourceID != object.ID {
		t.Fatalf("assignment=%+v revision=%d err=%v", assignment, revision, err)
	}
	var events int
	if err = database.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM outbox_events WHERE resource_id IN (?,?)`, object.ID, laboratory.ID).Scan(&events); err != nil || events < 2 {
		t.Fatalf("events=%d err=%v", events, err)
	}
	failed := domain.NetworkObject{ID: domain.NewID(), LaboratoryID: laboratory.ID, Name: "failed", Kind: domain.NetworkBridge, Revision: 1, DesiredState: "active", ObservedState: "provisioning", Config: map[string]any{}, CreatedAt: now, UpdatedAt: now}
	_, _, err = repositories.CreateNetworkObjectWithPlacement(ctx, failed, revision, &domain.PlacementIntent{FootprintClass: domain.FootprintNodeStandard}, "test")
	if err == nil {
		t.Fatal("expected invalid footprint failure")
	}
	var count int
	if err = database.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM network_objects WHERE id=?`, failed.ID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("objects=%d err=%v", count, err)
	}
	if err = database.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM topology_placements WHERE resource_id=?`, failed.ID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("placements=%d err=%v", count, err)
	}
}

func TestNetworkObjectLinkCreatePublishesRevisionedEvent(t *testing.T) {
	ctx := context.Background()
	database, err := Open(ctx, "file:network-object-link-event?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	now := time.Now().UTC()
	topology := NewTopologyRepository(database)
	lab := domain.Laboratory{ID: "lab-link-event", Name: "links", Revision: 1, RecoveryPolicy: domain.RecoveryRemainStopped, LifecycleState: "active", CreatedAt: now, UpdatedAt: now}
	if err = topology.CreateLaboratory(ctx, lab); err != nil {
		t.Fatal(err)
	}
	repositories := NewRepositories(database)
	for _, object := range []domain.NetworkObject{{ID: "a", LaboratoryID: lab.ID, Name: "A", Kind: domain.NetworkSwitchL2, Revision: 1, DesiredState: "active", ObservedState: "active", Config: map[string]any{}, CreatedAt: now, UpdatedAt: now}, {ID: "b", LaboratoryID: lab.ID, Name: "B", Kind: domain.NetworkSwitchL2, Revision: 1, DesiredState: "active", ObservedState: "active", Config: map[string]any{}, CreatedAt: now, UpdatedAt: now}} {
		if err = repositories.CreateNetworkObject(ctx, object); err != nil {
			t.Fatal(err)
		}
	}
	link := domain.NetworkObjectLink{ID: "link-event", LaboratoryID: lab.ID, ObjectAID: "a", PortAName: "swp1", ObjectBID: "b", PortBName: "swp1", Revision: 1, DesiredState: "connected", ObservedState: "pending"}
	if err = repositories.CreateNetworkObjectLink(ctx, link); err != nil {
		t.Fatal(err)
	}
	var revision int
	if err = database.DB.QueryRowContext(ctx, `SELECT revision FROM outbox_events WHERE event_type='network_object_link.created' AND resource_id=?`, link.ID).Scan(&revision); err != nil {
		t.Fatal(err)
	}
	if revision != 1 {
		t.Fatalf("revision=%d", revision)
	}
	problem := &domain.Problem{Code: "runtime_failed", Message: "veth endpoint missing"}
	if err = repositories.SetNetworkObjectLinkState(ctx, link.ID, "failed", problem); err != nil {
		t.Fatal(err)
	}
	var payload string
	if err = database.DB.QueryRowContext(ctx, `SELECT payload_json FROM outbox_events WHERE event_type='network_object_link.state_changed' AND resource_id=?`, link.ID).Scan(&payload); err != nil {
		t.Fatal(err)
	}
	if payload == "" || !strings.Contains(payload, `"observed_state":"failed"`) || !strings.Contains(payload, `"last_error"`) {
		t.Fatalf("payload=%s", payload)
	}
	if err = repositories.SetNetworkObjectLinkState(ctx, link.ID, "connected", nil); err != nil {
		t.Fatal(err)
	}
	if err = repositories.PublishNetworkObjectLinkRecovered(ctx, link.ID, "recovery-task"); err != nil {
		t.Fatal(err)
	}
	var recoveredTaskID string
	if err = database.DB.QueryRowContext(ctx, `SELECT task_id FROM outbox_events WHERE event_type='network_object_link.recovered' AND resource_id=?`, link.ID).Scan(&recoveredTaskID); err != nil {
		t.Fatal(err)
	}
	if recoveredTaskID != "recovery-task" {
		t.Fatalf("task_id=%q", recoveredTaskID)
	}
}

func TestNetworkObjectLinkDeleteChecksRevisionReleasesEndpointsAndPublishesTask(t *testing.T) {
	ctx := context.Background()
	database, err := Open(ctx, "file:"+string(domain.NewID())+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	now := time.Now().UTC()
	topology := NewTopologyRepository(database)
	lab := domain.Laboratory{ID: domain.NewID(), Name: "delete", Revision: 1, RecoveryPolicy: domain.RecoveryRemainStopped, LifecycleState: "active", CreatedAt: now, UpdatedAt: now}
	if err = topology.CreateLaboratory(ctx, lab); err != nil {
		t.Fatal(err)
	}
	repositories := NewRepositories(database)
	for _, object := range []domain.NetworkObject{{ID: "delete-a", LaboratoryID: lab.ID, Name: "A", Kind: domain.NetworkSwitchL2, Revision: 1, DesiredState: "active", ObservedState: "active", Config: map[string]any{}, CreatedAt: now, UpdatedAt: now}, {ID: "delete-b", LaboratoryID: lab.ID, Name: "B", Kind: domain.NetworkSwitchL2, Revision: 1, DesiredState: "active", ObservedState: "active", Config: map[string]any{}, CreatedAt: now, UpdatedAt: now}} {
		if err = repositories.CreateNetworkObject(ctx, object); err != nil {
			t.Fatal(err)
		}
	}
	link := domain.NetworkObjectLink{ID: "delete-link", LaboratoryID: lab.ID, ObjectAID: "delete-a", PortAName: "swp1", ObjectBID: "delete-b", PortBName: "swp1", Revision: 1, DesiredState: "connected", ObservedState: "connected"}
	if err = repositories.CreateNetworkObjectLink(ctx, link); err != nil {
		t.Fatal(err)
	}
	for _, endpoint := range []string{"namespace-a:swp1", "namespace-b:swp1"} {
		if err = repositories.UpsertRuntimeOwnership(ctx, "network_object_link", link.ID, "network_object_link_endpoint", endpoint, nil, "active"); err != nil {
			t.Fatal(err)
		}
	}
	if err = repositories.DeleteNetworkObjectLinkRevision(ctx, link.ID, 2, "delete-task"); err == nil {
		t.Fatal("expected revision conflict")
	}
	if _, err = repositories.GetNetworkObjectLink(ctx, link.ID); err != nil {
		t.Fatalf("revision conflict removed link: %v", err)
	}
	var ownershipCount int
	if err = database.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM runtime_ownership WHERE resource_type='network_object_link' AND resource_id=?`, link.ID).Scan(&ownershipCount); err != nil || ownershipCount != 2 {
		t.Fatalf("revision conflict changed ownership count=%d err=%v", ownershipCount, err)
	}
	if err = repositories.DeleteNetworkObjectLinkRevision(ctx, link.ID, 1, "delete-task"); err != nil {
		t.Fatal(err)
	}
	if err = database.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM runtime_ownership WHERE resource_type='network_object_link' AND resource_id=?`, link.ID).Scan(&ownershipCount); err != nil || ownershipCount != 0 {
		t.Fatalf("network object link ownership count=%d err=%v", ownershipCount, err)
	}
	if err = repositories.CreateNetworkObjectLink(ctx, domain.NetworkObjectLink{ID: "replacement", LaboratoryID: lab.ID, ObjectAID: "delete-a", PortAName: "swp1", ObjectBID: "delete-b", PortBName: "swp1", Revision: 1, DesiredState: "connected", ObservedState: "pending"}); err != nil {
		t.Fatalf("released endpoints were not reusable: %v", err)
	}
	var taskID string
	var revision int
	if err = database.DB.QueryRowContext(ctx, `SELECT task_id,revision FROM outbox_events WHERE event_type='network_object_link.deleted' AND resource_id=?`, link.ID).Scan(&taskID, &revision); err != nil {
		t.Fatal(err)
	}
	if taskID != "delete-task" || revision != 2 {
		t.Fatalf("task=%q revision=%d", taskID, revision)
	}
}
