package mcp

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/netlab/netlab/internal/domain"
)

type TrafficWorkloadCommands interface {
	List(context.Context, domain.ID) ([]domain.TrafficWorkload, error)
	Get(context.Context, domain.ID) (domain.TrafficWorkload, error)
	Create(context.Context, domain.TrafficWorkload, string) (domain.OperationTask, error)
	Start(context.Context, domain.ID, domain.Revision, string) (domain.OperationTask, error)
	Stop(context.Context, domain.ID, domain.Revision, string) (domain.OperationTask, error)
	Delete(context.Context, domain.ID, domain.Revision, string) (domain.OperationTask, error)
}

func TrafficWorkloadTools(service TrafficWorkloadCommands) []Tool {
	if service == nil {
		return nil
	}
	definition := map[string]any{
		"laboratory_id": stringProperty("Laboratory ID"), "name": stringProperty("Workload name"),
		"source": map[string]any{"type": "object"}, "protocol": enumProperty("icmp", "http", "dns"),
		"address_family": enumProperty("auto", "ipv4", "ipv6"), "destination": map[string]any{"type": "object"},
		"interval_seconds": map[string]any{"type": "integer"}, "timeout_seconds": map[string]any{"type": "integer"},
	}
	return []Tool{
		{Name: "netlab.traffic_workloads.list", Description: "List durable traffic workloads and aggregates.", InputSchema: requiredObject(map[string]any{"laboratory_id": stringProperty("Laboratory ID")}, "laboratory_id"), Handler: func(c *gin.Context, args map[string]any) (any, error) {
			return service.List(c, domain.ID(optionalString(args, "laboratory_id")))
		}},
		{Name: "netlab.traffic_workloads.get", Description: "Get one durable traffic workload.", InputSchema: requiredObject(map[string]any{"workload_id": stringProperty("Workload ID")}, "workload_id"), Handler: func(c *gin.Context, args map[string]any) (any, error) {
			return service.Get(c, domain.ID(optionalString(args, "workload_id")))
		}},
		{Name: "netlab.traffic_workloads.create", Description: "Create a durable ICMP, HTTP, or DNS workload.", InputSchema: mutationSchema(definition, "laboratory_id", "name", "source", "protocol", "address_family", "destination", "interval_seconds", "timeout_seconds"), Handler: func(c *gin.Context, args map[string]any) (any, error) {
			var value domain.TrafficWorkload
			if err := decodeArgument(args, &value); err != nil {
				return nil, err
			}
			taskValue, err := service.Create(c, value, optionalString(args, "idempotency_key"))
			return map[string]any{"workload_id": taskValue.ResourceID, "task": taskValue}, err
		}},
		{Name: "netlab.traffic_workloads.start", Description: "Start a durable traffic workload.", InputSchema: mutationSchema(map[string]any{"workload_id": stringProperty("Workload ID")}, "workload_id", "expected_revision"), Handler: func(c *gin.Context, args map[string]any) (any, error) {
			taskValue, err := service.Start(c, domain.ID(optionalString(args, "workload_id")), revisionArgument(args), optionalString(args, "idempotency_key"))
			return map[string]any{"task": taskValue}, err
		}},
		{Name: "netlab.traffic_workloads.stop", Description: "Stop a durable traffic workload.", InputSchema: mutationSchema(map[string]any{"workload_id": stringProperty("Workload ID")}, "workload_id", "expected_revision"), Handler: func(c *gin.Context, args map[string]any) (any, error) {
			taskValue, err := service.Stop(c, domain.ID(optionalString(args, "workload_id")), revisionArgument(args), optionalString(args, "idempotency_key"))
			return map[string]any{"task": taskValue}, err
		}},
		{Name: "netlab.traffic_workloads.delete", Description: "Delete a durable traffic workload.", InputSchema: mutationSchema(map[string]any{"workload_id": stringProperty("Workload ID")}, "workload_id", "expected_revision"), Handler: func(c *gin.Context, args map[string]any) (any, error) {
			taskValue, err := service.Delete(c, domain.ID(optionalString(args, "workload_id")), revisionArgument(args), optionalString(args, "idempotency_key"))
			return map[string]any{"task": taskValue}, err
		}},
	}
}
