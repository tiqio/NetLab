package reconcile

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/netlab/netlab/internal/domain"
)

var networkObjectPortNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

type NetworkObjectRepository interface {
	CreateNetworkObject(context.Context, domain.NetworkObject) error
	UpdateNetworkObject(context.Context, domain.NetworkObject, domain.Revision) (domain.NetworkObject, error)
	GetNetworkObject(context.Context, domain.ID) (domain.NetworkObject, error)
	ListNetworkObjects(context.Context, domain.ID) ([]domain.NetworkObject, error)
	SetNetworkObjectState(context.Context, domain.ID, string, *domain.Problem) error
	DeleteNetworkObject(context.Context, domain.ID, domain.Revision) error
	CreateNetworkAttachment(context.Context, domain.ID, domain.ID, string, map[string]any) error
	ListNetworkObjectAttachments(context.Context, domain.ID) ([]domain.NetworkAttachment, error)
	CreateNetworkObjectLink(context.Context, domain.NetworkObjectLink) error
	GetNetworkObjectLink(context.Context, domain.ID) (domain.NetworkObjectLink, error)
	ListNetworkObjectLinks(context.Context, domain.ID) ([]domain.NetworkObjectLink, error)
	DeleteNetworkObjectLink(context.Context, domain.ID) error
}

type natObservationRepository interface {
	SaveNATServiceObservation(context.Context, domain.NATServiceObservation) error
	DeleteNATServiceObservation(context.Context, domain.ID) error
}
type natObservationRuntime interface {
	HelperObservation(domain.ID) (domain.NATServiceObservation, bool)
}

type NetworkObjectRuntime interface {
	Configure(context.Context, domain.NetworkObject) error
	Delete(context.Context, domain.ID) error
}

type NetworkRuntimeDispatch struct {
	Bridge   NetworkObjectRuntime
	NAT      NetworkObjectRuntime
	PC       NetworkObjectRuntime
	SwitchL2 NetworkObjectRuntime
	SwitchL3 NetworkObjectRuntime
}

type NetworkObjectService struct {
	repository  NetworkObjectRepository
	runtimes    NetworkRuntimeDispatch
	attachments interface {
		DeleteAttachment(context.Context, domain.NetworkAttachment) error
	}
	objectLinks interface {
		DeleteNetworkObjectLink(context.Context, domain.NetworkObjectLink, domain.NetworkObject, domain.NetworkObject) error
	}
	objectLinkObservers []interface{ StopNetworkObjectLink(domain.ID) }
}

func (s *NetworkObjectService) SetAttachmentRuntime(runtime interface {
	DeleteAttachment(context.Context, domain.NetworkAttachment) error
}) {
	s.attachments = runtime
}

func (s *NetworkObjectService) SetObjectLinkRuntime(runtime interface {
	DeleteNetworkObjectLink(context.Context, domain.NetworkObjectLink, domain.NetworkObject, domain.NetworkObject) error
}) {
	s.objectLinks = runtime
}

func (s *NetworkObjectService) AddObjectLinkObserverCleanup(cleaner interface{ StopNetworkObjectLink(domain.ID) }) {
	s.objectLinkObservers = append(s.objectLinkObservers, cleaner)
}

func NewNetworkObjectService(repository NetworkObjectRepository, runtimes NetworkRuntimeDispatch) *NetworkObjectService {
	return &NetworkObjectService{repository: repository, runtimes: runtimes}
}

func (s *NetworkObjectService) Create(ctx context.Context, labID domain.ID, name, kind string, config map[string]any) (domain.NetworkObject, error) {
	return s.CreateAs(ctx, domain.NewID(), labID, name, kind, config)
}

func (s *NetworkObjectService) CreateAs(ctx context.Context, id, labID domain.ID, name, kind string, config map[string]any) (value domain.NetworkObject, err error) {
	defer normalizeTerminalError(&err, terminalProblem("network_object", id, "provisioning"))
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 128 {
		return domain.NetworkObject{}, fmt.Errorf("network object name must be 1-128 characters")
	}
	if err := domain.ValidateNetworkKind(kind); err != nil {
		return domain.NetworkObject{}, err
	}
	value, err = s.repository.GetNetworkObject(ctx, id)
	if err == nil && value.ObservedState == "active" {
		return value, nil
	}
	if err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "not found") {
			return domain.NetworkObject{}, err
		}
		value = domain.NetworkObject{ID: id, LaboratoryID: labID, Name: name, Kind: kind, Revision: 1, DesiredState: "active", ObservedState: "provisioning", Config: config, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
		if err = s.repository.CreateNetworkObject(ctx, value); err != nil {
			return domain.NetworkObject{}, err
		}
	}
	runtime := s.runtime(kind)
	if runtime == nil {
		problem := structuredProblem(nil, domain.Problem{Code: "capability_unsupported", Message: "network runtime is unavailable", ResourceType: "network_object", ResourceID: value.ID, Phase: "provisioning", Cleanup: "database record retained in failed state", OperatorHint: "enable the required network runtime or delete the object"})
		_ = s.repository.SetNetworkObjectState(ctx, value.ID, "failed", problem)
		return value, *problem
	}
	if err := runtime.Configure(ctx, value); err != nil {
		problem := structuredProblem(err, domain.Problem{Code: "reconciliation_failed", Retryable: true, ResourceType: "network_object", ResourceID: value.ID, Phase: "provisioning", Cleanup: "runtime adapter retains owned partial state for retry", OperatorHint: "inspect owned network resources and retry", RetryAfterSeconds: 3})
		_ = s.repository.SetNetworkObjectState(context.Background(), value.ID, "failed", problem)
		return value, *problem
	}
	value.ObservedState = "active"
	_ = s.repository.SetNetworkObjectState(context.Background(), value.ID, "active", nil)
	if value.Kind == domain.NetworkNAT {
		if runtime, ok := runtime.(natObservationRuntime); ok {
			if observation, exists := runtime.HelperObservation(value.ID); exists {
				if repository, ok := s.repository.(natObservationRepository); ok {
					_ = repository.SaveNATServiceObservation(context.Background(), observation)
				}
			}
		}
	}
	return value, nil
}

func (s *NetworkObjectService) List(ctx context.Context, labID domain.ID) (values []domain.NetworkObject, err error) {
	defer normalizeTerminalError(&err, terminalProblem("laboratory", labID, "network_object_list"))
	return s.repository.ListNetworkObjects(ctx, labID)
}

func (s *NetworkObjectService) Get(ctx context.Context, id domain.ID) (value domain.NetworkObject, err error) {
	defer normalizeTerminalError(&err, terminalProblem("network_object", id, "network_object_get"))
	return s.repository.GetNetworkObject(ctx, id)
}

func (s *NetworkObjectService) Update(ctx context.Context, id domain.ID, revision domain.Revision, name string, config map[string]any) (value domain.NetworkObject, err error) {
	defer normalizeTerminalError(&err, terminalProblem("network_object", id, "updating"))
	current, err := s.repository.GetNetworkObject(ctx, id)
	if err != nil {
		return domain.NetworkObject{}, err
	}
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 128 {
		return domain.NetworkObject{}, fmt.Errorf("network object name must be 1-128 characters")
	}
	if err = validateSwitchConfiguration(current.Kind, config); err != nil {
		return domain.NetworkObject{}, err
	}
	current.Name = name
	current.Config = config
	value, err = s.repository.UpdateNetworkObject(ctx, current, revision)
	if err != nil {
		return domain.NetworkObject{}, err
	}
	runtime := s.runtime(value.Kind)
	if runtime == nil {
		problem := structuredProblem(nil, domain.Problem{Code: "capability_unsupported", Message: "network runtime is unavailable", ResourceType: "network_object", ResourceID: value.ID, Phase: "updating", Cleanup: "updated configuration retained in failed state", OperatorHint: "enable the required network runtime or correct the object kind"})
		_ = s.repository.SetNetworkObjectState(context.Background(), value.ID, "failed", problem)
		return value, *problem
	}
	if err = runtime.Configure(ctx, value); err != nil {
		problem := structuredProblem(err, domain.Problem{Code: "reconciliation_failed", Retryable: true, ResourceType: "network_object", ResourceID: value.ID, Phase: "updating", Cleanup: "updated configuration retained for retry", OperatorHint: "inspect the network namespace and retry", RetryAfterSeconds: 3})
		_ = s.repository.SetNetworkObjectState(context.Background(), value.ID, "failed", problem)
		return value, *problem
	}
	value.ObservedState = "active"
	_ = s.repository.SetNetworkObjectState(context.Background(), value.ID, "active", nil)
	return value, nil
}

func validateSwitchConfiguration(kind string, config map[string]any) error {
	decode := func(destination any) error {
		body, err := json.Marshal(config)
		if err != nil {
			return err
		}
		return json.Unmarshal(body, destination)
	}
	switch kind {
	case domain.NetworkSwitchL2:
		var value domain.SwitchL2Config
		if err := decode(&value); err != nil {
			return err
		}
		return domain.ValidateSwitchL2Config(value)
	case domain.NetworkSwitchL3:
		var value domain.SwitchL3Config
		if err := decode(&value); err != nil {
			return err
		}
		return domain.ValidateSwitchL3Config(value)
	default:
		return nil
	}
}

func (s *NetworkObjectService) Delete(ctx context.Context, id domain.ID, revision domain.Revision) (err error) {
	defer normalizeTerminalError(&err, terminalProblem("network_object", id, "deleting"))
	value, err := s.repository.GetNetworkObject(ctx, id)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			return nil
		}
		return err
	}
	if runtime := s.runtime(value.Kind); runtime != nil {
		if err = runtime.Delete(ctx, id); err != nil {
			return err
		}
	}
	if err = s.deleteAttachments(ctx, id); err != nil {
		return err
	}
	if repository, ok := s.repository.(natObservationRepository); ok {
		_ = repository.DeleteNATServiceObservation(ctx, id)
	}
	return s.repository.DeleteNetworkObject(ctx, id, revision)
}

func (s *NetworkObjectService) Attach(ctx context.Context, objectID, interfaceID domain.ID, portName string, config map[string]any) (err error) {
	defer normalizeTerminalError(&err, terminalProblem("network_object", objectID, "attachment_create"))
	return s.repository.CreateNetworkAttachment(ctx, objectID, interfaceID, portName, config)
}

func (s *NetworkObjectService) ListObjectLinks(ctx context.Context, laboratoryID domain.ID) ([]domain.NetworkObjectLink, error) {
	return s.repository.ListNetworkObjectLinks(ctx, laboratoryID)
}

func (s *NetworkObjectService) CreateObjectLink(ctx context.Context, laboratoryID, objectAID domain.ID, portAName string, objectBID domain.ID, portBName string) (domain.NetworkObjectLink, error) {
	portAName = strings.TrimSpace(portAName)
	portBName = strings.TrimSpace(portBName)
	if objectAID == "" || objectBID == "" || objectAID == objectBID {
		return domain.NetworkObjectLink{}, fmt.Errorf("two different network objects are required")
	}
	if !networkObjectPortNamePattern.MatchString(portAName) || !networkObjectPortNamePattern.MatchString(portBName) {
		return domain.NetworkObjectLink{}, fmt.Errorf("network object port names must contain only letters, numbers, dots, underscores, or hyphens")
	}
	value := domain.NetworkObjectLink{ID: domain.NewID(), LaboratoryID: laboratoryID, ObjectAID: objectAID, PortAName: portAName, ObjectBID: objectBID, PortBName: portBName, Revision: 1, DesiredState: "connected", ObservedState: "pending"}
	if err := s.repository.CreateNetworkObjectLink(ctx, value); err != nil {
		return domain.NetworkObjectLink{}, err
	}
	return value, nil
}

func (s *NetworkObjectService) DeleteObjectLink(ctx context.Context, id domain.ID) error {
	link, err := s.repository.GetNetworkObjectLink(ctx, id)
	if err != nil {
		return err
	}
	if s.objectLinks != nil {
		for _, cleaner := range s.objectLinkObservers {
			cleaner.StopNetworkObjectLink(id)
		}
		objectA, err := s.repository.GetNetworkObject(ctx, link.ObjectAID)
		if err != nil {
			return err
		}
		objectB, err := s.repository.GetNetworkObject(ctx, link.ObjectBID)
		if err != nil {
			return err
		}
		if err := s.objectLinks.DeleteNetworkObjectLink(ctx, link, objectA, objectB); err != nil {
			return err
		}
	}
	return s.repository.DeleteNetworkObjectLink(ctx, id)
}

func (s *NetworkObjectService) runtime(kind string) NetworkObjectRuntime {
	switch kind {
	case domain.NetworkBridge:
		return s.runtimes.Bridge
	case domain.NetworkNAT:
		return s.runtimes.NAT
	case domain.NetworkPC:
		return s.runtimes.PC
	case domain.NetworkSwitchL2:
		return s.runtimes.SwitchL2
	case domain.NetworkSwitchL3:
		return s.runtimes.SwitchL3
	default:
		return nil
	}
}

func (s *NetworkObjectService) DeleteOwned(ctx context.Context, value domain.NetworkObject) (err error) {
	defer normalizeTerminalError(&err, terminalProblem("network_object", value.ID, "owned_cleanup"))
	if err := s.deleteAttachments(ctx, value.ID); err != nil {
		return err
	}
	if runtime := s.runtime(value.Kind); runtime != nil {
		if err := runtime.Delete(ctx, value.ID); err != nil {
			return err
		}
	}
	return nil
}

func (s *NetworkObjectService) deleteAttachments(ctx context.Context, objectID domain.ID) error {
	if s.attachments == nil {
		return nil
	}
	values, err := s.repository.ListNetworkObjectAttachments(ctx, objectID)
	if err != nil {
		return err
	}
	for _, value := range values {
		if err = s.attachments.DeleteAttachment(ctx, value); err != nil {
			return err
		}
	}
	return nil
}

func (s *NetworkObjectService) RestoreLaboratory(ctx context.Context, laboratoryID domain.ID) error {
	return s.RestoreLaboratoryWithCheckpoints(ctx, laboratoryID, func(RecoveryResourceOutcome) error { return nil })
}

func (s *NetworkObjectService) RestoreLaboratoryWithCheckpoints(ctx context.Context, laboratoryID domain.ID, checkpoint func(RecoveryResourceOutcome) error) (err error) {
	defer normalizeTerminalError(&err, terminalProblem("laboratory", laboratoryID, "network_recovery"))
	values, err := s.repository.ListNetworkObjects(ctx, laboratoryID)
	if err != nil {
		return err
	}
	for _, value := range values {
		if value.DesiredState != "active" {
			continue
		}
		runtime := s.runtime(value.Kind)
		if runtime == nil {
			if err = checkpoint(RecoveryResourceOutcome{ResourceType: "network_object", ResourceID: value.ID, State: "skipped", Error: "runtime unavailable"}); err != nil {
				return err
			}
			continue
		}
		if err = runtime.Configure(ctx, value); err != nil {
			problem := structuredProblem(err, domain.Problem{Code: "recovery_failed", Retryable: true, ResourceType: "network_object", ResourceID: value.ID, Phase: "recovery", Cleanup: "owned partial network state is retained for retry", OperatorHint: "inspect the network namespace or bridge and retry recovery", RetryAfterSeconds: 3})
			_ = s.repository.SetNetworkObjectState(ctx, value.ID, "failed", problem)
			if checkpointErr := checkpoint(RecoveryResourceOutcome{ResourceType: "network_object", ResourceID: value.ID, State: "failed", Error: err.Error()}); checkpointErr != nil {
				return checkpointErr
			}
			continue
		}
		_ = s.repository.SetNetworkObjectState(ctx, value.ID, "active", nil)
		if err = checkpoint(RecoveryResourceOutcome{ResourceType: "network_object", ResourceID: value.ID, State: "recovered"}); err != nil {
			return err
		}
	}
	return nil
}
