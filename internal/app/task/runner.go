package task

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/netlab/netlab/internal/domain"
)

type Store interface {
	CreateTask(context.Context, domain.OperationTask) error
	UpdateTask(context.Context, domain.OperationTask) error
	GetTask(context.Context, domain.ID) (domain.OperationTask, error)
}

type RecoverableStore interface {
	ListRecoverableTasks(context.Context, int) ([]domain.OperationTask, error)
}

type IdempotentStore interface {
	GetTaskByIdempotency(context.Context, string, string) (domain.OperationTask, error)
}

type Handler func(context.Context, *domain.OperationTask) (map[string]any, error)

type Runner struct {
	store     Store
	queue     chan domain.ID
	handlers  map[string]Handler
	cancel    map[domain.ID]context.CancelFunc
	mu        sync.Mutex
	enqueueMu sync.Mutex
	closeOnce sync.Once
	wg        sync.WaitGroup
}

func NewRunner(store Store, workers, queueSize int) *Runner {
	if workers < 1 {
		workers = 1
	}
	r := &Runner{store: store, queue: make(chan domain.ID, queueSize), handlers: map[string]Handler{}, cancel: map[domain.ID]context.CancelFunc{}}
	for range workers {
		r.wg.Add(1)
		go r.worker()
	}
	return r
}

func (r *Runner) Register(kind string, handler Handler) { r.handlers[kind] = handler }

func (r *Runner) GetByIdempotency(ctx context.Context, kind, key string) (domain.OperationTask, error) {
	if key == "" {
		return domain.OperationTask{}, fmt.Errorf("idempotency key is required")
	}
	store, ok := r.store.(IdempotentStore)
	if !ok {
		return domain.OperationTask{}, fmt.Errorf("task store does not support idempotency lookup")
	}
	return store.GetTaskByIdempotency(ctx, kind, key)
}

func (r *Runner) Checkpoint(ctx context.Context, task *domain.OperationTask) error {
	if task == nil {
		return fmt.Errorf("task is required")
	}
	return r.store.UpdateTask(ctx, *task)
}

func (r *Runner) Enqueue(ctx context.Context, task domain.OperationTask) error {
	r.enqueueMu.Lock()
	defer r.enqueueMu.Unlock()
	if len(r.queue) >= cap(r.queue) {
		return fmt.Errorf("task queue full")
	}
	if task.ID == "" {
		task.ID = domain.NewID()
	}
	if task.CreatedAt.IsZero() {
		task.CreatedAt = time.Now().UTC()
	}
	task.State = domain.TaskQueued
	if err := r.store.CreateTask(ctx, task); err != nil {
		return err
	}
	select {
	case r.queue <- task.ID:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return fmt.Errorf("task queue full")
	}
}

func (r *Runner) EnqueueOrGet(ctx context.Context, value domain.OperationTask) (domain.OperationTask, error) {
	if value.IdempotencyKey != "" {
		if store, ok := r.store.(IdempotentStore); ok {
			existing, err := store.GetTaskByIdempotency(ctx, value.Kind, value.IdempotencyKey)
			if err == nil {
				if existing.RequestFingerprint != "" && value.RequestFingerprint != "" && existing.RequestFingerprint != value.RequestFingerprint {
					return domain.OperationTask{}, domain.Problem{Code: "idempotency_conflict", Message: "idempotency key was already used with a different request", ResourceType: existing.ResourceType, ResourceID: existing.ResourceID, TaskID: existing.ID, Phase: "task_admission", Cleanup: "original task remains unchanged", OperatorHint: "reuse the original request or choose a new idempotency key"}
				}
				return existing, nil
			}
		}
	}
	if err := r.Enqueue(ctx, value); err != nil {
		if value.IdempotencyKey != "" {
			if store, ok := r.store.(IdempotentStore); ok {
				existing, getErr := store.GetTaskByIdempotency(ctx, value.Kind, value.IdempotencyKey)
				if getErr == nil {
					if existing.RequestFingerprint != "" && value.RequestFingerprint != "" && existing.RequestFingerprint != value.RequestFingerprint {
						return domain.OperationTask{}, domain.Problem{Code: "idempotency_conflict", Message: "idempotency key was already used with a different request", ResourceType: existing.ResourceType, ResourceID: existing.ResourceID, TaskID: existing.ID, Phase: "task_admission", Cleanup: "original task remains unchanged", OperatorHint: "reuse the original request or choose a new idempotency key"}
					}
					return existing, nil
				}
			}
		}
		return domain.OperationTask{}, err
	}
	return value, nil
}

func (r *Runner) Cancel(ctx context.Context, id domain.ID) error {
	r.mu.Lock()
	cancel := r.cancel[id]
	r.mu.Unlock()
	if cancel != nil {
		task, err := r.store.GetTask(ctx, id)
		if err != nil {
			return err
		}
		if task.ProgressCurrent > 0 && task.Input["cancellation_mode"] == "before_commit" {
			return domain.Problem{Code: "task_not_cancellable", Message: "operation has passed its cancellation boundary", TaskID: task.ID, ResourceType: task.ResourceType, ResourceID: task.ResourceID, Phase: "task_cancellation", Cleanup: "the committed operation continues to convergence", OperatorHint: "monitor the existing task until it reaches a terminal state"}
		}
		task.State = domain.TaskCancelling
		if err = r.store.UpdateTask(ctx, task); err != nil {
			return err
		}
		cancel()
		return nil
	}
	task, err := r.store.GetTask(ctx, id)
	if err != nil {
		return err
	}
	if task.State == domain.TaskQueued || task.State == domain.TaskCancelling {
		task.State = domain.TaskCancelled
		now := time.Now().UTC()
		task.FinishedAt = &now
		return r.store.UpdateTask(ctx, task)
	}
	return nil
}

func (r *Runner) Recover(ctx context.Context) error {
	store, ok := r.store.(RecoverableStore)
	if !ok {
		return nil
	}
	values, err := store.ListRecoverableTasks(ctx, cap(r.queue))
	if err != nil {
		return err
	}
	for _, value := range values {
		if r.handlers[value.Kind] == nil {
			continue
		}
		if value.State != domain.TaskQueued {
			value.State = domain.TaskQueued
			value.StartedAt = nil
			value.FinishedAt = nil
			value.Error = nil
			if err = r.store.UpdateTask(ctx, value); err != nil {
				return err
			}
		}
		select {
		case r.queue <- value.ID:
		case <-ctx.Done():
			return ctx.Err()
		default:
			return fmt.Errorf("task recovery queue full")
		}
	}
	return nil
}

func (r *Runner) Close() {
	r.closeOnce.Do(func() {
		r.mu.Lock()
		cancels := make([]context.CancelFunc, 0, len(r.cancel))
		for _, cancel := range r.cancel {
			cancels = append(cancels, cancel)
		}
		r.mu.Unlock()
		for _, cancel := range cancels {
			cancel()
		}
		close(r.queue)
		r.wg.Wait()
	})
}

func (r *Runner) worker() {
	defer r.wg.Done()
	for id := range r.queue {
		task, err := r.store.GetTask(context.Background(), id)
		if err != nil || task.State == domain.TaskCancelled {
			continue
		}
		handler := r.handlers[task.Kind]
		if handler == nil {
			task.State = domain.TaskFailed
			task.Error = &domain.Problem{Code: "handler_missing", Message: "task handler not registered"}
			now := time.Now().UTC()
			task.FinishedAt = &now
			_ = r.store.UpdateTask(context.Background(), task)
			continue
		}
		ctx, cancel := context.WithCancel(context.Background())
		r.mu.Lock()
		r.cancel[id] = cancel
		r.mu.Unlock()
		now := time.Now().UTC()
		task.StartedAt = &now
		task.State = domain.TaskRunning
		_ = r.store.UpdateTask(ctx, task)
		result, runErr := handler(ctx, &task)
		finished := time.Now().UTC()
		task.FinishedAt = &finished
		if ctx.Err() != nil {
			task.State = domain.TaskCancelled
		} else if runErr != nil {
			task.State = domain.TaskFailed
			problem := domain.NormalizeProblem(runErr, domain.Problem{Code: "task_failed", TaskID: task.ID, ResourceType: task.ResourceType, ResourceID: task.ResourceID, Phase: "task_execution", Cleanup: "handler-specific cleanup status unavailable", OperatorHint: "inspect the task error and retry only when marked retryable"})
			task.Error = &problem
		} else {
			task.State = domain.TaskSucceeded
			task.Result = result
		}
		_ = r.store.UpdateTask(context.Background(), task)
		cancel()
		r.mu.Lock()
		delete(r.cancel, id)
		r.mu.Unlock()
	}
}
