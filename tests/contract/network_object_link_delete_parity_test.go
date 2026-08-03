package contract

import (
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

func TestNetworkObjectLinkDeleteRESTAndMCPParity(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	database, err := storesqlite.Open(ctx, "file:"+string(domain.NewID())+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	repositories := storesqlite.NewRepositories(database)
	topology := storesqlite.NewTopologyRepository(database)
	now := time.Now().UTC()
	lab := domain.Laboratory{ID: domain.NewID(), Name: "delete parity", Revision: 1, RecoveryPolicy: domain.RecoveryRemainStopped, LifecycleState: "active", CreatedAt: now, UpdatedAt: now}
	if err = topology.CreateLaboratory(ctx, lab); err != nil {
		t.Fatal(err)
	}
	for _, object := range []domain.NetworkObject{{ID: domain.NewID(), LaboratoryID: lab.ID, Name: "A", Kind: domain.NetworkSwitchL2, Revision: 1, DesiredState: "active", ObservedState: "active", Config: map[string]any{}, CreatedAt: now, UpdatedAt: now}, {ID: domain.NewID(), LaboratoryID: lab.ID, Name: "B", Kind: domain.NetworkSwitchL2, Revision: 1, DesiredState: "active", ObservedState: "active", Config: map[string]any{}, CreatedAt: now, UpdatedAt: now}} {
		if err = repositories.CreateNetworkObject(ctx, object); err != nil {
			t.Fatal(err)
		}
	}
	objects, err := repositories.ListNetworkObjects(ctx, lab.ID)
	if err != nil || len(objects) != 2 {
		t.Fatalf("objects=%+v err=%v", objects, err)
	}
	link := domain.NetworkObjectLink{ID: domain.NewID(), LaboratoryID: lab.ID, ObjectAID: objects[0].ID, PortAName: "swp1", ObjectBID: objects[1].ID, PortBName: "swp1", Revision: 1, DesiredState: "connected", ObservedState: "connected"}
	if err = repositories.CreateNetworkObjectLink(ctx, link); err != nil {
		t.Fatal(err)
	}
	runner := task.NewRunner(repositories, 1, 8)
	defer runner.Close()
	service := reconcile.NewNetworkObjectService(repositories, reconcile.NetworkRuntimeDispatch{})
	operations := reconcile.NewNetworkObjectTaskService(service, runner)
	engine := gin.New()
	httpapi.NewNetworkHandlers(service, operations, nil).Register(engine)

	missingRevision := httptest.NewRequest(http.MethodDelete, "/api/v1/network-object-links/"+string(link.ID), nil)
	missingResponse := httptest.NewRecorder()
	engine.ServeHTTP(missingResponse, missingRevision)
	if missingResponse.Code != http.StatusPreconditionRequired {
		t.Fatalf("missing revision status=%d body=%s", missingResponse.Code, missingResponse.Body.String())
	}

	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/v1/network-object-links/"+string(link.ID), nil)
	deleteRequest.Header.Set("If-Match", "1")
	deleteRequest.Header.Set("Idempotency-Key", "delete-parity-key")
	deleteResponse := httptest.NewRecorder()
	engine.ServeHTTP(deleteResponse, deleteRequest)
	if deleteResponse.Code != http.StatusAccepted {
		t.Fatalf("delete status=%d body=%s", deleteResponse.Code, deleteResponse.Body.String())
	}
	var restEnvelope struct {
		NetworkObjectLink domain.NetworkObjectLink `json:"network_object_link"`
		Task              domain.OperationTask     `json:"task"`
	}
	if err = json.Unmarshal(deleteResponse.Body.Bytes(), &restEnvelope); err != nil {
		t.Fatal(err)
	}
	if restEnvelope.NetworkObjectLink.ObservedState != "disconnecting" || restEnvelope.Task.Kind != "network_object_link.delete" {
		t.Fatalf("REST=%+v", restEnvelope)
	}

	var deleteTool mcp.Tool
	for _, tool := range mcp.NetworkTools(service, operations) {
		if tool.Name == "netlab.network_object_links.delete" {
			deleteTool = tool
		}
	}
	mcpContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	result, err := deleteTool.Handler(mcpContext, map[string]any{"link_id": string(link.ID), "expected_revision": float64(1), "idempotency_key": "delete-parity-key"})
	if err != nil {
		t.Fatal(err)
	}
	mcpEnvelope := result.(map[string]any)
	if mcpEnvelope["task"].(domain.OperationTask).ID != restEnvelope.Task.ID || mcpEnvelope["network_object_link"].(domain.NetworkObjectLink).ID != link.ID {
		t.Fatalf("REST=%+v MCP=%+v", restEnvelope, mcpEnvelope)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		current, getErr := repositories.GetTask(ctx, restEnvelope.Task.ID)
		if getErr == nil && current.State == domain.TaskSucceeded {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	retryRequest := httptest.NewRequest(http.MethodDelete, "/api/v1/network-object-links/"+string(link.ID), nil)
	retryRequest.Header.Set("If-Match", "1")
	retryRequest.Header.Set("Idempotency-Key", "delete-parity-key")
	retryResponse := httptest.NewRecorder()
	engine.ServeHTTP(retryResponse, retryRequest)
	if retryResponse.Code != http.StatusAccepted {
		t.Fatalf("retry status=%d body=%s", retryResponse.Code, retryResponse.Body.String())
	}
	var retryEnvelope struct {
		Task domain.OperationTask `json:"task"`
	}
	if err = json.Unmarshal(retryResponse.Body.Bytes(), &retryEnvelope); err != nil || retryEnvelope.Task.ID != restEnvelope.Task.ID {
		t.Fatalf("retry=%+v err=%v", retryEnvelope, err)
	}
}
