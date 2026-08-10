package task

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/domain"
)

type memoryStore struct {
	mu    sync.Mutex
	tasks map[domain.ID]domain.OperationTask
}

func newMemoryStore() *memoryStore { return &memoryStore{tasks: map[domain.ID]domain.OperationTask{}} }
func (s *memoryStore) CreateTask(_ context.Context, task domain.OperationTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tasks[task.ID]; ok {
		return errors.New("duplicate")
	}
	s.tasks[task.ID] = task
	return nil
}
func (s *memoryStore) UpdateTask(_ context.Context, task domain.OperationTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks[task.ID] = task
	return nil
}
func (s *memoryStore) GetTask(_ context.Context, id domain.ID) (domain.OperationTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	task, ok := s.tasks[id]
	if !ok {
		return task, errors.New("missing")
	}
	return task, nil
}

func TestRunnerSuccessAndCancellation(t *testing.T) {
	store := newMemoryStore()
	runner := NewRunner(store, 1, 4)
	defer runner.Close()
	runner.Register("ok", func(context.Context, *domain.OperationTask) (map[string]any, error) {
		return map[string]any{"ok": true}, nil
	})
	id := domain.NewID()
	if err := runner.Enqueue(context.Background(), domain.OperationTask{ID: id, Kind: "ok"}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		task, _ := store.GetTask(context.Background(), id)
		if task.State == domain.TaskSucceeded {
			return
		}
		time.Sleep(time.Millisecond * 5)
	}
	t.Fatal("task did not finish")
}

func TestRunnerPreservesStructuredProblem(t *testing.T) {
	store := newMemoryStore()
	runner := NewRunner(store, 1, 4)
	defer runner.Close()
	runner.Register("structured", func(context.Context, *domain.OperationTask) (map[string]any, error) {
		return nil, fmt.Errorf("wrapped: %w", &domain.Problem{Code: "resource_exhausted", Message: "capacity reached", Phase: "start", Cleanup: "complete", OperatorHint: "stop another node"})
	})
	id := domain.NewID()
	resourceID := domain.NewID()
	if err := runner.Enqueue(context.Background(), domain.OperationTask{ID: id, Kind: "structured", ResourceType: "node", ResourceID: resourceID}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		value, _ := store.GetTask(context.Background(), id)
		if value.State == domain.TaskFailed {
			if value.Error == nil || value.Error.Code != "resource_exhausted" || value.Error.TaskID != id || value.Error.ResourceID != resourceID || value.Error.Phase != "start" {
				t.Fatalf("structured problem was not preserved: %+v", value.Error)
			}
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("task did not fail")
}

func TestRunnerCancellationPersistsCancellingState(t *testing.T) {
	store := newMemoryStore()
	runner := NewRunner(store, 1, 4)
	defer runner.Close()
	started := make(chan struct{})
	release := make(chan struct{})
	runner.Register("blocking", func(ctx context.Context, _ *domain.OperationTask) (map[string]any, error) {
		close(started)
		<-ctx.Done()
		close(release)
		return nil, ctx.Err()
	})
	id := domain.NewID()
	if err := runner.Enqueue(context.Background(), domain.OperationTask{ID: id, Kind: "blocking"}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("task did not start")
	}
	if err := runner.Cancel(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	value, _ := store.GetTask(context.Background(), id)
	if value.State != domain.TaskCancelling && value.State != domain.TaskCancelled {
		t.Fatalf("unexpected cancellation state: %s", value.State)
	}
	select {
	case <-release:
	case <-time.After(time.Second):
		t.Fatal("handler was not cancelled")
	}
}

func TestRunnerCloseCancelsActiveHandlers(t *testing.T) {
	store := newMemoryStore()
	runner := NewRunner(store, 1, 4)
	started := make(chan struct{})
	finished := make(chan struct{})
	runner.Register("close-blocking", func(ctx context.Context, _ *domain.OperationTask) (map[string]any, error) {
		close(started)
		<-ctx.Done()
		close(finished)
		return nil, ctx.Err()
	})
	id := domain.NewID()
	if err := runner.Enqueue(context.Background(), domain.OperationTask{ID: id, Kind: "close-blocking"}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("task did not start")
	}
	closed := make(chan struct{})
	go func() {
		runner.Close()
		close(closed)
	}()
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("close did not cancel active handler")
	}
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("close did not wait for worker shutdown")
	}
	runner.Close()
}
