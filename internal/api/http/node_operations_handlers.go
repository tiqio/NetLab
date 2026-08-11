package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/netlab/netlab/internal/app/command"
	"github.com/netlab/netlab/internal/app/reconcile"
	"github.com/netlab/netlab/internal/domain"
	qemuRuntime "github.com/netlab/netlab/internal/runtime/qemu"
)

type NodeOperationsHandlers struct {
	interfaces *command.InterfaceService
	guest      *command.GuestCommandService
	mappings   *command.PortMappingService
	nodes      interface {
		GetNode(context.Context, domain.ID) (domain.Node, error)
		ListAllNodes(context.Context) ([]domain.Node, error)
		UpdateNodeResources(context.Context, domain.ID, domain.Revision, int, int64, int) (domain.Node, error)
		UpdateNodeSettings(context.Context, domain.ID, domain.Revision, domain.NodeSettings) (domain.Node, error)
	}
	resources   *reconcile.ResourceManager
	credentials interface {
		Credentials(context.Context, string) (qemuRuntime.BootstrapCredentials, error)
		PrepareNetworkConfig(context.Context, string, string) (*qemuRuntime.PreparedSeedUpdate, error)
	}
	nodeCredentials interface {
		CredentialsForNode(context.Context, domain.Node) (qemuRuntime.BootstrapCredentials, error)
	}
	networkDiagnostics interface {
		NetworkDiagnostics(context.Context, domain.Node) (map[string]any, error)
	}
}

func (h *NodeOperationsHandlers) SetNodeCredentialReader(reader interface {
	CredentialsForNode(context.Context, domain.Node) (qemuRuntime.BootstrapCredentials, error)
}) {
	h.nodeCredentials = reader
}

func (h *NodeOperationsHandlers) SetNetworkDiagnostics(reader interface {
	NetworkDiagnostics(context.Context, domain.Node) (map[string]any, error)
}) {
	h.networkDiagnostics = reader
}

func NewNodeOperationsHandlers(interfaces *command.InterfaceService, guest *command.GuestCommandService, mappings *command.PortMappingService, nodes interface {
	GetNode(context.Context, domain.ID) (domain.Node, error)
	ListAllNodes(context.Context) ([]domain.Node, error)
	UpdateNodeResources(context.Context, domain.ID, domain.Revision, int, int64, int) (domain.Node, error)
	UpdateNodeSettings(context.Context, domain.ID, domain.Revision, domain.NodeSettings) (domain.Node, error)
}, resources *reconcile.ResourceManager, credentialReaders ...interface {
	Credentials(context.Context, string) (qemuRuntime.BootstrapCredentials, error)
	PrepareNetworkConfig(context.Context, string, string) (*qemuRuntime.PreparedSeedUpdate, error)
}) *NodeOperationsHandlers {
	handlers := &NodeOperationsHandlers{interfaces: interfaces, guest: guest, mappings: mappings, nodes: nodes, resources: resources}
	if len(credentialReaders) > 0 {
		handlers.credentials = credentialReaders[0]
	}
	return handlers
}

func (h *NodeOperationsHandlers) Register(engine *gin.Engine) {
	api := engine.Group("/api/v1")
	api.GET("/nodes/:nodeId", h.getNode)
	api.GET("/nodes/:nodeId/bootstrap-credentials", h.getBootstrapCredentials)
	api.POST("/nodes/:nodeId/interfaces", h.addInterface)
	api.DELETE("/interfaces/:interfaceId", h.removeInterface)
	api.POST("/nodes/:nodeId/guest-exec", h.guestExec)
	api.GET("/nodes/:nodeId/port-mappings", h.listMappings)
	api.POST("/nodes/:nodeId/port-mappings", h.createMapping)
	api.DELETE("/port-mappings/:mappingId", h.deleteMapping)
	api.PUT("/nodes/:nodeId/resources", h.updateResources)
	api.PUT("/nodes/:nodeId/settings", h.updateSettings)
	api.GET("/nodes/:nodeId/resources", h.getResources)
	api.GET("/nodes/:nodeId/network-diagnostics", h.getNetworkDiagnostics)
}

func (h *NodeOperationsHandlers) getNetworkDiagnostics(c *gin.Context) {
	if h.networkDiagnostics == nil {
		writeProblem(c, http.StatusConflict, domain.Problem{Code: "capability_unsupported", Message: "node network diagnostics are unavailable"})
		return
	}
	node, err := h.nodes.GetNode(c, domain.ID(c.Param("nodeId")))
	if err != nil {
		handleError(c, err)
		return
	}
	if node.Kind != string(domain.RuntimeDocker) {
		writeProblem(c, http.StatusConflict, domain.Problem{Code: "capability_unsupported", Message: "forwarding diagnostics are supported only for Docker nodes", ResourceType: "node", ResourceID: node.ID})
		return
	}
	diagnostics, err := h.networkDiagnostics.NetworkDiagnostics(c, node)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, diagnostics)
}

func (h *NodeOperationsHandlers) listMappings(c *gin.Context) {
	if !h.available(c, h.mappings) {
		return
	}
	values, err := h.mappings.ListNode(c, domain.ID(c.Param("nodeId")))
	if err != nil {
		handleError(c, err)
		return
	}
	if values == nil {
		values = []domain.PortMapping{}
	}
	c.JSON(http.StatusOK, values)
}

func (h *NodeOperationsHandlers) getBootstrapCredentials(c *gin.Context) {
	if h.nodes == nil || h.nodeCredentials == nil {
		writeProblem(c, http.StatusConflict, domain.Problem{Code: "capability_unsupported", Message: "bootstrap credential reader unavailable"})
		return
	}
	node, err := h.nodes.GetNode(c, domain.ID(c.Param("nodeId")))
	if err != nil {
		handleError(c, err)
		return
	}
	credentials, err := h.nodeCredentials.CredentialsForNode(c, node)
	if err != nil {
		writeProblem(c, http.StatusNotFound, domain.Problem{Code: "bootstrap_credentials_unavailable", Message: err.Error(), ResourceType: "node", ResourceID: node.ID})
		return
	}
	c.JSON(http.StatusOK, gin.H{"username": credentials.Username, "password": credentials.Password, "source": "managed-console-credentials"})
}

func (h *NodeOperationsHandlers) updateSettings(c *gin.Context) {
	if h.nodes == nil {
		writeProblem(c, http.StatusConflict, domain.Problem{Code: "capability_unsupported", Message: "node repository unavailable"})
		return
	}
	revision, ok := RevisionFromRequest(c)
	if !ok {
		return
	}
	var body domain.NodeSettings
	if err := c.ShouldBindJSON(&body); err != nil {
		handleError(c, err)
		return
	}
	current, err := h.nodes.GetNode(c, domain.ID(c.Param("nodeId")))
	if err != nil {
		handleError(c, err)
		return
	}
	if current.DesiredState != domain.DesiredStopped || current.ObservedState != domain.ObservedStopped {
		writeProblem(c, http.StatusConflict, domain.Problem{Code: "node_not_stopped", Message: "node settings can only be changed while the node is stopped", ResourceType: "node", ResourceID: current.ID, OperatorHint: "stop the node and wait for it to reach stopped state"})
		return
	}
	candidate := current
	candidate.Name = body.Name
	candidate.CPUCount = body.CPUCount
	candidate.CPUQuotaMicros = body.CPUQuotaMicros
	candidate.MemoryMiB = body.MemoryMiB
	candidate.InterfaceLimit = body.InterfaceLimit
	candidate.ProcessLimit = body.ProcessLimit
	if h.resources != nil {
		nodes, listErr := h.nodes.ListAllNodes(c)
		if listErr != nil {
			handleError(c, listErr)
			return
		}
		if err = h.resources.Admit(c, candidate, nodes); err != nil {
			handleError(c, err)
			return
		}
	}
	if err = validateNodeForwardingSettings(current, body); err != nil {
		handleError(c, err)
		return
	}
	var prepared *qemuRuntime.PreparedSeedUpdate
	if len(body.NetworkInterfaces) > 0 {
		interfaces, rawNetwork, validationErr := validateNetworkSettings(current, body.NetworkInterfaces)
		if validationErr != nil {
			handleError(c, validationErr)
			return
		}
		switch current.Kind {
		case string(domain.RuntimeQEMU):
			if h.credentials == nil {
				writeProblem(c, http.StatusConflict, domain.Problem{Code: "capability_unsupported", Message: "QEMU interface network settings require cloud-init seed support", ResourceType: "node", ResourceID: current.ID})
				return
			}
			networkConfig, buildErr := command.BuildCloudInitNetworkConfig(interfaces, rawNetwork)
			if buildErr != nil {
				handleError(c, buildErr)
				return
			}
			seedPath, _ := current.Config["seed_iso"].(string)
			prepared, err = h.credentials.PrepareNetworkConfig(c, seedPath, networkConfig)
			if err != nil {
				handleError(c, err)
				return
			}
		case string(domain.RuntimeDocker):
		default:
			writeProblem(c, http.StatusConflict, domain.Problem{Code: "capability_unsupported", Message: "interface network settings are supported only for QEMU and Docker nodes", ResourceType: "node", ResourceID: current.ID})
			return
		}
		if prepared != nil {
			defer prepared.Cleanup()
		}
	}
	updated, err := h.nodes.UpdateNodeSettings(c, current.ID, revision, body)
	if err != nil {
		handleError(c, err)
		return
	}
	if prepared != nil {
		if err = prepared.Commit(); err != nil {
			writeProblem(c, http.StatusInternalServerError, domain.Problem{Code: "seed_update_failed", Message: err.Error(), Retryable: true, ResourceType: "node", ResourceID: current.ID, Phase: "seed_commit", Cleanup: "settings saved; prepared seed commit failed", OperatorHint: "retry the same settings before starting the node"})
			return
		}
	}
	c.JSON(http.StatusOK, updated)
}

func validateNodeForwardingSettings(node domain.Node, settings domain.NodeSettings) error {
	if err := domain.ValidateNodeForwardingSettings(node.Kind, settings.ForwardIPv4, settings.ForwardIPv6); err != nil {
		return domain.Problem{Code: "invalid_node_network", Message: err.Error(), ResourceType: "node", ResourceID: node.ID}
	}
	return nil
}

func validateNetworkSettings(node domain.Node, values []domain.NodeNetworkInterfaceSettings) ([]domain.Interface, []map[string]any, error) {
	if err := domain.ValidateNodeNetworkInterfaces(values); err != nil {
		return nil, nil, domain.Problem{Code: "invalid_node_network", Message: err.Error(), ResourceType: "node", ResourceID: node.ID}
	}
	descriptors, _ := node.Config["interfaces"].([]any)
	if direct, ok := node.Config["interfaces"].([]map[string]any); ok {
		descriptors = make([]any, len(direct))
		for index := range direct {
			descriptors[index] = direct[index]
		}
	}
	if len(values) != len(descriptors) {
		return nil, nil, domain.Problem{Code: "invalid_node_settings", Message: "network settings must include every current interface", ResourceType: "node", ResourceID: node.ID}
	}
	requested := make(map[domain.ID]domain.NodeNetworkInterfaceSettings, len(values))
	allowedDrivers := map[string]bool{"virtio-net-pci": true, "e1000": true, "e1000e": true, "vmxnet3": true}
	allowedModes := map[string]bool{"dhcpv4": true, "dhcpv6": true, "slaac": true, "static": true}
	for _, value := range values {
		if requested[value.ID].ID != "" || (node.Kind == string(domain.RuntimeQEMU) && !allowedDrivers[value.Driver]) {
			return nil, nil, domain.Problem{Code: "invalid_node_settings", Message: "interface ID or driver is invalid", ResourceType: "interface", ResourceID: value.ID}
		}
		for _, mode := range value.Modes {
			if !allowedModes[strings.ToLower(mode)] {
				return nil, nil, domain.Problem{Code: "invalid_node_settings", Message: fmt.Sprintf("unsupported network mode %q", mode), ResourceType: "interface", ResourceID: value.ID}
			}
		}
		for _, address := range value.Addresses {
			if _, err := netip.ParsePrefix(address); err != nil {
				return nil, nil, domain.Problem{Code: "invalid_node_settings", Message: fmt.Sprintf("invalid interface address %q", address), ResourceType: "interface", ResourceID: value.ID}
			}
		}
		requested[value.ID] = value
	}
	interfaces := make([]domain.Interface, 0, len(descriptors))
	rawNetwork := make([]map[string]any, 0, len(descriptors))
	for index, raw := range descriptors {
		descriptor, _ := raw.(map[string]any)
		id, _ := descriptor["id"].(string)
		name, _ := descriptor["name"].(string)
		driver, _ := descriptor["driver"].(string)
		mac, _ := descriptor["mac_address"].(string)
		value, ok := requested[domain.ID(id)]
		if !ok || value.Name != name || (node.Kind == string(domain.RuntimeDocker) && value.Driver != driver) {
			return nil, nil, domain.Problem{Code: "invalid_node_settings", Message: "interface identity does not match the node", ResourceType: "interface", ResourceID: domain.ID(id)}
		}
		interfaces = append(interfaces, domain.Interface{ID: domain.ID(id), NodeID: node.ID, Slot: index, Name: name, Driver: value.Driver, MACAddress: mac})
		rawNetwork = append(rawNetwork, map[string]any{"name": name, "modes": value.Modes, "addresses": value.Addresses, "routes": value.Routes})
	}
	return interfaces, rawNetwork, nil
}

func (h *NodeOperationsHandlers) getNode(c *gin.Context) {
	if h.nodes == nil {
		writeProblem(c, http.StatusConflict, domain.Problem{Code: "capability_unsupported", Message: "node repository unavailable"})
		return
	}
	value, err := h.nodes.GetNode(c, domain.ID(c.Param("nodeId")))
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, value)
}

func (h *NodeOperationsHandlers) getResources(c *gin.Context) {
	if h.resources == nil {
		writeProblem(c, http.StatusConflict, domain.Problem{Code: "capability_unsupported", Message: "resource metrics unavailable"})
		return
	}
	node, err := h.nodes.GetNode(c, domain.ID(c.Param("nodeId")))
	if err != nil {
		handleError(c, err)
		return
	}
	metrics, err := h.resources.Metrics(c, node)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, metrics)
}

func (h *NodeOperationsHandlers) available(c *gin.Context, value any) bool {
	if value != nil {
		return true
	}
	writeProblem(c, http.StatusConflict, domain.Problem{Code: "capability_unsupported", Message: "runtime capability is unavailable on this host"})
	return false
}

func (h *NodeOperationsHandlers) addInterface(c *gin.Context) {
	if !h.available(c, h.interfaces) {
		return
	}
	var body struct {
		Driver string `json:"driver"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		handleError(c, err)
		return
	}
	iface, taskValue, err := h.interfaces.Add(c, domain.ID(c.Param("nodeId")), body.Driver, c.GetHeader("Idempotency-Key"))
	if err != nil {
		handleError(c, err)
		return
	}
	if taskValue == nil {
		c.JSON(http.StatusCreated, iface)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"interface": iface, "task": taskValue})
}

func (h *NodeOperationsHandlers) removeInterface(c *gin.Context) {
	if !h.available(c, h.interfaces) {
		return
	}
	revision, ok := RevisionFromRequest(c)
	if !ok {
		return
	}
	taskValue, err := h.interfaces.Remove(c, domain.ID(c.Param("interfaceId")), revision, c.GetHeader("Idempotency-Key"))
	if err != nil {
		handleError(c, err)
		return
	}
	if taskValue.ID == "" {
		c.Status(http.StatusNoContent)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"task": taskValue})
}

func (h *NodeOperationsHandlers) guestExec(c *gin.Context) {
	if !h.available(c, h.guest) {
		return
	}
	var body struct {
		Argv           []string `json:"argv"`
		TimeoutSeconds int      `json:"timeout_seconds"`
		OutputLimit    int      `json:"output_limit"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		handleError(c, err)
		return
	}
	value, err := h.guest.Execute(c, domain.ID(c.Param("nodeId")), body.Argv, time.Duration(body.TimeoutSeconds)*time.Second, body.OutputLimit, c.GetHeader("Idempotency-Key"))
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"task": value})
}

func (h *NodeOperationsHandlers) createMapping(c *gin.Context) {
	if !h.available(c, h.mappings) {
		return
	}
	var body domain.PortMapping
	if err := c.ShouldBindJSON(&body); err != nil {
		handleError(c, err)
		return
	}
	body.NodeID = domain.ID(c.Param("nodeId"))
	mapping, taskValue, err := h.mappings.Create(c, body, c.GetHeader("Idempotency-Key"))
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"port_mapping": mapping, "task": taskValue})
}

func (h *NodeOperationsHandlers) deleteMapping(c *gin.Context) {
	if !h.available(c, h.mappings) {
		return
	}
	value, err := h.mappings.Delete(c, domain.ID(c.Param("mappingId")), c.GetHeader("Idempotency-Key"))
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"task": value})
}

func (h *NodeOperationsHandlers) updateResources(c *gin.Context) {
	if h.nodes == nil {
		writeProblem(c, http.StatusConflict, domain.Problem{Code: "capability_unsupported", Message: "resource manager unavailable"})
		return
	}
	revision, ok := RevisionFromRequest(c)
	if !ok {
		return
	}
	var body struct {
		CPUCount       int   `json:"cpu_count"`
		CPUQuotaMicros int64 `json:"cpu_quota_micros"`
		MemoryMiB      int   `json:"memory_mib"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		handleError(c, err)
		return
	}
	if body.CPUCount < 1 || body.CPUCount > 256 || body.CPUQuotaMicros < 1000 || body.MemoryMiB < 64 {
		writeProblem(c, http.StatusBadRequest, domain.Problem{Code: "invalid_resources", Message: "invalid CPU, quota, or memory values"})
		return
	}
	node, err := h.nodes.UpdateNodeResources(c, domain.ID(c.Param("nodeId")), revision, body.CPUCount, body.CPUQuotaMicros, body.MemoryMiB)
	if err != nil {
		handleError(c, err)
		return
	}
	if h.resources != nil {
		if err = h.resources.Apply(c, node); err != nil {
			handleError(c, err)
			return
		}
	}
	c.JSON(http.StatusOK, node)
}
