package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/netlab/netlab/internal/api/mcp"
	"github.com/netlab/netlab/internal/app/command"
	"github.com/netlab/netlab/internal/app/reconcile"
	"github.com/netlab/netlab/internal/app/task"
	"github.com/netlab/netlab/internal/domain"
	storesqlite "github.com/netlab/netlab/internal/store/sqlite"
)

type topologyConnectionControlPlaneResult struct {
	TaskID       domain.ID
	ProblemCode  string
	Key          string
	ControlPlane string
}

func TestTopologyConnectionRealControlPlanesSerializeSharedPort(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	database, err := storesqlite.Open(ctx, "file:"+string(domain.NewID())+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	repositories := storesqlite.NewRepositories(database)
	topology := storesqlite.NewTopologyRepository(database)
	laboratory, err := command.NewLaboratoryService(topology).Create(ctx, "control-plane-contention", "", domain.RecoveryAutoRestore)
	if err != nil {
		t.Fatal(err)
	}
	nodeService := command.NewNodeService(topology)
	sourceNode, sourceInterfaces, err := nodeService.CreateConfigured(ctx, laboratory.ID, command.CreateNodeRequest{Name: "source", Kind: "docker", InterfaceCount: 1, InterfaceLimit: 4})
	if err != nil {
		t.Fatal(err)
	}
	firstTargetNode, firstTargetInterfaces, err := nodeService.CreateConfigured(ctx, laboratory.ID, command.CreateNodeRequest{Name: "target-http", Kind: "docker", InterfaceCount: 1, InterfaceLimit: 4})
	if err != nil {
		t.Fatal(err)
	}
	secondTargetNode, secondTargetInterfaces, err := nodeService.CreateConfigured(ctx, laboratory.ID, command.CreateNodeRequest{Name: "target-mcp", Kind: "docker", InterfaceCount: 1, InterfaceLimit: 4})
	if err != nil {
		t.Fatal(err)
	}
	runner := task.NewRunner(repositories, 2, 16)
	defer runner.Close()
	linkTasks := command.NewTopologyTaskService(topology, runner)
	networkTasks := reconcile.NewNetworkObjectTaskService(reconcile.NewNetworkObjectService(repositories, reconcile.NetworkRuntimeDispatch{}), runner)
	connections := command.NewTopologyConnectionService(repositories, linkTasks, networkTasks)
	engine := gin.New()
	NewTopologyConnectionHandlers(connections, repositories).Register(engine)
	mcp.NewServer(mcp.TopologyConnectionTools(connections, repositories)).Register(engine)

	source := domain.ConnectionEndpoint{Kind: domain.ConnectionEndpointNodeInterface, ResourceID: sourceNode.ID, PortID: sourceInterfaces[0].ID}
	httpTarget := domain.ConnectionEndpoint{Kind: domain.ConnectionEndpointNodeInterface, ResourceID: firstTargetNode.ID, PortID: firstTargetInterfaces[0].ID}
	mcpTarget := domain.ConnectionEndpoint{Kind: domain.ConnectionEndpointNodeInterface, ResourceID: secondTargetNode.ID, PortID: secondTargetInterfaces[0].ID}
	start := make(chan struct{})
	results := make(chan topologyConnectionControlPlaneResult, 2)
	var waitGroup sync.WaitGroup
	waitGroup.Add(2)
	go func() {
		defer waitGroup.Done()
		<-start
		results <- invokeTopologyConnectionHTTP(t, engine, laboratory, source, httpTarget, "http-contention")
	}()
	go func() {
		defer waitGroup.Done()
		<-start
		results <- invokeTopologyConnectionMCP(t, engine, laboratory, source, mcpTarget, "mcp-contention")
	}()
	close(start)
	waitGroup.Wait()
	close(results)

	values := make([]topologyConnectionControlPlaneResult, 0, 2)
	for result := range results {
		values = append(values, result)
	}
	if len(values) != 2 {
		t.Fatalf("results=%+v", values)
	}
	tasks := make([]domain.OperationTask, 0, 2)
	for _, value := range values {
		if value.TaskID != "" {
			tasks = append(tasks, waitForTopologyConnectionTask(t, repositories, value.TaskID))
		}
	}
	var winner, loser domain.OperationTask
	conflicts := 0
	for _, value := range values {
		if value.ProblemCode == "port_in_use" {
			conflicts++
		}
	}
	for _, value := range tasks {
		switch value.State {
		case domain.TaskSucceeded:
			winner = value
		case domain.TaskFailed:
			loser = value
			if value.Error != nil && value.Error.Code == "port_in_use" {
				conflicts++
			}
		}
	}
	if winner.ID == "" || conflicts != 1 {
		t.Fatalf("tasks=%+v", tasks)
	}
	if loser.ID != "" && (loser.Error == nil || loser.Error.Code != "port_in_use") {
		t.Fatalf("loser=%+v", loser)
	}

	refreshRequest := httptest.NewRequest(http.MethodGet, "/api/v1/labs/"+string(laboratory.ID)+"/connections", nil)
	refreshResponse := httptest.NewRecorder()
	engine.ServeHTTP(refreshResponse, refreshRequest)
	if refreshResponse.Code != http.StatusOK {
		t.Fatalf("refresh status=%d body=%s", refreshResponse.Code, refreshResponse.Body.String())
	}
	var refresh struct {
		Connections []domain.TopologyConnection `json:"connections"`
		Endpoints   []domain.ConnectionEndpoint `json:"endpoints"`
	}
	if err = json.Unmarshal(refreshResponse.Body.Bytes(), &refresh); err != nil {
		t.Fatal(err)
	}
	if len(refresh.Connections) != 1 || refresh.Connections[0].ID != winner.ResourceID {
		t.Fatalf("refresh=%+v winner=%+v", refresh, winner)
	}
	for _, endpoint := range refresh.Endpoints {
		if endpoint.PortID == sourceInterfaces[0].ID && endpoint.Availability != domain.ConnectionEndpointOccupied {
			t.Fatalf("source endpoint=%+v", endpoint)
		}
	}

	var replay topologyConnectionControlPlaneResult
	if winner.IdempotencyKey == "http-contention" {
		replay = invokeTopologyConnectionHTTP(t, engine, laboratory, source, httpTarget, winner.IdempotencyKey)
	} else {
		replay = invokeTopologyConnectionMCP(t, engine, laboratory, source, mcpTarget, winner.IdempotencyKey)
	}
	if replay.TaskID != winner.ID {
		t.Fatalf("replay task=%s winner=%s", replay.TaskID, winner.ID)
	}
}

func invokeTopologyConnectionHTTP(t *testing.T, engine *gin.Engine, laboratory domain.Laboratory, source, target domain.ConnectionEndpoint, key string) topologyConnectionControlPlaneResult {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"source": source, "target": target})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/labs/"+string(laboratory.ID)+"/connections", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("If-Match", "1")
	request.Header.Set("Idempotency-Key", key)
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted {
		var problem domain.Problem
		if err := json.Unmarshal(response.Body.Bytes(), &problem); err != nil {
			t.Fatal(err)
		}
		return topologyConnectionControlPlaneResult{ProblemCode: problem.Code, Key: key, ControlPlane: "http"}
	}
	var value struct {
		Task domain.OperationTask `json:"task"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &value); err != nil {
		t.Fatal(err)
	}
	return topologyConnectionControlPlaneResult{TaskID: value.Task.ID, Key: key, ControlPlane: "http"}
}

func invokeTopologyConnectionMCP(t *testing.T, engine *gin.Engine, laboratory domain.Laboratory, source, target domain.ConnectionEndpoint, key string) topologyConnectionControlPlaneResult {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": key, "method": "tools/call", "params": map[string]any{"name": "netlab.topology_connections.create", "arguments": map[string]any{"laboratory_id": laboratory.ID, "expected_revision": laboratory.Revision, "idempotency_key": key, "source": source, "target": target}}})
	request := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("MCP status=%d body=%s", response.Code, response.Body.String())
	}
	var value struct {
		Result struct {
			StructuredContent json.RawMessage `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &value); err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		Task domain.OperationTask `json:"task"`
	}
	if err := json.Unmarshal(value.Result.StructuredContent, &envelope); err == nil && envelope.Task.ID != "" {
		return topologyConnectionControlPlaneResult{TaskID: envelope.Task.ID, Key: key, ControlPlane: "mcp"}
	}
	var problem domain.Problem
	if err := json.Unmarshal(value.Result.StructuredContent, &problem); err != nil {
		t.Fatal(err)
	}
	return topologyConnectionControlPlaneResult{ProblemCode: problem.Code, Key: key, ControlPlane: "mcp"}
}

func waitForTopologyConnectionTask(t *testing.T, repositories *storesqlite.Repositories, id domain.ID) domain.OperationTask {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		value, err := repositories.GetTask(context.Background(), id)
		if err == nil && (value.State == domain.TaskSucceeded || value.State == domain.TaskFailed) {
			return value
		}
		time.Sleep(5 * time.Millisecond)
	}
	value, _ := repositories.GetTask(context.Background(), id)
	t.Fatalf("task did not finish: %+v", value)
	return domain.OperationTask{}
}
