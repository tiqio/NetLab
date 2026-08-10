package command

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	apptask "github.com/netlab/netlab/internal/app/task"
	"github.com/netlab/netlab/internal/domain"
)

type topologyConnectionTaskStore struct {
	mu    sync.Mutex
	tasks map[domain.ID]domain.OperationTask
}

func newTopologyConnectionTaskStore() *topologyConnectionTaskStore {
	return &topologyConnectionTaskStore{tasks: map[domain.ID]domain.OperationTask{}}
}

func (s *topologyConnectionTaskStore) CreateTask(_ context.Context, value domain.OperationTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.tasks {
		if value.IdempotencyKey != "" && existing.Kind == value.Kind && existing.IdempotencyKey == value.IdempotencyKey {
			return errors.New("duplicate idempotency key")
		}
	}
	s.tasks[value.ID] = value
	return nil
}

func (s *topologyConnectionTaskStore) UpdateTask(_ context.Context, value domain.OperationTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks[value.ID] = value
	return nil
}

func (s *topologyConnectionTaskStore) GetTask(_ context.Context, id domain.ID) (domain.OperationTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.tasks[id]
	if !ok {
		return domain.OperationTask{}, errors.New("task not found")
	}
	return value, nil
}

func (s *topologyConnectionTaskStore) GetTaskByIdempotency(_ context.Context, kind, key string) (domain.OperationTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, value := range s.tasks {
		if value.Kind == kind && value.IdempotencyKey == key {
			return value, nil
		}
	}
	return domain.OperationTask{}, errors.New("task not found")
}

func waitForTopologyConnectionTask(t *testing.T, store *topologyConnectionTaskStore, id domain.ID, states ...domain.TaskState) domain.OperationTask {
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
	value, _ := store.GetTask(context.Background(), id)
	t.Fatalf("task %s did not reach %v; current=%s", id, states, value.State)
	return domain.OperationTask{}
}

func TestTopologyConnectionTaskLifecycleAndIdempotentResult(t *testing.T) {
	store := newTopologyConnectionTaskStore()
	runner := apptask.NewRunner(store, 1, 8)
	defer runner.Close()
	started := make(chan struct{})
	release := make(chan struct{})
	runner.Register(TopologyConnectionCreateTaskKind, func(ctx context.Context, value *domain.OperationTask) (map[string]any, error) {
		value.ProgressCurrent = 1
		if err := runner.Checkpoint(ctx, value); err != nil {
			return nil, err
		}
		close(started)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-release:
		}
		value.ProgressCurrent = value.ProgressTotal
		return map[string]any{"connection_id": value.ResourceID}, nil
	})

	operation := NewTopologyConnectionOperation(TopologyConnectionCreateTaskKind, "connection-1", "same-key", "fingerprint", map[string]any{"entry_point": "http"})
	queued, err := runner.EnqueueOrGet(context.Background(), operation)
	if err != nil || queued.State != domain.TaskQueued || queued.ProgressTotal != 2 {
		t.Fatalf("unexpected queued task: %+v err=%v", queued, err)
	}
	<-started
	running := waitForTopologyConnectionTask(t, store, operation.ID, domain.TaskRunning)
	if running.ProgressCurrent != 1 {
		t.Fatalf("expected progress checkpoint, got %+v", running)
	}
	replayed, err := runner.EnqueueOrGet(context.Background(), NewTopologyConnectionOperation(TopologyConnectionCreateTaskKind, "connection-2", "same-key", "fingerprint", nil))
	if err != nil || replayed.ID != operation.ID {
		t.Fatalf("idempotent replay returned %+v err=%v", replayed, err)
	}
	close(release)
	finished := waitForTopologyConnectionTask(t, store, operation.ID, domain.TaskSucceeded)
	if finished.ProgressCurrent != finished.ProgressTotal || finished.Result["connection_id"] != domain.ID("connection-1") {
		t.Fatalf("unexpected final task: %+v", finished)
	}
}

func TestTopologyConnectionTaskFailureAndCancellation(t *testing.T) {
	store := newTopologyConnectionTaskStore()
	runner := apptask.NewRunner(store, 1, 8)
	defer runner.Close()
	runner.Register(TopologyConnectionDeleteTaskKind, func(context.Context, *domain.OperationTask) (map[string]any, error) {
		return nil, domain.Problem{Code: "runtime_disconnect_failed", Message: "disconnect failed", Cleanup: "reserved endpoints retained for retry"}
	})
	failedOperation := NewTopologyConnectionOperation(TopologyConnectionDeleteTaskKind, "connection-failed", "failed-key", "failed-fingerprint", nil)
	if _, err := runner.EnqueueOrGet(context.Background(), failedOperation); err != nil {
		t.Fatal(err)
	}
	failed := waitForTopologyConnectionTask(t, store, failedOperation.ID, domain.TaskFailed)
	if failed.Error == nil || failed.Error.Code != "runtime_disconnect_failed" || failed.Error.Cleanup == "" {
		t.Fatalf("missing structured failure: %+v", failed)
	}

	started := make(chan struct{})
	runner.Register(TopologyConnectionCreateTaskKind, func(ctx context.Context, value *domain.OperationTask) (map[string]any, error) {
		close(started)
		<-ctx.Done()
		return nil, ctx.Err()
	})
	cancelOperation := NewTopologyConnectionOperation(TopologyConnectionCreateTaskKind, "connection-cancel", "cancel-key", "cancel-fingerprint", nil)
	if _, err := runner.EnqueueOrGet(context.Background(), cancelOperation); err != nil {
		t.Fatal(err)
	}
	<-started
	waitForTopologyConnectionTask(t, store, cancelOperation.ID, domain.TaskRunning)
	if err := runner.Cancel(context.Background(), cancelOperation.ID); err != nil {
		t.Fatal(err)
	}
	waitForTopologyConnectionTask(t, store, cancelOperation.ID, domain.TaskCancelling, domain.TaskCancelled)
	cancelled := waitForTopologyConnectionTask(t, store, cancelOperation.ID, domain.TaskCancelled)
	if cancelled.FinishedAt == nil {
		t.Fatalf("cancelled task missing final timestamp: %+v", cancelled)
	}
}
