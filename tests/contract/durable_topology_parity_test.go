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
	"github.com/netlab/netlab/internal/api/mcp"
	"github.com/netlab/netlab/internal/app/artifact"
	"github.com/netlab/netlab/internal/app/command"
	"github.com/netlab/netlab/internal/app/query"
	"github.com/netlab/netlab/internal/app/reconcile"
	"github.com/netlab/netlab/internal/app/task"
	"github.com/netlab/netlab/internal/domain"
	storesqlite "github.com/netlab/netlab/internal/store/sqlite"
)

func TestDurableNodeLifecycleReturnsSameRESTAndMCPTaskEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	database, err := storesqlite.Open(ctx, "file:durable-topology-parity?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	repositories := storesqlite.NewRepositories(database)
	topology := storesqlite.NewTopologyRepository(database)
	lab, err := command.NewLaboratoryService(topology).Create(ctx, "durable-parity", "", domain.RecoveryRemainStopped)
	if err != nil {
		t.Fatal(err)
	}
	node, _, err := command.NewNodeService(topology).Create(ctx, lab.ID, "node", "pc", 1)
	if err != nil {
		t.Fatal(err)
	}
	runner := task.NewRunner(repositories, 1, 8)
	defer runner.Close()
	operations := command.NewTopologyTaskService(topology, runner)

	engine := gin.New()
	httpapi.NewTopologyHandlers(command.NewLaboratoryService(topology), query.NewLaboratoryService(topology), command.NewNodeService(topology), command.NewLinkService(topology), repositories, operations).Register(engine)
	body := bytes.NewBufferString(`{"desired_state":"running"}`)
	request := httptest.NewRequest(http.MethodPut, "/api/v1/nodes/"+string(node.ID)+"/state", body)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("If-Match", "1")
	request.Header.Set("Idempotency-Key", "shared-lifecycle-key")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var restEnvelope struct {
		Task domain.OperationTask `json:"task"`
	}
	if err = json.Unmarshal(response.Body.Bytes(), &restEnvelope); err != nil || restEnvelope.Task.ID == "" {
		t.Fatalf("rest envelope=%+v err=%v", restEnvelope, err)
	}

	var setStateTool mcp.Tool
	for _, tool := range mcp.Tools(mcp.Services{TopologyOps: operations}) {
		if tool.Name == "netlab.nodes.set_state" {
			setStateTool = tool
			break
		}
	}
	if setStateTool.Handler == nil {
		t.Fatal("set-state MCP tool missing")
	}
	mcpContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	result, err := setStateTool.Handler(mcpContext, map[string]any{"node_id": string(node.ID), "desired_state": "running", "expected_revision": float64(1), "idempotency_key": "shared-lifecycle-key"})
	if err != nil {
		t.Fatal(err)
	}
	mcpEnvelope, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("mcp envelope=%T", result)
	}
	mcpTask, ok := mcpEnvelope["task"].(domain.OperationTask)
	if !ok || mcpTask.ID != restEnvelope.Task.ID || mcpTask.Kind != restEnvelope.Task.Kind || mcpTask.ResourceID != restEnvelope.Task.ResourceID {
		t.Fatalf("rest=%+v mcp=%+v", restEnvelope.Task, mcpEnvelope["task"])
	}
	if err = runner.Cancel(ctx, restEnvelope.Task.ID); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		current, getErr := repositories.GetTask(ctx, restEnvelope.Task.ID)
		if getErr == nil && current.State == domain.TaskCancelled {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("durable lifecycle task was not cancelled")
}

func TestDurableLaboratoryImportReturnsSameRESTAndMCPTaskEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	database, err := storesqlite.Open(ctx, "file:durable-automation-parity?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	repositories := storesqlite.NewRepositories(database)
	topology := storesqlite.NewTopologyRepository(database)
	runner := task.NewRunner(repositories, 1, 8)
	defer runner.Close()
	exporter := command.NewExportService(topology, artifact.NewService(repositories, t.TempDir()))
	importer := command.NewImportService(topology, nil)
	automation := command.NewAutomationTaskService(exporter, importer, runner)
	bundle := command.LaboratoryExport{SchemaVersion: 1, Laboratory: command.ExportLaboratory{Name: "import parity", RecoveryPolicy: domain.RecoveryRemainStopped}, Redaction: command.ExportRedaction{ImagesExcluded: true, CredentialsExcluded: true, BootstrapSecretsExcluded: true, CapturesExcluded: true}}
	body, err := json.Marshal(bundle)
	if err != nil {
		t.Fatal(err)
	}

	engine := gin.New()
	httpapi.NewAutomationHandlers(query.NewTaskService(repositories, runner), exporter, importer, automation, nil).Register(engine)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/lab-imports", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "shared-import-key")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var restEnvelope struct {
		Task domain.OperationTask `json:"task"`
	}
	if err = json.Unmarshal(response.Body.Bytes(), &restEnvelope); err != nil || restEnvelope.Task.ID == "" {
		t.Fatalf("rest envelope=%+v err=%v", restEnvelope, err)
	}

	var importTool mcp.Tool
	for _, tool := range mcp.Tools(mcp.Services{Automation: automation}) {
		if tool.Name == "netlab.labs.import" {
			importTool = tool
			break
		}
	}
	if importTool.Handler == nil {
		t.Fatal("import MCP tool missing")
	}
	var bundleArgument map[string]any
	if err = json.Unmarshal(body, &bundleArgument); err != nil {
		t.Fatal(err)
	}
	mcpContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	result, err := importTool.Handler(mcpContext, map[string]any{"bundle": bundleArgument, "idempotency_key": "shared-import-key"})
	if err != nil {
		t.Fatal(err)
	}
	mcpEnvelope, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("mcp envelope=%T", result)
	}
	mcpTask, ok := mcpEnvelope["task"].(domain.OperationTask)
	if !ok || mcpTask.ID != restEnvelope.Task.ID || mcpTask.Kind != restEnvelope.Task.Kind || mcpTask.ResourceID != restEnvelope.Task.ResourceID {
		t.Fatalf("rest=%+v mcp=%+v", restEnvelope.Task, mcpEnvelope["task"])
	}
}

func TestDurableNetworkObjectCreateReturnsSameRESTAndMCPTaskEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	database, err := storesqlite.Open(ctx, "file:durable-network-parity?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	repositories := storesqlite.NewRepositories(database)
	topology := storesqlite.NewTopologyRepository(database)
	lab, err := command.NewLaboratoryService(topology).Create(ctx, "network parity", "", domain.RecoveryRemainStopped)
	if err != nil {
		t.Fatal(err)
	}
	runner := task.NewRunner(repositories, 1, 8)
	defer runner.Close()
	service := reconcile.NewNetworkObjectService(repositories, reconcile.NetworkRuntimeDispatch{})
	operations := reconcile.NewNetworkObjectTaskService(service, runner)

	engine := gin.New()
	httpapi.NewNetworkHandlers(service, operations, nil).Register(engine)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/labs/"+string(lab.ID)+"/network-objects", bytes.NewBufferString(`{"name":"bridge","kind":"bridge","config":{}}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "shared-network-key")
	request.Header.Set("If-Match", "1")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var restEnvelope struct {
		Task domain.OperationTask `json:"task"`
	}
	if err = json.Unmarshal(response.Body.Bytes(), &restEnvelope); err != nil || restEnvelope.Task.ID == "" {
		t.Fatalf("rest envelope=%+v err=%v", restEnvelope, err)
	}

	var createTool mcp.Tool
	for _, tool := range mcp.NetworkTools(service, operations) {
		if tool.Name == "netlab.network_objects.create" {
			createTool = tool
			break
		}
	}
	if createTool.Handler == nil {
		t.Fatal("network create MCP tool missing")
	}
	mcpContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	result, err := createTool.Handler(mcpContext, map[string]any{"lab_id": string(lab.ID), "name": "bridge", "kind": "bridge", "config": map[string]any{}, "expected_revision": float64(1), "idempotency_key": "shared-network-key"})
	if err != nil {
		t.Fatal(err)
	}
	mcpEnvelope, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("mcp envelope=%T", result)
	}
	mcpTask, ok := mcpEnvelope["task"].(domain.OperationTask)
	if !ok || mcpTask.ID != restEnvelope.Task.ID || mcpTask.Kind != restEnvelope.Task.Kind || mcpTask.ResourceID != restEnvelope.Task.ResourceID {
		t.Fatalf("rest=%+v mcp=%+v", restEnvelope.Task, mcpEnvelope["task"])
	}
}

func TestDurableCaptureStartReturnsSameRESTAndMCPTaskEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	database, err := storesqlite.Open(ctx, "file:durable-capture-parity?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	repositories := storesqlite.NewRepositories(database)
	runner := task.NewRunner(repositories, 1, 8)
	defer runner.Close()
	captures := reconcile.NewCaptureManager(t.TempDir(), 2, 1<<20, time.Hour)
	filters := reconcile.NewTrafficFilterManager(captures)
	operations := reconcile.NewCaptureTaskService(captures, filters, runner)

	engine := gin.New()
	httpapi.NewCaptureHandlers(captures, filters, operations).Register(engine)
	body := bytes.NewBufferString(`{"laboratory_id":"lab","source_type":"interface","source_id":"iface","interface":"missing0","format":"pcap","max_bytes":1048576,"duration_seconds":1}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/captures", body)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "shared-capture-key")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var restEnvelope struct {
		Task domain.OperationTask `json:"task"`
	}
	if err = json.Unmarshal(response.Body.Bytes(), &restEnvelope); err != nil || restEnvelope.Task.ID == "" {
		t.Fatalf("rest envelope=%+v err=%v", restEnvelope, err)
	}

	var startTool mcp.Tool
	for _, tool := range mcp.Tools(mcp.Services{Captures: captures, Filters: filters, CaptureOps: operations}) {
		if tool.Name == "netlab.captures.start" {
			startTool = tool
			break
		}
	}
	if startTool.Handler == nil {
		t.Fatal("capture start MCP tool missing")
	}
	mcpContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	result, err := startTool.Handler(mcpContext, map[string]any{"laboratory_id": "lab", "source_type": "interface", "source_id": "iface", "interface": "missing0", "format": "pcap", "max_bytes": float64(1048576), "duration_seconds": float64(1), "idempotency_key": "shared-capture-key"})
	if err != nil {
		t.Fatal(err)
	}
	mcpEnvelope, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("mcp envelope=%T", result)
	}
	mcpTask, ok := mcpEnvelope["task"].(domain.OperationTask)
	if !ok || mcpTask.ID != restEnvelope.Task.ID || mcpTask.Kind != restEnvelope.Task.Kind || mcpTask.ResourceID != restEnvelope.Task.ResourceID {
		t.Fatalf("rest=%+v mcp=%+v", restEnvelope.Task, mcpEnvelope["task"])
	}
}

func TestDurableLaboratoryDeleteReturnsSameRESTAndMCPTaskEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	database, err := storesqlite.Open(ctx, "file:durable-lab-delete-parity?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	repositories := storesqlite.NewRepositories(database)
	topology := storesqlite.NewTopologyRepository(database)
	labs := command.NewLaboratoryService(topology)
	lab, err := labs.Create(ctx, "delete parity", "", domain.RecoveryRemainStopped)
	if err != nil {
		t.Fatal(err)
	}
	runner := task.NewRunner(repositories, 1, 8)
	defer runner.Close()
	operations := command.NewLaboratoryTaskService(topology, runner)

	engine := gin.New()
	httpapi.NewTopologyHandlers(labs, query.NewLaboratoryService(topology), command.NewNodeService(topology), command.NewLinkService(topology), repositories, operations).Register(engine)
	request := httptest.NewRequest(http.MethodDelete, "/api/v1/labs/"+string(lab.ID), nil)
	request.Header.Set("If-Match", "1")
	request.Header.Set("Idempotency-Key", "shared-lab-delete-key")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var restEnvelope struct {
		Task domain.OperationTask `json:"task"`
	}
	if err = json.Unmarshal(response.Body.Bytes(), &restEnvelope); err != nil || restEnvelope.Task.ID == "" {
		t.Fatalf("rest envelope=%+v err=%v", restEnvelope, err)
	}

	var deleteTool mcp.Tool
	for _, tool := range mcp.Tools(mcp.Services{LabOps: operations}) {
		if tool.Name == "netlab.labs.delete" {
			deleteTool = tool
			break
		}
	}
	if deleteTool.Handler == nil {
		t.Fatal("laboratory delete MCP tool missing")
	}
	mcpContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	result, err := deleteTool.Handler(mcpContext, map[string]any{"lab_id": string(lab.ID), "expected_revision": float64(1), "idempotency_key": "shared-lab-delete-key"})
	if err != nil {
		t.Fatal(err)
	}
	mcpEnvelope, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("mcp envelope=%T", result)
	}
	mcpTask, ok := mcpEnvelope["task"].(domain.OperationTask)
	if !ok || mcpTask.ID != restEnvelope.Task.ID || mcpTask.Kind != restEnvelope.Task.Kind || mcpTask.ResourceID != restEnvelope.Task.ResourceID {
		t.Fatalf("rest=%+v mcp=%+v", restEnvelope.Task, mcpEnvelope["task"])
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		current, getErr := topology.GetLaboratory(ctx, lab.ID)
		if getErr == nil && current.LifecycleState == "deleting" {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if err = topology.FinalizeLaboratoryDeletion(ctx, lab.ID); err != nil {
		t.Fatal(err)
	}
	deadline = time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		current, getErr := repositories.GetTask(ctx, restEnvelope.Task.ID)
		if getErr == nil && current.State == domain.TaskSucceeded {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("laboratory deletion task did not converge")
}
