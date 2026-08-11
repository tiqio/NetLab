package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/netlab/netlab/internal/domain"
)

func TestValidateDockerNetworkSettingsPreservesRoutesAndDriver(t *testing.T) {
	node := domain.Node{ID: "node", Kind: string(domain.RuntimeDocker), Config: map[string]any{
		"interfaces": []any{map[string]any{"id": "if-1", "name": "eth0", "driver": "", "mac_address": "02:00:00:00:00:01"}},
	}}
	values := []domain.NodeNetworkInterfaceSettings{{
		ID:        "if-1",
		Name:      "eth0",
		Driver:    "",
		Modes:     []string{"static"},
		Addresses: []string{"192.0.2.10/24"},
		Routes:    []domain.RouteConfig{{Destination: "0.0.0.0/0", Gateway: "192.0.2.1", Metric: 10}},
	}}
	interfaces, raw, err := validateNetworkSettings(node, values)
	if err != nil {
		t.Fatal(err)
	}
	if len(interfaces) != 1 || interfaces[0].Driver != "" {
		t.Fatalf("interfaces=%+v", interfaces)
	}
	routes, ok := raw[0]["routes"].([]domain.RouteConfig)
	if !ok || len(routes) != 1 || routes[0].Gateway != "192.0.2.1" {
		t.Fatalf("raw=%#v", raw)
	}
}

func TestValidateDockerNetworkSettingsRejectsUnreachableGateway(t *testing.T) {
	node := domain.Node{ID: "node", Kind: string(domain.RuntimeDocker), Config: map[string]any{
		"interfaces": []any{map[string]any{"id": "if-1", "name": "eth0", "driver": "", "mac_address": "02:00:00:00:00:01"}},
	}}
	_, _, err := validateNetworkSettings(node, []domain.NodeNetworkInterfaceSettings{{
		ID:        "if-1",
		Name:      "eth0",
		Addresses: []string{"192.0.2.10/24"},
		Routes:    []domain.RouteConfig{{Destination: "0.0.0.0/0", Gateway: "198.51.100.1"}},
	}})
	problem, ok := err.(domain.Problem)
	if !ok || problem.Code != "invalid_node_network" {
		t.Fatalf("err=%v", err)
	}
}

func TestValidateNodeForwardingSettingsAcceptsDockerAndRejectsQEMU(t *testing.T) {
	enabled := true
	settings := domain.NodeSettings{ForwardIPv4: &enabled, ForwardIPv6: &enabled}
	if err := validateNodeForwardingSettings(domain.Node{ID: "docker", Kind: string(domain.RuntimeDocker)}, settings); err != nil {
		t.Fatal(err)
	}
	err := validateNodeForwardingSettings(domain.Node{ID: "qemu", Kind: string(domain.RuntimeQEMU)}, settings)
	problem, ok := err.(domain.Problem)
	if !ok || problem.Code != "invalid_node_network" || problem.ResourceID != "qemu" {
		t.Fatalf("err=%v", err)
	}
}

type nodeNetworkDiagnosticsServiceFake struct {
	node domain.Node
}

func (f nodeNetworkDiagnosticsServiceFake) GetNode(context.Context, domain.ID) (domain.Node, error) {
	return f.node, nil
}
func (nodeNetworkDiagnosticsServiceFake) ListAllNodes(context.Context) ([]domain.Node, error) {
	return nil, nil
}
func (nodeNetworkDiagnosticsServiceFake) UpdateNodeResources(context.Context, domain.ID, domain.Revision, int, int64, int) (domain.Node, error) {
	return domain.Node{}, nil
}
func (nodeNetworkDiagnosticsServiceFake) UpdateNodeSettings(context.Context, domain.ID, domain.Revision, domain.NodeSettings) (domain.Node, error) {
	return domain.Node{}, nil
}

type nodeNetworkDiagnosticsRuntimeFake struct{}

func (nodeNetworkDiagnosticsRuntimeFake) NetworkDiagnostics(context.Context, domain.Node) (map[string]any, error) {
	return map[string]any{
		"desired":    map[string]any{"forward_ipv4": true, "forward_ipv6": true},
		"observed":   map[string]any{"forward_ipv4": false, "forward_ipv6": true, "available": true},
		"mismatches": []string{"forward_ipv4 desired=true observed=false"},
	}, nil
}

func TestDockerNetworkDiagnosticsReturnsRequestedVersusObservedMismatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	handler := NewNodeOperationsHandlers(nil, nil, nil, nodeNetworkDiagnosticsServiceFake{node: domain.Node{ID: "router", Kind: string(domain.RuntimeDocker)}}, nil)
	handler.SetNetworkDiagnostics(nodeNetworkDiagnosticsRuntimeFake{})
	handler.Register(engine)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/nodes/router/network-diagnostics", nil)
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"forward_ipv4 desired=true observed=false"`) {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}
