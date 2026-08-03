package mcp

import (
	"github.com/gin-gonic/gin"
	"github.com/netlab/netlab/internal/app/reconcile"
	"github.com/netlab/netlab/internal/domain"
)

func NetworkTools(service *reconcile.NetworkObjectService, operations *reconcile.NetworkObjectTaskService) []Tool {
	if service == nil {
		return nil
	}
	return []Tool{
		{Name: "netlab.network_objects.create", Description: "Create a PC, switch, bridge, or NAT network object.", InputSchema: mutationSchema(map[string]any{"lab_id": stringProperty("Laboratory ID"), "name": stringProperty("Object name"), "kind": map[string]any{"type": "string", "enum": []string{"pc", "switch_l2", "switch_l3", "bridge", "nat_bridge"}}, "config": map[string]any{"type": "object"}}, "lab_id", "name", "kind"), Handler: func(c *gin.Context, args map[string]any) (any, error) {
			if operations == nil {
				return unavailable("network object create")
			}
			labID, err := argumentString(args, "lab_id")
			if err != nil {
				return nil, err
			}
			name, err := argumentString(args, "name")
			if err != nil {
				return nil, err
			}
			kind, err := argumentString(args, "kind")
			if err != nil {
				return nil, err
			}
			config, _ := args["config"].(map[string]any)
			object, taskValue, err := operations.Create(c, domain.ID(labID), name, kind, config, optionalString(args, "idempotency_key"))
			if err != nil {
				return nil, err
			}
			return map[string]any{"network_object": object, "task": taskValue}, nil
		}},
		{Name: "netlab.network_objects.get", Description: "Get network object state and configuration.", InputSchema: requiredObject(map[string]any{"object_id": stringProperty("Network object ID")}, "object_id"), Handler: func(c *gin.Context, args map[string]any) (any, error) {
			id, err := argumentString(args, "object_id")
			if err != nil {
				return nil, err
			}
			return service.Get(c, domain.ID(id))
		}},
		{Name: "netlab.network_objects.delete", Description: "Delete a network object and its owned runtime resources.", InputSchema: mutationSchema(map[string]any{"object_id": stringProperty("Network object ID")}, "object_id", "expected_revision"), Handler: func(c *gin.Context, args map[string]any) (any, error) {
			if operations == nil {
				return unavailable("network object delete")
			}
			id, err := argumentString(args, "object_id")
			if err != nil {
				return nil, err
			}
			taskValue, err := operations.Delete(c, domain.ID(id), revisionArgument(args), optionalString(args, "idempotency_key"))
			if err != nil {
				return nil, err
			}
			return map[string]any{"task": taskValue}, nil
		}},
	}
}
