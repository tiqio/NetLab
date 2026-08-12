package httpapi

import (
	"context"
	"net/http"

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

type TrafficWorkloadHandlers struct{ commands TrafficWorkloadCommands }

func NewTrafficWorkloadHandlers(commands TrafficWorkloadCommands) *TrafficWorkloadHandlers {
	return &TrafficWorkloadHandlers{commands: commands}
}

func (h *TrafficWorkloadHandlers) Register(engine *gin.Engine) {
	api := engine.Group("/api/v1")
	api.GET("/traffic-workloads", h.list)
	api.POST("/traffic-workloads", h.create)
	api.GET("/traffic-workloads/:workloadId", h.get)
	api.POST("/traffic-workloads/:workloadId/start", h.start)
	api.POST("/traffic-workloads/:workloadId/stop", h.stop)
	api.DELETE("/traffic-workloads/:workloadId", h.delete)
}

func (h *TrafficWorkloadHandlers) list(c *gin.Context) {
	values, err := h.commands.List(c, domain.ID(c.Query("laboratory_id")))
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, values)
}

func (h *TrafficWorkloadHandlers) get(c *gin.Context) {
	value, err := h.commands.Get(c, domain.ID(c.Param("workloadId")))
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, value)
}

func (h *TrafficWorkloadHandlers) create(c *gin.Context) {
	var value domain.TrafficWorkload
	if err := c.ShouldBindJSON(&value); err != nil {
		handleError(c, err)
		return
	}
	taskValue, err := h.commands.Create(c, value, c.GetHeader("Idempotency-Key"))
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"workload_id": taskValue.ResourceID, "task": taskValue})
}

func (h *TrafficWorkloadHandlers) start(c *gin.Context) { h.state(c, true) }
func (h *TrafficWorkloadHandlers) stop(c *gin.Context)  { h.state(c, false) }
func (h *TrafficWorkloadHandlers) state(c *gin.Context, running bool) {
	revision, err := ParseRevision(c.GetHeader("If-Match"))
	if err != nil {
		writeProblem(c, http.StatusPreconditionRequired, domain.Problem{Code: "precondition_required", Message: "valid If-Match revision required"})
		return
	}
	var taskValue domain.OperationTask
	if running {
		taskValue, err = h.commands.Start(c, domain.ID(c.Param("workloadId")), revision, c.GetHeader("Idempotency-Key"))
	} else {
		taskValue, err = h.commands.Stop(c, domain.ID(c.Param("workloadId")), revision, c.GetHeader("Idempotency-Key"))
	}
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"task": taskValue})
}

func (h *TrafficWorkloadHandlers) delete(c *gin.Context) {
	revision, err := ParseRevision(c.GetHeader("If-Match"))
	if err != nil {
		writeProblem(c, http.StatusPreconditionRequired, domain.Problem{Code: "precondition_required", Message: "valid If-Match revision required"})
		return
	}
	taskValue, err := h.commands.Delete(c, domain.ID(c.Param("workloadId")), revision, c.GetHeader("Idempotency-Key"))
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"task": taskValue})
}
