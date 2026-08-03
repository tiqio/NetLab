package command

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/netlab/netlab/internal/app/ports"
	"github.com/netlab/netlab/internal/app/task"
	"github.com/netlab/netlab/internal/domain"
)

type TopologyTaskRepository interface {
	GetNode(context.Context, domain.ID) (domain.Node, error)
	ListNodeLinks(context.Context, domain.ID) ([]domain.Link, error)
	SetNodeDesiredState(context.Context, domain.ID, domain.Revision, domain.DesiredState) (domain.Node, error)
	DeleteNode(context.Context, domain.ID, domain.Revision) error
	CreateLink(context.Context, domain.Link) error
	GetLink(context.Context, domain.ID) (domain.Link, error)
	LinkEndpointsReady(context.Context, domain.Link) (bool, error)
	MarkLinkDisconnected(context.Context, domain.ID) error
	MarkLinkConnected(context.Context, domain.ID) error
	DeleteLink(context.Context, domain.ID) error
}

type TopologyTaskService struct {
	repository       TopologyTaskRepository
	runner           *task.Runner
	poll             time.Duration
	timeout          time.Duration
	deleteTimeout    time.Duration
	deleteRuntimes   map[string]ports.NodeRuntime
	deleteResources  interface{ Cleanup(domain.ID) error }
	deleteInterfaces interface {
		Delete(context.Context, string) error
	}
	deleteLinks interface {
		DeleteLink(context.Context, domain.ID) error
	}
}

func NewTopologyTaskService(repository TopologyTaskRepository, runner *task.Runner) *TopologyTaskService {
	service := &TopologyTaskService{repository: repository, runner: runner, poll: 250 * time.Millisecond, timeout: 2 * time.Minute, deleteTimeout: 30 * time.Second, deleteRuntimes: map[string]ports.NodeRuntime{}}
	runner.Register("node.set_state", service.handleNodeState)
	runner.Register("node.delete", service.handleNodeDelete)
	runner.Register("link.connect", service.handleLinkConnect)
	runner.Register("link.disconnect", service.handleLinkDisconnect)
	return service
}

func (s *TopologyTaskService) SetNodeDeletionRuntime(kind string, runtime ports.NodeRuntime) {
	if kind != "" && runtime != nil {
		s.deleteRuntimes[kind] = runtime
	}
}

func (s *TopologyTaskService) SetNodeDeletionCleanup(resources interface{ Cleanup(domain.ID) error }, interfaces interface {
	Delete(context.Context, string) error
}) {
	s.deleteResources = resources
	s.deleteInterfaces = interfaces
}

func (s *TopologyTaskService) SetNodeDeletionLinkRuntime(runtime interface {
	DeleteLink(context.Context, domain.ID) error
}) {
	s.deleteLinks = runtime
}

func (s *TopologyTaskService) SetNodeState(ctx context.Context, id domain.ID, revision domain.Revision, state domain.DesiredState, idempotencyKey string) (domain.OperationTask, error) {
	if state != domain.DesiredRunning && state != domain.DesiredStopped {
		return domain.OperationTask{}, domain.Problem{Code: "invalid_node_transition", Message: "desired state must be running or stopped", ResourceType: "node", ResourceID: id}
	}
	node, err := s.repository.GetNode(ctx, id)
	if err != nil {
		return domain.OperationTask{}, err
	}
	value := s.operation("node.set_state", "node", id, idempotencyKey, 3, map[string]any{"revision": int64(revision), "desired_state": string(state), "previous_desired_state": string(node.DesiredState)}, map[string]any{"node_id": id, "revision": revision, "desired_state": state})
	return s.runner.EnqueueOrGet(ctx, value)
}

func (s *TopologyTaskService) DeleteNode(ctx context.Context, id domain.ID, revision domain.Revision, idempotencyKey string) (domain.OperationTask, error) {
	value := s.operation("node.delete", "node", id, idempotencyKey, 3, map[string]any{"revision": int64(revision)}, map[string]any{"node_id": id, "revision": revision})
	return s.runner.EnqueueOrGet(ctx, value)
}

func (s *TopologyTaskService) ConnectLink(ctx context.Context, labID, endpointAID, endpointBID domain.ID, idempotencyKey string) (domain.Link, domain.OperationTask, error) {
	if endpointAID == endpointBID {
		return domain.Link{}, domain.OperationTask{}, fmt.Errorf("link endpoints must differ")
	}
	link := domain.Link{ID: domain.NewID(), LaboratoryID: labID, EndpointAID: endpointAID, EndpointBID: endpointBID, Revision: 1, DesiredState: "connected", ObservedState: "pending"}
	value := s.operation("link.connect", "link", link.ID, idempotencyKey, 2, map[string]any{"laboratory_id": string(labID), "endpoint_a_id": string(endpointAID), "endpoint_b_id": string(endpointBID)}, map[string]any{"laboratory_id": labID, "endpoint_a_id": endpointAID, "endpoint_b_id": endpointBID})
	queued, err := s.runner.EnqueueOrGet(ctx, value)
	if err != nil {
		return domain.Link{}, domain.OperationTask{}, err
	}
	if queued.ID != value.ID {
		link = domain.Link{ID: queued.ResourceID, LaboratoryID: domain.ID(taskText(queued.Input["laboratory_id"])), EndpointAID: domain.ID(taskText(queued.Input["endpoint_a_id"])), EndpointBID: domain.ID(taskText(queued.Input["endpoint_b_id"])), Revision: 1, DesiredState: "connected", ObservedState: "pending"}
	}
	return link, queued, nil
}

func (s *TopologyTaskService) DisconnectLink(ctx context.Context, id domain.ID, idempotencyKey string) (domain.OperationTask, error) {
	value := s.operation("link.disconnect", "link", id, idempotencyKey, 2, nil, map[string]any{"link_id": id})
	return s.runner.EnqueueOrGet(ctx, value)
}

func (s *TopologyTaskService) operation(kind, resourceType string, resourceID domain.ID, idempotencyKey string, total int, input, fingerprintInput map[string]any) domain.OperationTask {
	body, _ := json.Marshal(fingerprintInput)
	return domain.OperationTask{ID: domain.NewID(), Kind: kind, ResourceType: resourceType, ResourceID: resourceID, IdempotencyKey: idempotencyKey, RequestFingerprint: RequestFingerprint(body), State: domain.TaskQueued, ProgressTotal: total, Input: input, CreatedAt: time.Now().UTC()}
}

func (s *TopologyTaskService) handleNodeState(ctx context.Context, value *domain.OperationTask) (map[string]any, error) {
	id := value.ResourceID
	desired := domain.DesiredState(taskText(value.Input["desired_state"]))
	previous := domain.DesiredState(taskText(value.Input["previous_desired_state"]))
	node, err := s.repository.GetNode(ctx, id)
	if err != nil {
		return nil, err
	}
	if node.DesiredState != desired {
		node, err = s.repository.SetNodeDesiredState(ctx, id, domain.Revision(taskNumber(value.Input["revision"])), desired)
		if err != nil {
			return nil, err
		}
	}
	value.ProgressCurrent = 1
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
			current, getErr := s.repository.GetNode(context.Background(), id)
			if getErr == nil && previous != "" && current.DesiredState == desired && previous != desired {
				_, _ = s.repository.SetNodeDesiredState(context.Background(), id, current.Revision, previous)
			}
			return nil, ctx.Err()
		case <-deadline.C:
			return nil, domain.Problem{Code: "node_lifecycle_timeout", Message: "node did not reach desired state before the task deadline", Retryable: true, TaskID: value.ID, ResourceType: "node", ResourceID: id, Phase: "lifecycle_convergence", Cleanup: "desired state remains durable for reconciliation", OperatorHint: "inspect node diagnostics and retry", RetryAfterSeconds: 3}
		case <-ticker.C:
			node, err = s.repository.GetNode(ctx, id)
			if err != nil {
				return nil, err
			}
			if node.ObservedState == domain.ObservedFailed {
				if node.LastError != nil {
					return nil, node.LastError
				}
				return nil, domain.Problem{Code: "node_failed", Message: "node entered failed state", Retryable: true, TaskID: value.ID, ResourceType: "node", ResourceID: id, Phase: "lifecycle_convergence", Cleanup: "desired state remains durable for reconciliation", OperatorHint: "inspect node diagnostics and retry", RetryAfterSeconds: 3}
			}
			if observedMatchesDesired(node.ObservedState, desired) {
				value.ProgressCurrent = value.ProgressTotal
				return map[string]any{"node": node}, nil
			}
			if value.ProgressCurrent < 2 {
				value.ProgressCurrent = 2
				if err = s.runner.Checkpoint(ctx, value); err != nil {
					return nil, err
				}
			}
		}
	}
}

func (s *TopologyTaskService) handleNodeDelete(ctx context.Context, value *domain.OperationTask) (map[string]any, error) {
	node, err := s.repository.GetNode(ctx, value.ResourceID)
	if err != nil {
		if taskNotFound(err) {
			return map[string]any{"node_id": value.ResourceID, "deleted": true}, nil
		}
		return nil, err
	}
	observer, _ := s.repository.(interface {
		SetNodeObservedState(context.Context, domain.ID, domain.ObservedState, *domain.Problem) error
	})
	if node.ObservedState != domain.ObservedDeleting && observer != nil {
		if err = observer.SetNodeObservedState(ctx, node.ID, domain.ObservedDeleting, nil); err != nil {
			return nil, err
		}
	}
	value.ProgressCurrent = 1
	if err = s.runner.Checkpoint(ctx, value); err != nil {
		return nil, err
	}
	fail := func(cause error, code, cleanup, hint string) error {
		problem := domain.Problem{Code: code, Message: cause.Error(), Retryable: true, TaskID: value.ID, ResourceType: "node", ResourceID: node.ID, Phase: "deleting", Cleanup: cleanup, OperatorHint: hint, RetryAfterSeconds: 3}
		if observer != nil {
			_ = observer.SetNodeObservedState(context.Background(), node.ID, domain.ObservedFailed, &problem)
		}
		return problem
	}
	links, listLinksErr := s.repository.ListNodeLinks(ctx, node.ID)
	if listLinksErr != nil {
		return nil, fail(listLinksErr, "node_delete_link_inventory_failed", "node, links, and runtime left intact", "inspect the node link inventory and retry deletion")
	}
	if len(links) > 0 && s.deleteLinks == nil {
		return nil, fail(errors.New("link runtime unavailable"), "node_delete_link_runtime_unavailable", "node, links, and runtime left intact", "restore the data-plane runtime and retry deletion")
	}
	for _, link := range links {
		if err = s.deleteLinks.DeleteLink(ctx, link.ID); err != nil {
			return nil, fail(err, "node_delete_link_cleanup_failed", "node row and connected link rows retained", "inspect the owned link bridge and retry deletion")
		}
	}
	if runtime := s.deleteRuntimes[node.Kind]; runtime != nil {
		deleteCtx, cancel := context.WithTimeout(ctx, s.deleteTimeout)
		err = runtime.Delete(deleteCtx, node)
		cancel()
		if err != nil {
			code := "node_delete_runtime_failed"
			if errors.Is(err, context.DeadlineExceeded) {
				code = "node_delete_timeout"
			}
			return nil, fail(err, code, "node row and ownership records retained; completed cleanup steps remain durable", "inspect the owned runtime, remove the blocking resource, and retry deletion")
		}
	} else if node.Kind != "" {
		return nil, fail(errors.New("node runtime unavailable"), "node_delete_runtime_unavailable", "node row and owned resources retained", "restore the required runtime and retry deletion")
	}
	if s.deleteResources != nil {
		if err = s.deleteResources.Cleanup(node.ID); err != nil {
			return nil, fail(err, "node_delete_resource_cleanup_failed", "runtime cleanup completed but node row and resource ownership remain", "inspect cgroup and runtime ownership, then retry deletion")
		}
	}
	if lister, ok := s.repository.(interface {
		ListNodeOwnedTaps(context.Context, domain.ID) ([]string, error)
	}); ok && s.deleteInterfaces != nil {
		taps, listErr := lister.ListNodeOwnedTaps(ctx, node.ID)
		if listErr != nil {
			return nil, fail(listErr, "node_delete_inventory_failed", "runtime cleanup completed; node row retained", "inspect interface inventory and retry deletion")
		}
		for _, tap := range taps {
			if cleanupErr := s.deleteInterfaces.Delete(ctx, tap); cleanupErr != nil {
				return nil, fail(cleanupErr, "node_delete_interface_cleanup_failed", "runtime cleanup completed; node row and remaining TAP ownership retained", "remove the owned TAP and retry deletion")
			}
		}
	}
	value.ProgressCurrent = 2
	if err = s.runner.Checkpoint(ctx, value); err != nil {
		return nil, err
	}
	err = s.repository.DeleteNode(ctx, value.ResourceID, domain.Revision(taskNumber(value.Input["revision"])))
	if err != nil && !taskNotFound(err) {
		return nil, fail(err, "node_delete_persistence_failed", "owned runtime cleanup completed; node row retained for retry", "resolve the database conflict and retry deletion")
	}
	value.ProgressCurrent = value.ProgressTotal
	return map[string]any{"node_id": value.ResourceID, "deleted": true}, nil
}

func (s *TopologyTaskService) handleLinkConnect(ctx context.Context, value *domain.OperationTask) (map[string]any, error) {
	link, err := s.repository.GetLink(ctx, value.ResourceID)
	if err != nil {
		if !taskNotFound(err) {
			return nil, err
		}
		link = domain.Link{ID: value.ResourceID, LaboratoryID: domain.ID(taskText(value.Input["laboratory_id"])), EndpointAID: domain.ID(taskText(value.Input["endpoint_a_id"])), EndpointBID: domain.ID(taskText(value.Input["endpoint_b_id"])), Revision: 1, DesiredState: "connected", ObservedState: "pending"}
		if err = s.repository.CreateLink(ctx, link); err != nil {
			return nil, err
		}
	}
	value.ProgressCurrent = 1
	if err = s.runner.Checkpoint(ctx, value); err != nil {
		return nil, err
	}
	ready, err := s.repository.LinkEndpointsReady(ctx, link)
	if err != nil {
		return nil, err
	}
	if !ready {
		value.ProgressCurrent = value.ProgressTotal
		return map[string]any{
			"link":        link,
			"convergence": "pending",
			"reason":      "endpoint nodes are not running",
		}, nil
	}
	return s.waitLink(ctx, value, true)
}

func (s *TopologyTaskService) handleLinkDisconnect(ctx context.Context, value *domain.OperationTask) (map[string]any, error) {
	link, err := s.repository.GetLink(ctx, value.ResourceID)
	if err != nil {
		if taskNotFound(err) {
			value.ProgressCurrent = value.ProgressTotal
			return map[string]any{"link_id": value.ResourceID, "disconnected": true}, nil
		}
		return nil, err
	}
	if link.DesiredState != "disconnected" {
		if err = s.repository.MarkLinkDisconnected(ctx, value.ResourceID); err != nil {
			return nil, err
		}
	}
	value.ProgressCurrent = 1
	if err = s.runner.Checkpoint(ctx, value); err != nil {
		return nil, err
	}
	return s.waitLink(ctx, value, false)
}

func (s *TopologyTaskService) waitLink(ctx context.Context, value *domain.OperationTask, connecting bool) (map[string]any, error) {
	deadline := time.NewTimer(s.timeout)
	defer deadline.Stop()
	ticker := time.NewTicker(s.poll)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			if connecting {
				_ = s.repository.MarkLinkDisconnected(context.Background(), value.ResourceID)
			} else {
				_ = s.repository.MarkLinkConnected(context.Background(), value.ResourceID)
			}
			return nil, ctx.Err()
		case <-deadline.C:
			return nil, domain.Problem{Code: "link_convergence_timeout", Message: "link did not converge before the task deadline", Retryable: true, TaskID: value.ID, ResourceType: "link", ResourceID: value.ResourceID, Phase: "link_convergence", Cleanup: "desired link state remains durable for reconciliation", OperatorHint: "inspect interface and bridge diagnostics then retry", RetryAfterSeconds: 2}
		case <-ticker.C:
			link, err := s.repository.GetLink(ctx, value.ResourceID)
			if err != nil {
				if !connecting && taskNotFound(err) {
					value.ProgressCurrent = value.ProgressTotal
					return map[string]any{"link_id": value.ResourceID, "disconnected": true}, nil
				}
				return nil, err
			}
			if connecting && link.ObservedState == "connected" {
				value.ProgressCurrent = value.ProgressTotal
				return map[string]any{"link": link}, nil
			}
			if link.ObservedState == "failed" {
				return nil, domain.Problem{Code: "link_failed", Message: "link reconciliation failed", Retryable: true, TaskID: value.ID, ResourceType: "link", ResourceID: value.ResourceID, Phase: "link_convergence", Cleanup: "link record retained for retry or deletion", OperatorHint: "inspect endpoint and bridge diagnostics", RetryAfterSeconds: 2}
			}
		}
	}
}

func observedMatchesDesired(observed domain.ObservedState, desired domain.DesiredState) bool {
	return desired == domain.DesiredRunning && observed == domain.ObservedRunning || desired == domain.DesiredStopped && observed == domain.ObservedStopped
}

func taskText(value any) string {
	text, _ := value.(string)
	return text
}

func taskNumber(value any) int64 {
	switch number := value.(type) {
	case int64:
		return number
	case float64:
		return int64(number)
	case int:
		return int64(number)
	default:
		return 0
	}
}

func taskNotFound(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "not found")
}
