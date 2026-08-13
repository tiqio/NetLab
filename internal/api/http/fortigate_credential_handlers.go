package httpapi

import (
	"context"
	"net"
	"net/http"
	"net/netip"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/netlab/netlab/internal/domain"
)

type FortiGateCredentialService interface {
	Metadata(context.Context, domain.ID) (domain.NodeCredentialMetadata, error)
	Put(context.Context, domain.ID, string, []byte, []byte) (domain.NodeCredentialMetadata, error)
	Delete(context.Context, domain.ID) error
	Verify(context.Context, domain.ID, string) (domain.OperationTask, error)
	Bootstrap(context.Context, domain.ID, string) (domain.OperationTask, error)
}

type FortiGateCredentialHandlers struct {
	service FortiGateCredentialService
	scopes  []netip.Prefix
}

func NewFortiGateCredentialHandlers(service FortiGateCredentialService, managementScopes []string) *FortiGateCredentialHandlers {
	handler := &FortiGateCredentialHandlers{service: service}
	for _, value := range managementScopes {
		if prefix, err := netip.ParsePrefix(strings.TrimSpace(value)); err == nil {
			handler.scopes = append(handler.scopes, prefix)
		}
	}
	return handler
}

func (h *FortiGateCredentialHandlers) Register(engine *gin.Engine) {
	api := engine.Group("/api/v1/nodes/:nodeId")
	api.Use(h.requireManagementSource)
	api.GET("/credentials", h.metadata)
	api.PUT("/credentials/console-admin", h.put)
	api.DELETE("/credentials/console-admin", h.delete)
	api.POST("/credentials/console-admin/verify", h.verify)
	api.POST("/bootstrap/fortigate", h.bootstrap)
}

func (h *FortiGateCredentialHandlers) requireManagementSource(c *gin.Context) {
	address := c.Request.RemoteAddr
	if host, _, err := net.SplitHostPort(address); err == nil {
		address = host
	}
	ip, err := netip.ParseAddr(strings.Trim(address, "[]"))
	if err != nil || !h.allowed(ip) {
		writeProblem(c, http.StatusForbidden, domain.Problem{Code: "management_scope_required", Message: "FortiGate credential operations are restricted to the configured management network"})
		c.Abort()
		return
	}
	c.Next()
}

func (h *FortiGateCredentialHandlers) allowed(address netip.Addr) bool {
	if address.IsLoopback() {
		return true
	}
	for _, prefix := range h.scopes {
		if prefix.Contains(address) {
			return true
		}
	}
	return false
}

func (h *FortiGateCredentialHandlers) metadata(c *gin.Context) {
	value, err := h.service.Metadata(c, domain.ID(c.Param("nodeId")))
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, value)
}

func (h *FortiGateCredentialHandlers) put(c *gin.Context) {
	var request struct {
		Username        string `json:"username"`
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		handleError(c, err)
		return
	}
	currentPassword := []byte(request.CurrentPassword)
	newPassword := []byte(request.NewPassword)
	defer clearSecretBytes(currentPassword)
	defer clearSecretBytes(newPassword)
	value, err := h.service.Put(c, domain.ID(c.Param("nodeId")), request.Username, currentPassword, newPassword)
	request.CurrentPassword = ""
	request.NewPassword = ""
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, value)
}

func (h *FortiGateCredentialHandlers) delete(c *gin.Context) {
	if err := h.service.Delete(c, domain.ID(c.Param("nodeId"))); err != nil {
		handleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *FortiGateCredentialHandlers) verify(c *gin.Context) {
	value, err := h.service.Verify(c, domain.ID(c.Param("nodeId")), c.GetHeader("Idempotency-Key"))
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"task": value})
}

func (h *FortiGateCredentialHandlers) bootstrap(c *gin.Context) {
	value, err := h.service.Bootstrap(c, domain.ID(c.Param("nodeId")), c.GetHeader("Idempotency-Key"))
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"task": value})
}

func clearSecretBytes(value []byte) {
	for index := range value {
		value[index] = 0
	}
}
