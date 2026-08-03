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
	"github.com/netlab/netlab/internal/app/reconcile"
	"github.com/netlab/netlab/internal/app/task"
	"github.com/netlab/netlab/internal/domain"
	storesqlite "github.com/netlab/netlab/internal/store/sqlite"
)

func TestNetworkObjectLinkRESTAndMCPCreateParity(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	database, err := storesqlite.Open(ctx, "file:object-link-control-parity?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	repositories := storesqlite.NewRepositories(database)
	topology := storesqlite.NewTopologyRepository(database)
	now := time.Now().UTC()
	lab := domain.Laboratory{ID: "lab-parity", Name: "parity", Revision: 1, RecoveryPolicy: domain.RecoveryRemainStopped, LifecycleState: "active", CreatedAt: now, UpdatedAt: now}
	if err = topology.CreateLaboratory(ctx, lab); err != nil {
		t.Fatal(err)
	}
	for _, object := range []domain.NetworkObject{{ID: "object-a", LaboratoryID: lab.ID, Name: "A", Kind: domain.NetworkSwitchL2, Revision: 1, DesiredState: "active", ObservedState: "active", Config: map[string]any{}, CreatedAt: now, UpdatedAt: now}, {ID: "object-b", LaboratoryID: lab.ID, Name: "B", Kind: domain.NetworkSwitchL2, Revision: 1, DesiredState: "active", ObservedState: "active", Config: map[string]any{}, CreatedAt: now, UpdatedAt: now}} {
		if err = repositories.CreateNetworkObject(ctx, object); err != nil {
			t.Fatal(err)
		}
	}
	runner := task.NewRunner(repositories, 1, 8)
	defer runner.Close()
	service := reconcile.NewNetworkObjectService(repositories, reconcile.NetworkRuntimeDispatch{})
	operations := reconcile.NewNetworkObjectTaskService(service, runner)
	engine := gin.New()
	httpapi.NewNetworkHandlers(service, operations, nil).Register(engine)
	requestBody := []byte(`{"object_a_id":"object-a","port_a_name":"swp1","object_b_id":"object-b","port_b_name":"swp1"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/labs/lab-parity/network-object-links", bytes.NewReader(requestBody))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "shared-link-key")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var restEnvelope struct {
		NetworkObjectLink domain.NetworkObjectLink `json:"network_object_link"`
		Task              domain.OperationTask     `json:"task"`
	}
	if err = json.Unmarshal(response.Body.Bytes(), &restEnvelope); err != nil {
		t.Fatal(err)
	}
	var createTool mcp.Tool
	for _, tool := range mcp.NetworkTools(service, operations) {
		if tool.Name == "netlab.network_object_links.create" {
			createTool = tool
		}
	}
	mcpContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	result, err := createTool.Handler(mcpContext, map[string]any{"laboratory_id": "lab-parity", "object_a_id": "object-a", "port_a_name": "swp1", "object_b_id": "object-b", "port_b_name": "swp1", "idempotency_key": "shared-link-key"})
	if err != nil {
		t.Fatal(err)
	}
	mcpEnvelope := result.(map[string]any)
	mcpTask := mcpEnvelope["task"].(domain.OperationTask)
	if mcpTask.ID != restEnvelope.Task.ID || mcpEnvelope["network_object_link"].(domain.NetworkObjectLink).ID != restEnvelope.NetworkObjectLink.ID {
		t.Fatalf("rest=%+v mcp=%+v", restEnvelope, mcpEnvelope)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		current, _ := repositories.GetTask(ctx, restEnvelope.Task.ID)
		if current.State == domain.TaskSucceeded {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	getRequest := httptest.NewRequest(http.MethodGet, "/api/v1/network-object-links/"+string(restEnvelope.NetworkObjectLink.ID), nil)
	getResponse := httptest.NewRecorder()
	engine.ServeHTTP(getResponse, getRequest)
	if getResponse.Code != http.StatusOK {
		t.Fatalf("get status=%d body=%s", getResponse.Code, getResponse.Body.String())
	}
	conflictRequest := httptest.NewRequest(http.MethodPost, "/api/v1/labs/lab-parity/network-object-links", bytes.NewReader([]byte(`{"object_a_id":"object-a","port_a_name":"swp1","object_b_id":"object-b","port_b_name":"swp2"}`)))
	conflictRequest.Header.Set("Content-Type", "application/json")
	conflictResponse := httptest.NewRecorder()
	engine.ServeHTTP(conflictResponse, conflictRequest)
	if conflictResponse.Code != http.StatusConflict || !bytes.Contains(conflictResponse.Body.Bytes(), []byte("port_in_use")) {
		t.Fatalf("conflict status=%d body=%s", conflictResponse.Code, conflictResponse.Body.String())
	}
}
