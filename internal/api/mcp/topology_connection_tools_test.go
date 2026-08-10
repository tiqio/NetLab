package mcp

import (
	"context"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/netlab/netlab/internal/domain"
)

type topologyConnectionToolCommandsStub struct{}

func (topologyConnectionToolCommandsStub) Create(_ context.Context, laboratoryID domain.ID, source, target domain.ConnectionEndpoint, _ domain.TopologyConnectionConfig, _ string) (domain.TopologyConnection, domain.OperationTask, error) {
	return domain.TopologyConnection{ID: "connection", LaboratoryID: laboratoryID, Source: source, Target: target, BackingKind: domain.ConnectionBackingLink, Revision: 1, DesiredState: "connected", ObservedState: "pending"}, domain.OperationTask{ID: "task", Kind: "link.connect", State: domain.TaskQueued}, nil
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
