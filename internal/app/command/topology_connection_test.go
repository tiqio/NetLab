package command

import (
	"context"
	"testing"

	"github.com/netlab/netlab/internal/domain"
)

type topologyConnectionRepositoryStub struct {
	endpoints map[string]domain.ConnectionEndpoint
}

func (s topologyConnectionRepositoryStub) ResolveConnectionEndpoint(_ context.Context, endpoint domain.ConnectionEndpoint) (domain.ConnectionEndpoint, error) {
	value, ok := s.endpoints[endpoint.Key()]
	if !ok {
		return domain.ConnectionEndpoint{}, domain.ErrNotFound
	}
	return value, nil
}
func (topologyConnectionRepositoryStub) ListTopologyConnections(context.Context, domain.ID) ([]domain.TopologyConnection, error) {
	return nil, nil
}
func (topologyConnectionRepositoryStub) GetTopologyConnection(context.Context, domain.ID) (domain.TopologyConnection, error) {
	return domain.TopologyConnection{}, domain.ErrNotFound
}

type topologyLinkOperationsStub struct {
	link domain.Link
	task domain.OperationTask
}

func (s *topologyLinkOperationsStub) ConnectLink(_ context.Context, labID, a, b domain.ID, _ string) (domain.Link, domain.OperationTask, error) {
	s.link = domain.Link{ID: "link", LaboratoryID: labID, EndpointAID: a, EndpointBID: b, Revision: 1, DesiredState: "connected", ObservedState: "pending"}
	s.task = domain.OperationTask{ID: "task-link", Kind: "link.connect", ResourceType: "link", ResourceID: s.link.ID, State: domain.TaskQueued}
	return s.link, s.task, nil
}
func (*topologyLinkOperationsStub) DisconnectLink(context.Context, domain.ID, string) (domain.OperationTask, error) {
	return domain.OperationTask{}, nil
}

type topologyNetworkOperationsStub struct {
	attachment domain.NetworkAttachment
	objectLink domain.NetworkObjectLink
}

func (s *topologyNetworkOperationsStub) CreateAttachment(_ context.Context, _ domain.ID, objectID, interfaceID domain.ID, portName string, config map[string]any, _ string) (domain.NetworkAttachment, domain.OperationTask, error) {
	s.attachment = domain.NetworkAttachment{ID: "attachment", NetworkObjectID: objectID, InterfaceID: interfaceID, PortName: portName, Config: config, ObservedState: "pending"}
	return s.attachment, domain.OperationTask{ID: "task-attachment", Kind: "network_attachment.create", ResourceType: "network_attachment", ResourceID: s.attachment.ID, State: domain.TaskQueued}, nil
}
func (*topologyNetworkOperationsStub) DeleteAttachment(context.Context, domain.ID, string) (domain.NetworkAttachment, domain.OperationTask, error) {
	return domain.NetworkAttachment{}, domain.OperationTask{}, nil
}
func (s *topologyNetworkOperationsStub) CreateObjectLink(_ context.Context, labID, objectAID domain.ID, portAName string, objectBID domain.ID, portBName, _ string) (domain.NetworkObjectLink, domain.OperationTask, error) {
	s.objectLink = domain.NetworkObjectLink{ID: "object-link", LaboratoryID: labID, ObjectAID: objectAID, PortAName: portAName, ObjectBID: objectBID, PortBName: portBName, Revision: 1, DesiredState: "connected", ObservedState: "pending"}
	return s.objectLink, domain.OperationTask{ID: "task-object-link", Kind: "network_object_link.create", ResourceType: "network_object_link", ResourceID: s.objectLink.ID, State: domain.TaskQueued}, nil
}
func (*topologyNetworkOperationsStub) DeleteObjectLink(context.Context, domain.ID, domain.Revision, string) (domain.NetworkObjectLink, domain.OperationTask, error) {
	return domain.NetworkObjectLink{}, domain.OperationTask{}, nil
}

func TestTopologyConnectionServiceChoosesBackingFromEndpoints(t *testing.T) {
	labID := domain.ID("lab")
	nodeA := domain.ConnectionEndpoint{Kind: domain.ConnectionEndpointNodeInterface, LaboratoryID: labID, ResourceID: "node-a", PortID: "if-a", PortName: "eth0", Availability: domain.ConnectionEndpointFree}
	nodeB := domain.ConnectionEndpoint{Kind: domain.ConnectionEndpointNodeInterface, LaboratoryID: labID, ResourceID: "node-b", PortID: "if-b", PortName: "eth0", Availability: domain.ConnectionEndpointFree}
	objectA := domain.ConnectionEndpoint{Kind: domain.ConnectionEndpointNetworkObjectPort, LaboratoryID: labID, ResourceID: "switch-a", PortName: "eth0", Availability: domain.ConnectionEndpointFree}
	objectB := domain.ConnectionEndpoint{Kind: domain.ConnectionEndpointNetworkObjectPort, LaboratoryID: labID, ResourceID: "switch-b", PortName: "eth1", Availability: domain.ConnectionEndpointFree}
	repository := topologyConnectionRepositoryStub{endpoints: map[string]domain.ConnectionEndpoint{nodeA.Key(): nodeA, nodeB.Key(): nodeB, objectA.Key(): objectA, objectB.Key(): objectB}}
	links := &topologyLinkOperationsStub{}
	networks := &topologyNetworkOperationsStub{}
	service := NewTopologyConnectionService(repository, links, networks)

	connection, taskValue, err := service.Create(context.Background(), labID, nodeA, nodeB, domain.TopologyConnectionConfig{}, "node-link")
	if err != nil || connection.BackingKind != domain.ConnectionBackingLink || taskValue.ID != "task-link" {
		t.Fatalf("connection=%+v task=%+v err=%v", connection, taskValue, err)
	}
	connection, _, err = service.Create(context.Background(), labID, nodeA, objectA, domain.TopologyConnectionConfig{PVID: 10, TaggedVLANs: []int{20}}, "attachment")
	if err != nil || connection.BackingKind != domain.ConnectionBackingAttachment || networks.attachment.PortName != "eth0" {
		t.Fatalf("connection=%+v attachment=%+v err=%v", connection, networks.attachment, err)
	}
	connection, _, err = service.Create(context.Background(), labID, objectA, objectB, domain.TopologyConnectionConfig{}, "object-link")
	if err != nil || connection.BackingKind != domain.ConnectionBackingObjectLink {
		t.Fatalf("connection=%+v err=%v", connection, err)
	}
}

func TestTopologyConnectionServiceRejectsUnavailableAuthoritativeEndpoint(t *testing.T) {
	labID := domain.ID("lab")
	nodeA := domain.ConnectionEndpoint{Kind: domain.ConnectionEndpointNodeInterface, LaboratoryID: labID, ResourceID: "node-a", PortID: "if-a", Availability: domain.ConnectionEndpointOccupied}
	nodeB := domain.ConnectionEndpoint{Kind: domain.ConnectionEndpointNodeInterface, LaboratoryID: labID, ResourceID: "node-b", PortID: "if-b", Availability: domain.ConnectionEndpointFree}
	repository := topologyConnectionRepositoryStub{endpoints: map[string]domain.ConnectionEndpoint{nodeA.Key(): nodeA, nodeB.Key(): nodeB}}
	service := NewTopologyConnectionService(repository, &topologyLinkOperationsStub{}, &topologyNetworkOperationsStub{})
	_, _, err := service.Create(context.Background(), labID, nodeA, nodeB, domain.TopologyConnectionConfig{}, "occupied")
	problem := domain.NormalizeProblem(err, domain.Problem{})
	if problem.Code != "port_in_use" {
		t.Fatalf("problem=%+v", problem)
	}
}
