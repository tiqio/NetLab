package mcp

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/netlab/netlab/internal/app/command"
	"github.com/netlab/netlab/internal/domain"
)

type TopologyPlacementCommand interface {
	Update(context.Context, domain.ID, domain.Revision, []domain.PlacementUpdate) (command.TopologyPlacementResult, error)
}

func TopologyPlacementTools(service TopologyPlacementCommand) []Tool {
	if service == nil {
		return nil
	}
	placementSchema := map[string]any{
		"type": "object", "additionalProperties": false,
		"properties": map[string]any{
			"resource_id":   stringProperty("Node or network object ID"),
			"resource_type": map[string]any{"type": "string", "enum": []string{"node", "network_object"}},
			"x":             map[string]any{"type": "number"}, "y": map[string]any{"type": "number"},
			"revision": map[string]any{"type": "integer", "minimum": 1},
		},
		"required": []string{"resource_id", "resource_type", "x", "y"},
	}
	return []Tool{{
		Name: "netlab.topology.set_positions", Description: "Atomically update authoritative shared positions after an explicit manual move. This tool never performs initial placement; create tools return the authoritative initial assignment.",
		InputSchema: mutationSchema(map[string]any{
			"laboratory_id": stringProperty("Laboratory ID"),
			"placements":    map[string]any{"type": "array", "minItems": 1, "maxItems": 100, "items": placementSchema},
		}, "laboratory_id", "expected_revision", "placements", "idempotency_key"),
		Handler: func(c *gin.Context, args map[string]any) (any, error) {
			laboratoryID, err := argumentString(args, "laboratory_id")
			if err != nil {
				return nil, err
			}
			values, ok := args["placements"].([]any)
			if !ok {
				return nil, fmt.Errorf("placements is required")
			}
			updates := make([]domain.PlacementUpdate, 0, len(values))
			for _, raw := range values {
				value, ok := raw.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("placement must be an object")
				}
				resourceID, _ := value["resource_id"].(string)
				resourceType, _ := value["resource_type"].(string)
				x, xOK := value["x"].(float64)
				y, yOK := value["y"].(float64)
				if resourceID == "" || resourceType == "" || !xOK || !yOK {
					return nil, fmt.Errorf("placement resource_id, resource_type, x, and y are required")
				}
				updates = append(updates, domain.PlacementUpdate{ResourceID: domain.ID(resourceID), ResourceType: domain.PlacementResourceType(resourceType), X: x, Y: y, Revision: numericRevision(value["revision"])})
			}
			return service.Update(c, domain.ID(laboratoryID), revisionArgument(args), updates)
		},
	}}
}

func numericRevision(value any) domain.Revision {
	if number, ok := value.(float64); ok && number >= 0 {
		return domain.Revision(number)
	}
	return 0
}
