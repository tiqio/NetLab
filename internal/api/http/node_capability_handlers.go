package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/netlab/netlab/internal/app/query"
	"github.com/netlab/netlab/internal/domain"
)

type NodeCapabilityHandlers struct {
	service *query.RuntimeCapabilityService
}

func NewNodeCapabilityHandlers(service *query.RuntimeCapabilityService) *NodeCapabilityHandlers {
	return &NodeCapabilityHandlers{service: service}
}

func (h *NodeCapabilityHandlers) Register(engine *gin.Engine) {
	engine.GET("/api/v1/nodes/:nodeId/capabilities", h.get)
}

func (h *NodeCapabilityHandlers) get(c *gin.Context) {
	observations, err := h.service.Get(c.Request.Context(), domain.ID(c.Param("nodeId")))
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"node_id": c.Param("nodeId"), "observations": observations})
}
