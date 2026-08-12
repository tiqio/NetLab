package reconcile

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/domain"
	storesqlite "github.com/netlab/netlab/internal/store/sqlite"
)

func TestNetworkRecoveryPreservesLegacySinglePortConfiguration(t *testing.T) {
	ctx := context.Background()
	database, err := storesqlite.Open(ctx, "file:"+string(domain.NewID())+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	repositories := storesqlite.NewRepositories(database)
	topology := storesqlite.NewTopologyRepository(database)
	now := time.Now().UTC()
	lab := domain.Laboratory{ID: domain.NewID(), Name: "legacy", Revision: 1, RecoveryPolicy: domain.RecoveryAutoRestore, LifecycleState: "active", CreatedAt: now, UpdatedAt: now}
	if err = topology.CreateLaboratory(ctx, lab); err != nil {
		t.Fatal(err)
	}
	object := domain.NetworkObject{ID: domain.NewID(), LaboratoryID: lab.ID, Name: "legacy", Kind: domain.NetworkSwitchL2, Revision: 1, DesiredState: "active", ObservedState: "pending", Config: map[string]any{"ports": []any{map[string]any{"name": "lan0"}}}, CreatedAt: now, UpdatedAt: now}
	if err = repositories.CreateNetworkObject(ctx, object); err != nil {
		t.Fatal(err)
	}
	runtime := &networkTaskRuntimeFake{configured: map[domain.ID]bool{}, values: map[domain.ID]domain.NetworkObject{}}
	service := NewNetworkObjectService(repositories, NetworkRuntimeDispatch{SwitchL2: runtime})
	if err = service.RestoreLaboratory(ctx, lab.ID); err != nil {
		t.Fatal(err)
	}
	ports := runtime.values[object.ID].Config["ports"].([]any)
	if len(ports) != 1 || ports[0].(map[string]any)["name"] != "lan0" {
		t.Fatalf("recovery expanded legacy ports: %+v", runtime.values[object.ID].Config)
	}
}

type pendingRecoveryRuntime struct {
	configured bool
}

func (r *pendingRecoveryRuntime) Configure(context.Context, domain.NetworkObject) error {
	r.configured = true
	return nil
}

func (*pendingRecoveryRuntime) Delete(context.Context, domain.ID) error { return nil }

func (*pendingRecoveryRuntime) InspectNetworkObject(context.Context, domain.NetworkObject) (domain.RuntimeBackingObservation, error) {
	return domain.RuntimeBackingObservation{Kind: "namespace", RuntimeName: "router", Owned: true, Usable: true}, nil
}

func (*pendingRecoveryRuntime) ConfigurationConverged(context.Context, domain.NetworkObject) (bool, map[string]any, error) {
	return false, map[string]any{"mismatches": []string{"eth0 unavailable"}}, nil
}

func TestNetworkRecoveryKeepsUsableButUnconvergedObjectPending(t *testing.T) {
	ctx := context.Background()
	database, err := storesqlite.Open(ctx, "file:"+string(domain.NewID())+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	repositories := storesqlite.NewRepositories(database)
	topology := storesqlite.NewTopologyRepository(database)
	now := time.Now().UTC()
	lab := domain.Laboratory{ID: domain.NewID(), Name: "pending recovery", Revision: 1, RecoveryPolicy: domain.RecoveryAutoRestore, LifecycleState: "active", CreatedAt: now, UpdatedAt: now}
	if err = topology.CreateLaboratory(ctx, lab); err != nil {
		t.Fatal(err)
	}
	object := domain.NetworkObject{ID: domain.NewID(), LaboratoryID: lab.ID, Name: "router", Kind: domain.NetworkSwitchL3, Revision: 1, DesiredState: "active", ObservedState: "active", Config: map[string]any{"interfaces": []any{map[string]any{"name": "eth0", "addresses": []any{"192.0.2.1/24"}}}}, CreatedAt: now, UpdatedAt: now}
	if err = repositories.CreateNetworkObject(ctx, object); err != nil {
		t.Fatal(err)
	}
	runtime := &pendingRecoveryRuntime{}
	service := NewNetworkObjectService(repositories, NetworkRuntimeDispatch{SwitchL3: runtime})
	var outcome RecoveryResourceOutcome
	if err = service.RestoreLaboratoryWithCheckpoints(ctx, lab.ID, func(value RecoveryResourceOutcome) error {
		outcome = value
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	restored, err := repositories.GetNetworkObject(ctx, object.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !runtime.configured || restored.ObservedState != "pending" {
		t.Fatalf("runtime=%+v object=%+v", runtime, restored)
	}
	if outcome.State != "pending" || outcome.ResourceID != object.ID || !strings.Contains(outcome.Error, "waiting for runtime ports") {
		t.Fatalf("outcome=%+v", outcome)
	}
}

type recoveryTaskStoreFake struct {
	created   []domain.OperationTask
	updated   []domain.OperationTask
	recovered []domain.ID
}

func (s *recoveryTaskStoreFake) PublishNetworkObjectLinkRecovered(_ context.Context, id, _ domain.ID) error {
	s.recovered = append(s.recovered, id)
	return nil
}

func (s *recoveryTaskStoreFake) CreateTask(_ context.Context, task domain.OperationTask) error {
	s.created = append(s.created, task)
	return nil
}

func (s *recoveryTaskStoreFake) UpdateTask(_ context.Context, task domain.OperationTask) error {
	s.updated = append(s.updated, task)
	return nil
}

type recoveryParticipantFake struct {
	name  string
	err   error
	runs  int
	order *[]string
}

type durableTaskRecovererFake struct{ calls int }

func (r *durableTaskRecovererFake) Recover(context.Context) error {
	r.calls++
	return nil
}

func TestDurableTaskRecoveryParticipantRunsInsideStartupRecovery(t *testing.T) {
	store := &recoveryTaskStoreFake{}
	recoverer := &durableTaskRecovererFake{}
	participant := NewDurableTaskRecoveryReconciler(recoverer)
	if _, err := NewRecoveryCoordinator(store, participant).Execute(context.Background(), "service_restart", nil); err != nil {
		t.Fatal(err)
	}
	if recoverer.calls != 1 {
		t.Fatalf("recover calls=%d", recoverer.calls)
	}
}

type checkpointParticipantFake struct{ recoveryParticipantFake }

func (p *checkpointParticipantFake) ReconcileWithCheckpoints(_ context.Context, checkpoint func(RecoveryResourceOutcome) error) error {
	p.runs++
	if err := checkpoint(RecoveryResourceOutcome{ResourceType: "node", ResourceID: "node-1", State: "recovered", RuntimeID: "pid-42", Details: map[string]string{"kind": "qemu"}}); err != nil {
		return err
	}
	return p.err
}

type objectLinkCheckpointParticipant struct{ recoveryParticipantFake }

func (p *objectLinkCheckpointParticipant) ReconcileWithCheckpoints(_ context.Context, checkpoint func(RecoveryResourceOutcome) error) error {
	p.runs++
	return checkpoint(RecoveryResourceOutcome{ResourceType: "network_object_link", ResourceID: "object-link-1", State: "recovered"})
}

func (p *recoveryParticipantFake) Name() string { return p.name }
func (p *recoveryParticipantFake) Reconcile(context.Context) error {
	p.runs++
	if p.order != nil {
		*p.order = append(*p.order, p.name)
	}
	return p.err
}

func TestStartupRecoveryCoordinatorOrdersBackingBeforeConnections(t *testing.T) {
	store := &recoveryTaskStoreFake{}
	var order []string
	participant := func(name string) *recoveryParticipantFake {
		return &recoveryParticipantFake{name: name, order: &order}
	}
	coordinator := NewStartupRecoveryCoordinator(store, StartupRecoveryParticipants{
		Captures:       participant("captures"),
		DataPlane:      participant("data-plane"),
		Reservations:   participant("reservations"),
		DurableTasks:   participant("durable-tasks"),
		NetworkObjects: participant("network-objects"),
		Nodes:          participant("nodes"),
		PortMappings:   participant("port-mappings"),
	})
	if _, err := coordinator.Execute(context.Background(), "service_restart", nil); err != nil {
		t.Fatal(err)
	}
	want := []string{"nodes", "network-objects", "durable-tasks", "reservations", "data-plane", "port-mappings", "captures"}
	if strings.Join(order, ",") != strings.Join(want, ",") {
		t.Fatalf("order=%v want=%v", order, want)
	}
}

func TestRecoveryCoordinatorPublishesProgressAndCompletion(t *testing.T) {
	store := &recoveryTaskStoreFake{}
	first := &recoveryParticipantFake{name: "nodes"}
	second := &recoveryParticipantFake{name: "links"}
	prepared := false
	task, err := NewRecoveryCoordinator(store, first, second).Execute(context.Background(), "host_restart", func(context.Context) error {
		prepared = true
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !prepared || first.runs != 1 || second.runs != 1 {
		t.Fatalf("prepared=%t first=%d second=%d", prepared, first.runs, second.runs)
	}
	if task.State != domain.TaskSucceeded || task.ProgressCurrent != 3 || task.ProgressTotal != 3 {
		t.Fatalf("task=%+v", task)
	}
	if len(store.created) != 1 || len(store.updated) != 4 {
		t.Fatalf("created=%d updated=%d", len(store.created), len(store.updated))
	}
}

func TestRecoveryCoordinatorContinuesWithActionableParticipantFailure(t *testing.T) {
	store := &recoveryTaskStoreFake{}
	failed := &recoveryParticipantFake{name: "port-mappings", err: errors.New("nft unavailable")}
	next := &recoveryParticipantFake{name: "data-plane"}
	task, err := NewRecoveryCoordinator(store, failed, next).Execute(context.Background(), "service_restart", nil)
	if err == nil || !strings.Contains(err.Error(), "port-mappings") {
		t.Fatalf("err=%v", err)
	}
	if next.runs != 1 || task.State != domain.TaskFailed || task.Error == nil || !strings.Contains(task.Error.Message, "nft unavailable") {
		t.Fatalf("next=%d task=%+v", next.runs, task)
	}
	if task.Error.TaskID != task.ID || task.Error.ResourceType != "host" || task.Error.Phase != "recovery" || task.Error.Cleanup == "" || task.Error.OperatorHint == "" || task.Error.RetryAfterSeconds == 0 {
		t.Fatalf("unstructured recovery error: %+v", task.Error)
	}
}

func TestRecoveryCoordinatorPersistsResourceCheckpoints(t *testing.T) {
	store := &recoveryTaskStoreFake{}
	participant := &checkpointParticipantFake{recoveryParticipantFake: recoveryParticipantFake{name: "nodes"}}
	task, err := NewRecoveryCoordinator(store, participant).Execute(context.Background(), "service_restart", nil)
	if err != nil {
		t.Fatal(err)
	}
	outcomes, ok := task.Result["resource_outcomes"].([]map[string]any)
	if !ok || len(outcomes) != 2 || outcomes[0]["resource_id"] != domain.ID("node-1") || outcomes[0]["runtime_id"] != "pid-42" {
		t.Fatalf("outcomes=%#v", task.Result["resource_outcomes"])
	}
	if len(store.updated) < 3 {
		t.Fatalf("checkpoint was not durably updated: %d", len(store.updated))
	}
}

func TestRecoveryCoordinatorPublishesRecoveredObjectLinkBeforeTaskCheckpoint(t *testing.T) {
	store := &recoveryTaskStoreFake{}
	participant := &objectLinkCheckpointParticipant{recoveryParticipantFake: recoveryParticipantFake{name: "data-plane"}}
	if _, err := NewRecoveryCoordinator(store, participant).Execute(context.Background(), "service_restart", nil); err != nil {
		t.Fatal(err)
	}
	if len(store.recovered) != 1 || store.recovered[0] != "object-link-1" {
		t.Fatalf("recovered=%v", store.recovered)
	}
}
