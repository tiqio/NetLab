package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNodeJSONIncludesZeroResourceValues(t *testing.T) {
	body, err := json.Marshal(Node{})
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"cpu_count", "cpu_quota_micros", "memory_mib", "storage_gib", "interface_limit", "process_limit"} {
		if !strings.Contains(string(body), `"`+field+`":0`) {
			t.Fatalf("missing explicit zero %s in %s", field, body)
		}
	}
}
