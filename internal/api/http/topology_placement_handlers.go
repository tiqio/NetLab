package httpapi

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/netlab/netlab/internal/app/command"
	"github.com/netlab/netlab/internal/domain"
)

type TopologyPlacementCommand interface {
	Update(context.Context, domain.ID, domain.Revision, []domain.PlacementUpdate) (command.TopologyPlacementResult, error)
}

type TopologyPlacementHandlers struct{ commands TopologyPlacementCommand }

func NewTopologyPlacementHandlers(commands TopologyPlacementCommand) *TopologyPlacementHandlers {
	return &TopologyPlacementHandlers{commands: commands}
}

func (h *TopologyPlacementHandlers) Register(engine *gin.Engine) {
	engine.PUT("/api/v1/labs/:labId/placements", h.update)
}

func (h *TopologyPlacementHandlers) update(c *gin.Context) {
	if h.commands == nil {
		writeProblem(c, http.StatusServiceUnavailable, domain.Problem{Code: "temporary_unavailable", Message: "topology placement service unavailable", Retryable: true})
		return
	}
	revision, err := ParseRevision(c.GetHeader("If-Match"))
	if err != nil {
		writeProblem(c, http.StatusPreconditionRequired, domain.Problem{Code: "precondition_required", Message: "valid If-Match revision required"})
		return
	}
	var input struct {
		Placements []domain.PlacementUpdate `json:"placements"`
	}
	if err = c.ShouldBindJSON(&input); err != nil {
		writeProblem(c, http.StatusBadRequest, domain.Problem{Code: "invalid_request", Message: err.Error()})
		return
	}
	result, err := h.commands.Update(c, domain.ID(c.Param("labId")), revision, input.Placements)
	if err != nil {
		handleError(c, err)
		return
	}
	c.Header("ETag", strconv.FormatInt(int64(result.LaboratoryRevision), 10))
	c.JSON(http.StatusOK, result)
}
