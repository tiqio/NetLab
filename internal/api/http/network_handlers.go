package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/netlab/netlab/internal/app/reconcile"
	"github.com/netlab/netlab/internal/domain"
	"github.com/netlab/netlab/internal/runtime/linuxnet"
)

type NetworkHandlers struct {
	service    *reconcile.NetworkObjectService
	operations *reconcile.NetworkObjectTaskService
	pc         *linuxnet.PCRuntime
	nat        *linuxnet.NATRuntime
	switchL3   *linuxnet.SwitchL3Runtime
}

func NewNetworkHandlers(service *reconcile.NetworkObjectService, operations *reconcile.NetworkObjectTaskService, pc *linuxnet.PCRuntime, runtimes ...any) *NetworkHandlers {
	handler := &NetworkHandlers{service: service, operations: operations, pc: pc}
	for _, runtime := range runtimes {
		if value, ok := runtime.(*linuxnet.NATRuntime); ok {
			handler.nat = value
		}
		if value, ok := runtime.(*linuxnet.SwitchL3Runtime); ok {
			handler.switchL3 = value
		}
	}
	return handler
}

func (h *NetworkHandlers) Register(engine *gin.Engine) {
	api := engine.Group("/api/v1")
	api.GET("/labs/:labId/network-objects", h.list)
	api.POST("/labs/:labId/network-objects", h.create)
	api.GET("/network-objects/:objectId", h.get)
	api.PATCH("/network-objects/:objectId", h.update)
	api.DELETE("/network-objects/:objectId", h.delete)
	api.POST("/network-objects/:objectId/attachments", h.attach)
	api.GET("/labs/:labId/network-object-links", h.listObjectLinks)
	api.POST("/labs/:labId/network-object-links", h.createObjectLink)
	api.GET("/network-object-links/:linkId", h.getObjectLink)
	api.DELETE("/network-object-links/:linkId", h.deleteObjectLink)
	api.GET("/network-objects/:objectId/diagnostics", h.diagnostics)
}

func (h *NetworkHandlers) listObjectLinks(c *gin.Context) {
	values, err := h.service.ListObjectLinks(c, domain.ID(c.Param("labId")))
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, values)
}

func (h *NetworkHandlers) createObjectLink(c *gin.Context) {
	var body struct {
		ObjectAID domain.ID `json:"object_a_id"`
		PortAName string    `json:"port_a_name"`
		ObjectBID domain.ID `json:"object_b_id"`
		PortBName string    `json:"port_b_name"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		handleError(c, err)
		return
	}
	if h.operations == nil {
		handleError(c, domain.Problem{Code: "operation_unavailable", Message: "network object link automation unavailable"})
		return
	}
	value, taskValue, err := h.operations.CreateObjectLink(c, domain.ID(c.Param("labId")), body.ObjectAID, body.PortAName, body.ObjectBID, body.PortBName, c.GetHeader("Idempotency-Key"))
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"network_object_link": value, "task": taskValue})
}

func (h *NetworkHandlers) getObjectLink(c *gin.Context) {
	value, err := h.service.GetObjectLink(c, domain.ID(c.Param("linkId")))
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, value)
}

func (h *NetworkHandlers) deleteObjectLink(c *gin.Context) {
	if h.operations == nil {
		handleError(c, domain.Problem{Code: "operation_unavailable", Message: "network object link automation unavailable"})
		return
	}
	revision, ok := RevisionFromRequest(c)
	if !ok {
		return
	}
	link, taskValue, err := h.operations.DeleteObjectLink(c, domain.ID(c.Param("linkId")), revision, c.GetHeader("Idempotency-Key"))
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"network_object_link": link, "task": taskValue})
}

func (h *NetworkHandlers) update(c *gin.Context) {
	revision, ok := RevisionFromRequest(c)
	if !ok {
		return
	}
	var body struct {
		Name   string         `json:"name"`
		Config map[string]any `json:"config"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		handleError(c, err)
		return
	}
	if h.operations == nil {
		handleError(c, domain.Problem{Code: "operation_unavailable", Message: "network object automation unavailable"})
		return
	}
	value, taskValue, err := h.operations.Update(c, domain.ID(c.Param("objectId")), revision, body.Name, body.Config, c.GetHeader("Idempotency-Key"))
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"network_object": value, "task": taskValue})
}

func (h *NetworkHandlers) list(c *gin.Context) {
	values, err := h.service.List(c, domain.ID(c.Param("labId")))
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, values)
}

func (h *NetworkHandlers) create(c *gin.Context) {
	var body struct {
		Name   string         `json:"name"`
		Kind   string         `json:"kind"`
		Config map[string]any `json:"config"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		handleError(c, err)
		return
	}
	if h.operations == nil {
		handleError(c, domain.Problem{Code: "operation_unavailable", Message: "network object automation unavailable"})
		return
	}
	value, taskValue, err := h.operations.Create(c, domain.ID(c.Param("labId")), body.Name, body.Kind, body.Config, c.GetHeader("Idempotency-Key"))
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"network_object": value, "task": taskValue})
}

func (h *NetworkHandlers) get(c *gin.Context) {
	value, err := h.service.Get(c, domain.ID(c.Param("objectId")))
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, value)
}

func (h *NetworkHandlers) delete(c *gin.Context) {
	revision, ok := RevisionFromRequest(c)
	if !ok {
		return
	}
	if h.operations == nil {
		handleError(c, domain.Problem{Code: "operation_unavailable", Message: "network object automation unavailable"})
		return
	}
	value, err := h.operations.Delete(c, domain.ID(c.Param("objectId")), revision, c.GetHeader("Idempotency-Key"))
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"task": value})
}

func (h *NetworkHandlers) attach(c *gin.Context) {
	var body struct {
		InterfaceID domain.ID      `json:"interface_id"`
		PortName    string         `json:"port_name"`
		Config      map[string]any `json:"config"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		handleError(c, err)
		return
	}
	if err := h.service.Attach(c, domain.ID(c.Param("objectId")), body.InterfaceID, body.PortName, body.Config); err != nil {
		handleError(c, err)
		return
	}
	c.Status(http.StatusCreated)
}

func (h *NetworkHandlers) diagnostics(c *gin.Context) {
	value, err := h.service.Get(c, domain.ID(c.Param("objectId")))
	if err != nil {
		handleError(c, err)
		return
	}
	var diagnostics any
	switch value.Kind {
	case domain.NetworkPC:
		if h.pc != nil {
			diagnostics, err = h.pc.Diagnostics(c, value.ID)
		}
	case domain.NetworkNAT:
		if h.nat != nil {
			diagnostics, err = h.nat.Diagnostics(c, value.ID)
		}
	case domain.NetworkSwitchL3:
		if h.switchL3 != nil {
			diagnostics, err = h.switchL3.Diagnostics(c, value.ID)
		}
	}
	if diagnostics == nil && err == nil {
		writeProblem(c, http.StatusConflict, domain.Problem{Code: "capability_unsupported", Message: "diagnostics are unavailable for this network object"})
		return
	}
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, diagnostics)
}
