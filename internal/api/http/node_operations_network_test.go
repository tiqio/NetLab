package httpapi

import (
	"testing"

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
