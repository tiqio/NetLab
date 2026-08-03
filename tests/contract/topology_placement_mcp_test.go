package contract

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/netlab/netlab/internal/api/mcp"
	"github.com/netlab/netlab/internal/app/command"
	"github.com/netlab/netlab/internal/domain"
)

type mcpPlacementStub struct{ calls int }

func (s *mcpPlacementStub) Update(_ context.Context, laboratoryID domain.ID, revision domain.Revision, updates []domain.PlacementUpdate) (command.TopologyPlacementResult, error) {
	s.calls++
	return command.TopologyPlacementResult{LaboratoryRevision: revision.Next(), Placements: []domain.TopologyPlacement{{LaboratoryID: laboratoryID, ResourceID: updates[0].ResourceID, ResourceType: updates[0].ResourceType, X: updates[0].X, Y: updates[0].Y, Revision: 1}}}, nil
}

func TestTopologySetPositionsIsDocumented(t *testing.T) {
	body, err := os.ReadFile("../../specs/004-topology-interaction-ux/contracts/mcp-tools.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "netlab.topology.set_positions") {
		t.Fatal("MCP placement tool missing from feature contract")
	}
}

func TestTopologySetPositionsMCPParity(t *testing.T) {
	service := &mcpPlacementStub{}
	tools := mcp.TopologyPlacementTools(service)
	if len(tools) != 1 || tools[0].Name != "netlab.topology.set_positions" {
		t.Fatalf("tools=%+v", tools)
	}
	value, err := tools[0].Handler(&gin.Context{}, map[string]any{
		"laboratory_id": "lab-1", "expected_revision": float64(2), "idempotency_key": "position-key",
		"placements": []any{map[string]any{"resource_id": "node-1", "resource_type": "node", "x": float64(10), "y": float64(20)}},
	})
	if err != nil || service.calls != 1 {
		t.Fatalf("value=%+v calls=%d err=%v", value, service.calls, err)
	}
}
