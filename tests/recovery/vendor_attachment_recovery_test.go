package recovery

import (
	"encoding/json"
	"github.com/netlab/netlab/internal/domain"
	"testing"
)

func TestVendorRoleMetadataRecoversWithoutCredentials(t *testing.T) {
	node := domain.Node{Config: map[string]any{"device_roles": []any{map[string]any{"interface_id": "if-1", "role": "management", "address": "10.30.30.2/24"}}}}
	body, err := json.Marshal(node)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) == "" {
		t.Fatal("empty export")
	}
	var restored domain.Node
	if err = json.Unmarshal(body, &restored); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(restored.Config["device_roles"])
	if string(raw) == "null" {
		t.Fatal("roles lost")
	}
}
