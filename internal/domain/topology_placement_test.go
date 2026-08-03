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
