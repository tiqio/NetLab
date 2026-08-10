package mcp

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/netlab/netlab/internal/domain"
)

type TopologyConnectionCommands interface {
	Create(context.Context, domain.ID, domain.ConnectionEndpoint, domain.ConnectionEndpoint, domain.TopologyConnectionConfig, string) (domain.TopologyConnection, domain.OperationTask, error)
	List(context.Context, domain.ID) ([]domain.TopologyConnection, error)
	Get(context.Context, domain.ID) (domain.TopologyConnection, error)
	Delete(context.Context, domain.ID, domain.Revision, string) (domain.TopologyConnection, domain.OperationTask, error)
}

type TopologyConnectionReadRepository interface {
	GetLaboratory(context.Context, domain.ID) (domain.Laboratory, error)
	ListConnectionEndpoints(context.Context, domain.ID) ([]domain.ConnectionEndpoint, error)
}

func TopologyConnectionTools(commands TopologyConnectionCommands, read TopologyConnectionReadRepository) []Tool {
	endpointSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"kind":        map[string]any{"type": "string", "enum": []string{"node_interface", "network_object_port", "network_object_access"}},
			"resource_id": stringProperty("Node or network object ID"),
			"port_id":     stringProperty("Node interface ID"),
			"port_name":   stringProperty("Network object port name"),
		},
		"required": []string{"kind", "resource_id"},
	}
	return []Tool{
		{Name: "netlab.topology_connections.create", Description: "Create a topology connection from two normalized endpoints without selecting the backing model.", InputSchema: mutationSchema(map[string]any{"laboratory_id": stringProperty("Laboratory ID"), "source": endpointSchema, "target": endpointSchema, "config": map[string]any{"type": "object"}}, "laboratory_id", "expected_revision", "idempotency_key", "source", "target"), Handler: func(c *gin.Context, args map[string]any) (any, error) {
			laboratoryText, err := argumentString(args, "laboratory_id")
			if err != nil {
				return nil, err
			}
			laboratoryID := domain.ID(laboratoryText)
			laboratory, err := read.GetLaboratory(c, laboratoryID)
			if err != nil {
				return nil, err
			}
			if laboratory.Revision != revisionArgument(args) {
				return nil, domain.Problem{Code: "revision_conflict", Message: "laboratory revision changed", Retryable: true, ResourceType: "laboratory", ResourceID: laboratoryID, Phase: "connection_admission", Cleanup: "no topology mutation was submitted", OperatorHint: "refresh the topology and retry with a new idempotency key"}
			}
			var source, target domain.ConnectionEndpoint
			if err = decodeArgument(args["source"], &source); err != nil {
				return nil, err
			}
			if err = decodeArgument(args["target"], &target); err != nil {
				return nil, err
			}
			var config domain.TopologyConnectionConfig
			if args["config"] != nil {
				if err = decodeArgument(args["config"], &config); err != nil {
					return nil, err
				}
			}
			connection, taskValue, err := commands.Create(c, laboratoryID, source, target, config, optionalString(args, "idempotency_key"))
			if err != nil {
				return nil, err
			}
			return map[string]any{"connection": connection, "task": taskValue, "laboratory_revision": laboratory.Revision}, nil
		}},
		{Name: "netlab.topology_connections.list", Description: "List authoritative unified connections and endpoint availability.", InputSchema: requiredObject(map[string]any{"laboratory_id": stringProperty("Laboratory ID")}, "laboratory_id"), Handler: func(c *gin.Context, args map[string]any) (any, error) {
			laboratoryText, err := argumentString(args, "laboratory_id")
			if err != nil {
				return nil, err
			}
			laboratoryID := domain.ID(laboratoryText)
			laboratory, err := read.GetLaboratory(c, laboratoryID)
			if err != nil {
				return nil, err
			}
			connections, err := commands.List(c, laboratoryID)
			if err != nil {
				return nil, err
			}
			endpoints, err := read.ListConnectionEndpoints(c, laboratoryID)
			if err != nil {
				return nil, err
			}
			return map[string]any{"laboratory_revision": laboratory.Revision, "connections": connections, "endpoints": endpoints}, nil
		}},
		{Name: "netlab.topology_connections.get", Description: "Get authoritative state for one unified topology connection.", InputSchema: requiredObject(map[string]any{"connection_id": stringProperty("Connection ID")}, "connection_id"), Handler: func(c *gin.Context, args map[string]any) (any, error) {
			id, err := argumentString(args, "connection_id")
			if err != nil {
				return nil, err
			}
			return commands.Get(c, domain.ID(id))
		}},
		{Name: "netlab.topology_connections.delete", Description: "Delete a unified topology connection through a durable task.", InputSchema: mutationSchema(map[string]any{"connection_id": stringProperty("Connection ID")}, "connection_id", "expected_revision", "idempotency_key"), Handler: func(c *gin.Context, args map[string]any) (any, error) {
			id, err := argumentString(args, "connection_id")
			if err != nil {
				return nil, err
			}
			connection, taskValue, err := commands.Delete(c, domain.ID(id), revisionArgument(args), optionalString(args, "idempotency_key"))
			if err != nil {
				return nil, err
			}
			return map[string]any{"connection": connection, "task": taskValue, "laboratory_revision": connection.Revision}, nil
		}},
	}
}
