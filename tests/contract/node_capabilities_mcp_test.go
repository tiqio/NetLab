package contract_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/netlab/netlab/internal/api/mcp"
	"github.com/netlab/netlab/internal/app/query"
	"github.com/netlab/netlab/internal/domain"
)

func TestNodeCapabilitiesMCPParity(t *testing.T) {
	service := query.NewRuntimeCapabilityService(capabilityStoreStub{values: []domain.RuntimeCapabilityObservation{{NodeID: "node-1", Capability: domain.CapabilityQMP, Revision: 2, State: domain.CapabilityReady, Required: true, ObservedAt: time.Now()}}})
	tool := mcp.NodeCapabilityTool(service)
	request, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/mcp", nil)
	value, err := tool.Handler(&gin.Context{Request: request}, map[string]any{"node_id": "node-1"})
	if err != nil {
		t.Fatal(err)
	}
	result := value.(map[string]any)
	if result["node_id"] != "node-1" || len(result["observations"].([]domain.RuntimeCapabilityObservation)) != 1 {
		t.Fatalf("unexpected result: %#v", result)
	}
}
