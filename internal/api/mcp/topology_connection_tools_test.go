package mcp

import (
	"context"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/netlab/netlab/internal/domain"
)

type topologyConnectionToolCommandsStub struct{}

func (topologyConnectionToolCommandsStub) Create(_ context.Context, laboratoryID domain.ID, source, target domain.ConnectionEndpoint, _ domain.TopologyConnectionConfig, _ string) (domain.TopologyConnection, domain.OperationTask, error) {
	return domain.TopologyConnection{ID: "connection", LaboratoryID: laboratoryID, Source: source, Target: target, BackingKind: domain.ConnectionBackingLink, Revision: 1, DesiredState: "connected", ObservedState: "pending"}, domain.OperationTask{ID: "task", Kind: "link.connect", State: domain.TaskQueued}, nil
}

type contendedTopologyConnectionToolCommands struct {
	mu       sync.Mutex
	occupied bool
}

func (s *contendedTopologyConnectionToolCommands) Create(_ context.Context, laboratoryID domain.ID, source, target domain.ConnectionEndpoint, _ domain.TopologyConnectionConfig, _ string) (domain.TopologyConnection, domain.OperationTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.occupied {
		return domain.TopologyConnection{}, domain.OperationTask{}, domain.Problem{Code: "port_in_use", Message: "endpoint already reserved", Retryable: true}
	}
	s.occupied = true
	connection := domain.TopologyConnection{ID: "winner", LaboratoryID: laboratoryID, Source: source, Target: target, BackingKind: domain.ConnectionBackingLink, Revision: 1}
	return connection, domain.OperationTask{ID: "winner-task", ResourceType: "link", ResourceID: connection.ID, State: domain.TaskQueued}, nil
}
func (*contendedTopologyConnectionToolCommands) List(context.Context, domain.ID) ([]domain.TopologyConnection, error) {
	return []domain.TopologyConnection{{ID: "winner", LaboratoryID: "lab", BackingKind: domain.ConnectionBackingLink, Revision: 1}}, nil
}
func (*contendedTopologyConnectionToolCommands) Get(context.Context, domain.ID) (domain.TopologyConnection, error) {
	return domain.TopologyConnection{}, domain.ErrNotFound
}
func (*contendedTopologyConnectionToolCommands) Delete(context.Context, domain.ID, domain.Revision, string) (domain.TopologyConnection, domain.OperationTask, error) {
	return domain.TopologyConnection{}, domain.OperationTask{}, nil
}

func TestTopologyConnectionMCPReportsPortConflictAndSupportsAuthoritativeRefresh(t *testing.T) {
	commands := &contendedTopologyConnectionToolCommands{}
	tools := TopologyConnectionTools(commands, topologyConnectionToolReadStub{})
	contextValue, _ := gin.CreateTestContext(nil)
	args := map[string]any{"laboratory_id": "lab", "expected_revision": float64(4), "idempotency_key": "first", "source": map[string]any{"kind": "node_interface", "resource_id": "node-a", "port_id": "if-a"}, "target": map[string]any{"kind": "node_interface", "resource_id": "node-b", "port_id": "if-b"}}
	if _, err := tools[0].Handler(contextValue, args); err != nil {
		t.Fatal(err)
	}
	args["idempotency_key"] = "second"
	_, err := tools[0].Handler(contextValue, args)
	problem := domain.NormalizeProblem(err, domain.Problem{})
	if problem.Code != "port_in_use" || !problem.Retryable {
		t.Fatalf("problem=%+v", problem)
	}
	result, err := tools[1].Handler(contextValue, map[string]any{"laboratory_id": "lab"})
	if err != nil || len(result.(map[string]any)["connections"].([]domain.TopologyConnection)) != 1 {
		t.Fatalf("refresh=%+v err=%v", result, err)
	}
}
func (topologyConnectionToolCommandsStub) List(context.Context, domain.ID) ([]domain.TopologyConnection, error) {
	return []domain.TopologyConnection{{ID: "connection", LaboratoryID: "lab", BackingKind: domain.ConnectionBackingLink, BackingID: "link", Revision: 1, DesiredState: "connected", ObservedState: "connected"}}, nil
}
func (topologyConnectionToolCommandsStub) Get(context.Context, domain.ID) (domain.TopologyConnection, error) {
	return domain.TopologyConnection{ID: "connection", LaboratoryID: "lab", BackingKind: domain.ConnectionBackingLink, BackingID: "link", Revision: 1, DesiredState: "connected", ObservedState: "connected"}, nil
}
func (topologyConnectionToolCommandsStub) Delete(context.Context, domain.ID, domain.Revision, string) (domain.TopologyConnection, domain.OperationTask, error) {
	return domain.TopologyConnection{ID: "connection", Revision: 1, DesiredState: "disconnected", ObservedState: "disconnecting"}, domain.OperationTask{ID: "delete-task", State: domain.TaskQueued}, nil
}

type topologyConnectionToolReadStub struct{}

func (topologyConnectionToolReadStub) GetLaboratory(context.Context, domain.ID) (domain.Laboratory, error) {
	return domain.Laboratory{ID: "lab", Revision: 4}, nil
}
func (topologyConnectionToolReadStub) ListConnectionEndpoints(context.Context, domain.ID) ([]domain.ConnectionEndpoint, error) {
	return []domain.ConnectionEndpoint{{Kind: domain.ConnectionEndpointNodeInterface, LaboratoryID: "lab", ResourceID: "node-a", PortID: "if-a", Availability: domain.ConnectionEndpointFree}}, nil
}

func TestTopologyConnectionToolsExposeSymmetricCreateListGetDelete(t *testing.T) {
	tools := TopologyConnectionTools(topologyConnectionToolCommandsStub{}, topologyConnectionToolReadStub{})
	if len(tools) != 4 {
		t.Fatalf("tools=%d", len(tools))
	}
	want := []string{"netlab.topology_connections.create", "netlab.topology_connections.list", "netlab.topology_connections.get", "netlab.topology_connections.delete"}
	for index, name := range want {
		if tools[index].Name != name {
			t.Fatalf("tool[%d]=%q want %q", index, tools[index].Name, name)
		}
	}
	contextValue, _ := gin.CreateTestContext(nil)
	result, err := tools[0].Handler(contextValue, map[string]any{
		"laboratory_id": "lab", "expected_revision": float64(4), "idempotency_key": "key",
		"source": map[string]any{"kind": "node_interface", "resource_id": "node-a", "port_id": "if-a"},
		"target": map[string]any{"kind": "node_interface", "resource_id": "node-b", "port_id": "if-b"},
	})
	if err != nil {
		t.Fatal(err)
	}
	value := result.(map[string]any)
	if value["connection"] == nil || value["task"] == nil || value["laboratory_revision"] != domain.Revision(4) {
		t.Fatalf("result=%+v", result)
	}
	listResult, err := tools[1].Handler(contextValue, map[string]any{"laboratory_id": "lab"})
	if err != nil {
		t.Fatal(err)
	}
	listValue := listResult.(map[string]any)
	if listValue["connections"] == nil || listValue["endpoints"] == nil || listValue["laboratory_revision"] != domain.Revision(4) {
		t.Fatalf("list result=%+v", listResult)
	}
	getResult, err := tools[2].Handler(contextValue, map[string]any{"connection_id": "connection"})
	if err != nil {
		t.Fatal(err)
	}
	if getResult.(domain.TopologyConnection).ObservedState != "connected" {
		t.Fatalf("get result=%+v", getResult)
	}
}
