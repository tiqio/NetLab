package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/netlab/netlab/internal/app/query"
)

type RuntimeOwnershipHandlers struct {
	service *query.RuntimeOwnershipService
}

func NewRuntimeOwnershipHandlers(service *query.RuntimeOwnershipService) *RuntimeOwnershipHandlers {
	return &RuntimeOwnershipHandlers{service: service}
}

func (h *RuntimeOwnershipHandlers) Register(engine *gin.Engine) {
	engine.GET("/api/v1/runtime-ownership", h.list)
}

func (h *RuntimeOwnershipHandlers) list(c *gin.Context) {
	values, err := h.service.List(c.Request.Context())
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, values)
}
