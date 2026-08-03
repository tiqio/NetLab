package integration

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
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

func TestAcceptedMutationCompletesOnlyAfterObservedConvergence(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	database, err := storesqlite.Open(ctx, "file:mutation-convergence?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	repositories := storesqlite.NewRepositories(database)
	topology := storesqlite.NewTopologyRepository(database)
	lab, _ := command.NewLaboratoryService(topology).Create(ctx, "lab", "", domain.RecoveryAutoRestore)
	node, _, _ := command.NewNodeService(topology).Create(ctx, lab.ID, "node", "pc", 1)
	runner := task.NewRunner(repositories, 1, 8)
	defer runner.Close()
	operations := command.NewTopologyTaskService(topology, runner)
	engine := gin.New()
	engine.Use(httpapi.MutationAutomation(command.NewIdempotencyService(repositories, time.Hour), repositories, audit.NewService(repositories)))
	httpapi.NewTopologyHandlers(command.NewLaboratoryService(topology), query.NewLaboratoryService(topology), command.NewNodeService(topology), command.NewLinkService(topology), repositories, operations).Register(engine)
	request := httptest.NewRequest(http.MethodPut, "/api/v1/nodes/"+string(node.ID)+"/state", bytes.NewBufferString(`{"desired_state":"running"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("If-Match", "1")
	request.Header.Set("Idempotency-Key", "lifecycle-convergence")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var tasks []domain.OperationTask
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		tasks, _ = repositories.ListTasks(ctx, 10)
		if len(tasks) == 1 && tasks[0].State == domain.TaskRunning && tasks[0].ProgressCurrent >= 1 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if len(tasks) != 1 || tasks[0].Kind != "node.set_state" || tasks[0].State != domain.TaskRunning {
		t.Fatalf("status=%d tasks=%+v", response.Code, tasks)
	}
	taskValue, _ := repositories.GetTask(ctx, tasks[0].ID)
	if taskValue.State != domain.TaskRunning {
		t.Fatalf("completed before convergence: %+v", taskValue)
	}
	if err = topology.SetNodeObservedState(ctx, node.ID, domain.ObservedStarting, nil); err != nil {
		t.Fatal(err)
	}
	if err = topology.SetNodeObservedState(ctx, node.ID, domain.ObservedRunning, nil); err != nil {
		t.Fatal(err)
	}
	deadline = time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		taskValue, _ = repositories.GetTask(ctx, tasks[0].ID)
		if taskValue.State == domain.TaskSucceeded {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if taskValue.State != domain.TaskSucceeded || taskValue.ProgressCurrent != taskValue.ProgressTotal {
		t.Fatalf("task did not converge: %+v", taskValue)
	}
}

func TestTaskCancellationPropagatesToRunningHandler(t *testing.T) {
	ctx := context.Background()
	database, err := storesqlite.Open(ctx, "file:task-cancel?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	repositories := storesqlite.NewRepositories(database)
	runner := task.NewRunner(repositories, 1, 4)
	defer runner.Close()
	started := make(chan struct{})
	runner.Register("blocking", func(ctx context.Context, _ *domain.OperationTask) (map[string]any, error) {
		close(started)
		<-ctx.Done()
		return nil, ctx.Err()
	})
	taskValue := domain.OperationTask{ID: domain.NewID(), Kind: "blocking", ResourceType: "node", ResourceID: "node-1", ProgressTotal: 1}
	if err = runner.Enqueue(ctx, taskValue); err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("task did not start")
	}
	service := query.NewTaskService(repositories, runner)
	if _, err = service.Cancel(ctx, taskValue.ID); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		current, getErr := repositories.GetTask(ctx, taskValue.ID)
		if getErr == nil && current.State == domain.TaskCancelled {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	current, _ := repositories.GetTask(ctx, taskValue.ID)
	t.Fatalf("task was not cancelled: %+v", current)
}
