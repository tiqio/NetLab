package domain

import (
	"fmt"
	"math"
)

type PlacementResourceType string
type PlacementFootprintClass string
type PlacementAssignmentReason string

const (
	PlacementNode                     PlacementResourceType     = "node"
	PlacementNetworkObject            PlacementResourceType     = "network_object"
	MaxPlacementCoordinate                                      = 1_000_000
	FootprintNodeStandard             PlacementFootprintClass   = "node-standard"
	FootprintNodeWide                 PlacementFootprintClass   = "node-wide"
	FootprintNetworkObjectStandard    PlacementFootprintClass   = "network-object-standard"
	FootprintNetworkObjectWide        PlacementFootprintClass   = "network-object-wide"
	PlacementReasonPreferredAvailable PlacementAssignmentReason = "preferred_available"
	PlacementReasonCollisionAvoided   PlacementAssignmentReason = "collision_avoided"
	PlacementReasonDefaultAnchor      PlacementAssignmentReason = "default_anchor"
	PlacementAlgorithmVersion                                   = 1
)

type PlacementIntent struct {
	PreferredX     *float64                `json:"preferred_x,omitempty"`
	PreferredY     *float64                `json:"preferred_y,omitempty"`
	FootprintClass PlacementFootprintClass `json:"footprint_class,omitempty"`
}

type PlacementPoint struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type PlacementFootprint struct {
	Class                                 PlacementFootprintClass `json:"class"`
	Width, Height, ClearanceX, ClearanceY float64
}

type PlacementOccupancy struct {
	ResourceID     ID
	X, Y           float64
	FootprintClass PlacementFootprintClass
}

type PlacementAssignment struct {
	Placement        TopologyPlacement         `json:"placement"`
	RequestedCenter  *PlacementPoint           `json:"requested_center,omitempty"`
	AssignedCenter   PlacementPoint            `json:"assigned_center"`
	Adjusted         bool                      `json:"adjusted"`
	Reason           PlacementAssignmentReason `json:"reason"`
	FootprintClass   PlacementFootprintClass   `json:"footprint_class"`
	AlgorithmVersion int                       `json:"algorithm_version"`
}

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

func DefaultPlacementFootprintClass(resourceType PlacementResourceType) PlacementFootprintClass {
	if resourceType == PlacementNetworkObject {
		return FootprintNetworkObjectStandard
	}
	return FootprintNodeStandard
}

func ValidatePlacementIntent(resourceType PlacementResourceType, intent *PlacementIntent) error {
	if intent == nil {
		return nil
	}
	if (intent.PreferredX == nil) != (intent.PreferredY == nil) {
		return fmt.Errorf("preferred placement coordinates must be provided together")
	}
	if intent.PreferredX != nil {
		if math.IsNaN(*intent.PreferredX) || math.IsNaN(*intent.PreferredY) || math.IsInf(*intent.PreferredX, 0) || math.IsInf(*intent.PreferredY, 0) || math.Abs(*intent.PreferredX) > MaxPlacementCoordinate || math.Abs(*intent.PreferredY) > MaxPlacementCoordinate {
			return fmt.Errorf("preferred placement coordinates out of bounds")
		}
	}
	class := intent.FootprintClass
	if class == "" {
		return nil
	}
	if resourceType == PlacementNode && class != FootprintNodeStandard && class != FootprintNodeWide {
		return fmt.Errorf("footprint class %q is not valid for nodes", class)
	}
	if resourceType == PlacementNetworkObject && class != FootprintNetworkObjectStandard && class != FootprintNetworkObjectWide {
		return fmt.Errorf("footprint class %q is not valid for network objects", class)
	}
	return nil
}
