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
	runner.Register("network_attachment.create", value.handleAttachmentCreate)
	runner.Register("network_attachment.delete", value.handleAttachmentDelete)
	runner.Register("network_object_link.create", value.handleObjectLinkCreate)
	runner.Register("network_object_link.delete", value.handleObjectLinkDelete)
	return value
}

func (s *NetworkObjectTaskService) CreateAttachment(ctx context.Context, laboratoryID, objectID, interfaceID domain.ID, portName string, config map[string]any, idempotencyKey string) (domain.NetworkAttachment, domain.OperationTask, error) {
	input := map[string]any{"laboratory_id": laboratoryID, "network_object_id": objectID, "interface_id": interfaceID, "port_name": strings.TrimSpace(portName), "config": config, "entry_point": command.TopologyConnectionEntryPoint(ctx, "compatibility_http")}
	attachment := domain.NetworkAttachment{ID: domain.NewID(), NetworkObjectID: objectID, InterfaceID: interfaceID, PortName: strings.TrimSpace(portName), Config: config, Revision: 1, ObservedState: "pending"}
	operation := networkObjectOperation("network_attachment.create", attachment.ID, idempotencyKey, input)
	operation.ResourceType = "network_attachment"
	queued, err := s.runner.EnqueueOrGet(ctx, operation)
	if err != nil {
		return domain.NetworkAttachment{}, domain.OperationTask{}, err
	}
	if queued.ID != operation.ID {
		attachment.ID = queued.ResourceID
		attachment.NetworkObjectID = domain.ID(networkTaskText(queued.Input["network_object_id"]))
		attachment.InterfaceID = domain.ID(networkTaskText(queued.Input["interface_id"]))
		attachment.PortName = networkTaskText(queued.Input["port_name"])
		attachment.Config, _ = networkTaskMap(queued.Input["config"])
	}
	return attachment, queued, nil
}

func (s *NetworkObjectTaskService) DeleteAttachment(ctx context.Context, id domain.ID, revision domain.Revision, idempotencyKey string) (domain.NetworkAttachment, domain.OperationTask, error) {
	attachment, err := s.service.GetAttachment(ctx, id)
	if err != nil {
		if idempotencyKey != "" {
			if existing, lookupErr := s.runner.GetByIdempotency(ctx, "network_attachment.delete", idempotencyKey); lookupErr == nil && existing.ResourceID == id {
				if domain.Revision(networkTaskInt64(existing.Input["revision"])) != revision {
					return domain.NetworkAttachment{}, domain.OperationTask{}, domain.Problem{Code: "idempotency_conflict", Message: "idempotency key was already used with a different attachment revision", ResourceType: "network_attachment", ResourceID: id, TaskID: existing.ID, Phase: "delete_admission"}
				}
				return domain.NetworkAttachment{ID: id, Revision: revision, ObservedState: "disconnected"}, existing, nil
			}
		}
		return domain.NetworkAttachment{}, domain.OperationTask{}, err
	}
	if attachment.Revision != revision {
		return domain.NetworkAttachment{}, domain.OperationTask{}, domain.Problem{Code: "revision_conflict", Message: fmt.Sprintf("expected revision %d, current revision is %d", revision, attachment.Revision), ResourceType: "network_attachment", ResourceID: id, Phase: "delete_admission"}
	}
	input := map[string]any{"revision": int64(revision), "network_object_id": attachment.NetworkObjectID, "interface_id": attachment.InterfaceID, "port_name": attachment.PortName, "config": attachment.Config, "entry_point": command.TopologyConnectionEntryPoint(ctx, "compatibility_http")}
	operation := networkObjectOperation("network_attachment.delete", id, idempotencyKey, input)
	operation.ResourceType = "network_attachment"
	queued, err := s.runner.EnqueueOrGet(ctx, operation)
	if err != nil {
		return domain.NetworkAttachment{}, domain.OperationTask{}, err
	}
	attachment.ObservedState = "disconnecting"
	return attachment, queued, nil
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
	input := map[string]any{"revision": int64(revision), "laboratory_id": link.LaboratoryID, "object_a_id": link.ObjectAID, "port_a_name": link.PortAName, "object_b_id": link.ObjectBID, "port_b_name": link.PortBName, "entry_point": command.TopologyConnectionEntryPoint(ctx, "compatibility_http")}
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
	input := map[string]any{"laboratory_id": laboratoryID, "object_a_id": objectAID, "port_a_name": portAName, "object_b_id": objectBID, "port_b_name": portBName, "entry_point": command.TopologyConnectionEntryPoint(ctx, "compatibility_http")}
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

func (s *NetworkObjectTaskService) handleAttachmentCreate(ctx context.Context, value *domain.OperationTask) (map[string]any, error) {
	value.ProgressCurrent = 1
	if err := s.runner.Checkpoint(ctx, value); err != nil {
		return nil, err
	}
	config, err := networkTaskMap(value.Input["config"])
	if err != nil {
		return nil, err
	}
	attachment, err := s.service.AttachAs(ctx, value.ResourceID, domain.ID(networkTaskText(value.Input["network_object_id"])), domain.ID(networkTaskText(value.Input["interface_id"])), networkTaskText(value.Input["port_name"]), config, value.ID)
	if err != nil {
		return nil, *command.NormalizeOperationProblem(err, domain.Problem{Code: "network_attachment_create_failed", Message: "network attachment creation failed", ResourceType: "network_attachment", ResourceID: value.ResourceID, TaskID: value.ID, Phase: "endpoint_reservation", Cleanup: "no endpoint reservations retained", OperatorHint: "refresh the topology and choose free endpoints"}, false)
	}
	value.ProgressCurrent = value.ProgressTotal
	return map[string]any{"network_attachment": attachment}, nil
}

func (s *NetworkObjectTaskService) handleAttachmentDelete(ctx context.Context, value *domain.OperationTask) (map[string]any, error) {
	value.ProgressCurrent = 1
	if err := s.runner.Checkpoint(ctx, value); err != nil {
		return nil, err
	}
	if err := s.service.DeleteAttachment(ctx, value.ResourceID, domain.Revision(networkTaskInt64(value.Input["revision"])), value.ID); err != nil {
		return nil, *command.NormalizeOperationProblem(err, domain.Problem{Code: "network_attachment_delete_failed", Message: "network attachment deletion failed", Retryable: true, ResourceType: "network_attachment", ResourceID: value.ResourceID, TaskID: value.ID, Phase: "cleanup", Cleanup: "attachment remains authoritative until cleanup succeeds", OperatorHint: "inspect runtime ownership and retry deletion"}, true)
	}
	value.ProgressCurrent = value.ProgressTotal
	return map[string]any{"network_attachment_id": value.ResourceID, "deleted": true}, nil
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
	config = domain.ApplyLightweightPortDefaultsOnCreate(kind, config)
	if idempotencyKey != "" {
		if existing, lookupErr := s.runner.GetByIdempotency(ctx, "network_object.create", idempotencyKey); lookupErr == nil {
			existingConfig, _ := networkTaskMap(existing.Input["config"])
			requestedConfig, _ := json.Marshal(config)
			storedConfig, _ := json.Marshal(existingConfig)
			if domain.ID(networkTaskText(existing.Input["laboratory_id"])) != laboratoryID || networkTaskText(existing.Input["name"]) != strings.TrimSpace(name) || networkTaskText(existing.Input["kind"]) != kind || string(requestedConfig) != string(storedConfig) {
				return domain.NetworkObject{}, domain.OperationTask{}, domain.Problem{Code: "idempotency_conflict", Message: "idempotency key was already used with a different create request", ResourceType: "network_object", ResourceID: existing.ResourceID, TaskID: existing.ID, Phase: "create_admission"}
			}
			object, err := s.service.Get(ctx, existing.ResourceID)
			return object, existing, err
		}
	}
	revision, err := s.service.LaboratoryRevision(ctx, laboratoryID)
	if err != nil {
		return domain.NetworkObject{}, domain.OperationTask{}, err
	}
	object, _, _, taskValue, err := s.CreateWithPlacement(ctx, laboratoryID, revision, name, kind, config, nil, idempotencyKey, "application")
	return object, taskValue, err
}

func (s *NetworkObjectTaskService) CreateWithPlacement(ctx context.Context, laboratoryID domain.ID, expectedRevision domain.Revision, name, kind string, config map[string]any, intent *domain.PlacementIntent, idempotencyKey, entry string) (domain.NetworkObject, domain.PlacementAssignment, domain.Revision, domain.OperationTask, error) {
	config = domain.ApplyLightweightPortDefaultsOnCreate(kind, config)
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 128 {
		return domain.NetworkObject{}, domain.PlacementAssignment{}, 0, domain.OperationTask{}, fmt.Errorf("network object name must be 1-128 characters")
	}
	if err := domain.ValidateNetworkKind(kind); err != nil {
		return domain.NetworkObject{}, domain.PlacementAssignment{}, 0, domain.OperationTask{}, err
	}
	if err := domain.ValidateUniqueLightweightPortNames(kind, config); err != nil {
		return domain.NetworkObject{}, domain.PlacementAssignment{}, 0, domain.OperationTask{}, err
	}
	object := domain.NetworkObject{ID: domain.NewID(), LaboratoryID: laboratoryID, Name: name, Kind: kind, Revision: 1, DesiredState: "active", ObservedState: "provisioning", Config: config}
	input := map[string]any{"laboratory_id": laboratoryID, "laboratory_revision": int64(expectedRevision), "name": name, "kind": kind, "config": config, "placement_intent": intent}
	value := networkObjectOperation("network_object.create", object.ID, idempotencyKey, input)
	if idempotencyKey != "" {
		if existing, lookupErr := s.runner.GetByIdempotency(ctx, value.Kind, idempotencyKey); lookupErr == nil {
			if existing.RequestFingerprint != value.RequestFingerprint {
				return domain.NetworkObject{}, domain.PlacementAssignment{}, 0, domain.OperationTask{}, domain.Problem{Code: "idempotency_conflict", Message: "idempotency key was already used with a different create request", ResourceType: "network_object", ResourceID: existing.ResourceID, TaskID: existing.ID, Phase: "create_admission"}
			}
			existingObject, getErr := s.service.Get(ctx, existing.ResourceID)
			if getErr != nil {
				return domain.NetworkObject{}, domain.PlacementAssignment{}, 0, domain.OperationTask{}, getErr
			}
			assignment, decodeErr := networkTaskPlacementAssignment(existing.Input["placement_assignment"])
			if decodeErr != nil {
				return domain.NetworkObject{}, domain.PlacementAssignment{}, 0, domain.OperationTask{}, decodeErr
			}
			return existingObject, assignment, domain.Revision(networkTaskInt64(existing.Input["assigned_laboratory_revision"])), existing, nil
		}
	}
	created, assignment, laboratoryRevision, err := s.service.CreateRecordWithPlacement(ctx, object.ID, laboratoryID, name, kind, config, expectedRevision, intent, entry)
	if err != nil {
		return domain.NetworkObject{}, domain.PlacementAssignment{}, 0, domain.OperationTask{}, err
	}
	value.Input["placement_assignment"] = assignment
	value.Input["assigned_laboratory_revision"] = int64(laboratoryRevision)
	value.Input["entry"] = entry
	queued, err := s.runner.EnqueueOrGet(ctx, value)
	if err != nil {
		_ = s.service.Delete(context.Background(), created.ID, created.Revision)
		return domain.NetworkObject{}, domain.PlacementAssignment{}, 0, domain.OperationTask{}, err
	}
	if queued.ID != value.ID {
		created, err = s.service.Get(ctx, queued.ResourceID)
		if err != nil {
			return domain.NetworkObject{}, domain.PlacementAssignment{}, 0, domain.OperationTask{}, err
		}
		assignment, err = networkTaskPlacementAssignment(queued.Input["placement_assignment"])
		if err != nil {
			return domain.NetworkObject{}, domain.PlacementAssignment{}, 0, domain.OperationTask{}, err
		}
		laboratoryRevision = domain.Revision(networkTaskInt64(queued.Input["assigned_laboratory_revision"]))
	}
	return created, assignment, laboratoryRevision, queued, nil
}

func networkTaskPlacementAssignment(raw any) (domain.PlacementAssignment, error) {
	encoded, err := json.Marshal(raw)
	if err != nil {
		return domain.PlacementAssignment{}, err
	}
	var assignment domain.PlacementAssignment
	if err = json.Unmarshal(encoded, &assignment); err != nil {
		return domain.PlacementAssignment{}, err
	}
	if assignment.Placement.ResourceID == "" {
		return domain.PlacementAssignment{}, fmt.Errorf("task placement assignment is unavailable")
	}
	return assignment, nil
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
