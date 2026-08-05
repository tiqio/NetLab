package stream

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/netlab/netlab/internal/domain"
	consoleRuntime "github.com/netlab/netlab/internal/runtime/console"
	"github.com/netlab/netlab/internal/runtime/ownership"
)

type ConsoleNetworkObjectReader interface {
	GetNetworkObject(context.Context, domain.ID) (domain.NetworkObject, error)
}

type NetworkObjectConsoleRuntime interface {
	OpenConsole(domain.NetworkObject) (io.ReadWriteCloser, error)
}

type NetworkObjectConsoleHandlers struct {
	objects  ConsoleNetworkObjectReader
	runtime  NetworkObjectConsoleRuntime
	limits   consoleRuntime.Limits
	mu       sync.RWMutex
	sessions map[string]*consoleRuntime.PersistentSession
	active   map[string]ownership.Record
}

func NewNetworkObjectConsoleHandlers(objects ConsoleNetworkObjectReader, runtime NetworkObjectConsoleRuntime, limits consoleRuntime.Limits) *NetworkObjectConsoleHandlers {
	return &NetworkObjectConsoleHandlers{objects: objects, runtime: runtime, limits: limits, sessions: map[string]*consoleRuntime.PersistentSession{}, active: map[string]ownership.Record{}}
}

func (h *NetworkObjectConsoleHandlers) Register(engine *gin.Engine) {
	engine.GET("/api/v1/network-objects/:objectId/consoles", h.discover)
	engine.GET("/api/v1/network-objects/:objectId/consoles/:mode/stream", h.stream)
	engine.DELETE("/api/v1/network-objects/:objectId/consoles/:mode/sessions/:sessionId", h.closeSession)
}

func (h *NetworkObjectConsoleHandlers) discover(c *gin.Context) {
	object, err := h.pc(c.Request.Context(), domain.ID(c.Param("objectId")))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": "console_unavailable", "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, []any{map[string]any{
		"mode": "telnet", "stream_url": fmt.Sprintf("/api/v1/network-objects/%s/consoles/telnet/stream", object.ID),
		"idle_seconds": int(h.limits.IdleTimeout.Seconds()), "reconnectable": true,
	}})
}

func (h *NetworkObjectConsoleHandlers) stream(c *gin.Context) {
	if c.Param("mode") != "telnet" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "capability_unsupported", "message": "PC network objects expose a shell console only"})
		return
	}
	object, err := h.pc(c.Request.Context(), domain.ID(c.Param("objectId")))
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"code": "console_unavailable", "message": err.Error(), "retryable": true})
		return
	}
	sessionID := c.Query("session_id")
	if sessionID == "" {
		sessionID = string(domain.NewID())
	}
	if !consoleSessionIDPattern.MatchString(sessionID) {
		c.JSON(http.StatusBadRequest, gin.H{"code": "validation_failed", "message": "invalid console session ID"})
		return
	}
	session, created, err := h.getOrCreate(object, sessionID)
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

func (h *NetworkObjectConsoleHandlers) getOrCreate(object domain.NetworkObject, sessionID string) (*consoleRuntime.PersistentSession, bool, error) {
	key := string(object.ID) + "\x00telnet\x00" + sessionID
	h.mu.RLock()
	existing := h.sessions[key]
	h.mu.RUnlock()
	if existing != nil {
		return existing, false, nil
	}
	backend, err := h.runtime.OpenConsole(object)
	if err != nil {
		return nil, false, err
	}
	var candidate *consoleRuntime.PersistentSession
	candidate = consoleRuntime.NewPersistentSession(backend, h.limits, 2*time.Minute, 256<<10, func() { h.remove(key, candidate) })
	h.mu.Lock()
	if existing = h.sessions[key]; existing != nil {
		h.mu.Unlock()
		candidate.Close()
		return existing, false, nil
	}
	h.sessions[key] = candidate
	h.active[key] = ownership.Record{ResourceType: "network_object", ResourceID: object.ID, ObjectKind: "console_proxy", ObjectName: sessionID, Metadata: map[string]string{"mode": "telnet", "kind": "pc"}, CleanupState: "active"}
	h.mu.Unlock()
	return candidate, true, nil
}

func (h *NetworkObjectConsoleHandlers) closeSession(c *gin.Context) {
	if !consoleSessionIDPattern.MatchString(c.Param("sessionId")) {
		c.JSON(http.StatusBadRequest, gin.H{"code": "validation_failed", "message": "invalid console session ID"})
		return
	}
	key := c.Param("objectId") + "\x00" + c.Param("mode") + "\x00" + c.Param("sessionId")
	h.mu.RLock()
	session := h.sessions[key]
	h.mu.RUnlock()
	if session != nil {
		session.Close()
	}
	c.Status(http.StatusNoContent)
}

func (h *NetworkObjectConsoleHandlers) remove(key string, session *consoleRuntime.PersistentSession) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.sessions[key] == session {
		delete(h.sessions, key)
		delete(h.active, key)
	}
}

func (h *NetworkObjectConsoleHandlers) pc(ctx context.Context, id domain.ID) (domain.NetworkObject, error) {
	if h.objects == nil || h.runtime == nil {
		return domain.NetworkObject{}, fmt.Errorf("PC console runtime is unavailable")
	}
	object, err := h.objects.GetNetworkObject(ctx, id)
	if err != nil {
		return domain.NetworkObject{}, err
	}
	if object.Kind != domain.NetworkPC {
		return domain.NetworkObject{}, fmt.Errorf("console is only available for PC network objects")
	}
	if object.ObservedState != "active" {
		return domain.NetworkObject{}, fmt.Errorf("PC network object is not active")
	}
	return object, nil
}

func (h *NetworkObjectConsoleHandlers) DiscoverRuntimeOwnership(context.Context) ([]ownership.Record, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	values := make([]ownership.Record, 0, len(h.active))
	for _, record := range h.active {
		values = append(values, record)
	}
	return values, nil
}
