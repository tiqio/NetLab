package httpapi

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/netlab/netlab/internal/domain"
)

type LinkReconnectCommand interface {
	Reconnect(context.Context, domain.ID, domain.Revision, domain.ID, domain.ID, string) (domain.OperationTask, error)
}

type LinkReconnectHandlers struct{ commands LinkReconnectCommand }

func NewLinkReconnectHandlers(commands LinkReconnectCommand) *LinkReconnectHandlers {
	return &LinkReconnectHandlers{commands: commands}
}

func (h *LinkReconnectHandlers) Register(engine *gin.Engine) {
	engine.POST("/api/v1/links/:linkId/reconnect", h.reconnect)
}

func (h *LinkReconnectHandlers) reconnect(c *gin.Context) {
	if h.commands == nil {
		writeProblem(c, http.StatusServiceUnavailable, domain.Problem{Code: "temporary_unavailable", Message: "link reconnect service unavailable", Retryable: true})
		return
	}
	revision, err := ParseRevision(c.GetHeader("If-Match"))
	if err != nil {
		writeProblem(c, http.StatusPreconditionRequired, domain.Problem{Code: "precondition_required", Message: "valid If-Match revision required"})
		return
	}
	var input struct {
		RetainedEndpointID    domain.ID `json:"retained_endpoint_id"`
		ReplacementEndpointID domain.ID `json:"replacement_endpoint_id"`
	}
	if err = c.ShouldBindJSON(&input); err != nil {
		handleError(c, err)
		return
	}
	value, err := h.commands.Reconnect(c, domain.ID(c.Param("linkId")), revision, input.RetainedEndpointID, input.ReplacementEndpointID, c.GetHeader("Idempotency-Key"))
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"task": value})
}
