package httpapi

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/netlab/netlab/internal/app/command"
	"github.com/netlab/netlab/internal/domain"
)

type TopologyConnectionCommands interface {
	Create(context.Context, domain.ID, domain.ConnectionEndpoint, domain.ConnectionEndpoint, domain.TopologyConnectionConfig, string) (domain.TopologyConnection, domain.OperationTask, error)
	List(context.Context, domain.ID) ([]domain.TopologyConnection, error)
	Get(context.Context, domain.ID) (domain.TopologyConnection, error)
	Delete(context.Context, domain.ID, domain.Revision, string) (domain.TopologyConnection, domain.OperationTask, error)
}

type TopologyConnectionReadRepository interface {
	ListConnectionEndpoints(context.Context, domain.ID) ([]domain.ConnectionEndpoint, error)
	GetLaboratory(context.Context, domain.ID) (domain.Laboratory, error)
}

type TopologyConnectionHandlers struct {
	commands TopologyConnectionCommands
	read     TopologyConnectionReadRepository
}

func NewTopologyConnectionHandlers(commands TopologyConnectionCommands, read TopologyConnectionReadRepository) *TopologyConnectionHandlers {
	return &TopologyConnectionHandlers{commands: commands, read: read}
}

func (h *TopologyConnectionHandlers) Register(engine *gin.Engine) {
	api := engine.Group("/api/v1")
	api.GET("/labs/:labId/connections", h.list)
	api.POST("/labs/:labId/connections", h.create)
	api.GET("/connections/:connectionId", h.get)
	api.DELETE("/connections/:connectionId", h.delete)
}

func (h *TopologyConnectionHandlers) list(c *gin.Context) {
	laboratoryID := domain.ID(c.Param("labId"))
	laboratory, err := h.read.GetLaboratory(c, laboratoryID)
	if err != nil {
		handleError(c, err)
		return
	}
	connections, err := h.commands.List(c, laboratoryID)
	if err != nil {
		handleError(c, err)
		return
	}
	endpoints, err := h.read.ListConnectionEndpoints(c, laboratoryID)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"laboratory_revision": laboratory.Revision, "connections": connections, "endpoints": endpoints})
}

func (h *TopologyConnectionHandlers) create(c *gin.Context) {
	laboratoryID := domain.ID(c.Param("labId"))
	revision, err := ParseRevision(c.GetHeader("If-Match"))
	if err != nil {
		writeProblem(c, http.StatusPreconditionRequired, domain.Problem{Code: "precondition_required", Message: "valid If-Match laboratory revision required"})
		return
	}
	laboratory, err := h.read.GetLaboratory(c, laboratoryID)
	if err != nil {
		handleError(c, err)
		return
	}
	if laboratory.Revision != revision {
		handleError(c, domain.Problem{Code: "revision_conflict", Message: "laboratory revision changed", Retryable: true, ResourceType: "laboratory", ResourceID: laboratoryID, Phase: "connection_admission", Cleanup: "no topology mutation was submitted", OperatorHint: "refresh the topology and retry with a new idempotency key"})
		return
	}
	var body struct {
		Source domain.ConnectionEndpoint       `json:"source"`
		Target domain.ConnectionEndpoint       `json:"target"`
		Config domain.TopologyConnectionConfig `json:"config"`
		Entry  string                          `json:"entry_point"`
	}
	if err = c.ShouldBindJSON(&body); err != nil {
		handleError(c, err)
		return
	}
	entryPoint := body.Entry
	if entryPoint == "" {
		entryPoint = "http"
	}
	entryContext := command.WithTopologyConnectionEntryPoint(c, entryPoint)
	entryPoint = command.TopologyConnectionEntryPoint(entryContext, "http")
	c.Set("topology_entry_point", entryPoint)
	connection, taskValue, err := h.commands.Create(entryContext, laboratoryID, body.Source, body.Target, body.Config, c.GetHeader("Idempotency-Key"))
	if err != nil {
		handleError(c, err)
		return
	}
	c.Header("ETag", strconv.FormatInt(int64(laboratory.Revision), 10))
	c.JSON(http.StatusAccepted, gin.H{"connection": connection, "task": taskValue, "laboratory_revision": laboratory.Revision})
}

func (h *TopologyConnectionHandlers) get(c *gin.Context) {
	connection, err := h.commands.Get(c, domain.ID(c.Param("connectionId")))
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, connection)
}

func (h *TopologyConnectionHandlers) delete(c *gin.Context) {
	revision, err := ParseRevision(c.GetHeader("If-Match"))
	if err != nil {
		writeProblem(c, http.StatusPreconditionRequired, domain.Problem{Code: "precondition_required", Message: "valid If-Match connection revision required"})
		return
	}
	entryPoint := c.Query("entry_point")
	if entryPoint == "" {
		entryPoint = "http"
	}
	entryContext := command.WithTopologyConnectionEntryPoint(c, entryPoint)
	entryPoint = command.TopologyConnectionEntryPoint(entryContext, "http")
	c.Set("topology_entry_point", entryPoint)
	connection, taskValue, err := h.commands.Delete(entryContext, domain.ID(c.Param("connectionId")), revision, c.GetHeader("Idempotency-Key"))
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"connection": connection, "task": taskValue, "laboratory_revision": connection.Revision})
}
