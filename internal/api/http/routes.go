package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/netlab/netlab/internal/app/audit"
	"github.com/netlab/netlab/internal/app/command"
	"github.com/netlab/netlab/internal/app/query"
	"github.com/netlab/netlab/internal/domain"
)

type AutomationHandlers struct {
	tasks      *query.TaskService
	exporter   *command.ExportService
	importer   *command.ImportService
	automation *command.AutomationTaskService
	audit      *audit.Service
	release    *query.ReleaseService
}

func (h *AutomationHandlers) SetReleaseService(service *query.ReleaseService) {
	h.release = service
}

func NewAutomationHandlers(tasks *query.TaskService, exporter *command.ExportService, importer *command.ImportService, automation *command.AutomationTaskService, auditService *audit.Service) *AutomationHandlers {
	return &AutomationHandlers{tasks: tasks, exporter: exporter, importer: importer, automation: automation, audit: auditService}
}

func (h *AutomationHandlers) Register(engine *gin.Engine) {
	api := engine.Group("/api/v1")
	api.GET("/capabilities", h.capabilities)
	api.GET("/tasks", h.listTasks)
	api.GET("/tasks/:taskId", h.getTask)
	api.POST("/tasks/:taskId/cancel", h.cancelTask)
	api.GET("/audit-events", h.listAudit)
	api.POST("/labs/:labId/exports", h.exportLab)
	api.POST("/labs/:labId/duplicate", h.duplicateLab)
	api.POST("/lab-imports", h.importLab)
}

func (h *AutomationHandlers) duplicateLab(c *gin.Context) {
	var body struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		handleError(c, err)
		return
	}
	if h.automation == nil {
		handleError(c, domain.Problem{Code: "operation_unavailable", Message: "laboratory automation unavailable"})
		return
	}
	value, err := h.automation.Duplicate(c, domain.ID(c.Param("labId")), body.Name, c.GetHeader("Idempotency-Key"))
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"task": value})
}

func (h *AutomationHandlers) capabilities(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"api_version": "v1", "single_host": true, "authentication": false,
		"runtimes":        []string{"qemu", "docker", "namespace"},
		"network_objects": []string{"bridge", "nat_bridge", "pc", "switch_l2", "switch_l3"},
		"console_modes":   []string{"telnet", "vnc"},
		"features":        []string{"mcp", "live_capture", "traffic_filter", "qmp_hotplug", "qga_exec", "port_mapping", "cpu_quota"},
		"release":         h.release.Get(),
	})
}

func (h *AutomationHandlers) listTasks(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	values, err := h.tasks.List(c, limit)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, values)
}

func (h *AutomationHandlers) getTask(c *gin.Context) {
	value, err := h.tasks.Get(c, domain.ID(c.Param("taskId")))
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, value)
}

func (h *AutomationHandlers) cancelTask(c *gin.Context) {
	value, err := h.tasks.Cancel(c, domain.ID(c.Param("taskId")))
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, value)
}

func (h *AutomationHandlers) listAudit(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	values, err := h.audit.List(c, limit)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, values)
}

func (h *AutomationHandlers) exportLab(c *gin.Context) {
	if h.automation == nil {
		handleError(c, domain.Problem{Code: "operation_unavailable", Message: "laboratory automation unavailable"})
		return
	}
	value, err := h.automation.Export(c, domain.ID(c.Param("labId")), 24*time.Hour, c.GetHeader("Idempotency-Key"))
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"task": value})
}

func (h *AutomationHandlers) importLab(c *gin.Context) {
	var bundle command.LaboratoryExport
	if err := c.ShouldBindJSON(&bundle); err != nil {
		writeProblem(c, http.StatusBadRequest, domain.Problem{Code: "invalid_export", Message: err.Error()})
		return
	}
	if h.automation == nil {
		handleError(c, domain.Problem{Code: "operation_unavailable", Message: "laboratory automation unavailable"})
		return
	}
	value, err := h.automation.Import(c, bundle, c.GetHeader("Idempotency-Key"))
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"task": value})
}
