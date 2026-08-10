package command

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/netlab/netlab/internal/app/task"
	"github.com/netlab/netlab/internal/domain"
)

const (
	TopologyConnectionCreateTaskKind = "topology_connection.create"
	TopologyConnectionDeleteTaskKind = "topology_connection.delete"
)

type TopologyConnectionTaskEnvelope struct {
	Connection         domain.TopologyConnection `json:"connection"`
	Task               domain.OperationTask      `json:"task"`
	LaboratoryRevision domain.Revision           `json:"laboratory_revision"`
}

type TopologyConnectionTaskExecutor interface {
	CreateTopologyConnection(context.Context, *domain.OperationTask) (map[string]any, error)
	DeleteTopologyConnection(context.Context, *domain.OperationTask) (map[string]any, error)
	CleanupTopologyConnectionOperation(context.Context, domain.OperationTask) error
}

type TopologyConnectionTaskRunner struct {
	runner   *task.Runner
	executor TopologyConnectionTaskExecutor
}

func NewTopologyConnectionTaskRunner(runner *task.Runner, executor TopologyConnectionTaskExecutor) *TopologyConnectionTaskRunner {
	value := &TopologyConnectionTaskRunner{runner: runner, executor: executor}
	runner.Register(TopologyConnectionCreateTaskKind, value.handleCreate)
	runner.Register(TopologyConnectionDeleteTaskKind, value.handleDelete)
	return value
}

func (r *TopologyConnectionTaskRunner) Cancel(ctx context.Context, taskID domain.ID) error {
	return r.runner.Cancel(ctx, taskID)
}

func (r *TopologyConnectionTaskRunner) handleCreate(ctx context.Context, value *domain.OperationTask) (map[string]any, error) {
	return r.run(ctx, value, r.executor.CreateTopologyConnection, true)
}

func (r *TopologyConnectionTaskRunner) handleDelete(ctx context.Context, value *domain.OperationTask) (map[string]any, error) {
	return r.run(ctx, value, r.executor.DeleteTopologyConnection, false)
}

func (r *TopologyConnectionTaskRunner) run(ctx context.Context, value *domain.OperationTask, execute func(context.Context, *domain.OperationTask) (map[string]any, error), compensate bool) (map[string]any, error) {
	value.ProgressCurrent = 1
	if err := r.runner.Checkpoint(ctx, value); err != nil {
		return nil, err
	}
	result, err := execute(ctx, value)
	if err != nil {
		if compensate {
			cleanupErr := r.executor.CleanupTopologyConnectionOperation(context.Background(), *value)
			if cleanupErr != nil {
				problem := domain.NormalizeProblem(err, domain.Problem{Code: "topology_connection_failed", ResourceType: value.ResourceType, ResourceID: value.ResourceID, TaskID: value.ID, Phase: "connection_runtime", Cleanup: "operation-owned cleanup failed", OperatorHint: "inspect operation-owned resources before retrying"})
				problem.Cleanup = "operation-owned cleanup failed: " + cleanupErr.Error()
				value.Error = &problem
				_ = r.runner.Checkpoint(context.Background(), value)
				if errors.Is(err, context.Canceled) {
					return nil, context.Canceled
				}
				return nil, problem
			}
		}
		return nil, err
	}
	value.ProgressCurrent = value.ProgressTotal
	if err = r.runner.Checkpoint(ctx, value); err != nil {
		return nil, err
	}
	return result, nil
}

func TopologyConnectionRequestFingerprint(laboratoryID domain.ID, source, target domain.ConnectionEndpoint, config domain.TopologyConnectionConfig) string {
	body, _ := json.Marshal(map[string]any{"laboratory_id": laboratoryID, "source": source, "target": target, "config": config})
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func NewTopologyConnectionOperation(kind string, resourceID domain.ID, idempotencyKey, fingerprint string, input map[string]any) domain.OperationTask {
	return domain.OperationTask{ID: domain.NewID(), Kind: kind, ResourceType: "topology_connection", ResourceID: resourceID, IdempotencyKey: idempotencyKey, RequestFingerprint: fingerprint, State: domain.TaskQueued, ProgressTotal: 2, Input: input, CreatedAt: time.Now().UTC()}
}
