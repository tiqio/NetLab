package reconcile

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/app/task"
	"github.com/netlab/netlab/internal/domain"
	storesqlite "github.com/netlab/netlab/internal/store/sqlite"
)

type networkTaskRuntimeFake struct {
	mu             sync.Mutex
	configured     map[domain.ID]bool
	values         map[domain.ID]domain.NetworkObject
	blockConfigure bool
}

func (r *networkTaskRuntimeFake) Configure(ctx context.Context, value domain.NetworkObject) error {
	if r.blockConfigure {
		<-ctx.Done()
		return ctx.Err()
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.configured[value.ID] = true
	if r.values != nil {
		r.values[value.ID] = value
	}
	return nil
}

func TestNetworkObjectUpdatePersistsVersionedSwitchConfiguration(t *testing.T) {
	ctx := context.Background()
	database, err := storesqlite.Open(ctx, "file:"+string(domain.NewID())+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	repositories := storesqlite.NewRepositories(database)
	topology := storesqlite.NewTopologyRepository(database)
	now := time.Now().UTC()
	lab := domain.Laboratory{ID: domain.NewID(), Name: "switch update", Revision: 1, RecoveryPolicy: domain.RecoveryRemainStopped, LifecycleState: "active", CreatedAt: now, UpdatedAt: now}
	if err = topology.CreateLaboratory(ctx, lab); err != nil {
		t.Fatal(err)
	}
	runtime := &networkTaskRuntimeFake{configured: map[domain.ID]bool{}, values: map[domain.ID]domain.NetworkObject{}}
	runner := task.NewRunner(repositories, 1, 8)
	defer runner.Close()
	service := NewNetworkObjectTaskService(NewNetworkObjectService(repositories, NetworkRuntimeDispatch{SwitchL2: runtime}), runner)
	created, createTask, err := service.Create(ctx, lab.ID, "switch", domain.NetworkSwitchL2, map[string]any{"vlan_filtering": true, "ports": []any{map[string]any{"name": "eth0", "pvid": 1, "tagged": []any{}}}}, "create-switch")
	if err != nil {
		t.Fatal(err)
	}
	waitForNetworkTask(t, repositories, createTask.ID, func(value domain.OperationTask) bool { return value.State == domain.TaskSucceeded })
	current, err := repositories.GetNetworkObject(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	config := map[string]any{"vlan_filtering": true, "ports": []any{map[string]any{"name": "lan0", "pvid": 10, "tagged": []any{20}}}}
	predicted, updateTask, err := service.Update(ctx, current.ID, current.Revision, "configured-switch", config, "update-switch")
	if err != nil {
		t.Fatal(err)
	}
	if predicted.Revision != current.Revision+1 || predicted.ObservedState != "provisioning" {
		t.Fatalf("predicted=%+v current=%+v", predicted, current)
	}
	waitForNetworkTask(t, repositories, updateTask.ID, func(value domain.OperationTask) bool { return value.State == domain.TaskSucceeded })
	updated, err := repositories.GetNetworkObject(ctx, current.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "configured-switch" || updated.Revision != current.Revision+1 || updated.ObservedState != "active" {
		t.Fatalf("updated=%+v", updated)
	}
	runtime.mu.Lock()
	applied := runtime.values[current.ID]
	runtime.mu.Unlock()
	if applied.Name != "configured-switch" || applied.Config["ports"] == nil {
		t.Fatalf("applied=%+v", applied)
	}
}

func (r *networkTaskRuntimeFake) Delete(_ context.Context, id domain.ID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.configured, id)
	return nil
}

func newNetworkTaskFixture(t *testing.T, runtime NetworkObjectRuntime) (context.Context, *storesqlite.Database, *storesqlite.Repositories, *NetworkObjectTaskService, *task.Runner, domain.Laboratory) {
	t.Helper()
	ctx := context.Background()
	database, err := storesqlite.Open(ctx, "file:"+string(domain.NewID())+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	repositories := storesqlite.NewRepositories(database)
	topology := storesqlite.NewTopologyRepository(database)
	now := time.Now().UTC()
	lab := domain.Laboratory{ID: domain.NewID(), Name: "network tasks", Revision: 1, RecoveryPolicy: domain.RecoveryRemainStopped, LifecycleState: "active", CreatedAt: now, UpdatedAt: now}
	if err = topology.CreateLaboratory(ctx, lab); err != nil {
		database.Close()
		t.Fatal(err)
	}
	runner := task.NewRunner(repositories, 1, 8)
	service := NewNetworkObjectService(repositories, NetworkRuntimeDispatch{Bridge: runtime})
	return ctx, database, repositories, NewNetworkObjectTaskService(service, runner), runner, lab
}

func waitForNetworkTask(t *testing.T, repository *storesqlite.Repositories, id domain.ID, condition func(domain.OperationTask) bool) domain.OperationTask {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		value, err := repository.GetTask(context.Background(), id)
		if err == nil && condition(value) {
			return value
		}
		time.Sleep(5 * time.Millisecond)
	}
	value, _ := repository.GetTask(context.Background(), id)
	t.Fatalf("task did not reach expected state: %+v", value)
	return domain.OperationTask{}
}

func TestNetworkObjectTaskIdempotencyAndRecovery(t *testing.T) {
	runtime := &networkTaskRuntimeFake{configured: map[domain.ID]bool{}}
	ctx, database, repositories, service, runner, lab := newNetworkTaskFixture(t, runtime)
	first, firstTask, err := service.Create(ctx, lab.ID, "bridge", domain.NetworkBridge, map[string]any{}, "network-key")
	if err != nil {
		t.Fatal(err)
	}
	second, secondTask, err := service.Create(ctx, lab.ID, "bridge", domain.NetworkBridge, map[string]any{}, "network-key")
	if err != nil || secondTask.ID != firstTask.ID || second.ID != first.ID {
		t.Fatalf("first=%+v/%+v second=%+v/%+v err=%v", first, firstTask, second, secondTask, err)
	}
	if _, _, err = service.Create(ctx, lab.ID, "different", domain.NetworkBridge, map[string]any{}, "network-key"); err == nil {
		t.Fatal("expected idempotency conflict")
	}
	waitForNetworkTask(t, repositories, firstTask.ID, func(value domain.OperationTask) bool { return value.State == domain.TaskSucceeded })
	runner.Close()

	current, err := repositories.GetTask(ctx, firstTask.ID)
	if err != nil {
		t.Fatal(err)
	}
	current.State = domain.TaskRunning
	current.FinishedAt = nil
	current.Result = nil
	if err = repositories.UpdateTask(ctx, current); err != nil {
		t.Fatal(err)
	}
	recoveryRunner := task.NewRunner(repositories, 1, 8)
	defer recoveryRunner.Close()
	NewNetworkObjectTaskService(NewNetworkObjectService(repositories, NetworkRuntimeDispatch{Bridge: runtime}), recoveryRunner)
	if err = recoveryRunner.Recover(ctx); err != nil {
		t.Fatal(err)
	}
	waitForNetworkTask(t, repositories, firstTask.ID, func(value domain.OperationTask) bool { return value.State == domain.TaskSucceeded })
	database.Close()
}

func TestNetworkObjectCreateCancellationCompensatesOwnedObject(t *testing.T) {
	runtime := &networkTaskRuntimeFake{configured: map[domain.ID]bool{}, blockConfigure: true}
	ctx, database, repositories, service, runner, lab := newNetworkTaskFixture(t, runtime)
	defer database.Close()
	defer runner.Close()
	object, value, err := service.Create(ctx, lab.ID, "bridge", domain.NetworkBridge, map[string]any{}, "cancel-network")
	if err != nil {
		t.Fatal(err)
	}
	waitForNetworkTask(t, repositories, value.ID, func(value domain.OperationTask) bool {
		return value.State == domain.TaskRunning && value.ProgressCurrent == 1
	})
	if err = runner.Cancel(ctx, value.ID); err != nil {
		t.Fatal(err)
	}
	waitForNetworkTask(t, repositories, value.ID, func(value domain.OperationTask) bool { return value.State == domain.TaskCancelled })
	if _, err = repositories.GetNetworkObject(ctx, object.ID); err == nil {
		t.Fatal("cancelled create left network object behind")
	}
}
