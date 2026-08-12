package mcp

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/netlab/netlab/internal/app/query"
	"github.com/netlab/netlab/internal/domain"
	"net/http/httptest"
	"testing"
)

type readinessRepositoryFake struct{}

func (readinessRepositoryFake) GetNode(context.Context, domain.ID) (domain.Node, error) {
	return domain.Node{ID: "node", LaboratoryID: "lab", Config: map[string]any{"device_roles": []any{map[string]any{"interface_id": "if-1", "role": "management"}}}}, nil
}
func (readinessRepositoryFake) Snapshot(context.Context, domain.ID) (domain.TopologySnapshot, error) {
	return domain.TopologySnapshot{Interfaces: []domain.Interface{{ID: "if-1", NodeID: "node"}}, Attachments: []domain.NetworkAttachment{{InterfaceID: "if-1", ObservedState: "active"}}}, nil
}
func (readinessRepositoryFake) ListRuntimeCapabilities(context.Context, domain.ID) ([]domain.RuntimeCapabilityObservation, error) {
	return []domain.RuntimeCapabilityObservation{{Capability: domain.CapabilitySerial, State: domain.CapabilityReady}}, nil
}
func TestDeviceReadinessMCPParity(t *testing.T) {
	service := query.NewDeviceReadinessService(readinessRepositoryFake{}, readinessRepositoryFake{}, readinessRepositoryFake{})
	result, err := DeviceReadinessTool(service).Handler(&gin.Context{Request: httptest.NewRequest("POST", "/mcp", nil)}, map[string]any{"node_id": "node"})
	if err != nil || result.(domain.DeviceReadiness).Cable.State != "ready" || result.(domain.DeviceReadiness).Management.State != "prerequisite" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}
