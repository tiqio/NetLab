package reconcile

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/app/command"
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

type objectLinkDeleteRuntimeFake struct {
	calls *[]string
}

func (r objectLinkDeleteRuntimeFake) DeleteNetworkObjectLink(context.Context, domain.NetworkObjectLink, domain.NetworkObject, domain.NetworkObject) error {
	*r.calls = append(*r.calls, "runtime")
	return nil
}

type objectLinkObserverCleanupFake struct {
	calls *[]string
}

func (c objectLinkObserverCleanupFake) StopNetworkObjectLink(domain.ID) {
	*c.calls = append(*c.calls, "observer")
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

func TestNetworkObjectReconcileIsRevisionedAndDurable(t *testing.T) {
	runtime := &networkTaskRuntimeFake{configured: map[domain.ID]bool{}}
	ctx, database, repositories, operations, runner, lab := newNetworkTaskFixture(t, runtime)
	defer database.Close()
	defer runner.Close()
	created, createTask, err := operations.Create(ctx, lab.ID, "bridge", domain.NetworkBridge, map[string]any{"mtu": 1500, "stp": true}, "reconcile-create")
	if err != nil {
		t.Fatal(err)
	}
	waitForNetworkTask(t, repositories, createTask.ID, func(value domain.OperationTask) bool { return value.State == domain.TaskSucceeded })
	current, err := repositories.GetNetworkObject(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = operations.ReconcileObject(ctx, current.ID, current.Revision+1, "wrong-revision"); domain.NormalizeProblem(err, domain.Problem{}).Code != "revision_conflict" {
		t.Fatalf("err=%v", err)
	}
	_, reconcileTask, err := operations.ReconcileObject(ctx, current.ID, current.Revision, "reconcile-object")
	if err != nil {
		t.Fatal(err)
	}
	waitForNetworkTask(t, repositories, reconcileTask.ID, func(value domain.OperationTask) bool { return value.State == domain.TaskSucceeded })
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

func TestNetworkObjectCreateValidatesExplicitL3ConfigurationBeforePersistence(t *testing.T) {
	runtime := &networkTaskRuntimeFake{configured: map[domain.ID]bool{}}
	ctx, database, repositories, service, runner, lab := newNetworkTaskFixture(t, runtime)
	defer database.Close()
	defer runner.Close()

	_, _, err := service.Create(ctx, lab.ID, "invalid-l3", domain.NetworkSwitchL3, map[string]any{
		"interfaces": []any{map[string]any{"name": "eth0", "addresses": []any{"not-cidr"}}},
	}, "invalid-l3")
	if err == nil {
		t.Fatal("expected invalid L3 configuration to be rejected")
	}
	objects, listErr := repositories.ListNetworkObjects(ctx, lab.ID)
	if listErr != nil {
		t.Fatal(listErr)
	}
	if len(objects) != 0 {
		t.Fatalf("invalid L3 configuration persisted objects: %+v", objects)
	}
}

func TestNetworkAttachmentDeleteIdempotencyIncludesRevision(t *testing.T) {
	runtime := &networkTaskRuntimeFake{configured: map[domain.ID]bool{}}
	ctx, database, repositories, service, runner, lab := newNetworkTaskFixture(t, runtime)
	defer database.Close()
	defer runner.Close()
	topology := storesqlite.NewTopologyRepository(database)
	_, interfaces, err := command.NewNodeService(topology).CreateConfigured(ctx, lab.ID, command.CreateNodeRequest{Name: "node", Kind: "docker", InterfaceCount: 1, InterfaceLimit: 4})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	object := domain.NetworkObject{ID: domain.NewID(), LaboratoryID: lab.ID, Name: "bridge", Kind: domain.NetworkBridge, Revision: 1, DesiredState: "active", ObservedState: "active", Config: map[string]any{}, CreatedAt: now, UpdatedAt: now}
	if err = repositories.CreateNetworkObject(ctx, object); err != nil {
		t.Fatal(err)
	}
	attachment, err := repositories.CreateTopologyNetworkAttachment(ctx, object.ID, interfaces[0].ID, "eth0", nil, "create-attachment")
	if err != nil {
		t.Fatal(err)
	}
	_, deleteTask, err := service.DeleteAttachment(ctx, attachment.ID, attachment.Revision, "delete-attachment")
	if err != nil {
		t.Fatal(err)
	}
	waitForNetworkTask(t, repositories, deleteTask.ID, func(value domain.OperationTask) bool { return value.State == domain.TaskSucceeded })
	if _, _, err = service.DeleteAttachment(ctx, attachment.ID, attachment.Revision+1, "delete-attachment"); err == nil {
		t.Fatal("expected idempotency conflict for different attachment revision")
	} else if problem, ok := domain.ProblemFromError(err); !ok || problem.Code != "idempotency_conflict" {
		t.Fatalf("err=%v", err)
	}
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

func TestNetworkObjectLinkCreateIsDurableIdempotentAndObservable(t *testing.T) {
	ctx, database, repositories, operations, runner, lab := newNetworkTaskFixture(t, &networkTaskRuntimeFake{configured: map[domain.ID]bool{}})
	defer database.Close()
	defer runner.Close()
	now := time.Now().UTC()
	for _, object := range []domain.NetworkObject{{ID: "switch-a", LaboratoryID: lab.ID, Name: "A", Kind: domain.NetworkSwitchL2, Revision: 1, DesiredState: "active", ObservedState: "active", Config: map[string]any{}, CreatedAt: now, UpdatedAt: now}, {ID: "switch-b", LaboratoryID: lab.ID, Name: "B", Kind: domain.NetworkSwitchL2, Revision: 1, DesiredState: "active", ObservedState: "active", Config: map[string]any{}, CreatedAt: now, UpdatedAt: now}} {
		if err := repositories.CreateNetworkObject(ctx, object); err != nil {
			t.Fatal(err)
		}
	}
	entryContext := command.WithTopologyConnectionEntryPoint(ctx, "mcp")
	predicted, queued, err := operations.CreateObjectLink(entryContext, lab.ID, "switch-a", "swp1", "switch-b", "swp1", "link-create-key")
	if err != nil {
		t.Fatal(err)
	}
	if queued.Input["entry_point"] != "mcp" {
		t.Fatalf("task input=%+v", queued.Input)
	}
	replayed, replayedTask, err := operations.CreateObjectLink(entryContext, lab.ID, "switch-a", "swp1", "switch-b", "swp1", "link-create-key")
	if err != nil {
		t.Fatal(err)
	}
	if replayed.ID != predicted.ID || replayedTask.ID != queued.ID {
		t.Fatalf("predicted=%+v replayed=%+v tasks=%s/%s", predicted, replayed, queued.ID, replayedTask.ID)
	}
	completed := waitForNetworkTask(t, repositories, queued.ID, func(value domain.OperationTask) bool { return value.State == domain.TaskSucceeded })
	if completed.ResourceType != "network_object_link" || completed.ResourceID != predicted.ID {
		t.Fatalf("task=%+v", completed)
	}
	stored, err := operations.service.GetObjectLink(ctx, predicted.ID)
	if err != nil || stored.Revision != 1 || stored.PortAName != "swp1" {
		t.Fatalf("stored=%+v err=%v", stored, err)
	}
	listed, err := operations.service.ListObjectLinks(ctx, lab.ID)
	if err != nil || len(listed) != 1 {
		t.Fatalf("listed=%+v err=%v", listed, err)
	}
	var createdSequence, finalTaskSequence int64
	if err = database.DB.QueryRowContext(ctx, `SELECT sequence FROM outbox_events WHERE event_type='network_object_link.created' AND resource_id=?`, predicted.ID).Scan(&createdSequence); err != nil {
		t.Fatal(err)
	}
	if err = database.DB.QueryRowContext(ctx, `SELECT MAX(sequence) FROM outbox_events WHERE event_type='task.updated' AND resource_id=?`, queued.ID).Scan(&finalTaskSequence); err != nil {
		t.Fatal(err)
	}
	if createdSequence <= 0 || finalTaskSequence <= createdSequence {
		t.Fatalf("created=%d final_task=%d", createdSequence, finalTaskSequence)
	}
	if _, _, err = operations.CreateObjectLink(ctx, lab.ID, "switch-a", "swp1", "switch-b", "swp2", "occupied-key"); err == nil {
		t.Fatal("expected occupied port conflict")
	}
}

func TestNetworkObjectLinkDeleteIsRevisionedIdempotentOrderedAndReleasesPorts(t *testing.T) {
	ctx, database, repositories, operations, runner, lab := newNetworkTaskFixture(t, &networkTaskRuntimeFake{configured: map[domain.ID]bool{}})
	defer database.Close()
	defer runner.Close()
	now := time.Now().UTC()
	for _, object := range []domain.NetworkObject{{ID: "delete-a", LaboratoryID: lab.ID, Name: "A", Kind: domain.NetworkSwitchL2, Revision: 1, DesiredState: "active", ObservedState: "active", Config: map[string]any{}, CreatedAt: now, UpdatedAt: now}, {ID: "delete-b", LaboratoryID: lab.ID, Name: "B", Kind: domain.NetworkSwitchL2, Revision: 1, DesiredState: "active", ObservedState: "active", Config: map[string]any{}, CreatedAt: now, UpdatedAt: now}} {
		if err := repositories.CreateNetworkObject(ctx, object); err != nil {
			t.Fatal(err)
		}
	}
	link, createTask, err := operations.CreateObjectLink(ctx, lab.ID, "delete-a", "swp1", "delete-b", "swp1", "delete-create")
	if err != nil {
		t.Fatal(err)
	}
	waitForNetworkTask(t, repositories, createTask.ID, func(value domain.OperationTask) bool { return value.State == domain.TaskSucceeded })
	if _, _, err = operations.DeleteObjectLink(ctx, link.ID, 2, "delete-key"); err == nil {
		t.Fatal("expected revision conflict")
	}
	calls := []string{}
	operations.service.SetObjectLinkRuntime(objectLinkDeleteRuntimeFake{calls: &calls})
	operations.service.AddObjectLinkObserverCleanup(objectLinkObserverCleanupFake{calls: &calls})
	predicted, queued, err := operations.DeleteObjectLink(ctx, link.ID, 1, "delete-key")
	if err != nil || predicted.ObservedState != "disconnecting" || predicted.DesiredState != "disconnected" {
		t.Fatalf("predicted=%+v task=%+v err=%v", predicted, queued, err)
	}
	completed := waitForNetworkTask(t, repositories, queued.ID, func(value domain.OperationTask) bool { return value.State == domain.TaskSucceeded })
	if len(calls) != 2 || calls[0] != "observer" || calls[1] != "runtime" {
		t.Fatalf("cleanup order=%v", calls)
	}
	if _, err = repositories.GetNetworkObjectLink(ctx, link.ID); err == nil {
		t.Fatal("deleted link still exists")
	}
	replayed, replayedTask, err := operations.DeleteObjectLink(ctx, link.ID, 1, "delete-key")
	if err != nil || replayedTask.ID != queued.ID || replayed.ID != link.ID {
		t.Fatalf("replayed=%+v task=%+v err=%v", replayed, replayedTask, err)
	}
	reused, reuseTask, err := operations.CreateObjectLink(ctx, lab.ID, "delete-a", "swp1", "delete-b", "swp1", "reuse-ports")
	if err != nil {
		t.Fatal(err)
	}
	waitForNetworkTask(t, repositories, reuseTask.ID, func(value domain.OperationTask) bool { return value.State == domain.TaskSucceeded })
	if reused.ID == link.ID {
		t.Fatal("port reuse unexpectedly resurrected deleted link identity")
	}
	var disconnectingSequence, deletedSequence, finalTaskSequence int64
	if err = database.DB.QueryRowContext(ctx, `SELECT MIN(sequence) FROM outbox_events WHERE event_type='network_object_link.state_changed' AND resource_id=?`, link.ID).Scan(&disconnectingSequence); err != nil {
		t.Fatal(err)
	}
	if err = database.DB.QueryRowContext(ctx, `SELECT sequence FROM outbox_events WHERE event_type='network_object_link.deleted' AND resource_id=? AND task_id=?`, link.ID, queued.ID).Scan(&deletedSequence); err != nil {
		t.Fatal(err)
	}
	if err = database.DB.QueryRowContext(ctx, `SELECT MAX(sequence) FROM outbox_events WHERE event_type='task.updated' AND resource_id=?`, completed.ID).Scan(&finalTaskSequence); err != nil {
		t.Fatal(err)
	}
	if !(disconnectingSequence < deletedSequence && deletedSequence < finalTaskSequence) {
		t.Fatalf("event order disconnecting=%d deleted=%d task=%d", disconnectingSequence, deletedSequence, finalTaskSequence)
	}
	cascadeTask, err := operations.Delete(ctx, "delete-a", 1, "cascade-object")
	if err != nil {
		t.Fatal(err)
	}
	waitForNetworkTask(t, repositories, cascadeTask.ID, func(value domain.OperationTask) bool { return value.State == domain.TaskSucceeded })
	if _, err = repositories.GetNetworkObjectLink(ctx, reused.ID); err == nil {
		t.Fatal("network object deletion left its object link behind")
	}
}
