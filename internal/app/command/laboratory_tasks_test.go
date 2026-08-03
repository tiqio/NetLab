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

type laboratoryTaskRepositoryFake struct {
	mu   sync.Mutex
	labs map[domain.ID]domain.Laboratory
}

func (r *laboratoryTaskRepositoryFake) GetLaboratory(_ context.Context, id domain.ID) (domain.Laboratory, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	value, ok := r.labs[id]
	if !ok {
		return domain.Laboratory{}, errors.New("laboratory not found")
	}
	return value, nil
}

func (r *laboratoryTaskRepositoryFake) MarkLaboratoryDeleting(_ context.Context, id domain.ID, revision domain.Revision) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	value, ok := r.labs[id]
	if !ok {
		return errors.New("laboratory not found")
	}
	if value.Revision != revision {
		return errors.New("revision conflict")
	}
	value.Revision++
	value.LifecycleState = "deleting"
	r.labs[id] = value
	return nil
}

func (r *laboratoryTaskRepositoryFake) remove(id domain.ID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.labs, id)
}

func TestLaboratoryDeleteTaskIdempotencyBoundaryAndConvergence(t *testing.T) {
	store := newTopologyTaskStore()
	repository := &laboratoryTaskRepositoryFake{labs: map[domain.ID]domain.Laboratory{"lab": {ID: "lab", Revision: 1, LifecycleState: "active"}}}
	runner := task.NewRunner(store, 1, 8)
	defer runner.Close()
	service := NewLaboratoryTaskService(repository, runner)
	service.poll = 5 * time.Millisecond
	service.timeout = time.Second
	first, err := service.Delete(context.Background(), "lab", 1, "delete-key")
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Delete(context.Background(), "lab", 1, "delete-key")
	if err != nil || second.ID != first.ID {
		t.Fatalf("first=%+v second=%+v err=%v", first, second, err)
	}
	if _, err = service.Delete(context.Background(), "lab", 2, "delete-key"); err == nil {
		t.Fatal("expected idempotency conflict")
	}
	waitForTopologyTask(t, store, first.ID, func(value domain.OperationTask) bool {
		return value.State == domain.TaskRunning && value.ProgressCurrent == 2
	})
	if err = runner.Cancel(context.Background(), first.ID); err == nil {
		t.Fatal("expected committed deletion cancellation rejection")
	} else if problem, ok := domain.ProblemFromError(err); !ok || problem.Code != "task_not_cancellable" {
		t.Fatalf("problem=%+v ok=%t", problem, ok)
	}
	repository.remove("lab")
	waitForTopologyTask(t, store, first.ID, func(value domain.OperationTask) bool { return value.State == domain.TaskSucceeded })
}

func TestLaboratoryDeleteTaskRecoveryResumesOriginalTask(t *testing.T) {
	store := newTopologyTaskStore()
	repository := &laboratoryTaskRepositoryFake{labs: map[domain.ID]domain.Laboratory{"lab": {ID: "lab", Revision: 2, LifecycleState: "deleting"}}}
	value := domain.OperationTask{ID: "delete-task", Kind: "laboratory.delete", ResourceType: "laboratory", ResourceID: "lab", State: domain.TaskRunning, ProgressCurrent: 2, ProgressTotal: 3, Input: map[string]any{"revision": int64(1), "cancellation_mode": "before_commit"}, CreatedAt: time.Now().UTC()}
	store.tasks[value.ID] = value
	runner := task.NewRunner(store, 1, 8)
	defer runner.Close()
	service := NewLaboratoryTaskService(repository, runner)
	service.poll = 5 * time.Millisecond
	service.timeout = time.Second
	if err := runner.Recover(context.Background()); err != nil {
		t.Fatal(err)
	}
	waitForTopologyTask(t, store, value.ID, func(value domain.OperationTask) bool { return value.State == domain.TaskRunning })
	repository.remove("lab")
	current := waitForTopologyTask(t, store, value.ID, func(value domain.OperationTask) bool { return value.State == domain.TaskSucceeded })
	if current.ID != value.ID || current.ResourceID != value.ResourceID {
		t.Fatalf("task identity changed: %+v", current)
	}
}
