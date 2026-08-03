package command

import (
	"context"
	"strings"
	"time"

	"github.com/netlab/netlab/internal/app/task"
	"github.com/netlab/netlab/internal/domain"
)

type LaboratoryTaskRepository interface {
	GetLaboratory(context.Context, domain.ID) (domain.Laboratory, error)
	MarkLaboratoryDeleting(context.Context, domain.ID, domain.Revision) error
}

type LaboratoryTaskService struct {
	repository LaboratoryTaskRepository
	runner     *task.Runner
	poll       time.Duration
	timeout    time.Duration
}

func NewLaboratoryTaskService(repository LaboratoryTaskRepository, runner *task.Runner) *LaboratoryTaskService {
	service := &LaboratoryTaskService{repository: repository, runner: runner, poll: 250 * time.Millisecond, timeout: 10 * time.Minute}
	runner.Register("laboratory.delete", service.handleDelete)
	return service
}

func (s *LaboratoryTaskService) Delete(ctx context.Context, id domain.ID, revision domain.Revision, idempotencyKey string) (domain.OperationTask, error) {
	if _, err := s.repository.GetLaboratory(ctx, id); err != nil {
		return domain.OperationTask{}, err
	}
	input := map[string]any{"revision": int64(revision), "cancellation_mode": "before_commit"}
	value := automationOperation("laboratory.delete", "laboratory", id, idempotencyKey, 3, input, input)
	return s.runner.EnqueueOrGet(ctx, value)
}

func (s *LaboratoryTaskService) handleDelete(ctx context.Context, value *domain.OperationTask) (map[string]any, error) {
	laboratory, err := s.repository.GetLaboratory(ctx, value.ResourceID)
	if isLaboratoryMissing(err) {
		value.ProgressCurrent = value.ProgressTotal
		return map[string]any{"laboratory_id": value.ResourceID, "deleted": true}, nil
	}
	if err != nil {
		return nil, err
	}
	value.ProgressCurrent = 1
	if err = s.runner.Checkpoint(ctx, value); err != nil {
		return nil, err
	}
	if laboratory.LifecycleState == "active" {
		if err = s.repository.MarkLaboratoryDeleting(ctx, value.ResourceID, domain.Revision(taskInt64(value.Input["revision"]))); err != nil {
			return nil, err
		}
	}
	value.ProgressCurrent = 2
	if err = s.runner.Checkpoint(ctx, value); err != nil {
		return nil, err
	}
	deadline := time.NewTimer(s.timeout)
	defer deadline.Stop()
	ticker := time.NewTicker(s.poll)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-deadline.C:
			return nil, domain.Problem{Code: "laboratory_delete_timeout", Message: "laboratory cleanup did not finish before the task deadline", Retryable: true, TaskID: value.ID, ResourceType: "laboratory", ResourceID: value.ResourceID, Phase: "deleting", Cleanup: "committed deletion remains scheduled for reconciliation", OperatorHint: "inspect owned-resource cleanup diagnostics and retry observation", RetryAfterSeconds: 5}
		case <-ticker.C:
			laboratory, err = s.repository.GetLaboratory(ctx, value.ResourceID)
			if isLaboratoryMissing(err) {
				value.ProgressCurrent = value.ProgressTotal
				return map[string]any{"laboratory_id": value.ResourceID, "deleted": true}, nil
			}
			if err != nil {
				return nil, err
			}
			if laboratory.LifecycleState == "delete_failed" {
				return nil, domain.Problem{Code: "laboratory_delete_failed", Message: "laboratory cleanup failed", Retryable: true, TaskID: value.ID, ResourceType: "laboratory", ResourceID: value.ResourceID, Phase: "deleting", Cleanup: "completed cleanup steps are retained for retry", OperatorHint: "inspect remaining owned resources and retry deletion", RetryAfterSeconds: 5}
			}
		}
	}
}

func isLaboratoryMissing(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "not found")
}
