package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/netlab/netlab/internal/app/command"
	"github.com/netlab/netlab/internal/app/query"
	"github.com/netlab/netlab/internal/app/reconcile"
	"github.com/netlab/netlab/internal/app/task"
	"github.com/netlab/netlab/internal/domain"
	storesqlite "github.com/netlab/netlab/internal/store/sqlite"
)

func TestTopologyCreationContractReturnsAuthoritativePlacement(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	database, err := storesqlite.Open(ctx, "file:topology-creation-contract?mode=memory&cache=shared")
	if err != nil { t.Fatal(err) }
	defer database.Close()
	topology := storesqlite.NewTopologyRepository(database)
	repositories := storesqlite.NewRepositories(database)
	laboratory, err := command.NewLaboratoryService(topology).Create(ctx, "creation contract", "", domain.RecoveryAutoRestore)
	if err != nil { t.Fatal(err) }
	runner := task.NewRunner(repositories, 1, 8)
	defer runner.Close()
	networkService := reconcile.NewNetworkObjectService(repositories, reconcile.NetworkRuntimeDispatch{})
	engine := gin.New()
	engine.Use(MutationAutomation(command.NewIdempotencyService(repositories, time.Hour), repositories, nil))
	NewTopologyHandlers(command.NewLaboratoryService(topology), query.NewLaboratoryService(topology), command.NewNodeService(topology), command.NewLinkService(topology), repositories).Register(engine)
	NewNetworkHandlers(networkService, reconcile.NewNetworkObjectTaskService(networkService, runner), nil).Register(engine)

	create := func(path, revision, key string, body map[string]any, expected int) map[string]any {
		encoded, _ := json.Marshal(body)
		request := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(encoded))
		request.Header.Set("Content-Type", "application/json")
		if revision != "" { request.Header.Set("If-Match", revision) }
		if key != "" { request.Header.Set("Idempotency-Key", key) }
		response := httptest.NewRecorder()
		engine.ServeHTTP(response, request)
		if response.Code != expected { t.Fatalf("status=%d want=%d body=%s", response.Code, expected, response.Body.String()) }
		var result map[string]any
		_ = json.Unmarshal(response.Body.Bytes(), &result)
		return result
	}

	path := "/api/v1/labs/" + string(laboratory.ID) + "/nodes"
	body := map[string]any{"name": "node-a", "kind": "docker", "interface_count": 1, "placement_intent": map[string]any{"preferred_x": 40, "preferred_y": 50, "footprint_class": "node-standard"}}
	first := create(path, "1", "node-key", body, http.StatusCreated)
	assignment := first["placement_assignment"].(map[string]any)
	if assignment["adjusted"].(bool) || assignment["reason"] != "preferred_available" { t.Fatalf("assignment=%+v", assignment) }
	if first["laboratory_revision"].(float64) != 2 { t.Fatalf("result=%+v", first) }
	create(path, "2", "node-key", map[string]any{"name": "different", "kind": "docker", "interface_count": 1}, http.StatusConflict)
	create(path, "", "missing-revision", map[string]any{"name": "missing", "kind": "docker", "interface_count": 1}, http.StatusPreconditionRequired)

	object := create("/api/v1/labs/"+string(laboratory.ID)+"/network-objects", "2", "object-key", map[string]any{"name": "bridge", "kind": "bridge", "config": map[string]any{}, "placement_intent": map[string]any{"preferred_x": 40, "preferred_y": 50, "footprint_class": "network-object-standard"}}, http.StatusAccepted)
	objectAssignment := object["placement_assignment"].(map[string]any)
	if !objectAssignment["adjusted"].(bool) || object["laboratory_revision"].(float64) != 3 { t.Fatalf("object=%+v", object) }
}
