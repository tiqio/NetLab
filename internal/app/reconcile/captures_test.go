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
)

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
