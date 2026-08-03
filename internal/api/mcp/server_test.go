package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/netlab/netlab/internal/app/audit"
	"github.com/netlab/netlab/internal/app/command"
	"github.com/netlab/netlab/internal/domain"
	storesqlite "github.com/netlab/netlab/internal/store/sqlite"
)

func TestMutatingToolReplaysSuccessFailureAndAudits(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	database, err := storesqlite.Open(ctx, "file:mcp-idempotency?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	repositories := storesqlite.NewRepositories(database)
	calls := 0
	tool := Tool{Name: "netlab.node.start", Handler: func(_ *gin.Context, arguments map[string]any) (any, error) {
		calls++
		if arguments["fail"] == true {
			return nil, domain.Problem{Code: "node_failed", Message: "start failed", Retryable: true}
		}
		return map[string]any{"node_id": arguments["node_id"], "call": calls}, nil
	}}
	server := NewServer([]Tool{tool}, command.NewIdempotencyService(repositories, time.Hour), audit.NewService(repositories))
	engine := gin.New()
	server.Register(engine)

	first := callTool(t, engine, map[string]any{"node_id": "node-1", "idempotency_key": "success-key"})
	second := callTool(t, engine, map[string]any{"node_id": "node-1", "idempotency_key": "success-key"})
	if string(first) != string(second) || calls != 1 {
		t.Fatalf("success replay mismatch calls=%d\n%s\n%s", calls, first, second)
	}
	conflict := callTool(t, engine, map[string]any{"node_id": "node-2", "idempotency_key": "success-key"})
	if !bytes.Contains(conflict, []byte("idempotency_conflict")) || calls != 1 {
		t.Fatalf("expected conflict without invocation: calls=%d body=%s", calls, conflict)
	}
	failure := callTool(t, engine, map[string]any{"node_id": "node-1", "fail": true, "idempotency_key": "failure-key"})
	failureReplay := callTool(t, engine, map[string]any{"node_id": "node-1", "fail": true, "idempotency_key": "failure-key"})
	if string(failure) != string(failureReplay) || !bytes.Contains(failureReplay, []byte("node_failed")) || calls != 2 {
		t.Fatalf("failure replay mismatch calls=%d\n%s\n%s", calls, failure, failureReplay)
	}
	audits, err := repositories.ListAuditEvents(ctx, 10)
	if err != nil || len(audits) != 5 {
		t.Fatalf("audits=%d err=%v", len(audits), err)
	}
}

func TestProblemFromErrorPreservesWrappedPointerProblem(t *testing.T) {
	problem := problemFromError(fmt.Errorf("tool failed: %w", &domain.Problem{Code: "temporary_unavailable", Message: "QMP unavailable", Retryable: true, ResourceType: "node", ResourceID: "node", Phase: "starting", Cleanup: "no runtime created", OperatorHint: "retry after QMP recovers", RetryAfterSeconds: 4}))
	if problem.Code != "temporary_unavailable" || problem.ResourceID != "node" || problem.Phase != "starting" || problem.RetryAfterSeconds != 4 {
		t.Fatalf("problem=%+v", problem)
	}
}

func callTool(t *testing.T, handler http.Handler, arguments map[string]any) []byte {
	t.Helper()
	payload := map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": "netlab.node.start", "arguments": arguments}}
	body, _ := json.Marshal(payload)
	request := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	return response.Body.Bytes()
}
