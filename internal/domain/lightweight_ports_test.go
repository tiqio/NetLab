package domain

import "testing"

func TestApplyLightweightPortDefaultsOnCreate(t *testing.T) {
	for _, kind := range []string{NetworkSwitchL2, NetworkSwitchL3} {
		config := ApplyLightweightPortDefaultsOnCreate(kind, nil)
		key := "ports"
		if kind == NetworkSwitchL3 {
			key = "interfaces"
		}
		values, ok := config[key].([]any)
		if !ok || len(values) != 4 {
			t.Fatalf("kind=%s config=%+v", kind, config)
		}
		for index, raw := range values {
			name := raw.(map[string]any)["name"]
			if name != "eth"+string(rune('0'+index)) {
				t.Fatalf("kind=%s index=%d name=%v", kind, index, name)
			}
		}
	}
}

func TestLightweightPortDefaultsPreserveExplicitAndLegacyConfig(t *testing.T) {
	explicit := map[string]any{"ports": []any{map[string]any{"name": "lan0"}}}
	created := ApplyLightweightPortDefaultsOnCreate(NetworkSwitchL2, explicit)
	if len(created["ports"].([]any)) != 1 {
		t.Fatalf("explicit config expanded: %+v", created)
	}
	if len(explicit["ports"].([]any)) != 1 {
		t.Fatalf("input mutated: %+v", explicit)
	}
}

func TestValidateUniqueLightweightPortNames(t *testing.T) {
	err := ValidateUniqueLightweightPortNames(NetworkSwitchL2, map[string]any{
		"ports": []any{map[string]any{"name": "eth0"}, map[string]any{"name": "eth0"}},
	})
	if err == nil {
		t.Fatal("expected duplicate name error")
	}
}
