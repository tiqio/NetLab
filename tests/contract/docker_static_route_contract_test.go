package contract

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	httpapi "github.com/netlab/netlab/internal/api/http"
	"github.com/netlab/netlab/internal/api/mcp"
	"github.com/netlab/netlab/internal/domain"
)

type dockerNodeSettingsContractService struct {
	node     domain.Node
	settings domain.NodeSettings
}

func (s *dockerNodeSettingsContractService) GetNode(context.Context, domain.ID) (domain.Node, error) {
	return s.node, nil
}

func (s *dockerNodeSettingsContractService) ListAllNodes(context.Context) ([]domain.Node, error) {
	return []domain.Node{s.node}, nil
}

func (s *dockerNodeSettingsContractService) UpdateNodeResources(context.Context, domain.ID, domain.Revision, int, int64, int) (domain.Node, error) {
	return s.node, nil
}

func (s *dockerNodeSettingsContractService) UpdateNodeSettings(_ context.Context, _ domain.ID, _ domain.Revision, settings domain.NodeSettings) (domain.Node, error) {
	s.settings = settings
	return s.node, nil
}

func TestDockerStaticRoutesHaveRESTAndMCPSettingsParity(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &dockerNodeSettingsContractService{node: domain.Node{
		ID: "node", Kind: string(domain.RuntimeDocker), Revision: 1,
		DesiredState: domain.DesiredStopped, ObservedState: domain.ObservedStopped,
		Config: map[string]any{"interfaces": []any{map[string]any{"id": "if-1", "name": "eth0", "driver": "", "mac_address": "02:00:00:00:00:01"}}},
	}}
	settings := map[string]any{
		"name": "docker", "cpu_count": 1, "cpu_quota_micros": 100000, "memory_mib": 128,
		"interface_limit": 4, "process_limit": 64,
		"network_interfaces": []any{map[string]any{
			"id": "if-1", "name": "eth0", "driver": "", "modes": []any{"static"}, "addresses": []any{"192.0.2.10/24"},
			"routes": []any{map[string]any{"destination": "0.0.0.0/0", "gateway": "192.0.2.1", "metric": 10}},
		}},
	}
	body, _ := json.Marshal(settings)
	engine := gin.New()
	httpapi.NewNodeOperationsHandlers(nil, nil, nil, service, nil).Register(engine)
	request := httptest.NewRequest(http.MethodPut, "/api/v1/nodes/node/settings", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("If-Match", "1")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	assertDockerRouteSettings(t, service.settings)

	var tool mcp.Tool
	for _, candidate := range mcp.Tools(mcp.Services{NodeSettings: service}) {
		if candidate.Name == "netlab.nodes.update_settings" {
			tool = candidate
			break
		}
	}
	if tool.Handler == nil {
		t.Fatal("Docker node settings MCP tool missing")
	}
	schema, _ := json.Marshal(tool.InputSchema)
	for _, expected := range []string{"network_interfaces", "routes", "destination", "gateway", "metric"} {
		if !bytes.Contains(schema, []byte(expected)) {
			t.Errorf("MCP schema missing %s: %s", expected, schema)
		}
	}
	args := map[string]any{"node_id": "node", "expected_revision": float64(1)}
	for key, value := range settings {
		args[key] = value
	}
	contextValue, _ := gin.CreateTestContext(httptest.NewRecorder())
	if _, err := tool.Handler(contextValue, args); err != nil {
		t.Fatal(err)
	}
	assertDockerRouteSettings(t, service.settings)
}

func TestGeneratedClientExposesTypedDockerRoutes(t *testing.T) {
	body, err := os.ReadFile("../../web/src/api/generated.ts")
	if err != nil {
		t.Fatal(err)
	}
	client := string(body)
	for _, expected := range []string{"export interface DockerStaticRoute", "destination: string", "routes: DockerStaticRoute[]", "network_interfaces?: NodeNetworkInterfaceSettings[]"} {
		if !strings.Contains(client, expected) {
			t.Errorf("generated client missing %q", expected)
		}
	}
}

func TestDockerStaticRouteHTTPValidationIsActionable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &dockerNodeSettingsContractService{node: domain.Node{
		ID: "node", Kind: string(domain.RuntimeDocker), Revision: 1,
		DesiredState: domain.DesiredStopped, ObservedState: domain.ObservedStopped,
		Config: map[string]any{"interfaces": []any{map[string]any{"id": "if-1", "name": "eth0", "driver": "", "mac_address": "02:00:00:00:00:01"}}},
	}}
	body := bytes.NewBufferString(`{"name":"docker","cpu_count":1,"cpu_quota_micros":100000,"memory_mib":128,"interface_limit":4,"process_limit":64,"network_interfaces":[{"id":"if-1","name":"eth0","driver":"","modes":["static"],"addresses":["192.0.2.10/24"],"routes":[{"destination":"0.0.0.0/0","gateway":"198.51.100.1"}]}]}`)
	engine := gin.New()
	httpapi.NewNodeOperationsHandlers(nil, nil, nil, service, nil).Register(engine)
	request := httptest.NewRequest(http.MethodPut, "/api/v1/nodes/node/settings", body)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("If-Match", "1")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var problem domain.Problem
	if err := json.Unmarshal(response.Body.Bytes(), &problem); err != nil {
		t.Fatal(err)
	}
	if problem.Code != "invalid_node_network" || !strings.Contains(problem.Message, "unreachable") {
		t.Fatalf("problem=%+v", problem)
	}
}

func assertDockerRouteSettings(t *testing.T, settings domain.NodeSettings) {
	t.Helper()
	if len(settings.NetworkInterfaces) != 1 || len(settings.NetworkInterfaces[0].Routes) != 1 {
		t.Fatalf("settings=%+v", settings)
	}
	route := settings.NetworkInterfaces[0].Routes[0]
	if route.Destination != "0.0.0.0/0" || route.Gateway != "192.0.2.1" || route.Metric != 10 {
		t.Fatalf("route=%+v", route)
	}
}
