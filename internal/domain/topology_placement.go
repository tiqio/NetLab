package domain

import (
	"fmt"
	"math"
)

type PlacementResourceType string

const (
	PlacementNode          PlacementResourceType = "node"
	PlacementNetworkObject PlacementResourceType = "network_object"
	MaxPlacementCoordinate                       = 1_000_000
)

type TopologyPlacement struct {
	LaboratoryID ID                    `json:"laboratory_id"`
	ResourceID   ID                    `json:"resource_id"`
	ResourceType PlacementResourceType `json:"resource_type"`
	X            float64               `json:"x"`
	Y            float64               `json:"y"`
	Revision     Revision              `json:"revision"`
}

type PlacementUpdate struct {
	ResourceID   ID                    `json:"resource_id"`
	ResourceType PlacementResourceType `json:"resource_type"`
	X            float64               `json:"x"`
	Y            float64               `json:"y"`
	Revision     Revision              `json:"revision,omitempty"`
}

func ValidatePlacementBatch(values []PlacementUpdate) error {
	if len(values) == 0 || len(values) > 100 {
		return fmt.Errorf("placement batch must contain 1 to 100 items")
	}
	seen := map[ID]struct{}{}
	for _, value := range values {
		if value.ResourceID == "" {
			return fmt.Errorf("placement resource id required")
		}
		if value.ResourceType != PlacementNode && value.ResourceType != PlacementNetworkObject {
			return fmt.Errorf("unsupported placement resource type %q", value.ResourceType)
		}
		if math.IsNaN(value.X) || math.IsNaN(value.Y) || math.IsInf(value.X, 0) || math.IsInf(value.Y, 0) || math.Abs(value.X) > MaxPlacementCoordinate || math.Abs(value.Y) > MaxPlacementCoordinate {
			return fmt.Errorf("placement coordinates out of bounds")
		}
		if _, ok := seen[value.ResourceID]; ok {
			return fmt.Errorf("duplicate placement resource %s", value.ResourceID)
		}
		seen[value.ResourceID] = struct{}{}
	}
	return nil
}
