package reconcile

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/domain"
	captureRuntime "github.com/netlab/netlab/internal/runtime/capture"
	"github.com/netlab/netlab/internal/runtime/linuxnet"
)

type captureNetworkObjectRepository struct {
	link   domain.NetworkObjectLink
	object domain.NetworkObject
}

func (r captureNetworkObjectRepository) GetNetworkObjectLink(context.Context, domain.ID) (domain.NetworkObjectLink, error) {
	return r.link, nil
}

func (r captureNetworkObjectRepository) GetNetworkObject(context.Context, domain.ID) (domain.NetworkObject, error) {
	return r.object, nil
}

func TestCaptureManagerResolvesNetworkObjectLinkEndpointA(t *testing.T) {
	object := domain.NetworkObject{ID: "object-a", Kind: domain.NetworkSwitchL3}
	manager := NewCaptureManager(t.TempDir(), 1, 1<<20, time.Hour)
	manager.SetNetworkObjectRepository(captureNetworkObjectRepository{
		link:   domain.NetworkObjectLink{ID: "object-link", ObjectAID: object.ID, PortAName: "swp1", ObjectBID: "object-b", PortBName: "swp9"},
		object: object,
	})
	interfaceName, namespace, err := manager.captureLocator(context.Background(), "network_object_link", "object-link")
	if err != nil {
		t.Fatal(err)
	}
	wantNamespace, err := linuxnet.NetworkObjectNamespaceName(object)
	if err != nil {
		t.Fatal(err)
	}
	if interfaceName != "swp1" || namespace != wantNamespace {
		t.Fatalf("locator=%s/%s want=swp1/%s", interfaceName, namespace, wantNamespace)
	}
}

func TestNetworkObjectLinkCapturePersistsMetadataAndStreamsUntilRestart(t *testing.T) {
	installFakeNamespacedDumpcap(t)
	stateDir := t.TempDir()
	object := domain.NetworkObject{ID: "object-a", Kind: domain.NetworkSwitchL3}
	repository := captureNetworkObjectRepository{
		link:   domain.NetworkObjectLink{ID: "object-link", LaboratoryID: "lab", ObjectAID: object.ID, PortAName: "swp1", ObjectBID: "object-b", PortBName: "swp9"},
		object: object,
	}
	manager := NewCaptureManager(stateDir, 1, 1<<20, time.Hour)
	manager.SetNetworkObjectRepository(repository)
	value, err := manager.StartAs(context.Background(), "capture-object-link", CaptureRequest{
		LaboratoryID: "lab", SourceType: "network_object_link", SourceID: "object-link", Format: "pcap", Retain: true, MaxBytes: 1 << 20, Duration: time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	stream, cancel, err := manager.Subscribe(value.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()
	select {
	case chunk := <-stream:
		if string(chunk) != "capture-data" {
			t.Fatalf("stream=%q", chunk)
		}
	case <-time.After(time.Second):
		t.Fatal("object-link capture did not expose a live stream")
	}
	request, err := manager.Request(value.ID)
	if err != nil {
		t.Fatal(err)
	}
	wantNamespace, _ := linuxnet.NetworkObjectNamespaceName(object)
	if request.Interface != "swp1" || request.Namespace != wantNamespace || !value.Retain || value.SourceType != "network_object_link" || value.SourceID != "object-link" {
		t.Fatalf("capture=%+v request=%+v", value, request)
	}
	if _, err = manager.Stop(value.ID); err != nil {
		t.Fatal(err)
	}
	waitForCaptureState(t, manager, value.ID, "cancelled")

	restarted := NewCaptureManager(stateDir, 1, 1<<20, time.Hour)
	recovered, err := restarted.Get(value.ID)
	if err != nil || recovered.SourceType != "network_object_link" || recovered.SourceID != "object-link" || !recovered.Retain {
		t.Fatalf("recovered=%+v err=%v", recovered, err)
	}
	request, err = restarted.Request(value.ID)
	if err != nil || request.Interface != "swp1" || request.Namespace != wantNamespace {
		t.Fatalf("recovered request=%+v err=%v", request, err)
	}
	if _, _, err = restarted.Subscribe(value.ID); err == nil {
		t.Fatal("restarted completed capture unexpectedly exposed a live stream")
	}
}

func installFakeNamespacedDumpcap(t *testing.T) {
	t.Helper()
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "dumpcap"), []byte("#!/bin/sh\nprintf capture-data\nexec sleep 30\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "ip"), []byte("#!/bin/sh\nif [ \"$1\" = netns ] && [ \"$2\" = exec ]; then shift 3; exec \"$@\"; fi\nexit 2\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", directory+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func waitForCaptureState(t *testing.T, manager *CaptureManager, id domain.ID, state string) domain.Capture {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		value, err := manager.Get(id)
		if err == nil && value.State == state {
			return value
		}
		time.Sleep(10 * time.Millisecond)
	}
	value, err := manager.Get(id)
	t.Fatalf("capture=%+v err=%v want state=%s", value, err, state)
	return domain.Capture{}
}

func TestCaptureStopNetworkObjectLinkUsesLinkDeletedReason(t *testing.T) {
	manager := NewCaptureManager(t.TempDir(), 1, 1<<20, time.Hour)
	worker, err := captureRuntime.NewWorker(captureRuntime.WorkerConfig{Interface: "swp1", MaxBytes: 1 << 20, Format: "pcap"})
	if err != nil {
		t.Fatal(err)
	}
	reader, writer := io.Pipe()
	worker.StartReader(context.Background(), reader)
	manager.values["capture-link"] = &managedCapture{metadata: domain.Capture{ID: "capture-link", SourceType: "network_object_link", SourceID: "object-link", State: "running"}, worker: worker}
	manager.StopNetworkObjectLink("object-link")
	_ = writer.Close()
	select {
	case <-worker.Done():
	case <-time.After(time.Second):
		t.Fatal("capture did not stop")
	}
	manager.watch("capture-link", manager.values["capture-link"])
	value, err := manager.Get("capture-link")
	if err != nil || value.CompletionReason != "link_deleted" {
		t.Fatalf("capture=%+v err=%v", value, err)
	}
}

func TestCaptureManagerRecoversAbandonedCapture(t *testing.T) {
	stateDir := t.TempDir()
	directory := filepath.Join(stateDir, "captures")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	records := []captureRecord{{Metadata: domain.Capture{ID: "capture-1", State: "running", CreatedAt: now, StartedAt: &now}}}
	body, _ := json.Marshal(records)
	if err := os.WriteFile(filepath.Join(directory, "index.json"), body, 0o600); err != nil {
		t.Fatal(err)
	}
	manager := NewCaptureManager(stateDir, 1, 1024, time.Hour)
	value, err := manager.Get("capture-1")
	if err != nil {
		t.Fatal(err)
	}
	if value.State != "failed" || value.CompletionReason != "service_restart" || value.LastError == nil {
		t.Fatalf("unexpected recovered capture: %+v", value)
	}
	if value.LastError.ResourceType != "capture" || value.LastError.ResourceID != value.ID || value.LastError.Phase != "recovery" || value.LastError.Cleanup == "" || value.LastError.OperatorHint == "" || value.LastError.RetryAfterSeconds == 0 {
		t.Fatalf("unstructured capture recovery error: %+v", value.LastError)
	}
}

func TestCaptureRecoveryCheckpointRequiresStreamReconnect(t *testing.T) {
	stateDir := t.TempDir()
	directory := filepath.Join(stateDir, "captures")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	records := []captureRecord{{Metadata: domain.Capture{ID: "capture-1", State: "running", CreatedAt: now, StartedAt: &now}}}
	body, _ := json.Marshal(records)
	if err := os.WriteFile(filepath.Join(directory, "index.json"), body, 0o600); err != nil {
		t.Fatal(err)
	}
	manager := NewCaptureManager(stateDir, 1, 1024, time.Hour)
	var outcomes []RecoveryResourceOutcome
	if err := NewCaptureRecoveryReconciler(manager).ReconcileWithCheckpoints(context.Background(), func(outcome RecoveryResourceOutcome) error { outcomes = append(outcomes, outcome); return nil }); err != nil {
		t.Fatal(err)
	}
	if len(outcomes) != 1 || outcomes[0].State != "reconnect_required" || outcomes[0].RuntimeID != "capture-1" {
		t.Fatalf("outcomes=%+v", outcomes)
	}
}

func TestCaptureInterfaceIsDerivedFromSource(t *testing.T) {
	interfaceName, err := captureInterface("interface", "interface-1")
	if err != nil || interfaceName == "" {
		t.Fatalf("interface=%q err=%v", interfaceName, err)
	}
	linkName, err := captureInterface("link", "link-1")
	if err != nil || linkName == "" || linkName == interfaceName {
		t.Fatalf("link=%q err=%v", linkName, err)
	}
	if _, err = captureInterface("node", "node-1"); err == nil {
		t.Fatal("expected invalid source type error")
	}
}

func TestCaptureMetadataPreservesInternalOwnership(t *testing.T) {
	installFakeDumpcap(t)
	manager := NewCaptureManager(t.TempDir(), 2, 128<<20, time.Hour)
	value, err := manager.Start(context.Background(), CaptureRequest{
		LaboratoryID: "lab",
		SourceType:   "interface",
		SourceID:     "interface-1",
		Purpose:      "traffic_filter",
		ParentID:     "filter-1",
		Format:       "pcap",
		MaxBytes:     64 << 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	if value.Purpose != "traffic_filter" || value.ParentResourceID != "filter-1" {
		t.Fatalf("capture=%+v", value)
	}
	_, _ = manager.Stop(value.ID)
}

func TestCaptureBudgetIgnoresFinishedNonRetainedData(t *testing.T) {
	manager := NewCaptureManager(t.TempDir(), 2, 64<<20, time.Hour)
	manager.values["finished"] = &managedCapture{metadata: domain.Capture{
		ID:           "finished",
		State:        "cancelled",
		Retain:       false,
		MaxBytes:     64 << 20,
		BytesWritten: 936,
	}}
	if available := manager.AvailableBytes(); available != 64<<20 {
		t.Fatalf("available=%d, want %d", available, 64<<20)
	}
	manager.values["active"] = &managedCapture{metadata: domain.Capture{
		ID:           "active",
		State:        "running",
		Retain:       false,
		MaxBytes:     16 << 20,
		BytesWritten: 1024,
	}}
	if available := manager.AvailableBytes(); available != 48<<20 {
		t.Fatalf("available=%d, want %d", available, 48<<20)
	}
}
