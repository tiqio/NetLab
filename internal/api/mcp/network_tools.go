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
		{Name: "netlab.network_objects.diagnostics", Description: "Inspect desired and observed network-object runtime backing.", InputSchema: requiredObject(map[string]any{"object_id": stringProperty("Network object ID")}, "object_id"), Handler: func(c *gin.Context, args map[string]any) (any, error) {
			id, err := argumentString(args, "object_id")
			if err != nil {
				return nil, err
			}
			object, err := service.Get(c, domain.ID(id))
			if err != nil {
				return nil, err
			}
			backing, err := service.InspectObject(c, object.ID)
			if err != nil {
				return nil, err
			}
			result := map[string]any{"object_id": object.ID, "desired_state": object.DesiredState, "observed_state": object.ObservedState, "backing": backing}
			if runtime, diagnosticsErr := service.DiagnosticsObject(c, object.ID); diagnosticsErr == nil {
				result["runtime"] = runtime
			}
			return result, nil
		}},
		{Name: "netlab.network_objects.reconcile", Description: "Retry network-object runtime reconciliation through a durable task.", InputSchema: mutationSchema(map[string]any{"object_id": stringProperty("Network object ID")}, "object_id", "expected_revision"), Handler: func(c *gin.Context, args map[string]any) (any, error) {
			if operations == nil {
				return unavailable("network object reconcile")
			}
			id, err := argumentString(args, "object_id")
			if err != nil {
				return nil, err
			}
			object, taskValue, err := operations.ReconcileObject(c, domain.ID(id), revisionArgument(args), optionalString(args, "idempotency_key"))
			if err != nil {
				return nil, err
			}
			return map[string]any{"network_object": object, "task": taskValue}, nil
		}},
		{Name: "netlab.network_object_links.reconcile", Description: "Retry network-object-link runtime reconciliation through a durable task.", InputSchema: mutationSchema(map[string]any{"link_id": stringProperty("Network object link ID")}, "link_id", "expected_revision"), Handler: func(c *gin.Context, args map[string]any) (any, error) {
			if operations == nil {
				return unavailable("network object link reconcile")
			}
			id, err := argumentString(args, "link_id")
			if err != nil {
				return nil, err
			}
			link, taskValue, err := operations.ReconcileObjectLink(c, domain.ID(id), revisionArgument(args), optionalString(args, "idempotency_key"))
			if err != nil {
				return nil, err
			}
			return map[string]any{"network_object_link": link, "task": taskValue}, nil
		}},
		{Name: "netlab.network_object_links.create", Description: "Create a durable direct link between two network-object ports.", InputSchema: mutationSchema(map[string]any{"laboratory_id": stringProperty("Laboratory ID"), "object_a_id": stringProperty("Endpoint A object ID"), "port_a_name": stringProperty("Endpoint A port"), "object_b_id": stringProperty("Endpoint B object ID"), "port_b_name": stringProperty("Endpoint B port")}, "laboratory_id", "object_a_id", "port_a_name", "object_b_id", "port_b_name"), Handler: func(c *gin.Context, args map[string]any) (any, error) {
			if operations == nil {
				return unavailable("network object link create")
			}
			laboratoryID, err := argumentString(args, "laboratory_id")
			if err != nil {
				return nil, err
			}
			objectAID, err := argumentString(args, "object_a_id")
			if err != nil {
				return nil, err
			}
			portAName, err := argumentString(args, "port_a_name")
			if err != nil {
				return nil, err
			}
			objectBID, err := argumentString(args, "object_b_id")
			if err != nil {
				return nil, err
			}
			portBName, err := argumentString(args, "port_b_name")
			if err != nil {
				return nil, err
			}
			link, taskValue, err := operations.CreateObjectLink(c, domain.ID(laboratoryID), domain.ID(objectAID), portAName, domain.ID(objectBID), portBName, optionalString(args, "idempotency_key"))
			if err != nil {
				return nil, err
			}
			return map[string]any{"network_object_link": link, "task": taskValue}, nil
		}},
		{Name: "netlab.network_object_links.get", Description: "Get authoritative network-object link state.", InputSchema: requiredObject(map[string]any{"link_id": stringProperty("Network object link ID")}, "link_id"), Handler: func(c *gin.Context, args map[string]any) (any, error) {
			id, err := argumentString(args, "link_id")
			if err != nil {
				return nil, err
			}
			return service.GetObjectLink(c, domain.ID(id))
		}},
		{Name: "netlab.network_object_links.delete", Description: "Delete a network-object link through a revisioned durable task.", InputSchema: mutationSchema(map[string]any{"link_id": stringProperty("Network object link ID")}, "link_id", "expected_revision"), Handler: func(c *gin.Context, args map[string]any) (any, error) {
			if operations == nil {
				return unavailable("network object link delete")
			}
			id, err := argumentString(args, "link_id")
			if err != nil {
				return nil, err
			}
			link, taskValue, err := operations.DeleteObjectLink(c, domain.ID(id), revisionArgument(args), optionalString(args, "idempotency_key"))
			if err != nil {
				return nil, err
			}
			return map[string]any{"network_object_link": link, "task": taskValue}, nil
		}},
		{Name: "netlab.network_objects.create", Description: "Create a PC, switch, bridge, or NAT network object with authoritative placement.", InputSchema: mutationSchema(map[string]any{"lab_id": stringProperty("Laboratory ID"), "name": stringProperty("Object name"), "kind": map[string]any{"type": "string", "enum": []string{"pc", "switch_l2", "switch_l3", "bridge", "nat_bridge"}}, "config": map[string]any{"type": "object"}, "placement_intent": map[string]any{"type": "object"}}, "lab_id", "name", "kind", "expected_revision"), Handler: func(c *gin.Context, args map[string]any) (any, error) {
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
			var intent *domain.PlacementIntent
			if args["placement_intent"] != nil {
				intent = &domain.PlacementIntent{}
				if err = decodeArgument(args["placement_intent"], intent); err != nil {
					return nil, err
				}
			}
			object, assignment, laboratoryRevision, taskValue, err := operations.CreateWithPlacement(c, domain.ID(labID), revisionArgument(args), name, kind, config, intent, optionalString(args, "idempotency_key"), "mcp")
			if err != nil {
				return nil, err
			}
			return map[string]any{"network_object": object, "task": taskValue, "placement_assignment": assignment, "laboratory_revision": laboratoryRevision}, nil
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
