package mcp

import (
	"github.com/gin-gonic/gin"
	"github.com/netlab/netlab/internal/app/query"
	"github.com/netlab/netlab/internal/domain"
)

func DeviceReadinessTool(service *query.DeviceReadinessService) Tool {
	return Tool{Name: "netlab.nodes.device_readiness", Description: "Read cable, guest, management and data-path readiness independently.", InputSchema: requiredObject(map[string]any{"node_id": stringProperty("Node ID")}, "node_id"), Handler: func(c *gin.Context, args map[string]any) (any, error) {
		if service == nil {
			return unavailable("device readiness")
		}
		id, err := argumentString(args, "node_id")
		if err != nil {
			return nil, err
		}
		return service.Get(c, domain.ID(id))
	}}
}
