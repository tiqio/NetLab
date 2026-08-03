package stream

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/netlab/netlab/internal/app/command"
	"github.com/netlab/netlab/internal/domain"
	consoleRuntime "github.com/netlab/netlab/internal/runtime/console"
	"github.com/netlab/netlab/internal/runtime/ownership"
)

type ConsoleNodeReader interface {
	GetNode(context.Context, domain.ID) (domain.Node, error)
}

type ConsoleHandlers struct {
	runtimeDir string
	limits     consoleRuntime.Limits
	nodes      ConsoleNodeReader
	docker     interface {
		OpenConsole(context.Context, domain.Node) (io.ReadWriteCloser, error)
	}
	ssh interface {
		OpenConsole(context.Context, domain.Node) (io.ReadWriteCloser, error)
	}
	mu                 sync.RWMutex
	active             map[string]ownership.Record
	sessions           map[string]*consoleRuntime.PersistentSession
	exclusiveSessions  map[string]string
	ruijieCommandLocks sync.Map
}

var consoleSessionIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

func (h *ConsoleHandlers) SetDockerConsole(runtime interface {
	OpenConsole(context.Context, domain.Node) (io.ReadWriteCloser, error)
}) {
	h.docker = runtime
}

func (h *ConsoleHandlers) SetSSHConsole(runtime interface {
	OpenConsole(context.Context, domain.Node) (io.ReadWriteCloser, error)
}) {
	h.ssh = runtime
}

func NewConsoleHandlers(runtimeDir string, limits consoleRuntime.Limits, readers ...ConsoleNodeReader) *ConsoleHandlers {
	handler := &ConsoleHandlers{
		runtimeDir:        runtimeDir,
		limits:            limits,
		active:            map[string]ownership.Record{},
		sessions:          map[string]*consoleRuntime.PersistentSession{},
		exclusiveSessions: map[string]string{},
	}
	if len(readers) > 0 {
		handler.nodes = readers[0]
	}
	return handler
}

func (h *ConsoleHandlers) Register(engine *gin.Engine) {
	engine.GET("/api/v1/nodes/:nodeId/consoles", h.discover)
	engine.GET("/api/v1/nodes/:nodeId/consoles/:mode/stream", h.stream)
	engine.DELETE("/api/v1/nodes/:nodeId/consoles/:mode/sessions/:sessionId", h.closeSession)
	engine.POST("/api/v1/nodes/:nodeId/ruijie/configure", h.configureRuijie)
}

func (h *ConsoleHandlers) configureRuijie(c *gin.Context) {
	if h.nodes == nil {
		c.JSON(http.StatusConflict, gin.H{"code": "operation_unavailable", "message": "node repository unavailable"})
		return
	}
	nodeID := domain.ID(c.Param("nodeId"))
	node, err := h.nodes.GetNode(c, nodeID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": "not_found", "message": err.Error()})
		return
	}
	if node.ObservedState != domain.ObservedRunning {
		c.JSON(http.StatusConflict, gin.H{"code": "node_not_running", "message": "start the Ruijie node before applying CLI configuration"})
		return
	}
	var request command.RuijieConfigRequest
	if err = c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "validation_failed", "message": err.Error()})
		return
	}
	commands, err := command.BuildRuijieCommands(node, request)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "validation_failed", "message": err.Error()})
		return
	}
	session, err := h.commandSession(c.Request.Context(), nodeID)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"code": "console_unavailable", "message": err.Error(), "retryable": true})
		return
	}
	lockValue, _ := h.ruijieCommandLocks.LoadOrStore(nodeID, &sync.Mutex{})
	lock := lockValue.(*sync.Mutex)
	lock.Lock()
	defer lock.Unlock()
	commandContext, cancel := context.WithTimeout(c.Request.Context(), 3*time.Minute)
	defer cancel()
	_, err = executeRuijieCommands(commandContext, session, commands)
	if err != nil {
		var executionErr *ruijieExecutionError
		if errors.As(err, &executionErr) {
			c.JSON(http.StatusConflict, gin.H{"code": executionErr.code, "message": executionErr.message, "retryable": executionErr.retryable})
			return
		}
		c.JSON(http.StatusConflict, gin.H{"code": "console_write_failed", "message": err.Error(), "retryable": true})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"commands": commands, "console_mode": "telnet", "verified": true})
}

func (h *ConsoleHandlers) commandSession(ctx context.Context, nodeID domain.ID) (*consoleRuntime.PersistentSession, error) {
	exclusiveKey := string(nodeID) + "\x00telnet"
	h.mu.RLock()
	ownerKey := h.exclusiveSessions[exclusiveKey]
	existing := h.sessions[ownerKey]
	h.mu.RUnlock()
	if existing != nil {
		return existing, nil
	}
	key := consoleSessionKey(nodeID, "telnet", "inspector")
	session, _, err := h.getOrCreateExclusiveSession(ctx, nodeID, "telnet", "inspector", key, exclusiveKey)
	return session, err
}

func (h *ConsoleHandlers) discover(c *gin.Context) {
	nodeID := c.Param("nodeId")
	modes, err := h.modes(c, domain.ID(nodeID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": "not_found", "message": err.Error()})
		return
	}
	values := make([]any, 0, len(modes))
	for _, mode := range modes {
		if mode == "vnc" {
			values = append(values, consoleRuntime.DescribeVNC(nodeID))
			continue
		}
		values = append(values, map[string]any{"mode": mode, "stream_url": fmt.Sprintf("/api/v1/nodes/%s/consoles/%s/stream", nodeID, mode), "idle_seconds": int(h.limits.IdleTimeout.Seconds()), "reconnectable": true})
	}
	c.JSON(http.StatusOK, values)
}

func (h *ConsoleHandlers) stream(c *gin.Context) {
	nodeID := domain.ID(c.Param("nodeId"))
	mode := c.Param("mode")
	modes, err := h.modes(c, nodeID)
	if err != nil || !containsMode(modes, mode) {
		c.JSON(http.StatusBadRequest, gin.H{"code": "capability_unsupported", "message": "console mode must be ssh, telnet, or vnc"})
		return
	}
	if mode == "telnet" || mode == "ssh" {
		h.streamPersistent(c, nodeID, mode)
		return
	}
	backend, err := h.openBackend(c.Request.Context(), nodeID, mode)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"code": "console_unavailable", "message": err.Error(), "retryable": true})
		return
	}
	connection, err := websocket.Accept(c.Writer, c.Request, &websocket.AcceptOptions{OriginPatterns: []string{c.Request.Host}})
	if err != nil {
		_ = backend.Close()
		return
	}
	client := websocket.NetConn(context.Background(), connection, websocket.MessageBinary)
	sessionID := domain.NewID()
	h.beginSession(sessionID, domain.ID(c.Param("nodeId")), mode)
	defer h.endSession(sessionID)
	_ = consoleRuntime.Bridge(c.Request.Context(), client, backend, h.limits)
}

func (h *ConsoleHandlers) streamPersistent(c *gin.Context, nodeID domain.ID, mode string) {
	sessionID := c.Query("session_id")
	if sessionID == "" {
		sessionID = string(domain.NewID())
	}
	if !consoleSessionIDPattern.MatchString(sessionID) {
		c.JSON(http.StatusBadRequest, gin.H{"code": "validation_failed", "message": "session_id must contain only letters, numbers, dots, underscores, or hyphens and be at most 128 characters"})
		return
	}
	session, created, err := h.getOrCreateSession(c.Request.Context(), nodeID, mode, sessionID)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"code": "console_unavailable", "message": err.Error(), "retryable": true})
		return
	}
	connection, err := websocket.Accept(c.Writer, c.Request, &websocket.AcceptOptions{OriginPatterns: []string{c.Request.Host}})
	if err != nil {
		if created {
			session.Close()
		}
		return
	}
	client := websocket.NetConn(context.Background(), connection, websocket.MessageBinary)
	_ = session.Attach(client)
}

func (h *ConsoleHandlers) closeSession(c *gin.Context) {
	nodeID := domain.ID(c.Param("nodeId"))
	mode := c.Param("mode")
	sessionID := c.Param("sessionId")
	if !consoleSessionIDPattern.MatchString(sessionID) {
		c.JSON(http.StatusBadRequest, gin.H{"code": "validation_failed", "message": "invalid console session ID"})
		return
	}
	key := consoleSessionKey(nodeID, mode, sessionID)
	h.mu.RLock()
	session := h.sessions[key]
	h.mu.RUnlock()
	if session != nil {
		session.Close()
	}
	c.Status(http.StatusNoContent)
}

func (h *ConsoleHandlers) getOrCreateSession(ctx context.Context, nodeID domain.ID, mode, sessionID string) (*consoleRuntime.PersistentSession, bool, error) {
	key := consoleSessionKey(nodeID, mode, sessionID)
	exclusiveKey, err := h.exclusiveSessionKey(ctx, nodeID, mode)
	if err != nil {
		return nil, false, err
	}
	if exclusiveKey != "" {
		return h.getOrCreateExclusiveSession(ctx, nodeID, mode, sessionID, key, exclusiveKey)
	}
	h.mu.RLock()
	existing := h.sessions[key]
	h.mu.RUnlock()
	if existing != nil {
		return existing, false, nil
	}
	backend, err := h.openBackend(ctx, nodeID, mode)
	if err != nil {
		return nil, false, err
	}
	var candidate *consoleRuntime.PersistentSession
	candidate = consoleRuntime.NewPersistentSession(backend, h.limits, 2*time.Minute, 256<<10, func() {
		h.removeSession(key, candidate)
	})
	h.mu.Lock()
	if existing = h.sessions[key]; existing != nil {
		h.mu.Unlock()
		candidate.Close()
		return existing, false, nil
	}
	h.sessions[key] = candidate
	h.active[key] = ownership.Record{
		ResourceType: "node",
		ResourceID:   nodeID,
		ObjectKind:   "console_proxy",
		ObjectName:   sessionID,
		Metadata:     map[string]string{"mode": mode},
		CleanupState: "active",
	}
	h.mu.Unlock()
	select {
	case <-candidate.Done():
		h.removeSession(key, candidate)
	default:
	}
	return candidate, true, nil
}

func (h *ConsoleHandlers) getOrCreateExclusiveSession(ctx context.Context, nodeID domain.ID, mode, sessionID, key, exclusiveKey string) (*consoleRuntime.PersistentSession, bool, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if existing := h.sessions[key]; existing != nil {
		return existing, false, nil
	}
	if ownerKey := h.exclusiveSessions[exclusiveKey]; ownerKey != "" {
		return nil, false, fmt.Errorf("QEMU serial console already has an active session; close it or reconnect the existing terminal")
	}
	backend, err := h.openBackend(ctx, nodeID, mode)
	if err != nil {
		return nil, false, err
	}
	var session *consoleRuntime.PersistentSession
	session = consoleRuntime.NewPersistentSession(backend, h.limits, 2*time.Minute, 256<<10, func() {
		h.removeSession(key, session)
	})
	h.sessions[key] = session
	h.exclusiveSessions[exclusiveKey] = key
	h.active[key] = ownership.Record{
		ResourceType: "node",
		ResourceID:   nodeID,
		ObjectKind:   "console_proxy",
		ObjectName:   sessionID,
		Metadata:     map[string]string{"mode": mode},
		CleanupState: "active",
	}
	return session, true, nil
}

func (h *ConsoleHandlers) removeSession(key string, session *consoleRuntime.PersistentSession) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.sessions[key] != session {
		return
	}
	delete(h.sessions, key)
	delete(h.active, key)
	for exclusiveKey, ownerKey := range h.exclusiveSessions {
		if ownerKey == key {
			delete(h.exclusiveSessions, exclusiveKey)
		}
	}
}

func (h *ConsoleHandlers) exclusiveSessionKey(ctx context.Context, nodeID domain.ID, mode string) (string, error) {
	if mode != "telnet" || h.nodes == nil {
		return "", nil
	}
	node, err := h.nodes.GetNode(ctx, nodeID)
	if err != nil {
		return "", err
	}
	if node.Kind == "docker" {
		return "", nil
	}
	return string(nodeID) + "\x00" + mode, nil
}

func (h *ConsoleHandlers) openBackend(ctx context.Context, nodeID domain.ID, mode string) (io.ReadWriteCloser, error) {
	if h.nodes == nil {
		return h.openSocket(string(nodeID), mode)
	}
	node, err := h.nodes.GetNode(ctx, nodeID)
	if err != nil {
		return nil, err
	}
	if mode == "ssh" {
		if node.Kind == "docker" || h.ssh == nil {
			return nil, fmt.Errorf("SSH console is unavailable")
		}
		return h.ssh.OpenConsole(ctx, node)
	}
	if node.Kind != "docker" {
		return h.openSocket(string(nodeID), mode)
	}
	if h.docker == nil || mode != "telnet" {
		return nil, fmt.Errorf("docker exec console is unavailable")
	}
	return h.docker.OpenConsole(ctx, node)
}

func consoleSessionKey(nodeID domain.ID, mode, sessionID string) string {
	return string(nodeID) + "\x00" + mode + "\x00" + sessionID
}

func (h *ConsoleHandlers) openSocket(nodeID, mode string) (io.ReadWriteCloser, error) {
	socketName := "serial.sock"
	if mode == "vnc" {
		socketName = "vnc.sock"
	}
	return net.DialTimeout("unix", filepath.Join(h.runtimeDir, nodeID, socketName), 3*time.Second)
}

func (h *ConsoleHandlers) beginSession(sessionID, nodeID domain.ID, mode string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.active[string(sessionID)] = ownership.Record{ResourceType: "node", ResourceID: nodeID, ObjectKind: "console_proxy", ObjectName: string(sessionID), Metadata: map[string]string{"mode": mode}, CleanupState: "active"}
}

func (h *ConsoleHandlers) endSession(sessionID domain.ID) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.active, string(sessionID))
}

func (h *ConsoleHandlers) DiscoverRuntimeOwnership(context.Context) ([]ownership.Record, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	values := make([]ownership.Record, 0, len(h.active))
	for _, record := range h.active {
		values = append(values, record)
	}
	return values, nil
}

func (h *ConsoleHandlers) modes(ctx context.Context, nodeID domain.ID) ([]string, error) {
	if h.nodes == nil {
		return []string{"telnet", "vnc"}, nil
	}
	node, err := h.nodes.GetNode(ctx, nodeID)
	if err != nil {
		return nil, err
	}
	if node.Kind == "docker" && h.docker != nil {
		return []string{"telnet"}, nil
	}
	raw := node.Config["console_modes"]
	if direct, ok := raw.([]string); ok {
		return direct, nil
	}
	values, _ := raw.([]any)
	result := make([]string, 0, len(values))
	for _, value := range values {
		if mode, ok := value.(string); ok && (mode == "telnet" || mode == "vnc") {
			result = append(result, mode)
		}
	}
	if h.ssh != nil && !containsMode(result, "ssh") {
		result = append([]string{"ssh"}, result...)
	}
	return result, nil
}

func containsMode(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
