package command_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/app/command"
	"github.com/netlab/netlab/internal/app/task"
	"github.com/netlab/netlab/internal/domain"
	storesqlite "github.com/netlab/netlab/internal/store/sqlite"
)

type reconnectRuntimeCall struct {
	link      domain.Link
	endpointA domain.Interface
	endpointB domain.Interface
}

type reconnectRuntimeFake struct {
	mu      sync.Mutex
	calls   []reconnectRuntimeCall
	called  chan struct{}
	handler func(context.Context, reconnectRuntimeCall, int) error
}

func newReconnectRuntimeFake(handler func(context.Context, reconnectRuntimeCall, int) error) *reconnectRuntimeFake {
	return &reconnectRuntimeFake{called: make(chan struct{}, 8), handler: handler}
}

func (f *reconnectRuntimeFake) EnsureLink(ctx context.Context, link domain.Link, endpointA, endpointB domain.Interface) error {
	call := reconnectRuntimeCall{link: link, endpointA: endpointA, endpointB: endpointB}
	f.mu.Lock()
	f.calls = append(f.calls, call)
	index := len(f.calls) - 1
	f.mu.Unlock()
	f.called <- struct{}{}
	if f.handler == nil {
		return nil
	}
	return f.handler(ctx, call, index)
}

func (f *reconnectRuntimeFake) call(index int) reconnectRuntimeCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls[index]
}

func TestLinkReconnectKeepsOriginalEndpointsAuthoritativeUntilRuntimeSuccess(t *testing.T) {
	fixture := newReconnectFixture(t, "link-reconnect-staged-success")
	blocked := make(chan struct{})
	runtime := newReconnectRuntimeFake(func(ctx context.Context, _ reconnectRuntimeCall, index int) error {
		if index == 0 {
			select {
			case <-blocked:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	})
	fixture.service.SetRuntime(runtime)

	value, err := fixture.service.Reconnect(context.Background(), fixture.link.ID, fixture.link.Revision, fixture.left.ID, fixture.replacement.ID, "staged-success")
	if err != nil {
		t.Fatal(err)
	}
	waitRuntimeCall(t, runtime)
	current, err := fixture.topology.GetLink(context.Background(), fixture.link.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.EndpointAID != fixture.left.ID || current.EndpointBID != fixture.right.ID || current.Revision != fixture.link.Revision {
		t.Fatalf("canonical link changed before runtime success: %+v", current)
	}
	candidate := runtime.call(0)
	if candidate.link.EndpointAID != fixture.left.ID || candidate.link.EndpointBID != fixture.replacement.ID {
		t.Fatalf("unexpected staged candidate: %+v", candidate.link)
	}
	close(blocked)
	waitTask(t, fixture.repositories, value.ID, domain.TaskSucceeded)
	committed := waitLink(t, fixture.topology, fixture.link.ID, func(link domain.Link) bool {
		return link.EndpointAID == fixture.left.ID && link.EndpointBID == fixture.replacement.ID
	})
	if committed.ObservedState != "connected" {
		t.Fatalf("committed link observed state = %q", committed.ObservedState)
	}
}

func TestLinkReconnectFailureKeepsDatabaseAndReEnsuresOriginalRuntime(t *testing.T) {
	fixture := newReconnectFixture(t, "link-reconnect-staged-failure")
	runtimeFailure := errors.New("replacement runtime failed")
	runtime := newReconnectRuntimeFake(func(_ context.Context, _ reconnectRuntimeCall, index int) error {
		if index == 0 {
			return runtimeFailure
		}
		return nil
	})
	fixture.service.SetRuntime(runtime)

	value, err := fixture.service.Reconnect(context.Background(), fixture.link.ID, fixture.link.Revision, fixture.left.ID, fixture.replacement.ID, "staged-failure")
	if err != nil {
		t.Fatal(err)
	}
	waitTask(t, fixture.repositories, value.ID, domain.TaskFailed)
	waitRuntimeCalls(t, runtime, 2)
	current, err := fixture.topology.GetLink(context.Background(), fixture.link.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.EndpointAID != fixture.left.ID || current.EndpointBID != fixture.right.ID || current.Revision != fixture.link.Revision {
		t.Fatalf("canonical link changed after runtime failure: %+v", current)
	}
	rollback := runtime.call(1)
	if rollback.link.EndpointAID != fixture.left.ID || rollback.link.EndpointBID != fixture.right.ID {
		t.Fatalf("original runtime was not re-ensured: %+v", rollback.link)
	}
}

func TestLinkReconnectCancellationKeepsDatabaseAndReEnsuresOriginalRuntime(t *testing.T) {
	fixture := newReconnectFixture(t, "link-reconnect-staged-cancel")
	runtime := newReconnectRuntimeFake(func(ctx context.Context, _ reconnectRuntimeCall, index int) error {
		if index == 0 {
			<-ctx.Done()
			return ctx.Err()
		}
		return nil
	})
	fixture.service.SetRuntime(runtime)

	value, err := fixture.service.Reconnect(context.Background(), fixture.link.ID, fixture.link.Revision, fixture.left.ID, fixture.replacement.ID, "staged-cancel")
	if err != nil {
		t.Fatal(err)
	}
	waitRuntimeCall(t, runtime)
	if err = fixture.runner.Cancel(context.Background(), value.ID); err != nil {
		t.Fatal(err)
	}
	waitTask(t, fixture.repositories, value.ID, domain.TaskCancelled)
	waitRuntimeCall(t, runtime)
	current, err := fixture.topology.GetLink(context.Background(), fixture.link.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.EndpointAID != fixture.left.ID || current.EndpointBID != fixture.right.ID || current.Revision != fixture.link.Revision {
		t.Fatalf("canonical link changed after cancellation: %+v", current)
	}
}

type reconnectFixture struct {
	topology     *storesqlite.TopologyRepository
	repositories *storesqlite.Repositories
	runner       *task.Runner
	service      *command.LinkReconnectService
	link         domain.Link
	left         domain.Interface
	right        domain.Interface
	replacement  domain.Interface
}

func newReconnectFixture(t *testing.T, databaseName string) reconnectFixture {
	t.Helper()
	ctx := context.Background()
	database, err := storesqlite.Open(ctx, "file:"+databaseName+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	topology := storesqlite.NewTopologyRepository(database)
	repositories := storesqlite.NewRepositories(database)
	lab, err := command.NewLaboratoryService(topology).Create(ctx, "reconnect", "", domain.RecoveryAutoRestore)
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
	runner := task.NewRunner(repositories, 1, 8)
	t.Cleanup(runner.Close)
	return reconnectFixture{
		topology: topology, repositories: repositories, runner: runner,
		service: command.NewLinkReconnectService(topology, runner), link: link,
		left: left[0], right: right[0], replacement: replacement[0],
	}
}

func waitRuntimeCall(t *testing.T, runtime *reconnectRuntimeFake) {
	t.Helper()
	select {
	case <-runtime.called:
	case <-time.After(3 * time.Second):
		t.Fatal("runtime call did not start")
	}
}

func waitRuntimeCalls(t *testing.T, runtime *reconnectRuntimeFake, count int) {
	t.Helper()
	for index := 0; index < count; index++ {
		waitRuntimeCall(t, runtime)
	}
}

func waitTask(t *testing.T, repository *storesqlite.Repositories, id domain.ID, state domain.TaskState) domain.OperationTask {
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

func waitLink(t *testing.T, repository *storesqlite.TopologyRepository, id domain.ID, predicate func(domain.Link) bool) domain.Link {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		value, err := repository.GetLink(context.Background(), id)
		if err == nil && predicate(value) {
			return value
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("link %s did not reach expected state", id)
	return domain.Link{}
}
