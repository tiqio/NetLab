package recovery_test

import (
	"context"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/app/reconcile"
	"github.com/netlab/netlab/internal/app/task"
	"github.com/netlab/netlab/internal/domain"
	storesqlite "github.com/netlab/netlab/internal/store/sqlite"
)

type recoveredObjectLinkRuntime struct{ calls int }

func (r *recoveredObjectLinkRuntime) DeleteNetworkObjectLink(context.Context, domain.NetworkObjectLink, domain.NetworkObject, domain.NetworkObject) error {
	r.calls++
	return nil
}

func TestInterruptedNetworkObjectLinkDeleteResumesWithoutResurrection(t *testing.T) {
	ctx := context.Background()
	database, err := storesqlite.Open(ctx, "file:"+string(domain.NewID())+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	repositories := storesqlite.NewRepositories(database)
	topology := storesqlite.NewTopologyRepository(database)
	now := time.Now().UTC()
	laboratory := domain.Laboratory{ID: domain.NewID(), Name: "delete recovery", Revision: 1, RecoveryPolicy: domain.RecoveryRemainStopped, LifecycleState: "active", CreatedAt: now, UpdatedAt: now}
	if err = topology.CreateLaboratory(ctx, laboratory); err != nil {
		t.Fatal(err)
	}
	objectA := domain.NetworkObject{ID: domain.NewID(), LaboratoryID: laboratory.ID, Name: "a", Kind: domain.NetworkSwitchL2, Revision: 1, DesiredState: "active", ObservedState: "active", Config: map[string]any{}}
	objectB := domain.NetworkObject{ID: domain.NewID(), LaboratoryID: laboratory.ID, Name: "b", Kind: domain.NetworkSwitchL2, Revision: 1, DesiredState: "active", ObservedState: "active", Config: map[string]any{}}
	if err = repositories.CreateNetworkObject(ctx, objectA); err != nil {
		t.Fatal(err)
	}
	if err = repositories.CreateNetworkObject(ctx, objectB); err != nil {
		t.Fatal(err)
	}
	link := domain.NetworkObjectLink{ID: domain.NewID(), LaboratoryID: laboratory.ID, ObjectAID: objectA.ID, PortAName: "swp1", ObjectBID: objectB.ID, PortBName: "swp1", Revision: 1, DesiredState: "connected", ObservedState: "disconnecting"}
	if err = repositories.CreateNetworkObjectLink(ctx, link); err != nil {
		t.Fatal(err)
	}
	if err = repositories.SetNetworkObjectLinkState(ctx, link.ID, "disconnecting", nil); err != nil {
		t.Fatal(err)
	}
	operation := domain.OperationTask{ID: domain.NewID(), Kind: "network_object_link.delete", ResourceType: "network_object_link", ResourceID: link.ID, RequestedRevision: link.Revision, State: domain.TaskRunning, ProgressCurrent: 1, ProgressTotal: 2, CreatedAt: now, StartedAt: &now, Input: map[string]any{"revision": int64(link.Revision), "laboratory_id": laboratory.ID, "object_a_id": objectA.ID, "port_a_name": link.PortAName, "object_b_id": objectB.ID, "port_b_name": link.PortBName}}
	if err = repositories.CreateTask(ctx, operation); err != nil {
		t.Fatal(err)
	}
	runtime := &recoveredObjectLinkRuntime{}
	service := reconcile.NewNetworkObjectService(repositories, reconcile.NetworkRuntimeDispatch{})
	service.SetObjectLinkRuntime(runtime)
	runner := task.NewRunner(repositories, 1, 8)
	defer runner.Close()
	reconcile.NewNetworkObjectTaskService(service, runner)
	if err = runner.Recover(ctx); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		current, getErr := repositories.GetTask(ctx, operation.ID)
		if getErr == nil && current.State == domain.TaskSucceeded {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	current, err := repositories.GetTask(ctx, operation.ID)
	if err != nil || current.State != domain.TaskSucceeded {
		t.Fatalf("task=%+v err=%v", current, err)
	}
	if runtime.calls != 1 {
		t.Fatalf("delete calls=%d", runtime.calls)
	}
	if _, err = repositories.GetNetworkObjectLink(ctx, link.ID); err == nil {
		t.Fatal("link resurrected after recovered deletion")
	}
	var captureSequence, deleteSequence int64
	_ = captureSequence
	if err = database.DB.QueryRowContext(ctx, `SELECT sequence FROM outbox_events WHERE event_type='network_object_link.deleted' AND resource_id=? AND task_id=?`, link.ID, operation.ID).Scan(&deleteSequence); err != nil {
		t.Fatal(err)
	}
	if deleteSequence == 0 {
		t.Fatal("missing terminal deletion event")
	}
}
