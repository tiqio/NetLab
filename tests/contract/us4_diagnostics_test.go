package contract

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	httpapi "github.com/netlab/netlab/internal/api/http"
	"github.com/netlab/netlab/internal/api/stream"
	"github.com/netlab/netlab/internal/app/reconcile"
	"github.com/netlab/netlab/internal/app/task"
	"github.com/netlab/netlab/internal/domain"
	consoleRuntime "github.com/netlab/netlab/internal/runtime/console"
	storesqlite "github.com/netlab/netlab/internal/store/sqlite"
)

func TestConsoleAndTrafficFilterContracts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	stream.NewConsoleHandlers(t.TempDir(), consoleRuntime.Limits{IdleTimeout: time.Minute}).Register(engine)
	database, err := storesqlite.Open(context.Background(), "file:diagnostics-contract?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	runner := task.NewRunner(storesqlite.NewRepositories(database), 1, 8)
	defer runner.Close()
	captures := reconcile.NewCaptureManager(t.TempDir(), 1, 1<<20, time.Hour)
	filters := reconcile.NewTrafficFilterManager(captures)
	httpapi.NewCaptureHandlers(captures, filters, reconcile.NewCaptureTaskService(captures, filters, runner)).Register(engine)

	discovery := httptest.NewRecorder()
	engine.ServeHTTP(discovery, httptest.NewRequest(http.MethodGet, "/api/v1/nodes/node-1/consoles", nil))
	if discovery.Code != http.StatusOK || !bytes.Contains(discovery.Body.Bytes(), []byte("telnet")) || !bytes.Contains(discovery.Body.Bytes(), []byte("vnc")) {
		t.Fatalf("discovery=%d %s", discovery.Code, discovery.Body.String())
	}
	body, _ := json.Marshal(map[string]any{"laboratory_id": "lab-1", "match": map[string]any{"protocol": "tcp", "destination_port": 443}, "max_observations": 100})
	start := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/traffic-filters", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(start, req)
	if start.Code != http.StatusAccepted || !bytes.Contains(start.Body.Bytes(), []byte("dst port 443")) {
		t.Fatalf("start=%d %s", start.Code, start.Body.String())
	}
}

type consoleNodeReader struct{ node domain.Node }

func (r consoleNodeReader) GetNode(context.Context, domain.ID) (domain.Node, error) {
	return r.node, nil
}

func TestConsoleDiscoveryHonorsDeclaredModes(t *testing.T) {
	engine := gin.New()
	stream.NewConsoleHandlers(t.TempDir(), consoleRuntime.Limits{IdleTimeout: time.Minute}, consoleNodeReader{node: domain.Node{ID: "node-1", Config: map[string]any{"console_modes": []any{"telnet"}}}}).Register(engine)
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/nodes/node-1/consoles", nil))
	if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte("telnet")) || bytes.Contains(response.Body.Bytes(), []byte("vnc")) {
		t.Fatal(response.Body.String())
	}
}
