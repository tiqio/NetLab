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
	"github.com/netlab/netlab/internal/app/reconcile"
	"github.com/netlab/netlab/internal/app/task"
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

func TestTopologyCreationToolsReturnAuthoritativePlacementAndConflicts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	database, err := storesqlite.Open(ctx, "file:mcp-topology-creation?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	topology := storesqlite.NewTopologyRepository(database)
	repositories := storesqlite.NewRepositories(database)
	laboratory, err := command.NewLaboratoryService(topology).Create(ctx, "mcp placement", "", domain.RecoveryAutoRestore)
	if err != nil {
		t.Fatal(err)
	}
	runner := task.NewRunner(repositories, 1, 8)
	defer runner.Close()
	networkService := reconcile.NewNetworkObjectService(repositories, reconcile.NetworkRuntimeDispatch{})
	networkOperations := reconcile.NewNetworkObjectTaskService(networkService, runner)
	tools := Tools(Services{Nodes: command.NewNodeService(topology)})
	tools = append(tools, NetworkTools(networkService, networkOperations)...)
	server := NewServer(tools, command.NewIdempotencyService(repositories, time.Hour), audit.NewService(repositories))
	engine := gin.New()
	server.Register(engine)

	nodeArguments := map[string]any{
		"lab_id": string(laboratory.ID), "name": "node-a", "kind": "docker",
		"interface_count": 1, "expected_revision": 1, "idempotency_key": "mcp-node",
		"placement_intent": map[string]any{"preferred_x": 80, "preferred_y": 120, "footprint_class": "node-standard"},
	}
	first := callNamedTool(t, engine, "netlab.nodes.create", nodeArguments)
	second := callNamedTool(t, engine, "netlab.nodes.create", nodeArguments)
	if string(first) != string(second) || !bytes.Contains(first, []byte(`"placement_assignment"`)) || !bytes.Contains(first, []byte(`"laboratory_revision":2`)) || !bytes.Contains(first, []byte(`"reason":"preferred_available"`)) {
		t.Fatalf("node create/replay mismatch\n%s\n%s", first, second)
	}
	conflictArguments := map[string]any{}
	for key, value := range nodeArguments {
		conflictArguments[key] = value
	}
	conflictArguments["name"] = "node-b"
	conflict := callNamedTool(t, engine, "netlab.nodes.create", conflictArguments)
	if !bytes.Contains(conflict, []byte("idempotency_conflict")) {
		t.Fatalf("expected idempotency conflict: %s", conflict)
	}

	object := callNamedTool(t, engine, "netlab.network_objects.create", map[string]any{
		"lab_id": string(laboratory.ID), "name": "bridge-a", "kind": "bridge",
		"config": map[string]any{}, "expected_revision": 2, "idempotency_key": "mcp-object",
		"placement_intent": map[string]any{"preferred_x": 80, "preferred_y": 120, "footprint_class": "network-object-standard"},
	})
	if !bytes.Contains(object, []byte(`"placement_assignment"`)) || !bytes.Contains(object, []byte(`"laboratory_revision":3`)) || !bytes.Contains(object, []byte(`"adjusted":true`)) {
		t.Fatalf("object result=%s", object)
	}
	stale := callNamedTool(t, engine, "netlab.network_objects.create", map[string]any{
		"lab_id": string(laboratory.ID), "name": "bridge-stale", "kind": "bridge",
		"config": map[string]any{}, "expected_revision": 2, "idempotency_key": "mcp-object-stale",
	})
	if !bytes.Contains(stale, []byte("revision_conflict")) || !bytes.Contains(stale, []byte(`"retryable":true`)) {
		t.Fatalf("expected structured revision conflict: %s", stale)
	}
	switchArguments := map[string]any{
		"lab_id": string(laboratory.ID), "name": "switch-a", "kind": "switch_l3",
		"expected_revision": 3, "idempotency_key": "mcp-switch",
	}
	switchFirst := callNamedTool(t, engine, "netlab.network_objects.create", switchArguments)
	switchReplay := callNamedTool(t, engine, "netlab.network_objects.create", switchArguments)
	for _, expected := range []string{`"name":"eth0"`, `"name":"eth1"`, `"name":"eth2"`, `"name":"eth3"`} {
		if !bytes.Contains(switchFirst, []byte(expected)) {
			t.Fatalf("missing %s in %s", expected, switchFirst)
		}
	}
	if string(switchFirst) != string(switchReplay) {
		t.Fatalf("switch replay mismatch\n%s\n%s", switchFirst, switchReplay)
	}
}

func callTool(t *testing.T, handler http.Handler, arguments map[string]any) []byte {
	return callNamedTool(t, handler, "netlab.node.start", arguments)
}

func callNamedTool(t *testing.T, handler http.Handler, name string, arguments map[string]any) []byte {
	t.Helper()
	payload := map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": name, "arguments": arguments}}
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
