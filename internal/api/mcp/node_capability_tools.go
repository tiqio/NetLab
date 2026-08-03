package mcp

import (
	"github.com/gin-gonic/gin"
	"github.com/netlab/netlab/internal/app/query"
	"github.com/netlab/netlab/internal/domain"
)

func NodeCapabilityTool(service *query.RuntimeCapabilityService) Tool {
	return Tool{Name: "netlab.nodes.capabilities", Description: "Return current runtime capability observations for one node.", InputSchema: requiredObject(map[string]any{"node_id": stringProperty("Node ID")}, "node_id"), Handler: func(c *gin.Context, arguments map[string]any) (any, error) {
		if service == nil {
			return unavailable("node capability observations")
		}
		nodeID, err := argumentString(arguments, "node_id")
		if err != nil {
			return nil, err
		}
		observations, err := service.Get(c.Request.Context(), domain.ID(nodeID))
		if err != nil {
			return nil, err
		}
		return map[string]any{"node_id": nodeID, "observations": observations}, nil
	}}
}
