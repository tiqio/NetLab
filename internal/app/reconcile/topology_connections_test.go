package reconcile

import (
	"context"
	"errors"
	"testing"

	"github.com/netlab/netlab/internal/domain"
)

type unifiedConnectionRepositoryFake struct {
	endpoints map[string]domain.ConnectionEndpoint
}

func (f unifiedConnectionRepositoryFake) ResolveConnectionEndpoint(_ context.Context, endpoint domain.ConnectionEndpoint) (domain.ConnectionEndpoint, error) {
	value, ok := f.endpoints[endpoint.Key()]
	if !ok {
		return domain.ConnectionEndpoint{}, domain.ErrNotFound
	}
	return value, nil
}
func (unifiedConnectionRepositoryFake) ListTopologyConnections(context.Context, domain.ID) ([]domain.TopologyConnection, error) {
	return nil, nil
}
func (unifiedConnectionRepositoryFake) GetTopologyConnection(context.Context, domain.ID) (domain.TopologyConnection, error) {
	return domain.TopologyConnection{}, domain.ErrNotFound
}

type unifiedLinkRuntimeFake struct{ err error }

func (f unifiedLinkRuntimeFake) ConnectLink(_ context.Context, lab, first, second domain.ID, _ string) (domain.Link, domain.OperationTask, error) {
	return domain.Link{ID: "link", LaboratoryID: lab, EndpointAID: first, EndpointBID: second, Revision: 1, DesiredState: "connected", ObservedState: "pending"}, domain.OperationTask{ID: "link-task", State: domain.TaskQueued}, f.err
}
func (unifiedLinkRuntimeFake) DisconnectLink(context.Context, domain.ID, string) (domain.OperationTask, error) {
	return domain.OperationTask{}, nil
}

type unifiedNetworkRuntimeFake struct{ attachmentErr, objectLinkErr error }

func (f unifiedNetworkRuntimeFake) CreateAttachment(_ context.Context, _ domain.ID, objectID, interfaceID domain.ID, port string, config map[string]any, _ string) (domain.NetworkAttachment, domain.OperationTask, error) {
	return domain.NetworkAttachment{ID: "attachment", NetworkObjectID: objectID, InterfaceID: interfaceID, PortName: port, Config: config, ObservedState: "pending"}, domain.OperationTask{ID: "attachment-task", State: domain.TaskQueued}, f.attachmentErr
}
func (unifiedNetworkRuntimeFake) DeleteAttachment(context.Context, domain.ID, domain.Revision, string) (domain.NetworkAttachment, domain.OperationTask, error) {
	return domain.NetworkAttachment{}, domain.OperationTask{}, nil
}
func (f unifiedNetworkRuntimeFake) CreateObjectLink(_ context.Context, lab, first domain.ID, firstPort string, second domain.ID, secondPort, _ string) (domain.NetworkObjectLink, domain.OperationTask, error) {
	return domain.NetworkObjectLink{ID: "object-link", LaboratoryID: lab, ObjectAID: first, PortAName: firstPort, ObjectBID: second, PortBName: secondPort, Revision: 1, DesiredState: "connected", ObservedState: "pending"}, domain.OperationTask{ID: "object-link-task", State: domain.TaskQueued}, f.objectLinkErr
}
func (unifiedNetworkRuntimeFake) DeleteObjectLink(context.Context, domain.ID, domain.Revision, string) (domain.NetworkObjectLink, domain.OperationTask, error) {
	return domain.NetworkObjectLink{}, domain.OperationTask{}, nil
}

func TestUnifiedTopologyConnectionServiceAdaptsAllBackingKinds(t *testing.T) {
	lab := domain.ID("lab")
	nodeA := domain.ConnectionEndpoint{Kind: domain.ConnectionEndpointNodeInterface, LaboratoryID: lab, ResourceID: "node-a", PortID: "if-a", Availability: domain.ConnectionEndpointFree}
	nodeB := domain.ConnectionEndpoint{Kind: domain.ConnectionEndpointNodeInterface, LaboratoryID: lab, ResourceID: "node-b", PortID: "if-b", Availability: domain.ConnectionEndpointFree}
	objectA := domain.ConnectionEndpoint{Kind: domain.ConnectionEndpointNetworkObjectPort, LaboratoryID: lab, ResourceID: "object-a", PortName: "eth0", Availability: domain.ConnectionEndpointFree}
	objectB := domain.ConnectionEndpoint{Kind: domain.ConnectionEndpointNetworkObjectPort, LaboratoryID: lab, ResourceID: "object-b", PortName: "eth1", Availability: domain.ConnectionEndpointFree}
	repository := unifiedConnectionRepositoryFake{endpoints: map[string]domain.ConnectionEndpoint{nodeA.Key(): nodeA, nodeB.Key(): nodeB, objectA.Key(): objectA, objectB.Key(): objectB}}
	service := NewUnifiedTopologyConnectionService(repository, unifiedLinkRuntimeFake{}, unifiedNetworkRuntimeFake{})
	for _, test := range []struct {
		source, target domain.ConnectionEndpoint
		want           domain.ConnectionBackingKind
	}{{nodeA, nodeB, domain.ConnectionBackingLink}, {nodeA, objectA, domain.ConnectionBackingAttachment}, {objectA, objectB, domain.ConnectionBackingObjectLink}} {
		connection, _, err := service.Create(context.Background(), lab, test.source, test.target, domain.TopologyConnectionConfig{}, "key-"+string(test.want))
		if err != nil || connection.BackingKind != test.want {
			t.Fatalf("backing=%s connection=%+v err=%v", test.want, connection, err)
		}
	}
}

func TestUnifiedTopologyConnectionServicePreservesRuntimeFailure(t *testing.T) {
	lab := domain.ID("lab")
	node := domain.ConnectionEndpoint{Kind: domain.ConnectionEndpointNodeInterface, LaboratoryID: lab, ResourceID: "node", PortID: "if-a", Availability: domain.ConnectionEndpointFree}
	object := domain.ConnectionEndpoint{Kind: domain.ConnectionEndpointNetworkObjectPort, LaboratoryID: lab, ResourceID: "object", PortName: "eth0", Availability: domain.ConnectionEndpointFree}
	repository := unifiedConnectionRepositoryFake{endpoints: map[string]domain.ConnectionEndpoint{node.Key(): node, object.Key(): object}}
	runtimeErr := errors.New("partial runtime failure")
	service := NewUnifiedTopologyConnectionService(repository, unifiedLinkRuntimeFake{}, unifiedNetworkRuntimeFake{attachmentErr: runtimeErr})
	_, _, err := service.Create(context.Background(), lab, node, object, domain.TopologyConnectionConfig{}, "failure")
	if !errors.Is(err, runtimeErr) {
		t.Fatalf("runtime failure was not preserved: %v", err)
	}
}
