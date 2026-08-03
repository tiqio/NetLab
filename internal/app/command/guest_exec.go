package command

import (
	"context"
	"encoding/base64"
	"time"

	"github.com/netlab/netlab/internal/app/audit"
	"github.com/netlab/netlab/internal/app/task"
	"github.com/netlab/netlab/internal/domain"
	qemuRuntime "github.com/netlab/netlab/internal/runtime/qemu"
)

type GuestNodeRepository interface {
	GetNode(context.Context, domain.ID) (domain.Node, error)
}
type GuestExecutor interface {
	GuestExec(context.Context, domain.Node, qemuRuntime.GuestExecRequest) (qemuRuntime.GuestExecResult, error)
}

type GuestCommandService struct {
	repository   GuestNodeRepository
	runner       *task.Runner
	executor     GuestExecutor
	audit        *audit.Service
	capabilities interface {
		ListRuntimeCapabilities(context.Context, domain.ID) ([]domain.RuntimeCapabilityObservation, error)
	}
}

func (s *GuestCommandService) SetCapabilityRepository(repository interface {
	ListRuntimeCapabilities(context.Context, domain.ID) ([]domain.RuntimeCapabilityObservation, error)
}) {
	s.capabilities = repository
}

func NewGuestCommandService(repository GuestNodeRepository, runner *task.Runner, executor GuestExecutor, auditService *audit.Service) *GuestCommandService {
	service := &GuestCommandService{repository: repository, runner: runner, executor: executor, audit: auditService}
	runner.Register("node.guest_exec", service.handle)
	return service
}

func (s *GuestCommandService) Execute(ctx context.Context, nodeID domain.ID, argv []string, timeout time.Duration, outputLimit int, idempotencyKey string) (domain.OperationTask, error) {
	if _, err := s.repository.GetNode(ctx, nodeID); err != nil {
		return domain.OperationTask{}, err
	}
	if s.capabilities != nil {
		observations, err := s.capabilities.ListRuntimeCapabilities(ctx, nodeID)
		if err != nil {
			return domain.OperationTask{}, err
		}
		ready := false
		for _, observation := range observations {
			if observation.Capability == domain.CapabilityGuestExec && observation.State == domain.CapabilityReady {
				ready = true
				break
			}
		}
		if !ready {
			return domain.OperationTask{}, domain.Problem{Code: domain.ProblemCodeCapabilityUnavailable, Message: "QEMU guest agent is not ready", Retryable: true, ResourceType: "node", ResourceID: nodeID, Phase: "guest_exec_preflight", Cleanup: "no command executed", OperatorHint: "install and enable qemu-guest-agent, then wait for capability readiness"}
		}
	}
	value := domain.OperationTask{ID: domain.NewID(), Kind: "node.guest_exec", ResourceType: "node", ResourceID: nodeID, IdempotencyKey: idempotencyKey, ProgressTotal: 2, Input: map[string]any{"argv": argv, "timeout_ms": timeout.Milliseconds(), "output_limit": outputLimit}, CreatedAt: time.Now().UTC()}
	return value, s.runner.Enqueue(ctx, value)
}

func (s *GuestCommandService) handle(ctx context.Context, value *domain.OperationTask) (map[string]any, error) {
	node, err := s.repository.GetNode(ctx, value.ResourceID)
	if err != nil {
		return nil, err
	}
	argvValues, _ := value.Input["argv"].([]any)
	argv := make([]string, 0, len(argvValues))
	for _, item := range argvValues {
		argv = append(argv, text(item))
	}
	if direct, ok := value.Input["argv"].([]string); ok {
		argv = direct
	}
	value.ProgressCurrent = 1
	result, err := s.executor.GuestExec(ctx, node, qemuRuntime.GuestExecRequest{Argv: argv, Timeout: time.Duration(number(value.Input["timeout_ms"])) * time.Millisecond, OutputLimit: int(number(value.Input["output_limit"]))})
	if s.audit != nil {
		outcome := "succeeded"
		if err != nil {
			outcome = "failed"
		}
		_, _ = s.audit.Record(context.Background(), "api", "node.guest_exec", "node", node.ID, value.ID, outcome, string(value.ID), map[string]any{
			"argument_count": len(argv),
			"timeout_ms":     number(value.Input["timeout_ms"]),
			"output_limit":   number(value.Input["output_limit"]),
			"stdout_bytes":   len(result.Stdout),
			"stderr_bytes":   len(result.Stderr),
		})
	}
	if err != nil {
		return nil, err
	}
	value.ProgressCurrent = 2
	return map[string]any{"exit_code": result.ExitCode, "stdout_base64": base64.StdEncoding.EncodeToString(result.Stdout), "stderr_base64": base64.StdEncoding.EncodeToString(result.Stderr), "truncated": result.Truncated}, nil
}
