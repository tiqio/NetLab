package contract

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/netlab/netlab/internal/api/mcp"
	"gopkg.in/yaml.v3"
)

func TestContractDocumentsAndGeneratedClientStayValid(t *testing.T) {
	openAPIBody, err := os.ReadFile("../../specs/001-network-simulator-platform/contracts/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var openAPI map[string]any
	if err = yaml.Unmarshal(openAPIBody, &openAPI); err != nil {
		t.Fatal(err)
	}
	if openAPI["openapi"] != "3.1.0" {
		t.Fatalf("openapi=%v", openAPI["openapi"])
	}
	paths, ok := openAPI["paths"].(map[string]any)
	if !ok || len(paths) < 20 {
		t.Fatalf("paths=%d", len(paths))
	}
	clientBody, err := os.ReadFile("../../web/src/api/generated.ts")
	if err != nil {
		t.Fatal(err)
	}
	for _, schema := range []string{"Problem", "Laboratory", "OperationTask", "Artifact"} {
		if !strings.Contains(string(clientBody), "interface "+schema) {
			t.Fatalf("generated client missing %s", schema)
		}
	}

	exportBody, err := os.ReadFile("../../specs/001-network-simulator-platform/contracts/lab-export.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	var exportSchema map[string]any
	if err = json.Unmarshal(exportBody, &exportSchema); err != nil {
		t.Fatal(err)
	}
	if exportSchema["$schema"] != "https://json-schema.org/draft/2020-12/schema" {
		t.Fatal("unexpected export schema draft")
	}

	seen := map[string]bool{}
	for _, tool := range mcp.Tools(mcp.Services{}) {
		if tool.Name == "" || seen[tool.Name] {
			t.Fatalf("invalid MCP tool %q", tool.Name)
		}
		seen[tool.Name] = true
		if tool.InputSchema["type"] != "object" {
			t.Fatalf("tool %s has invalid schema", tool.Name)
		}
	}
}
