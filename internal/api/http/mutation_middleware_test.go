package httpapi

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/netlab/netlab/internal/app/audit"
	"github.com/netlab/netlab/internal/app/command"
	storesqlite "github.com/netlab/netlab/internal/store/sqlite"
)

func TestMutationAutomationReplaysAndRejectsFingerprintConflicts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	database, err := storesqlite.Open(ctx, "file:mutation-middleware?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	repositories := storesqlite.NewRepositories(database)
	idempotency := command.NewIdempotencyService(repositories, time.Hour)
	audits := audit.NewService(repositories)
	engine := gin.New()
	engine.Use(MutationAutomation(idempotency, repositories, audits))
	calls := 0
	engine.POST("/api/v1/nodes/:nodeId/state", func(c *gin.Context) {
		calls++
		c.JSON(http.StatusAccepted, gin.H{"accepted": true, "call": calls})
	})

	first := mutationRequest(engine, `{"desired_state":"running"}`, "same-key")
	second := mutationRequest(engine, `{"desired_state":"running"}`, "same-key")
	conflict := mutationRequest(engine, `{"desired_state":"stopped"}`, "same-key")

	if first.Code != http.StatusAccepted || second.Code != first.Code || second.Body.String() != first.Body.String() {
		t.Fatalf("replay mismatch first=%d %s second=%d %s", first.Code, first.Body.String(), second.Code, second.Body.String())
	}
	if second.Header().Get("Idempotency-Replayed") != "true" {
		t.Fatal("expected replay marker")
	}
	if conflict.Code != http.StatusConflict || calls != 1 {
		t.Fatalf("conflict=%d calls=%d", conflict.Code, calls)
	}
	tasks, err := repositories.ListTasks(ctx, 10)
	if err != nil || len(tasks) != 0 {
		t.Fatalf("tasks=%d err=%v", len(tasks), err)
	}
	events, err := repositories.ListAuditEvents(ctx, 10)
	if err != nil || len(events) != 1 || events[0].Outcome != "succeeded" || events[0].Details["entry_point"] != "compatibility_http" {
		t.Fatalf("audits=%+v err=%v", events, err)
	}
}

func TestMutationAutomationSkipsDurableTaskEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	database, err := storesqlite.Open(ctx, "file:durable-mutation-middleware?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	repositories := storesqlite.NewRepositories(database)
	engine := gin.New()
	engine.Use(MutationAutomation(command.NewIdempotencyService(repositories, time.Hour), repositories, audit.NewService(repositories)))
	engine.PUT("/api/v1/nodes/:nodeId/state", func(c *gin.Context) {
		c.JSON(http.StatusAccepted, gin.H{"task": map[string]any{"id": "durable"}})
	})
	request := httptest.NewRequest(http.MethodPut, "/api/v1/nodes/node-1/state", bytes.NewBufferString(`{"desired_state":"running"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "durable-key")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	values, err := repositories.ListTasks(ctx, 10)
	if err != nil || len(values) != 0 {
		t.Fatalf("shadow tasks=%d err=%v", len(values), err)
	}
	events, err := repositories.ListAuditEvents(ctx, 10)
	if err != nil || len(events) != 1 || events[0].TaskID != "durable" || events[0].Outcome != "accepted" {
		t.Fatalf("audit events=%+v err=%v", events, err)
	}
}

func TestUnifiedConnectionAuditRecordsGestureEntryPoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	database, err := storesqlite.Open(ctx, "file:connection-entry-audit?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	repositories := storesqlite.NewRepositories(database)
	engine := gin.New()
	engine.Use(MutationAutomation(command.NewIdempotencyService(repositories, time.Hour), repositories, audit.NewService(repositories)))
	compatibilityEntryPoint := ""
	engine.POST("/api/v1/labs/:labId/connections", func(c *gin.Context) {
		compatibilityEntryPoint = command.TopologyConnectionEntryPoint(c, "")
		c.Set("topology_entry_point", "port_drag")
		c.JSON(http.StatusAccepted, gin.H{"task": map[string]any{"id": "connection-task", "resource_type": "link", "resource_id": "link-1"}})
	})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/labs/lab-1/connections", bytes.NewBufferString(`{"entry_point":"port_drag"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "connection-entry")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if compatibilityEntryPoint != "compatibility_http" {
		t.Fatalf("compatibility entry point=%q", compatibilityEntryPoint)
	}
	events, err := repositories.ListAuditEvents(ctx, 10)
	if err != nil || len(events) != 1 || events[0].TaskID != "connection-task" || events[0].Details["entry_point"] != "port_drag" {
		t.Fatalf("events=%+v err=%v", events, err)
	}
}

func TestDurableTaskMutationIncludesLaboratoryAutomation(t *testing.T) {
	tests := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/labs/lab-1/exports"},
		{http.MethodPost, "/api/v1/labs/lab-1/duplicate"},
		{http.MethodPost, "/api/v1/lab-imports"},
		{http.MethodPost, "/api/v1/labs/lab-1/network-objects"},
		{http.MethodPost, "/api/v1/labs/lab-1/connections"},
		{http.MethodDelete, "/api/v1/connections/connection-1"},
		{http.MethodDelete, "/api/v1/network-objects/object-1"},
		{http.MethodDelete, "/api/v1/labs/lab-1"},
		{http.MethodPost, "/api/v1/captures"},
		{http.MethodDelete, "/api/v1/captures/capture-1"},
		{http.MethodPost, "/api/v1/traffic-filters"},
		{http.MethodDelete, "/api/v1/traffic-filters/filter-1"},
	}
	for _, test := range tests {
		if !durableTaskMutation(test.method, test.path) {
			t.Fatalf("expected durable mutation: %s %s", test.method, test.path)
		}
	}
}

func mutationRequest(handler http.Handler, body, key string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/nodes/node-1/state", bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", key)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
