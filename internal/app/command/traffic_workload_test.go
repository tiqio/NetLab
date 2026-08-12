package command

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/app/task"
	"github.com/netlab/netlab/internal/domain"
)

type trafficWorkloadRepositoryFake struct {
	mu        sync.Mutex
	workloads map[domain.ID]domain.TrafficWorkload
	block     chan struct{}
	started   chan struct{}
}

func newTrafficWorkloadRepositoryFake() *trafficWorkloadRepositoryFake {
	return &trafficWorkloadRepositoryFake{workloads: map[domain.ID]domain.TrafficWorkload{}}
}

func (r *trafficWorkloadRepositoryFake) CreateTrafficWorkload(ctx context.Context, value domain.TrafficWorkload) error {
	if r.started != nil {
		close(r.started)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-r.block:
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.workloads[value.ID]; ok {
		return errors.New("duplicate workload")
	}
	r.workloads[value.ID] = value
	return nil
}

func (r *trafficWorkloadRepositoryFake) GetTrafficWorkload(_ context.Context, id domain.ID) (domain.TrafficWorkload, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	value, ok := r.workloads[id]
	if !ok {
		return value, domain.ErrNotFound
	}
	return value, nil
}

func (r *trafficWorkloadRepositoryFake) ListTrafficWorkloads(context.Context, domain.ID) ([]domain.TrafficWorkload, error) {
	return nil, nil
}

func (r *trafficWorkloadRepositoryFake) UpdateTrafficWorkloadState(_ context.Context, id domain.ID, expected domain.Revision, desired, observed string, _ *domain.Problem, _ domain.ID) (domain.TrafficWorkload, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	value, ok := r.workloads[id]
	if !ok {
		return value, domain.ErrNotFound
	}
	if value.Revision != expected {
		return value, workloadRevisionConflict(id)
	}
	value.Revision++
	value.DesiredState = desired
	value.ObservedState = observed
	r.workloads[id] = value
	return value, nil
}

func (r *trafficWorkloadRepositoryFake) DeleteTrafficWorkload(_ context.Context, id domain.ID, expected domain.Revision, _ domain.ID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	value, ok := r.workloads[id]
	if !ok {
		return domain.ErrNotFound
	}
	if value.Revision != expected {
		return workloadRevisionConflict(id)
	}
	delete(r.workloads, id)
	return nil
}

func testTrafficWorkload(id domain.ID) domain.TrafficWorkload {
	return domain.TrafficWorkload{ID: id, LaboratoryID: "lab", Name: "steady ping", Revision: 1, Source: domain.TrafficWorkloadEndpoint{Kind: "node", ResourceID: "node"}, Protocol: "icmp", AddressFamily: "ipv4", Destination: domain.TrafficWorkloadDestination{Address: "192.0.2.1"}, IntervalSeconds: 5, TimeoutSeconds: 2, DesiredState: "stopped", ObservedState: "stopped"}
}

func waitTrafficTask(t *testing.T, store *topologyTaskStore, id domain.ID, states ...domain.TaskState) domain.OperationTask {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		value, err := store.GetTask(context.Background(), id)
		if err == nil {
			for _, state := range states {
				if value.State == state {
					return value
				}
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("task %s did not reach %v", id, states)
	return domain.OperationTask{}
}

func TestTrafficWorkloadCreateIsDurableAndIdempotent(t *testing.T) {
	repository := newTrafficWorkloadRepositoryFake()
	store := newTopologyTaskStore()
	runner := task.NewRunner(store, 1, 8)
	defer runner.Close()
	service := NewTrafficWorkloadService(repository, runner)

	workload := testTrafficWorkload("workload")
	created, err := service.Create(context.Background(), workload, "create-key")
	if err != nil {
		t.Fatal(err)
	}
	finished := waitTrafficTask(t, store, created.ID, domain.TaskSucceeded)
	if finished.Result["workload_id"] != workload.ID {
		t.Fatalf("unexpected task result: %+v", finished)
	}
	replayed, err := service.Create(context.Background(), workload, "create-key")
	if err != nil || replayed.ID != created.ID {
		t.Fatalf("replay=%+v err=%v", replayed, err)
	}
	conflicting := workload
	conflicting.Name = "different"
	if _, err = service.Create(context.Background(), conflicting, "create-key"); err == nil {
		t.Fatal("expected idempotency conflict")
	}
}

func TestTrafficWorkloadCreateReplayWithoutClientSuppliedID(t *testing.T) {
	repository := newTrafficWorkloadRepositoryFake()
	store := newTopologyTaskStore()
	runner := task.NewRunner(store, 1, 8)
	defer runner.Close()
	service := NewTrafficWorkloadService(repository, runner)
	workload := testTrafficWorkload("")
	first, err := service.Create(context.Background(), workload, "generated-id-key")
	if err != nil {
		t.Fatal(err)
	}
	waitTrafficTask(t, store, first.ID, domain.TaskSucceeded)
	second, err := service.Create(context.Background(), workload, "generated-id-key")
	if err != nil || second.ID != first.ID || second.ResourceID != first.ResourceID {
		t.Fatalf("first=%+v second=%+v err=%v", first, second, err)
	}
}

func TestTrafficWorkloadStateDeleteAndRevisionConflicts(t *testing.T) {
	repository := newTrafficWorkloadRepositoryFake()
	repository.workloads["workload"] = testTrafficWorkload("workload")
	store := newTopologyTaskStore()
	runner := task.NewRunner(store, 1, 8)
	defer runner.Close()
	service := NewTrafficWorkloadService(repository, runner)

	started, err := service.Start(context.Background(), "workload", 1, "start-key")
	if err != nil {
		t.Fatal(err)
	}
	waitTrafficTask(t, store, started.ID, domain.TaskSucceeded)
	if _, err = service.Stop(context.Background(), "workload", 1, "stale-key"); err == nil {
		t.Fatal("expected stale revision rejection")
	}
	stopped, err := service.Stop(context.Background(), "workload", 2, "stop-key")
	if err != nil {
		t.Fatal(err)
	}
	waitTrafficTask(t, store, stopped.ID, domain.TaskSucceeded)
	deleted, err := service.Delete(context.Background(), "workload", 3, "delete-key")
	if err != nil {
		t.Fatal(err)
	}
	waitTrafficTask(t, store, deleted.ID, domain.TaskSucceeded)
	if _, err = repository.GetTrafficWorkload(context.Background(), "workload"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("workload not deleted: %v", err)
	}
}

func TestTrafficWorkloadCancellationBeforeCommit(t *testing.T) {
	repository := newTrafficWorkloadRepositoryFake()
	store := newTopologyTaskStore()
	runner := task.NewRunner(store, 1, 8)
	defer runner.Close()
	service := NewTrafficWorkloadService(repository, runner)
	blockStarted := make(chan struct{})
	blockRelease := make(chan struct{})
	runner.Register("test.block", func(context.Context, *domain.OperationTask) (map[string]any, error) {
		close(blockStarted)
		<-blockRelease
		return nil, nil
	})
	blocker := domain.OperationTask{ID: domain.NewID(), Kind: "test.block", ResourceType: "test", ResourceID: "block", ProgressTotal: 1}
	if err := runner.Enqueue(context.Background(), blocker); err != nil {
		t.Fatal(err)
	}
	<-blockStarted

	operation, err := service.Create(context.Background(), testTrafficWorkload("cancelled"), "cancel-key")
	if err != nil {
		t.Fatal(err)
	}
	if err = service.Cancel(context.Background(), operation.ID); err != nil {
		t.Fatal(err)
	}
	close(blockRelease)
	waitTrafficTask(t, store, operation.ID, domain.TaskCancelled)
	if _, err = repository.GetTrafficWorkload(context.Background(), "cancelled"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("cancelled create persisted workload: %v", err)
	}
}

func TestTrafficWorkloadRecoveryTreatsCommittedMutationAsSuccess(t *testing.T) {
	repository := newTrafficWorkloadRepositoryFake()
	workload := testTrafficWorkload("recovered")
	workload.Revision = 2
	workload.DesiredState = "running"
	workload.ObservedState = "queued"
	repository.workloads[workload.ID] = workload
	store := newTopologyTaskStore()
	operation := newTrafficWorkloadOperation(TrafficWorkloadStartTaskKind, workload.ID, 1, "recover-key", "recover-fingerprint", map[string]any{"desired_state": "running", "cancellation_mode": "before_commit"})
	operation.State = domain.TaskRunning
	store.tasks[operation.ID] = operation
	runner := task.NewRunner(store, 1, 8)
	defer runner.Close()
	NewTrafficWorkloadService(repository, runner)
	if err := runner.Recover(context.Background()); err != nil {
		t.Fatal(err)
	}
	waitTrafficTask(t, store, operation.ID, domain.TaskSucceeded)
	got, _ := repository.GetTrafficWorkload(context.Background(), workload.ID)
	if got.Revision != 2 {
		t.Fatalf("recovery repeated mutation: %+v", got)
	}
}
