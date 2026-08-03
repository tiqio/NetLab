package contract

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	httpapi "github.com/netlab/netlab/internal/api/http"
	streamapi "github.com/netlab/netlab/internal/api/stream"
	consoleRuntime "github.com/netlab/netlab/internal/runtime/console"
	"gopkg.in/yaml.v3"
)

func TestOpenAPIRoutesSchemasAndGeneratedClientMatchRuntime(t *testing.T) {
	document := loadOpenAPI(t)
	paths := object(t, document, "paths")
	runtimeRoutes := registeredRoutes()
	operationIDs := map[string]bool{}

	for path, raw := range paths {
		operations, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		for _, method := range []string{"get", "post", "put", "patch", "delete"} {
			operation, exists := operations[method].(map[string]any)
			if !exists {
				continue
			}
			operationID, _ := operation["operationId"].(string)
			if operationID == "" || operationIDs[operationID] {
				t.Fatalf("invalid duplicate operationId %q at %s", operationID, path)
			}
			operationIDs[operationID] = true
			runtimePath := "/api/v1" + openAPIToGinPath(path)
			if !runtimeRoutes[strings.ToUpper(method)+" "+runtimePath] {
				t.Errorf("OpenAPI operation %s %s is not registered by the runtime", strings.ToUpper(method), runtimePath)
			}
		}
	}

	clientBody, err := os.ReadFile("../../web/src/api/index.ts")
	if err != nil {
		t.Fatal(err)
	}
	clientOperations := generatedClientOperations(string(clientBody))
	for operationID := range operationIDs {
		if !clientOperations[operationID] {
			t.Errorf("generated client missing operationId %s", operationID)
		}
	}

	schemas := object(t, object(t, document, "components"), "schemas")
	requireSchemaProperties(t, schemas, "OperationTask", "progress_current", "progress_total", "resource_type", "resource_id")
	requireSchemaProperties(t, schemas, "CaptureSession", "max_bytes", "bytes_written", "packets")
	requireSchemaProperties(t, schemas, "ConsoleDescriptor", "stream_url", "idle_seconds")
	requireSchemaProperties(t, schemas, "LabSnapshot", "interfaces", "event_sequence")
	requireSchemaProperties(t, schemas, "CreateNode", "template_version_id", "image_version_id", "bootstrap")
	rejectSchemaProperties(t, schemas, "CaptureSession", "packet_count", "byte_count")
	rejectSchemaProperties(t, schemas, "StartCapture", "size_limit_bytes")
}

func loadOpenAPI(t *testing.T) map[string]any {
	t.Helper()
	body, err := os.ReadFile("../../specs/001-network-simulator-platform/contracts/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err = yaml.Unmarshal(body, &document); err != nil {
		t.Fatal(err)
	}
	return document
}

func registeredRoutes() map[string]bool {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	httpapi.NewTopologyHandlers(nil, nil, nil, nil, nil).Register(engine)
	httpapi.NewTopologyPlacementHandlers(nil).Register(engine)
	httpapi.NewLinkReconnectHandlers(nil).Register(engine)
	httpapi.NewTemplateHandlers(nil, nil, nil).Register(engine)
	httpapi.NewAutomationHandlers(nil, nil, nil, nil, nil).Register(engine)
	httpapi.NewNodeOperationsHandlers(nil, nil, nil, nil, nil).Register(engine)
	httpapi.NewCaptureHandlers(nil, nil, nil).Register(engine)
	httpapi.NewNetworkHandlers(nil, nil, nil).Register(engine)
	httpapi.NewArtifactHandlers(nil).Register(engine)
	httpapi.NewClientToolsHandlers("").Register(engine)
	httpapi.NewRuntimeOwnershipHandlers(nil).Register(engine)
	streamapi.NewConsoleHandlers("", consoleRuntime.Limits{}).Register(engine)
	routes := map[string]bool{}
	for _, route := range engine.Routes() {
		routes[route.Method+" "+route.Path] = true
	}
	return routes
}

func openAPIToGinPath(path string) string {
	replacer := regexp.MustCompile(`\{([^}]+)\}`)
	return replacer.ReplaceAllString(path, `:$1`)
}

func generatedClientOperations(body string) map[string]bool {
	start := strings.Index(body, "export const generatedApi = {")
	if start < 0 {
		return nil
	}
	operations := map[string]bool{}
	pattern := regexp.MustCompile(`(?m)^  ([A-Za-z][A-Za-z0-9]*):`)
	for _, match := range pattern.FindAllStringSubmatch(body[start:], -1) {
		operations[match[1]] = true
	}
	return operations
}

func object(t *testing.T, parent map[string]any, key string) map[string]any {
	t.Helper()
	value, ok := parent[key].(map[string]any)
	if !ok {
		t.Fatalf("%s is not an object", key)
	}
	return value
}

func requireSchemaProperties(t *testing.T, schemas map[string]any, name string, expected ...string) {
	t.Helper()
	schema := object(t, schemas, name)
	properties := object(t, schema, "properties")
	for _, property := range expected {
		if _, ok := properties[property]; !ok {
			t.Errorf("schema %s missing property %s", name, property)
		}
	}
}

func rejectSchemaProperties(t *testing.T, schemas map[string]any, name string, rejected ...string) {
	t.Helper()
	schema := object(t, schemas, name)
	properties := object(t, schema, "properties")
	for _, property := range rejected {
		if _, ok := properties[property]; ok {
			t.Errorf("schema %s still exposes obsolete property %s", name, property)
		}
	}
}
