package reconcile

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/app/task"
	"github.com/netlab/netlab/internal/domain"
	captureRuntime "github.com/netlab/netlab/internal/runtime/capture"
	storesqlite "github.com/netlab/netlab/internal/store/sqlite"
)

func installFakeDumpcap(t *testing.T) {
	t.Helper()
	directory := t.TempDir()
	path := filepath.Join(directory, "dumpcap")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nprintf capture-data\nexec sleep 5\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", directory+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestCaptureTaskIdempotencyStopAndRecovery(t *testing.T) {
	installFakeDumpcap(t)
	ctx := context.Background()
	database, err := storesqlite.Open(ctx, "file:capture-task-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	repositories := storesqlite.NewRepositories(database)
	directory := t.TempDir()
	manager := NewCaptureManager(directory, 2, 1<<20, time.Hour)
	filters := NewTrafficFilterManager(manager)
	runner := task.NewRunner(repositories, 1, 8)
	service := NewCaptureTaskService(manager, filters, runner)
	request := CaptureRequest{LaboratoryID: "lab", SourceType: "interface", SourceID: "iface", Interface: "fake0", Format: "pcap", MaxBytes: 1 << 20, Duration: 5 * time.Second}

	first, firstTask, err := service.StartCapture(ctx, request, "capture-key")
	if err != nil {
		t.Fatal(err)
	}
	second, secondTask, err := service.StartCapture(ctx, request, "capture-key")
	if err != nil || secondTask.ID != firstTask.ID || second.ID != first.ID {
		t.Fatalf("first=%+v/%+v second=%+v/%+v err=%v", first, firstTask, second, secondTask, err)
	}
	conflict := request
	conflict.Filter = "icmp"
	if _, _, err = service.StartCapture(ctx, conflict, "capture-key"); err == nil {
		t.Fatal("expected idempotency conflict")
	}
	waitForNetworkTask(t, repositories, firstTask.ID, func(value domain.OperationTask) bool { return value.State == domain.TaskSucceeded })
	stopTask, err := service.StopCapture(ctx, first.ID, "capture-stop")
	if err != nil {
		t.Fatal(err)
	}
	waitForNetworkTask(t, repositories, stopTask.ID, func(value domain.OperationTask) bool { return value.State == domain.TaskSucceeded })
	runner.Close()

	current, err := repositories.GetTask(ctx, firstTask.ID)
	if err != nil {
		t.Fatal(err)
	}
	current.State = domain.TaskRunning
	current.FinishedAt = nil
	current.Result = nil
	if err = repositories.UpdateTask(ctx, current); err != nil {
		t.Fatal(err)
	}
	recoveredManager := NewCaptureManager(directory, 2, 1<<20, time.Hour)
	recoveredFilters := NewTrafficFilterManager(recoveredManager)
	recoveryRunner := task.NewRunner(repositories, 1, 8)
	defer recoveryRunner.Close()
	NewCaptureTaskService(recoveredManager, recoveredFilters, recoveryRunner)
	if err = recoveryRunner.Recover(ctx); err != nil {
		t.Fatal(err)
	}
	waitForNetworkTask(t, repositories, firstTask.ID, func(value domain.OperationTask) bool { return value.State == domain.TaskSucceeded })
	recovered, err := recoveredManager.Get(first.ID)
	if err != nil || recovered.ID != first.ID || recovered.State != "running" {
		t.Fatalf("capture=%+v err=%v", recovered, err)
	}
	_, _ = recoveredManager.Stop(first.ID)
}

func TestCaptureTaskPreservesNetworkObjectLinkSource(t *testing.T) {
	installFakeNamespacedDumpcap(t)
	ctx := context.Background()
	database, err := storesqlite.Open(ctx, "file:capture-object-link-task?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	repositories := storesqlite.NewRepositories(database)
	runner := task.NewRunner(repositories, 1, 8)
	defer runner.Close()
	object := domain.NetworkObject{ID: "object-a", Kind: domain.NetworkSwitchL3}
	manager := NewCaptureManager(t.TempDir(), 1, 1<<20, time.Hour)
	manager.SetNetworkObjectRepository(captureNetworkObjectRepository{
		link:   domain.NetworkObjectLink{ID: "object-link", LaboratoryID: "lab", ObjectAID: object.ID, PortAName: "swp1", ObjectBID: "object-b", PortBName: "swp2"},
		object: object,
	})
	service := NewCaptureTaskService(manager, NewTrafficFilterManager(manager), runner)
	value, operation, err := service.StartCapture(ctx, CaptureRequest{
		LaboratoryID: "lab", SourceType: "network_object_link", SourceID: "object-link", Format: "pcap", MaxBytes: 1 << 20, Duration: time.Minute,
	}, "object-link-capture")
	if err != nil {
		t.Fatal(err)
	}
	completed := waitForNetworkTask(t, repositories, operation.ID, func(value domain.OperationTask) bool { return value.State == domain.TaskSucceeded })
	if completed.ResourceType != "capture" || completed.ResourceID != value.ID {
		t.Fatalf("capture=%+v task=%+v", value, completed)
	}
	request, err := manager.Request(value.ID)
	if err != nil || request.SourceType != "network_object_link" || request.SourceID != "object-link" || request.Interface != "swp1" || request.Namespace == "" {
		t.Fatalf("request=%+v err=%v", request, err)
	}
	_, _ = manager.Stop(value.ID)
}

func TestTrafficFilterTaskUsesDurableEnvelope(t *testing.T) {
	ctx := context.Background()
	database, err := storesqlite.Open(ctx, "file:filter-task-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	repositories := storesqlite.NewRepositories(database)
	runner := task.NewRunner(repositories, 1, 8)
	defer runner.Close()
	manager := NewCaptureManager(t.TempDir(), 2, 128<<20, time.Hour)
	filters := NewTrafficFilterManager(manager)
	service := NewCaptureTaskService(manager, filters, runner)
	filter, value, err := service.StartFilter(ctx, "lab", structToMatch(t, map[string]any{"protocol": "icmp"}), 10, []domain.ID{"iface"}, nil, "#f59e0b", "filter-key")
	if err != nil {
		t.Fatal(err)
	}
	current := waitForNetworkTask(t, repositories, value.ID, func(value domain.OperationTask) bool { return value.State == domain.TaskSucceeded })
	if current.ResourceID != filter.ID || current.Kind != "traffic_filter.start" {
		t.Fatalf("filter=%+v task=%+v", filter, current)
	}
	stop, err := service.StopFilter(ctx, filter.ID, "filter-stop")
	if err != nil {
		t.Fatal(err)
	}
	waitForNetworkTask(t, repositories, stop.ID, func(value domain.OperationTask) bool { return value.State == domain.TaskSucceeded })
}

func structToMatch(t *testing.T, value map[string]any) captureRuntime.Match {
	t.Helper()
	body, _ := json.Marshal(value)
	var match captureRuntime.Match
	if err := json.Unmarshal(body, &match); err != nil {
		t.Fatal(err)
	}
	return match
}
