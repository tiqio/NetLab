package compliance

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

var schemaFiles = []string{
	"compliance-ledger.schema.json",
	"evidence-record.schema.json",
	"deployment-authority.schema.json",
	"template-readiness.schema.json",
	"acceptance-run.schema.json",
}

func LoadContractSchemas(directory string) (map[string]json.RawMessage, error) {
	result := make(map[string]json.RawMessage, len(schemaFiles))
	for _, name := range schemaFiles {
		body, err := os.ReadFile(filepath.Join(directory, name))
		if err != nil {
			return nil, fmt.Errorf("read schema %s: %w", name, err)
		}
		var document map[string]any
		if err := json.Unmarshal(body, &document); err != nil {
			return nil, fmt.Errorf("parse schema %s: %w", name, err)
		}
		if document["$schema"] == nil || document["type"] != "object" {
			return nil, fmt.Errorf("schema %s lacks required object declaration", name)
		}
		result[name] = body
	}
	return result, nil
}
