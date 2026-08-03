package recovery

import (
	"context"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/app/command"
	"github.com/netlab/netlab/internal/app/task"
	"github.com/netlab/netlab/internal/domain"
	storesqlite "github.com/netlab/netlab/internal/store/sqlite"
)

type reconnectRecoveryRuntime struct {
	called  chan domain.Link
	release chan struct{}
}

func (r *reconnectRecoveryRuntime) EnsureLink(ctx context.Context, link domain.Link, _, _ domain.Interface) error {
	r.called <- link
	if r.release == nil {
		return nil
	}
	select {
	case <-r.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func TestInterruptedLinkReconnectRecoveryKeepsOriginalEndpointsUntilSuccess(t *testing.T) {
	fixture := newReconnectRecoveryFixture(t, "link-reconnect-recovery-staged")
	operation := fixture.operation("recover-staged")
	operation.State = domain.TaskRunning
	operation.ProgressCurrent = 0
	if err := fixture.repositories.CreateTask(context.Background(), operation); err != nil {
		t.Fatal(err)
	}
	runtime := &reconnectRecoveryRuntime{called: make(chan domain.Link, 2), release: make(chan struct{})}
	runner := task.NewRunner(fixture.repositories, 1, 8)
	t.Cleanup(runner.Close)
	service := command.NewLinkReconnectService(fixture.topology, runner)
	service.SetRuntime(runtime)
	if err := runner.Recover(context.Background()); err != nil {
		t.Fatal(err)
	}
	candidate := waitRecoveredRuntime(t, runtime)
	if candidate.EndpointAID != fixture.left.ID || candidate.EndpointBID != fixture.replacement.ID {
		t.Fatalf("candidate=%+v", candidate)
	}
	current, err := fixture.topology.GetLink(context.Background(), fixture.link.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.EndpointAID != fixture.left.ID || current.EndpointBID != fixture.right.ID {
		t.Fatalf("canonical endpoints changed during recovery: %+v", current)
	}
	close(runtime.release)
	waitRecoveredTask(t, fixture.repositories, operation.ID, domain.TaskSucceeded)
	committed, err := fixture.topology.GetLink(context.Background(), fixture.link.ID)
	if err != nil {
		t.Fatal(err)
	}
	if committed.EndpointAID != fixture.left.ID || committed.EndpointBID != fixture.replacement.ID {
		t.Fatalf("committed=%+v", committed)
	}
}

func TestLinkReconnectRecoveryIsIdempotentAfterEndpointCommit(t *testing.T) {
	fixture := newReconnectRecoveryFixture(t, "link-reconnect-recovery-committed")
	operation := fixture.operation("recover-committed")
	operation.State = domain.TaskRunning
	operation.ProgressCurrent = 1
	if err := fixture.repositories.CreateTask(context.Background(), operation); err != nil {
		t.Fatal(err)
	}
	committed, err := fixture.topology.CommitLinkReconnect(context.Background(), fixture.link.ID, fixture.link.Revision, fixture.left.ID, fixture.replacement.ID)
	if err != nil {
		t.Fatal(err)
	}
	runtime := &reconnectRecoveryRuntime{called: make(chan domain.Link, 1)}
	runner := task.NewRunner(fixture.repositories, 1, 8)
	t.Cleanup(runner.Close)
	service := command.NewLinkReconnectService(fixture.topology, runner)
	service.SetRuntime(runtime)
	if err = runner.Recover(context.Background()); err != nil {
		t.Fatal(err)
	}
	waitRecoveredTask(t, fixture.repositories, operation.ID, domain.TaskSucceeded)
	select {
	case call := <-runtime.called:
		t.Fatalf("committed reconnect unexpectedly replayed runtime: %+v", call)
	default:
	}
	current, err := fixture.topology.GetLink(context.Background(), fixture.link.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current != committed {
		t.Fatalf("current=%+v committed=%+v", current, committed)
	}
}

type reconnectRecoveryFixture struct {
	topology     *storesqlite.TopologyRepository
	repositories *storesqlite.Repositories
	link         domain.Link
	left         domain.Interface
	right        domain.Interface
	replacement  domain.Interface
}

func newReconnectRecoveryFixture(t *testing.T, name string) reconnectRecoveryFixture {
	t.Helper()
	ctx := context.Background()
	database, err := storesqlite.Open(ctx, "file:"+name+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	topology := storesqlite.NewTopologyRepository(database)
	repositories := storesqlite.NewRepositories(database)
	lab, err := command.NewLaboratoryService(topology).Create(ctx, "recovery", "", domain.RecoveryAutoRestore)
	if err != nil {
		t.Fatal(err)
	}
	nodes := command.NewNodeService(topology, nil)
	_, left, err := nodes.Create(ctx, lab.ID, "left", "pc", 1)
	if err != nil {
		t.Fatal(err)
	}
	_, right, err := nodes.Create(ctx, lab.ID, "right", "pc", 1)
	if err != nil {
		t.Fatal(err)
	}
	_, replacement, err := nodes.Create(ctx, lab.ID, "replacement", "pc", 1)
	if err != nil {
		t.Fatal(err)
	}
	link, err := command.NewLinkService(topology).Connect(ctx, lab.ID, left[0].ID, right[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	return reconnectRecoveryFixture{topology: topology, repositories: repositories, link: link, left: left[0], right: right[0], replacement: replacement[0]}
}

func (f reconnectRecoveryFixture) operation(idempotencyKey string) domain.OperationTask {
	return domain.OperationTask{
		ID: domain.NewID(), Kind: "link.reconnect", ResourceType: "link", ResourceID: f.link.ID,
		IdempotencyKey: idempotencyKey, RequestedRevision: f.link.Revision,
		State: domain.TaskQueued, ProgressTotal: 3, CreatedAt: time.Now().UTC(),
		Input: map[string]any{
			"revision": int64(f.link.Revision), "retained_endpoint_id": string(f.left.ID), "replacement_endpoint_id": string(f.replacement.ID),
			"previous_endpoint_a_id": string(f.left.ID), "previous_endpoint_b_id": string(f.right.ID), "cancellation_mode": "before_commit",
		},
	}
}

func waitRecoveredRuntime(t *testing.T, runtime *reconnectRecoveryRuntime) domain.Link {
	t.Helper()
	select {
	case link := <-runtime.called:
		return link
	case <-time.After(3 * time.Second):
		t.Fatal("recovered runtime did not execute")
		return domain.Link{}
	}
}

func waitRecoveredTask(t *testing.T, repository *storesqlite.Repositories, id domain.ID, state domain.TaskState) domain.OperationTask {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		value, err := repository.GetTask(context.Background(), id)
		if err == nil && value.State == state {
			return value
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("task %s did not reach %s", id, state)
	return domain.OperationTask{}
}
