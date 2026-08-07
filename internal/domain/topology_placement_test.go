package domain

import (
	"math"
	"testing"
)

func TestValidatePlacementBatch(t *testing.T) {
	valid := []PlacementUpdate{{ResourceID: "node-1", ResourceType: PlacementNode, X: 10, Y: -20}}
	if err := ValidatePlacementBatch(valid); err != nil {
		t.Fatal(err)
	}
	for name, values := range map[string][]PlacementUpdate{
		"empty":     {},
		"duplicate": {{ResourceID: "node-1", ResourceType: PlacementNode}, {ResourceID: "node-1", ResourceType: PlacementNode}},
		"nan":       {{ResourceID: "node-1", ResourceType: PlacementNode, X: math.NaN()}},
		"kind":      {{ResourceID: "node-1", ResourceType: "capture"}},
	} {
		t.Run(name, func(t *testing.T) {
			if err := ValidatePlacementBatch(values); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestValidatePlacementIntent(t *testing.T) {
	x, y := 10.0, -20.0
	if err := ValidatePlacementIntent(PlacementNode, &PlacementIntent{PreferredX: &x, PreferredY: &y, FootprintClass: FootprintNodeStandard}); err != nil {
		t.Fatal(err)
	}
	for name, intent := range map[string]PlacementIntent{
		"partial": {PreferredX: &x},
		"bounds":  {PreferredX: func() *float64 { value := float64(MaxPlacementCoordinate + 1); return &value }(), PreferredY: &y},
		"class":   {PreferredX: &x, PreferredY: &y, FootprintClass: FootprintNetworkObjectStandard},
	} {
		t.Run(name, func(t *testing.T) {
			if err := ValidatePlacementIntent(PlacementNode, &intent); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}
