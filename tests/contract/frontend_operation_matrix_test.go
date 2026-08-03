package contract

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestFrontendOperationMatrixAndWorkspaceSchema(t *testing.T) {
	registryBody, err := os.ReadFile("../../web/src/api/operationRegistry.ts")
	if err != nil {
		t.Fatal(err)
	}
	clientBody, err := os.ReadFile("../../web/src/api/index.ts")
	if err != nil {
		t.Fatal(err)
	}
	matrixBody, err := os.ReadFile("../../specs/002-frontend-ux-modernization/contracts/backend-integration-matrix.md")
	if err != nil {
		t.Fatal(err)
	}

	methods := regexp.MustCompile(`apiMethod: "([A-Za-z][A-Za-z0-9]+)"`).FindAllStringSubmatch(string(registryBody), -1)
	if len(methods) < 20 {
		t.Fatalf("frontend operation registry has only %d mutations", len(methods))
	}
	for _, method := range methods {
		name := method[1]
		if !strings.Contains(string(clientBody), name+":") {
			t.Errorf("operation registry references missing generated API method %s", name)
		}
		if !strings.Contains(string(matrixBody), "`"+name+"`") {
			t.Errorf("backend integration matrix does not document %s", name)
		}
	}

	schemaBody, err := os.ReadFile("../../specs/002-frontend-ux-modernization/contracts/workspace-preferences.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]any
	if err = json.Unmarshal(schemaBody, &schema); err != nil {
		t.Fatal(err)
	}
	if schema["$schema"] != "https://json-schema.org/draft/2020-12/schema" {
		t.Fatalf("unexpected workspace preference schema draft %v", schema["$schema"])
	}
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("workspace preference schema properties are missing")
	}
	for _, forbidden := range []string{"credential", "secret", "capture", "console", "bootstrap", "image_bytes"} {
		if _, exists := properties[forbidden]; exists {
			t.Errorf("browser-local schema must not persist %s", forbidden)
		}
	}
}
