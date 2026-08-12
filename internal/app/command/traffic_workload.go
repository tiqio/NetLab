package command

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/netlab/netlab/internal/app/task"
	"github.com/netlab/netlab/internal/domain"
)

const (
	TrafficWorkloadCreateTaskKind = "traffic_workload.create"
	TrafficWorkloadStartTaskKind  = "traffic_workload.start"
	TrafficWorkloadStopTaskKind   = "traffic_workload.stop"
	TrafficWorkloadDeleteTaskKind = "traffic_workload.delete"
)

type TrafficWorkloadRepository interface {
	CreateTrafficWorkload(context.Context, domain.TrafficWorkload) error
	GetTrafficWorkload(context.Context, domain.ID) (domain.TrafficWorkload, error)
	ListTrafficWorkloads(context.Context, domain.ID) ([]domain.TrafficWorkload, error)
	UpdateTrafficWorkloadState(context.Context, domain.ID, domain.Revision, string, string, *domain.Problem, domain.ID) (domain.TrafficWorkload, error)
	DeleteTrafficWorkload(context.Context, domain.ID, domain.Revision, domain.ID) error
}

type TrafficWorkloadService struct {
	repository TrafficWorkloadRepository
	runner     *task.Runner
}

func NewTrafficWorkloadService(repository TrafficWorkloadRepository, runner *task.Runner) *TrafficWorkloadService {
	service := &TrafficWorkloadService{repository: repository, runner: runner}
	runner.Register(TrafficWorkloadCreateTaskKind, service.handleCreate)
	runner.Register(TrafficWorkloadStartTaskKind, service.handleState)
	runner.Register(TrafficWorkloadStopTaskKind, service.handleState)
	runner.Register(TrafficWorkloadDeleteTaskKind, service.handleDelete)
	return service
}

func (s *TrafficWorkloadService) List(ctx context.Context, laboratoryID domain.ID) ([]domain.TrafficWorkload, error) {
	return s.repository.ListTrafficWorkloads(ctx, laboratoryID)
}

func (s *TrafficWorkloadService) Get(ctx context.Context, id domain.ID) (domain.TrafficWorkload, error) {
	return s.repository.GetTrafficWorkload(ctx, id)
}

func (s *TrafficWorkloadService) Create(ctx context.Context, workload domain.TrafficWorkload, idempotencyKey string) (domain.OperationTask, error) {
	if workload.ID == "" {
		workload.ID = domain.NewID()
	}
	if workload.Revision == 0 {
		workload.Revision = 1
	}
	if workload.DesiredState == "" {
		workload.DesiredState = "stopped"
	}
	if workload.ObservedState == "" {
		workload.ObservedState = "stopped"
	}
	now := time.Now().UTC()
	if workload.CreatedAt.IsZero() {
		workload.CreatedAt = now
	}
	if workload.UpdatedAt.IsZero() {
		workload.UpdatedAt = now
	}
	if err := workload.Validate(); err != nil {
		return domain.OperationTask{}, domain.Problem{Code: domain.ProblemCodeInvalidRequest, Message: err.Error(), ResourceType: "traffic_workload", ResourceID: workload.ID, Phase: "task_admission", Cleanup: "no workload created", OperatorHint: "correct the workload definition and retry"}
	}
	body, err := json.Marshal(workload)
	if err != nil {
		return domain.OperationTask{}, err
	}
	operation := newTrafficWorkloadOperation(TrafficWorkloadCreateTaskKind, workload.ID, 0, idempotencyKey, trafficWorkloadFingerprint(workload), map[string]any{"workload": string(body), "cancellation_mode": "before_commit"})
	return s.runner.EnqueueOrGet(ctx, operation)
}

func (s *TrafficWorkloadService) Start(ctx context.Context, id domain.ID, revision domain.Revision, idempotencyKey string) (domain.OperationTask, error) {
	return s.enqueueState(ctx, TrafficWorkloadStartTaskKind, id, revision, "running", idempotencyKey)
}

func (s *TrafficWorkloadService) Stop(ctx context.Context, id domain.ID, revision domain.Revision, idempotencyKey string) (domain.OperationTask, error) {
	return s.enqueueState(ctx, TrafficWorkloadStopTaskKind, id, revision, "stopped", idempotencyKey)
}

func (s *TrafficWorkloadService) Delete(ctx context.Context, id domain.ID, revision domain.Revision, idempotencyKey string) (domain.OperationTask, error) {
	if _, err := s.repository.GetTrafficWorkload(ctx, id); err != nil {
		return domain.OperationTask{}, err
	}
	fingerprint := RequestFingerprint([]byte(fmt.Sprintf("%s:%d", id, revision)))
	return s.runner.EnqueueOrGet(ctx, newTrafficWorkloadOperation(TrafficWorkloadDeleteTaskKind, id, revision, idempotencyKey, fingerprint, map[string]any{"cancellation_mode": "before_commit"}))
}

func (s *TrafficWorkloadService) Cancel(ctx context.Context, taskID domain.ID) error {
	return s.runner.Cancel(ctx, taskID)
}

func (s *TrafficWorkloadService) enqueueState(ctx context.Context, kind string, id domain.ID, revision domain.Revision, desired, idempotencyKey string) (domain.OperationTask, error) {
	workload, err := s.repository.GetTrafficWorkload(ctx, id)
	if err != nil {
		return domain.OperationTask{}, err
	}
	if workload.Revision != revision {
		return domain.OperationTask{}, workloadRevisionConflict(id)
	}
	fingerprint := RequestFingerprint([]byte(fmt.Sprintf("%s:%d:%s", id, revision, desired)))
	return s.runner.EnqueueOrGet(ctx, newTrafficWorkloadOperation(kind, id, revision, idempotencyKey, fingerprint, map[string]any{"desired_state": desired, "cancellation_mode": "before_commit"}))
}

func newTrafficWorkloadOperation(kind string, id domain.ID, revision domain.Revision, key, fingerprint string, input map[string]any) domain.OperationTask {
	return domain.OperationTask{ID: domain.NewID(), Kind: kind, ResourceType: "traffic_workload", ResourceID: id, IdempotencyKey: key, RequestFingerprint: fingerprint, RequestedRevision: revision, State: domain.TaskQueued, ProgressTotal: 2, Input: input, CreatedAt: time.Now().UTC()}
}

func (s *TrafficWorkloadService) handleCreate(ctx context.Context, operation *domain.OperationTask) (map[string]any, error) {
	var workload domain.TrafficWorkload
	if err := json.Unmarshal([]byte(text(operation.Input["workload"])), &workload); err != nil {
		return nil, domain.Problem{Code: domain.ProblemCodeInvalidRequest, Message: "invalid durable workload input", ResourceType: "traffic_workload", ResourceID: operation.ResourceID, Phase: "workload_create", Cleanup: "no workload created"}
	}
	if existing, err := s.repository.GetTrafficWorkload(ctx, workload.ID); err == nil {
		if trafficWorkloadFingerprint(existing) != trafficWorkloadFingerprint(workload) {
			return nil, domain.Problem{Code: "resource_conflict", Message: "traffic workload id already exists with different configuration", ResourceType: "traffic_workload", ResourceID: workload.ID, Phase: "workload_create", Cleanup: "existing workload remains unchanged"}
		}
		operation.ProgressCurrent = operation.ProgressTotal
		return map[string]any{"workload_id": workload.ID}, nil
	} else if !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	operation.ProgressCurrent = 1
	if err := s.runner.Checkpoint(ctx, operation); err != nil {
		return nil, err
	}
	if err := s.repository.CreateTrafficWorkload(ctx, workload); err != nil {
		return nil, err
	}
	operation.ProgressCurrent = operation.ProgressTotal
	if err := s.runner.Checkpoint(context.Background(), operation); err != nil {
		return nil, err
	}
	return map[string]any{"workload_id": workload.ID}, nil
}

func (s *TrafficWorkloadService) handleState(ctx context.Context, operation *domain.OperationTask) (map[string]any, error) {
	desired := text(operation.Input["desired_state"])
	existing, err := s.repository.GetTrafficWorkload(ctx, operation.ResourceID)
	if err != nil {
		return nil, err
	}
	if existing.Revision == operation.RequestedRevision.Next() && existing.DesiredState == desired {
		operation.ProgressCurrent = operation.ProgressTotal
		return map[string]any{"workload_id": existing.ID, "revision": existing.Revision}, nil
	}
	if existing.Revision != operation.RequestedRevision {
		return nil, workloadRevisionConflict(existing.ID)
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	operation.ProgressCurrent = 1
	if err = s.runner.Checkpoint(ctx, operation); err != nil {
		return nil, err
	}
	updated, err := s.repository.UpdateTrafficWorkloadState(ctx, existing.ID, operation.RequestedRevision, desired, "queued", nil, operation.ID)
	if err != nil {
		return nil, err
	}
	operation.ProgressCurrent = operation.ProgressTotal
	if err = s.runner.Checkpoint(context.Background(), operation); err != nil {
		return nil, err
	}
	return map[string]any{"workload_id": updated.ID, "revision": updated.Revision}, nil
}

func (s *TrafficWorkloadService) handleDelete(ctx context.Context, operation *domain.OperationTask) (map[string]any, error) {
	existing, err := s.repository.GetTrafficWorkload(ctx, operation.ResourceID)
	if errors.Is(err, domain.ErrNotFound) {
		operation.ProgressCurrent = operation.ProgressTotal
		return map[string]any{"workload_id": operation.ResourceID, "deleted": true}, nil
	}
	if err != nil {
		return nil, err
	}
	if existing.Revision != operation.RequestedRevision {
		return nil, workloadRevisionConflict(existing.ID)
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	operation.ProgressCurrent = 1
	if err = s.runner.Checkpoint(ctx, operation); err != nil {
		return nil, err
	}
	if err = s.repository.DeleteTrafficWorkload(ctx, existing.ID, operation.RequestedRevision, operation.ID); err != nil {
		return nil, err
	}
	operation.ProgressCurrent = operation.ProgressTotal
	if err = s.runner.Checkpoint(context.Background(), operation); err != nil {
		return nil, err
	}
	return map[string]any{"workload_id": existing.ID, "deleted": true}, nil
}

func trafficWorkloadFingerprint(workload domain.TrafficWorkload) string {
	workload.ID = ""
	workload.Revision = 0
	workload.Attempts = 0
	workload.Successes = 0
	workload.Failures = 0
	workload.MatchedBytes = 0
	workload.LastSuccessAt = nil
	workload.LastError = nil
	workload.CreatedAt = time.Time{}
	workload.UpdatedAt = time.Time{}
	body, _ := json.Marshal(workload)
	return RequestFingerprint(body)
}

func workloadRevisionConflict(id domain.ID) domain.Problem {
	return domain.Problem{Code: domain.ProblemCodeConflict, Message: "traffic workload revision mismatch", Retryable: true, ResourceType: "traffic_workload", ResourceID: id, Phase: "task_admission", Cleanup: "workload remains unchanged", OperatorHint: "refresh the workload and retry with its current revision"}
}
