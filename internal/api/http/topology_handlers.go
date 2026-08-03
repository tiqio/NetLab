package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/netlab/netlab/internal/app/command"
	"github.com/netlab/netlab/internal/app/query"
	"github.com/netlab/netlab/internal/domain"
	qemuRuntime "github.com/netlab/netlab/internal/runtime/qemu"
	storesqlite "github.com/netlab/netlab/internal/store/sqlite"
	"net/http"
	"strconv"
)

type TopologyHandlers struct {
	labs          *command.LaboratoryService
	labQueries    *query.LaboratoryService
	nodes         *command.NodeService
	links         *command.LinkService
	tasks         *storesqlite.Repositories
	idempotency   *command.IdempotencyService
	operations    *command.TopologyTaskService
	labOperations *command.LaboratoryTaskService
}

func NewTopologyHandlers(labs *command.LaboratoryService, labQueries *query.LaboratoryService, nodes *command.NodeService, links *command.LinkService, tasks *storesqlite.Repositories, options ...any) *TopologyHandlers {
	handler := &TopologyHandlers{labs: labs, labQueries: labQueries, nodes: nodes, links: links, tasks: tasks}
	for _, option := range options {
		if value, ok := option.(*command.IdempotencyService); ok {
			handler.idempotency = value
		}
		if value, ok := option.(*command.TopologyTaskService); ok {
			handler.operations = value
		}
		if value, ok := option.(*command.LaboratoryTaskService); ok {
			handler.labOperations = value
		}
	}
	return handler
}
func (h *TopologyHandlers) Register(engine *gin.Engine) {
	api := engine.Group("/api/v1")
	api.GET("/labs", h.listLabs)
	api.POST("/labs", h.createLab)
	api.GET("/labs/:labId", h.getLab)
	api.PATCH("/labs/:labId", h.updateLab)
	api.DELETE("/labs/:labId", h.deleteLab)
	api.POST("/labs/:labId/nodes", h.createNode)
	api.PUT("/nodes/:nodeId/state", h.setNodeState)
	api.DELETE("/nodes/:nodeId", h.deleteNode)
	api.POST("/labs/:labId/links", h.createLink)
	api.DELETE("/links/:linkId", h.deleteLink)
}
func (h *TopologyHandlers) listLabs(c *gin.Context) {
	values, err := h.labQueries.List(c)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, values)
}
func (h *TopologyHandlers) createLab(c *gin.Context) {
	bodyBytes, err := c.GetRawData()
	if err != nil {
		handleError(c, err)
		return
	}
	var body struct {
		Name           string                `json:"name"`
		Description    string                `json:"description"`
		RecoveryPolicy domain.RecoveryPolicy `json:"recovery_policy"`
	}
	if err = json.Unmarshal(bodyBytes, &body); err != nil {
		writeProblem(c, 400, domain.Problem{Code: "invalid_request", Message: err.Error()})
		return
	}
	if h.idempotency == nil || c.GetHeader("Idempotency-Key") == "" {
		lab, createErr := h.labs.Create(c, body.Name, body.Description, body.RecoveryPolicy)
		if createErr != nil {
			handleError(c, createErr)
			return
		}
		c.Header("ETag", strconv.FormatInt(int64(lab.Revision), 10))
		c.JSON(http.StatusCreated, lab)
		return
	}
	result, err := h.idempotency.Execute(c, "POST:/api/v1/labs", c.GetHeader("Idempotency-Key"), bodyBytes, func(ctx context.Context) (int, []byte, error) {
		lab, createErr := h.labs.Create(ctx, body.Name, body.Description, body.RecoveryPolicy)
		if createErr != nil {
			return http.StatusBadRequest, nil, createErr
		}
		encoded, encodeErr := json.Marshal(lab)
		return http.StatusCreated, encoded, encodeErr
	})
	if err != nil {
		if !WriteIdempotencyError(c, err) {
			handleError(c, err)
		}
		return
	}
	if result.Replay {
		c.Header("Idempotency-Replayed", "true")
	}
	c.Data(result.StatusCode, "application/json", result.Body)
}
func (h *TopologyHandlers) getLab(c *gin.Context) {
	value, err := h.labQueries.Snapshot(c, domain.ID(c.Param("labId")))
	if err != nil {
		handleError(c, err)
		return
	}
	c.Header("X-Event-Sequence", strconv.FormatInt(value.Sequence, 10))
	c.Header("ETag", strconv.FormatInt(int64(value.Laboratory.Revision), 10))
	c.JSON(http.StatusOK, value)
}
func (h *TopologyHandlers) updateLab(c *gin.Context) {
	revision, err := ParseRevision(c.GetHeader("If-Match"))
	if err != nil {
		writeProblem(c, 428, domain.Problem{Code: "precondition_required", Message: "valid If-Match revision required"})
		return
	}
	var body struct {
		Name           string                `json:"name"`
		Description    string                `json:"description"`
		RecoveryPolicy domain.RecoveryPolicy `json:"recovery_policy"`
	}
	if err = c.ShouldBindJSON(&body); err != nil {
		handleError(c, err)
		return
	}
	lab, err := h.labs.Update(c, domain.ID(c.Param("labId")), revision, body.Name, body.Description, body.RecoveryPolicy)
	if err != nil {
		handleError(c, err)
		return
	}
	c.Header("ETag", strconv.FormatInt(int64(lab.Revision), 10))
	c.JSON(200, lab)
}
func (h *TopologyHandlers) deleteLab(c *gin.Context) {
	revision, err := ParseRevision(c.GetHeader("If-Match"))
	if err != nil {
		writeProblem(c, 428, domain.Problem{Code: "precondition_required", Message: "valid If-Match revision required"})
		return
	}
	if h.labOperations == nil {
		handleError(c, domain.Problem{Code: "operation_unavailable", Message: "laboratory deletion automation unavailable"})
		return
	}
	value, err := h.labOperations.Delete(c, domain.ID(c.Param("labId")), revision, c.GetHeader("Idempotency-Key"))
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"task": value})
}
func (h *TopologyHandlers) createNode(c *gin.Context) {
	var body struct {
		Name              string         `json:"name"`
		Kind              string         `json:"kind"`
		TemplateVersionID domain.ID      `json:"template_version_id"`
		ImageVersionID    domain.ID      `json:"image_version_id"`
		CPUCount          int            `json:"cpu_count"`
		CPUQuotaMicros    int64          `json:"cpu_quota_micros"`
		MemoryMiB         int            `json:"memory_mib"`
		StorageGiB        int            `json:"storage_gib"`
		InterfaceLimit    int            `json:"interface_limit"`
		ProcessLimit      int            `json:"process_limit"`
		InterfaceCount    int            `json:"interface_count"`
		Config            map[string]any `json:"config"`
		Bootstrap         struct {
			UserData      string `json:"user_data"`
			MetaData      string `json:"meta_data"`
			NetworkConfig string `json:"network_config"`
		} `json:"bootstrap"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		handleError(c, err)
		return
	}
	node, interfaces, err := h.nodes.CreateConfigured(c, domain.ID(c.Param("labId")), command.CreateNodeRequest{Name: body.Name, Kind: body.Kind, TemplateVersionID: body.TemplateVersionID, ImageVersionID: body.ImageVersionID, CPUCount: body.CPUCount, CPUQuotaMicros: body.CPUQuotaMicros, MemoryMiB: body.MemoryMiB, StorageGiB: body.StorageGiB, InterfaceLimit: body.InterfaceLimit, ProcessLimit: body.ProcessLimit, InterfaceCount: body.InterfaceCount, Config: body.Config, Bootstrap: qemuRuntime.SeedSpec{UserData: body.Bootstrap.UserData, MetaData: body.Bootstrap.MetaData, NetworkConfig: body.Bootstrap.NetworkConfig}})
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"node": node, "interfaces": interfaces})
}
func (h *TopologyHandlers) setNodeState(c *gin.Context) {
	revision, err := ParseRevision(c.GetHeader("If-Match"))
	if err != nil {
		writeProblem(c, 428, domain.Problem{Code: "precondition_required", Message: "valid If-Match revision required"})
		return
	}
	var body struct {
		DesiredState domain.DesiredState `json:"desired_state"`
	}
	if err = c.ShouldBindJSON(&body); err != nil {
		handleError(c, err)
		return
	}
	if h.operations != nil {
		taskValue, taskErr := h.operations.SetNodeState(c, domain.ID(c.Param("nodeId")), revision, body.DesiredState, c.GetHeader("Idempotency-Key"))
		if taskErr != nil {
			handleError(c, taskErr)
			return
		}
		c.JSON(http.StatusAccepted, gin.H{"task": taskValue})
		return
	}
	node, err := h.nodes.SetState(c, domain.ID(c.Param("nodeId")), revision, body.DesiredState)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, node)
}
func (h *TopologyHandlers) deleteNode(c *gin.Context) {
	revision, err := ParseRevision(c.GetHeader("If-Match"))
	if err != nil {
		writeProblem(c, 428, domain.Problem{Code: "precondition_required", Message: "valid If-Match revision required"})
		return
	}
	if h.operations != nil {
		taskValue, taskErr := h.operations.DeleteNode(c, domain.ID(c.Param("nodeId")), revision, c.GetHeader("Idempotency-Key"))
		if taskErr != nil {
			handleError(c, taskErr)
			return
		}
		c.JSON(http.StatusAccepted, gin.H{"task": taskValue})
		return
	}
	if err = h.nodes.Delete(c, domain.ID(c.Param("nodeId")), revision); err != nil {
		handleError(c, err)
		return
	}
	c.Status(http.StatusAccepted)
}
func (h *TopologyHandlers) createLink(c *gin.Context) {
	var body struct {
		EndpointAID domain.ID `json:"endpoint_a_id"`
		EndpointBID domain.ID `json:"endpoint_b_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		handleError(c, err)
		return
	}
	if h.operations != nil {
		link, taskValue, taskErr := h.operations.ConnectLink(c, domain.ID(c.Param("labId")), body.EndpointAID, body.EndpointBID, c.GetHeader("Idempotency-Key"))
		if taskErr != nil {
			handleError(c, taskErr)
			return
		}
		c.JSON(http.StatusAccepted, gin.H{"link": link, "task": taskValue})
		return
	}
	link, err := h.links.Connect(c, domain.ID(c.Param("labId")), body.EndpointAID, body.EndpointBID)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, link)
}
func (h *TopologyHandlers) deleteLink(c *gin.Context) {
	if h.operations != nil {
		taskValue, taskErr := h.operations.DisconnectLink(c, domain.ID(c.Param("linkId")), c.GetHeader("Idempotency-Key"))
		if taskErr != nil {
			handleError(c, taskErr)
			return
		}
		c.JSON(http.StatusAccepted, gin.H{"task": taskValue})
		return
	}
	if err := h.links.Disconnect(c, domain.ID(c.Param("linkId"))); err != nil {
		handleError(c, err)
		return
	}
	c.Status(http.StatusAccepted)
}
func (h *TopologyHandlers) getTask(c *gin.Context) {
	task, err := h.tasks.GetTask(c, domain.ID(c.Param("taskId")))
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(200, task)
}
func handleError(c *gin.Context, err error) {
	if errors.Is(err, storesqlite.ErrNotFound) {
		writeProblem(c, 404, domain.Problem{Code: "not_found", Message: "resource not found"})
		return
	}
	problem := domain.NormalizeProblem(err, domain.Problem{Code: "invalid_request", Phase: "http_request", Cleanup: "no cleanup required", OperatorHint: "correct the request and retry"})
	SetRetryHeaders(c, problem)
	writeProblem(c, problemHTTPStatus(problem), problem)
}

func problemHTTPStatus(problem domain.Problem) int {
	switch problem.Code {
	case "revision_conflict", "precondition_failed":
		return http.StatusPreconditionFailed
	case "precondition_required":
		return http.StatusPreconditionRequired
	case "not_found":
		return http.StatusNotFound
	case "idempotency_conflict", "invalid_node_transition", "capability_unsupported", "port_in_use":
		return http.StatusConflict
	case "invalid_topology", "invalid_port_name":
		return http.StatusUnprocessableEntity
	case "temporary_unavailable", "runtime_unavailable":
		return http.StatusServiceUnavailable
	case "resource_exhausted":
		if problem.Retryable {
			return http.StatusServiceUnavailable
		}
		return http.StatusTooManyRequests
	default:
		if problem.Retryable {
			return http.StatusServiceUnavailable
		}
		return http.StatusBadRequest
	}
}
