package contract

import (
	"encoding/json"
	"os"
	"testing"
)

func TestFrontendAcceptanceSchemas(t *testing.T) {
	for _, name := range []string{
		"interaction-inventory.schema.json",
		"acceptance-evidence.schema.json",
	} {
		body, err := os.ReadFile("../../specs/003-frontend-interaction-acceptance/contracts/" + name)
		if err != nil {
			t.Fatal(err)
		}
		var schema map[string]any
		if err := json.Unmarshal(body, &schema); err != nil {
			t.Fatal(err)
		}
		if schema["$schema"] != "https://json-schema.org/draft/2020-12/schema" {
			t.Fatalf("%s uses unexpected draft %v", name, schema["$schema"])
		}
		if schema["type"] != "object" {
			t.Fatalf("%s root type=%v", name, schema["type"])
		}
		if _, ok := schema["required"].([]any); !ok {
			t.Fatalf("%s missing required properties", name)
		}
	}
}
