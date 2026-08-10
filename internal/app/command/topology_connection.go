package command

import (
	"context"

	"github.com/netlab/netlab/internal/domain"
)

type TopologyConnectionRepository interface {
	ResolveConnectionEndpoint(context.Context, domain.ConnectionEndpoint) (domain.ConnectionEndpoint, error)
	ListTopologyConnections(context.Context, domain.ID) ([]domain.TopologyConnection, error)
	GetTopologyConnection(context.Context, domain.ID) (domain.TopologyConnection, error)
}

type TopologyConnectionLinkOperations interface {
	ConnectLink(context.Context, domain.ID, domain.ID, domain.ID, string) (domain.Link, domain.OperationTask, error)
	DisconnectLink(context.Context, domain.ID, string) (domain.OperationTask, error)
}

type TopologyConnectionNetworkOperations interface {
	CreateAttachment(context.Context, domain.ID, domain.ID, domain.ID, string, map[string]any, string) (domain.NetworkAttachment, domain.OperationTask, error)
	DeleteAttachment(context.Context, domain.ID, domain.Revision, string) (domain.NetworkAttachment, domain.OperationTask, error)
	CreateObjectLink(context.Context, domain.ID, domain.ID, string, domain.ID, string, string) (domain.NetworkObjectLink, domain.OperationTask, error)
	DeleteObjectLink(context.Context, domain.ID, domain.Revision, string) (domain.NetworkObjectLink, domain.OperationTask, error)
}

type TopologyConnectionService struct {
	repository TopologyConnectionRepository
	links      TopologyConnectionLinkOperations
	networks   TopologyConnectionNetworkOperations
}

func NewTopologyConnectionService(repository TopologyConnectionRepository, links TopologyConnectionLinkOperations, networks TopologyConnectionNetworkOperations) *TopologyConnectionService {
	return &TopologyConnectionService{repository: repository, links: links, networks: networks}
}

func (s *TopologyConnectionService) Create(ctx context.Context, laboratoryID domain.ID, sourceRef, targetRef domain.ConnectionEndpoint, config domain.TopologyConnectionConfig, idempotencyKey string) (domain.TopologyConnection, domain.OperationTask, error) {
	sourceRef.LaboratoryID = laboratoryID
	targetRef.LaboratoryID = laboratoryID
	source, err := s.repository.ResolveConnectionEndpoint(ctx, sourceRef)
	if err != nil {
		return domain.TopologyConnection{}, domain.OperationTask{}, err
	}
	target, err := s.repository.ResolveConnectionEndpoint(ctx, targetRef)
	if err != nil {
		return domain.TopologyConnection{}, domain.OperationTask{}, err
	}
	if source.Availability != "" && source.Availability != domain.ConnectionEndpointFree {
		return domain.TopologyConnection{}, domain.OperationTask{}, endpointUnavailableProblem(source)
	}
	if target.Availability != "" && target.Availability != domain.ConnectionEndpointFree {
		return domain.TopologyConnection{}, domain.OperationTask{}, endpointUnavailableProblem(target)
	}
	backing, err := domain.ResolveTopologyConnectionBacking(source, target)
	if err != nil {
		return domain.TopologyConnection{}, domain.OperationTask{}, err
	}
	switch backing {
	case domain.ConnectionBackingLink:
		link, taskValue, createErr := s.links.ConnectLink(ctx, laboratoryID, source.PortID, target.PortID, idempotencyKey)
		if createErr != nil {
			return domain.TopologyConnection{}, domain.OperationTask{}, createErr
		}
		return connectionFromLink(link, source, target), taskValue, nil
	case domain.ConnectionBackingAttachment:
		nodeEndpoint, objectEndpoint := source, target
		if nodeEndpoint.Kind != domain.ConnectionEndpointNodeInterface {
			nodeEndpoint, objectEndpoint = target, source
		}
		attachmentConfig := map[string]any{}
		if config.PVID != 0 {
			attachmentConfig["pvid"] = config.PVID
		}
		if len(config.TaggedVLANs) > 0 {
			attachmentConfig["tagged"] = config.TaggedVLANs
		}
		attachment, taskValue, createErr := s.networks.CreateAttachment(ctx, laboratoryID, objectEndpoint.ResourceID, nodeEndpoint.PortID, objectEndpoint.PortName, attachmentConfig, idempotencyKey)
		if createErr != nil {
			return domain.TopologyConnection{}, domain.OperationTask{}, createErr
		}
		return connectionFromAttachment(laboratoryID, attachment, source, target, config), taskValue, nil
	case domain.ConnectionBackingObjectLink:
		link, taskValue, createErr := s.networks.CreateObjectLink(ctx, laboratoryID, source.ResourceID, source.PortName, target.ResourceID, target.PortName, idempotencyKey)
		if createErr != nil {
			return domain.TopologyConnection{}, domain.OperationTask{}, createErr
		}
		return connectionFromObjectLink(link, source, target), taskValue, nil
	default:
		return domain.TopologyConnection{}, domain.OperationTask{}, domain.Problem{Code: "endpoint_incompatible", Message: "unsupported connection endpoint combination", Phase: "connection_admission"}
	}
}

func (s *TopologyConnectionService) List(ctx context.Context, laboratoryID domain.ID) ([]domain.TopologyConnection, error) {
	return s.repository.ListTopologyConnections(ctx, laboratoryID)
}

func (s *TopologyConnectionService) Get(ctx context.Context, id domain.ID) (domain.TopologyConnection, error) {
	return s.repository.GetTopologyConnection(ctx, id)
}

func (s *TopologyConnectionService) Delete(ctx context.Context, id domain.ID, revision domain.Revision, idempotencyKey string) (domain.TopologyConnection, domain.OperationTask, error) {
	connection, err := s.repository.GetTopologyConnection(ctx, id)
	if err != nil {
		return domain.TopologyConnection{}, domain.OperationTask{}, err
	}
	if revision > 0 && connection.Revision != revision {
		return domain.TopologyConnection{}, domain.OperationTask{}, domain.Problem{Code: "revision_conflict", Message: "topology connection revision changed", ResourceType: "topology_connection", ResourceID: id, Phase: "delete_admission"}
	}
	switch connection.BackingKind {
	case domain.ConnectionBackingLink:
		taskValue, deleteErr := s.links.DisconnectLink(ctx, id, idempotencyKey)
		connection.DesiredState = "disconnected"
		connection.ObservedState = "disconnecting"
		return connection, taskValue, deleteErr
	case domain.ConnectionBackingAttachment:
		_, taskValue, deleteErr := s.networks.DeleteAttachment(ctx, id, connection.Revision, idempotencyKey)
		connection.DesiredState = "disconnected"
		connection.ObservedState = "disconnecting"
		return connection, taskValue, deleteErr
	case domain.ConnectionBackingObjectLink:
		_, taskValue, deleteErr := s.networks.DeleteObjectLink(ctx, id, connection.Revision, idempotencyKey)
		connection.DesiredState = "disconnected"
		connection.ObservedState = "disconnecting"
		return connection, taskValue, deleteErr
	default:
		return domain.TopologyConnection{}, domain.OperationTask{}, domain.Problem{Code: "invalid_topology", Message: "unknown topology connection backing kind", ResourceType: "topology_connection", ResourceID: id}
	}
}

func endpointUnavailableProblem(endpoint domain.ConnectionEndpoint) error {
	code := "endpoint_unavailable"
	if endpoint.Availability == domain.ConnectionEndpointOccupied || endpoint.Availability == domain.ConnectionEndpointReserved {
		code = "port_in_use"
	}
	return domain.Problem{Code: code, Message: "connection endpoint is not available", ResourceType: string(endpoint.Kind), ResourceID: endpoint.ResourceID, Phase: "connection_admission", Cleanup: "no topology mutation was submitted", OperatorHint: "refresh the topology and choose a free endpoint"}
}

func connectionCapabilities() []string {
	return []string{"select", "delete", "capture", "wireshark", "traffic_filter"}
}

func connectionFromLink(value domain.Link, source, target domain.ConnectionEndpoint) domain.TopologyConnection {
	return domain.TopologyConnection{ID: value.ID, LaboratoryID: value.LaboratoryID, Source: source, Target: target, BackingKind: domain.ConnectionBackingLink, BackingID: value.ID, Revision: value.Revision, DesiredState: value.DesiredState, ObservedState: value.ObservedState, Capabilities: connectionCapabilities()}
}

func connectionFromAttachment(laboratoryID domain.ID, value domain.NetworkAttachment, source, target domain.ConnectionEndpoint, config domain.TopologyConnectionConfig) domain.TopologyConnection {
	return domain.TopologyConnection{ID: value.ID, LaboratoryID: laboratoryID, Source: source, Target: target, BackingKind: domain.ConnectionBackingAttachment, BackingID: value.ID, Config: config, Revision: value.Revision, DesiredState: "connected", ObservedState: value.ObservedState, Capabilities: connectionCapabilities()}
}

func connectionFromObjectLink(value domain.NetworkObjectLink, source, target domain.ConnectionEndpoint) domain.TopologyConnection {
	return domain.TopologyConnection{ID: value.ID, LaboratoryID: value.LaboratoryID, Source: source, Target: target, BackingKind: domain.ConnectionBackingObjectLink, BackingID: value.ID, Revision: value.Revision, DesiredState: value.DesiredState, ObservedState: value.ObservedState, Capabilities: connectionCapabilities()}
}
