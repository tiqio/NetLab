package reconcile

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/netlab/netlab/internal/app/command"
	"github.com/netlab/netlab/internal/app/task"
	"github.com/netlab/netlab/internal/domain"
)

type NetworkObjectTaskService struct {
	service *NetworkObjectService
	runner  *task.Runner
}

func NewNetworkObjectTaskService(service *NetworkObjectService, runner *task.Runner) *NetworkObjectTaskService {
	value := &NetworkObjectTaskService{service: service, runner: runner}
	runner.Register("network_object.create", value.handleCreate)
	runner.Register("network_object.update", value.handleUpdate)
	runner.Register("network_object.delete", value.handleDelete)
	runner.Register("network_object_link.create", value.handleObjectLinkCreate)
	runner.Register("network_object_link.delete", value.handleObjectLinkDelete)
	return value
}

func (s *NetworkObjectTaskService) DeleteObjectLink(ctx context.Context, id domain.ID, revision domain.Revision, idempotencyKey string) (domain.NetworkObjectLink, domain.OperationTask, error) {
	link, err := s.service.GetObjectLink(ctx, id)
	if err != nil {
		if idempotencyKey != "" {
			if existing, lookupErr := s.runner.GetByIdempotency(ctx, "network_object_link.delete", idempotencyKey); lookupErr == nil {
				if existing.ResourceID != id || domain.Revision(networkTaskInt64(existing.Input["revision"])) != revision {
					return domain.NetworkObjectLink{}, domain.OperationTask{}, domain.Problem{Code: "idempotency_conflict", Message: "idempotency key was already used with a different delete request", ResourceType: "network_object_link", ResourceID: existing.ResourceID, TaskID: existing.ID, Phase: "delete_admission"}
				}
				return domain.NetworkObjectLink{ID: id, LaboratoryID: domain.ID(networkTaskText(existing.Input["laboratory_id"])), ObjectAID: domain.ID(networkTaskText(existing.Input["object_a_id"])), PortAName: networkTaskText(existing.Input["port_a_name"]), ObjectBID: domain.ID(networkTaskText(existing.Input["object_b_id"])), PortBName: networkTaskText(existing.Input["port_b_name"]), Revision: revision, DesiredState: "disconnected", ObservedState: "disconnected"}, existing, nil
			}
		}
		return domain.NetworkObjectLink{}, domain.OperationTask{}, err
	}
	if link.Revision != revision {
		return domain.NetworkObjectLink{}, domain.OperationTask{}, domain.Problem{Code: "revision_conflict", Message: fmt.Sprintf("expected revision %d, current revision is %d", revision, link.Revision), ResourceType: "network_object_link", ResourceID: id, Phase: "delete_admission"}
	}
	input := map[string]any{"revision": int64(revision), "laboratory_id": link.LaboratoryID, "object_a_id": link.ObjectAID, "port_a_name": link.PortAName, "object_b_id": link.ObjectBID, "port_b_name": link.PortBName}
	operation := networkObjectLinkOperation("network_object_link.delete", id, idempotencyKey, input)
	queued, err := s.runner.EnqueueOrGet(ctx, operation)
	if err != nil {
		return domain.NetworkObjectLink{}, domain.OperationTask{}, err
	}
	if queued.ID != operation.ID {
		link.ID = queued.ResourceID
		link.Revision = domain.Revision(networkTaskInt64(queued.Input["revision"]))
	}
	link.DesiredState = "disconnected"
	link.ObservedState = "disconnecting"
	return link, queued, nil
}

func (s *NetworkObjectTaskService) CreateObjectLink(ctx context.Context, laboratoryID, objectAID domain.ID, portAName string, objectBID domain.ID, portBName, idempotencyKey string) (domain.NetworkObjectLink, domain.OperationTask, error) {
	portAName, portBName = strings.TrimSpace(portAName), strings.TrimSpace(portBName)
	if objectAID == "" || objectBID == "" || objectAID == objectBID {
		return domain.NetworkObjectLink{}, domain.OperationTask{}, domain.Problem{Code: "invalid_topology", Message: "two different network objects are required", ResourceType: "network_object_link"}
	}
	if err := domain.ValidateNetworkObjectPortName(portAName); err != nil {
		return domain.NetworkObjectLink{}, domain.OperationTask{}, err
	}
	if err := domain.ValidateNetworkObjectPortName(portBName); err != nil {
		return domain.NetworkObjectLink{}, domain.OperationTask{}, err
	}
	link := domain.NetworkObjectLink{ID: domain.NewID(), LaboratoryID: laboratoryID, ObjectAID: objectAID, PortAName: portAName, ObjectBID: objectBID, PortBName: portBName, Revision: 1, DesiredState: "connected", ObservedState: "pending"}
	input := map[string]any{"laboratory_id": laboratoryID, "object_a_id": objectAID, "port_a_name": portAName, "object_b_id": objectBID, "port_b_name": portBName}
	operation := networkObjectLinkOperation("network_object_link.create", link.ID, idempotencyKey, input)
	if idempotencyKey != "" {
		if existing, lookupErr := s.runner.GetByIdempotency(ctx, operation.Kind, idempotencyKey); lookupErr == nil {
			if existing.RequestFingerprint != operation.RequestFingerprint {
				return domain.NetworkObjectLink{}, domain.OperationTask{}, domain.Problem{Code: "idempotency_conflict", Message: "idempotency key was already used with a different create request", ResourceType: "network_object_link", ResourceID: existing.ResourceID, TaskID: existing.ID, Phase: "task_admission"}
			}
			return domain.NetworkObjectLink{ID: existing.ResourceID, LaboratoryID: domain.ID(networkTaskText(existing.Input["laboratory_id"])), ObjectAID: domain.ID(networkTaskText(existing.Input["object_a_id"])), PortAName: networkTaskText(existing.Input["port_a_name"]), ObjectBID: domain.ID(networkTaskText(existing.Input["object_b_id"])), PortBName: networkTaskText(existing.Input["port_b_name"]), Revision: 1, DesiredState: "connected", ObservedState: "pending"}, existing, nil
		}
	}
	if err := s.service.ValidateObjectLinkAdmission(ctx, laboratoryID, objectAID, portAName, objectBID, portBName); err != nil {
		return domain.NetworkObjectLink{}, domain.OperationTask{}, err
	}
	queued, err := s.runner.EnqueueOrGet(ctx, operation)
	if err != nil {
		return domain.NetworkObjectLink{}, domain.OperationTask{}, err
	}
	if queued.ID != operation.ID {
		link.ID = queued.ResourceID
		link.LaboratoryID = domain.ID(networkTaskText(queued.Input["laboratory_id"]))
		link.ObjectAID = domain.ID(networkTaskText(queued.Input["object_a_id"]))
		link.PortAName = networkTaskText(queued.Input["port_a_name"])
		link.ObjectBID = domain.ID(networkTaskText(queued.Input["object_b_id"]))
		link.PortBName = networkTaskText(queued.Input["port_b_name"])
	}
	return link, queued, nil
}

func networkObjectLinkOperation(kind string, resourceID domain.ID, idempotencyKey string, input map[string]any) domain.OperationTask {
	body, _ := json.Marshal(input)
	sum := sha256.Sum256(body)
	return domain.OperationTask{ID: domain.NewID(), Kind: kind, ResourceType: "network_object_link", ResourceID: resourceID, IdempotencyKey: idempotencyKey, RequestFingerprint: hex.EncodeToString(sum[:]), State: domain.TaskQueued, ProgressTotal: 2, Input: input, CreatedAt: time.Now().UTC()}
}

func (s *NetworkObjectTaskService) handleObjectLinkCreate(ctx context.Context, value *domain.OperationTask) (map[string]any, error) {
	value.ProgressCurrent = 1
	if err := s.runner.Checkpoint(ctx, value); err != nil {
		return nil, err
	}
	link, err := s.service.CreateObjectLinkAs(ctx, value.ResourceID, domain.ID(networkTaskText(value.Input["laboratory_id"])), domain.ID(networkTaskText(value.Input["object_a_id"])), networkTaskText(value.Input["port_a_name"]), domain.ID(networkTaskText(value.Input["object_b_id"])), networkTaskText(value.Input["port_b_name"]))
	if err != nil {
		return nil, *command.NormalizeOperationProblem(err, domain.Problem{Code: "network_object_link_create_failed", Message: "network object link creation failed", ResourceType: "network_object_link", ResourceID: value.ResourceID, TaskID: value.ID, Phase: "reservation", Cleanup: "no endpoint reservations retained", OperatorHint: "choose two free ports in the same laboratory and retry"}, false)
	}
	value.ProgressCurrent = value.ProgressTotal
	return map[string]any{"network_object_link": link}, nil
}

func (s *NetworkObjectTaskService) handleObjectLinkDelete(ctx context.Context, value *domain.OperationTask) (map[string]any, error) {
	value.ProgressCurrent = 1
	if err := s.runner.Checkpoint(ctx, value); err != nil {
		return nil, err
	}
	err := s.service.DeleteObjectLinkRevision(ctx, value.ResourceID, domain.Revision(networkTaskInt64(value.Input["revision"])), value.ID)
	if err != nil {
		return nil, *command.NormalizeOperationProblem(err, domain.Problem{Code: "network_object_link_delete_failed", Message: "network object link deletion failed", Retryable: true, ResourceType: "network_object_link", ResourceID: value.ResourceID, TaskID: value.ID, Phase: "cleanup", Cleanup: "link remains authoritative until cleanup succeeds", OperatorHint: "inspect endpoint ownership and retry with the same idempotency key"}, true)
	}
	value.ProgressCurrent = value.ProgressTotal
	return map[string]any{"network_object_link_id": value.ResourceID, "deleted": true}, nil
}

func (s *NetworkObjectTaskService) Update(ctx context.Context, id domain.ID, revision domain.Revision, name string, config map[string]any, idempotencyKey string) (domain.NetworkObject, domain.OperationTask, error) {
	current, err := s.service.Get(ctx, id)
	if err != nil {
		return domain.NetworkObject{}, domain.OperationTask{}, err
	}
	if current.Revision != revision {
		return domain.NetworkObject{}, domain.OperationTask{}, domain.Problem{Code: "revision_conflict", Message: "network object revision mismatch", ResourceType: "network_object", ResourceID: id}
	}
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 128 {
		return domain.NetworkObject{}, domain.OperationTask{}, fmt.Errorf("network object name must be 1-128 characters")
	}
	if err = validateSwitchConfiguration(current.Kind, config); err != nil {
		return domain.NetworkObject{}, domain.OperationTask{}, err
	}
	input := map[string]any{
		"revision":        int64(revision),
		"laboratory_id":   current.LaboratoryID,
		"name":            name,
		"kind":            current.Kind,
		"config":          config,
		"previous_name":   current.Name,
		"previous_config": current.Config,
	}
	value := networkObjectOperation("network_object.update", id, idempotencyKey, input)
	queued, err := s.runner.EnqueueOrGet(ctx, value)
	if err != nil {
		return domain.NetworkObject{}, domain.OperationTask{}, err
	}
	predicted := current
	predicted.Name = name
	predicted.Config = config
	predicted.Revision = revision + 1
	predicted.ObservedState = "provisioning"
	return predicted, queued, nil
}

func (s *NetworkObjectTaskService) Create(ctx context.Context, laboratoryID domain.ID, name, kind string, config map[string]any, idempotencyKey string) (domain.NetworkObject, domain.OperationTask, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 128 {
		return domain.NetworkObject{}, domain.OperationTask{}, fmt.Errorf("network object name must be 1-128 characters")
	}
	if err := domain.ValidateNetworkKind(kind); err != nil {
		return domain.NetworkObject{}, domain.OperationTask{}, err
	}
	object := domain.NetworkObject{ID: domain.NewID(), LaboratoryID: laboratoryID, Name: name, Kind: kind, Revision: 1, DesiredState: "active", ObservedState: "provisioning", Config: config}
	input := map[string]any{"laboratory_id": laboratoryID, "name": name, "kind": kind, "config": config}
	value := networkObjectOperation("network_object.create", object.ID, idempotencyKey, input)
	queued, err := s.runner.EnqueueOrGet(ctx, value)
	if err != nil {
		return domain.NetworkObject{}, domain.OperationTask{}, err
	}
	if queued.ID != value.ID {
		object.ID = queued.ResourceID
		object.LaboratoryID = domain.ID(networkTaskText(queued.Input["laboratory_id"]))
		object.Name = networkTaskText(queued.Input["name"])
		object.Kind = networkTaskText(queued.Input["kind"])
		object.Config, _ = networkTaskMap(queued.Input["config"])
	}
	return object, queued, nil
}

func (s *NetworkObjectTaskService) Delete(ctx context.Context, id domain.ID, revision domain.Revision, idempotencyKey string) (domain.OperationTask, error) {
	object, err := s.service.Get(ctx, id)
	if err != nil {
		return domain.OperationTask{}, err
	}
	input := map[string]any{"revision": int64(revision), "laboratory_id": object.LaboratoryID, "name": object.Name, "kind": object.Kind, "config": object.Config}
	value := networkObjectOperation("network_object.delete", id, idempotencyKey, input)
	return s.runner.EnqueueOrGet(ctx, value)
}

func networkObjectOperation(kind string, resourceID domain.ID, idempotencyKey string, input map[string]any) domain.OperationTask {
	body, _ := json.Marshal(input)
	sum := sha256.Sum256(body)
	return domain.OperationTask{ID: domain.NewID(), Kind: kind, ResourceType: "network_object", ResourceID: resourceID, IdempotencyKey: idempotencyKey, RequestFingerprint: hex.EncodeToString(sum[:]), State: domain.TaskQueued, ProgressTotal: 2, Input: input, CreatedAt: time.Now().UTC()}
}

func (s *NetworkObjectTaskService) handleCreate(ctx context.Context, value *domain.OperationTask) (map[string]any, error) {
	value.ProgressCurrent = 1
	if err := s.runner.Checkpoint(ctx, value); err != nil {
		return nil, err
	}
	config, err := networkTaskMap(value.Input["config"])
	if err != nil {
		return nil, err
	}
	object, err := s.service.CreateAs(ctx, value.ResourceID, domain.ID(networkTaskText(value.Input["laboratory_id"])), networkTaskText(value.Input["name"]), networkTaskText(value.Input["kind"]), config)
	if ctx.Err() != nil {
		if current, getErr := s.service.Get(context.Background(), value.ResourceID); getErr == nil {
			_ = s.service.Delete(context.Background(), current.ID, current.Revision)
		}
		return nil, ctx.Err()
	}
	if err != nil {
		return nil, *command.NormalizeOperationProblem(err, domain.Problem{Code: "network_object_create_failed", Message: "network object creation failed", Retryable: true, ResourceType: "network_object", ResourceID: value.ResourceID, TaskID: value.ID, Phase: "configure", Cleanup: "owned runtime cleanup is pending verification", OperatorHint: "inspect network diagnostics and retry"}, false)
	}
	value.ProgressCurrent = value.ProgressTotal
	return map[string]any{"network_object": object}, nil
}

func (s *NetworkObjectTaskService) handleUpdate(ctx context.Context, value *domain.OperationTask) (map[string]any, error) {
	value.ProgressCurrent = 1
	if err := s.runner.Checkpoint(ctx, value); err != nil {
		return nil, err
	}
	config, err := networkTaskMap(value.Input["config"])
	if err != nil {
		return nil, err
	}
	updated, err := s.service.Update(ctx, value.ResourceID, domain.Revision(networkTaskInt64(value.Input["revision"])), networkTaskText(value.Input["name"]), config)
	if ctx.Err() != nil {
		previousConfig, _ := networkTaskMap(value.Input["previous_config"])
		if updated.Revision > 0 {
			_, _ = s.service.Update(context.Background(), value.ResourceID, updated.Revision, networkTaskText(value.Input["previous_name"]), previousConfig)
		}
		return nil, ctx.Err()
	}
	if err != nil {
		return nil, *command.NormalizeOperationProblem(err, domain.Problem{Code: "network_object_update_failed", Message: "network object configuration update failed", Retryable: true, ResourceType: "network_object", ResourceID: value.ResourceID, TaskID: value.ID, Phase: "configure", Cleanup: "requested configuration is retained with failed status", OperatorHint: "correct the configuration and retry"}, false)
	}
	value.ProgressCurrent = value.ProgressTotal
	return map[string]any{"network_object": updated}, nil
}

func (s *NetworkObjectTaskService) handleDelete(ctx context.Context, value *domain.OperationTask) (map[string]any, error) {
	value.ProgressCurrent = 1
	if err := s.runner.Checkpoint(ctx, value); err != nil {
		return nil, err
	}
	err := s.service.Delete(ctx, value.ResourceID, domain.Revision(networkTaskInt64(value.Input["revision"])))
	if ctx.Err() != nil {
		config, _ := networkTaskMap(value.Input["config"])
		_, _ = s.service.CreateAs(context.Background(), value.ResourceID, domain.ID(networkTaskText(value.Input["laboratory_id"])), networkTaskText(value.Input["name"]), networkTaskText(value.Input["kind"]), config)
		return nil, ctx.Err()
	}
	if err != nil {
		if problem := command.NormalizeOperationProblem(err, domain.Problem{Code: "network_object_delete_failed", Message: "network object deletion failed", Retryable: true, ResourceType: "network_object", ResourceID: value.ResourceID, TaskID: value.ID, Phase: "cleanup", Cleanup: "cleanup incomplete", OperatorHint: "inspect runtime ownership and retry deletion"}, true); problem != nil {
			return nil, *problem
		}
	}
	value.ProgressCurrent = value.ProgressTotal
	return map[string]any{"network_object_id": value.ResourceID, "deleted": true}, nil
}

func networkTaskText(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}

func networkTaskInt64(value any) int64 {
	switch typed := value.(type) {
	case int64:
		return typed
	case int:
		return int64(typed)
	case float64:
		return int64(typed)
	default:
		return 0
	}
}

func networkTaskMap(value any) (map[string]any, error) {
	if value == nil {
		return map[string]any{}, nil
	}
	body, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	result := map[string]any{}
	if err = json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return result, nil
}
