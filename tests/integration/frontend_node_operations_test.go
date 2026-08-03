package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	httpapi "github.com/netlab/netlab/internal/api/http"
	"github.com/netlab/netlab/internal/app/audit"
	"github.com/netlab/netlab/internal/app/command"
	"github.com/netlab/netlab/internal/app/query"
	"github.com/netlab/netlab/internal/app/task"
	"github.com/netlab/netlab/internal/domain"
	storesqlite "github.com/netlab/netlab/internal/store/sqlite"
)

func TestFrontendNodeOperationsExposeTasksReplayCancellationAndProblems(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	database, err := storesqlite.Open(ctx, "file:frontend-node-operations?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	repositories := storesqlite.NewRepositories(database)
	topology := storesqlite.NewTopologyRepository(database)
	lab, _ := command.NewLaboratoryService(topology).Create(ctx, "node operations", "", domain.RecoveryRemainStopped)
	node, _, _ := command.NewNodeService(topology).Create(ctx, lab.ID, "node", "pc", 1)
	runner := task.NewRunner(repositories, 1, 8)
	defer runner.Close()
	operations := command.NewTopologyTaskService(topology, runner)
	engine := gin.New()
	engine.Use(httpapi.MutationAutomation(command.NewIdempotencyService(repositories, time.Hour), repositories, audit.NewService(repositories)))
	httpapi.NewTopologyHandlers(command.NewLaboratoryService(topology), query.NewLaboratoryService(topology), command.NewNodeService(topology), command.NewLinkService(topology), repositories, operations).Register(engine)

	requestState := func(key string, revision int) *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodPut, "/api/v1/nodes/"+string(node.ID)+"/state", bytes.NewBufferString(`{"desired_state":"running"}`))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("If-Match", strconv.Itoa(revision))
		request.Header.Set("Idempotency-Key", key)
		response := httptest.NewRecorder()
		engine.ServeHTTP(response, request)
		return response
	}

	first := requestState("frontend-replay", 1)
	second := requestState("frontend-replay", 1)
	var firstEnvelope, secondEnvelope struct {
		Task domain.OperationTask `json:"task"`
	}
	if err = json.Unmarshal(first.Body.Bytes(), &firstEnvelope); err != nil || firstEnvelope.Task.ID == "" {
		t.Fatalf("invalid task envelope: %v %s", err, first.Body.String())
	}
	if err = json.Unmarshal(second.Body.Bytes(), &secondEnvelope); err != nil {
		t.Fatal(err)
	}
	if first.Code != http.StatusAccepted || second.Code != http.StatusAccepted || firstEnvelope.Task.ID != secondEnvelope.Task.ID {
		t.Fatalf("idempotent replay mismatch: first=%d %+v second=%d %+v", first.Code, firstEnvelope.Task, second.Code, secondEnvelope.Task)
	}
	if _, err = query.NewTaskService(repositories, runner).Cancel(ctx, firstEnvelope.Task.ID); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		current, getErr := repositories.GetTask(ctx, firstEnvelope.Task.ID)
		if getErr == nil && current.State == domain.TaskCancelled {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	conflict := requestState("frontend-conflict", 99)
	if conflict.Code != http.StatusAccepted {
		t.Fatalf("expected durable conflict task, got %d %s", conflict.Code, conflict.Body.String())
	}
	var conflictEnvelope struct {
		Task domain.OperationTask `json:"task"`
	}
	if err = json.Unmarshal(conflict.Body.Bytes(), &conflictEnvelope); err != nil {
		t.Fatal(err)
	}
	deadline = time.Now().Add(time.Second)
	var conflictTask domain.OperationTask
	for time.Now().Before(deadline) {
		conflictTask, err = repositories.GetTask(ctx, conflictEnvelope.Task.ID)
		if err == nil && conflictTask.State == domain.TaskFailed {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if conflictTask.Error == nil || conflictTask.Error.Code == "" || conflictTask.Error.ResourceType == "" || conflictTask.Error.ResourceID == "" {
		t.Fatalf("structured conflict problem missing fields: %+v", conflictTask)
	}
}
