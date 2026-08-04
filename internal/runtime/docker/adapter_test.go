package docker

import (
	"context"
	"errors"
	"io"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/moby/moby/api/types/container"
	dockerclient "github.com/moby/moby/client"
	"github.com/netlab/netlab/internal/domain"
)

type fakeEngine struct {
	items       []container.Summary
	inspection  container.InspectResponse
	inspections []container.InspectResponse
	created     dockerclient.ContainerCreateOptions
	startCalls  int
	stopCalls   int
	removeCalls int
	execOptions dockerclient.ExecCreateOptions
	attachConn  net.Conn
}

func (f *fakeEngine) ContainerCreate(_ context.Context, options dockerclient.ContainerCreateOptions) (dockerclient.ContainerCreateResult, error) {
	f.created = options
	return dockerclient.ContainerCreateResult{ID: "created-container"}, nil
}
func (f *fakeEngine) ContainerStart(context.Context, string, dockerclient.ContainerStartOptions) (dockerclient.ContainerStartResult, error) {
	f.startCalls++
	return dockerclient.ContainerStartResult{}, nil
}
func (f *fakeEngine) ContainerStop(context.Context, string, dockerclient.ContainerStopOptions) (dockerclient.ContainerStopResult, error) {
	f.stopCalls++
	return dockerclient.ContainerStopResult{}, nil
}
func (f *fakeEngine) ContainerRemove(context.Context, string, dockerclient.ContainerRemoveOptions) (dockerclient.ContainerRemoveResult, error) {
	f.removeCalls++
	return dockerclient.ContainerRemoveResult{}, nil
}
func (f *fakeEngine) ContainerInspect(context.Context, string, dockerclient.ContainerInspectOptions) (dockerclient.ContainerInspectResult, error) {
	if len(f.inspections) > 0 {
		inspection := f.inspections[0]
		f.inspections = f.inspections[1:]
		return dockerclient.ContainerInspectResult{Container: inspection}, nil
	}
	return dockerclient.ContainerInspectResult{Container: f.inspection}, nil
}
func (f *fakeEngine) ContainerList(context.Context, dockerclient.ContainerListOptions) (dockerclient.ContainerListResult, error) {
	return dockerclient.ContainerListResult{Items: f.items}, nil
}
func (f *fakeEngine) ExecCreate(_ context.Context, _ string, options dockerclient.ExecCreateOptions) (dockerclient.ExecCreateResult, error) {
	f.execOptions = options
	return dockerclient.ExecCreateResult{ID: "exec-1"}, nil
}
func (f *fakeEngine) ExecAttach(context.Context, string, dockerclient.ExecAttachOptions) (dockerclient.ExecAttachResult, error) {
	return dockerclient.ExecAttachResult{HijackedResponse: dockerclient.NewHijackedResponse(f.attachConn, "application/vnd.docker.raw-stream")}, nil
}

type fakeEndpoints struct {
	ensurePIDs   []int
	ensureErr    error
	cleanupCalls int
	cleanupErr   error
}

func (f *fakeEndpoints) Ensure(_ context.Context, _ domain.Node, pid int) error {
	f.ensurePIDs = append(f.ensurePIDs, pid)
	return f.ensureErr
}
func (f *fakeEndpoints) Cleanup(context.Context, domain.Node) error {
	f.cleanupCalls++
	return f.cleanupErr
}

func TestQuotaNormalizationAndName(t *testing.T) {
	if dockerClientTimeout <= 10*time.Second {
		t.Fatalf("Docker API timeout %s must exceed the graceful stop window", dockerClientTimeout)
	}
	if containerNamespaceReadyTimeout <= 10*time.Second {
		t.Fatalf("container namespace readiness timeout %s must tolerate Docker restart state propagation", containerNamespaceReadyTimeout)
	}
	if quotaToNano(100000) != 1000000000 {
		t.Fatal("quota normalization failed")
	}
	adapter := &Adapter{}
	if adapter.name(domain.ID("abc")) != "netlab-abc" {
		t.Fatal("ownership name failed")
	}
}

func TestStartRunningContainerRebuildsEndpointsWithoutReplacingContainer(t *testing.T) {
	engine := &fakeEngine{
		items:      []container.Summary{{ID: "stable-container", State: container.StateRunning}},
		inspection: container.InspectResponse{State: &container.State{Running: true, Pid: 4242}},
	}
	endpoints := &fakeEndpoints{}
	adapter := NewAdapterWithRuntime(engine, endpoints)

	if err := adapter.Start(context.Background(), domain.Node{ID: "node-1"}); err != nil {
		t.Fatal(err)
	}
	if engine.startCalls != 0 || engine.stopCalls != 0 || engine.removeCalls != 0 {
		t.Fatalf("running container was mutated: start=%d stop=%d remove=%d", engine.startCalls, engine.stopCalls, engine.removeCalls)
	}
	if !reflect.DeepEqual(endpoints.ensurePIDs, []int{4242}) {
		t.Fatalf("unexpected endpoint PIDs: %#v", endpoints.ensurePIDs)
	}
}

func TestStartStoppedContainerStartsThenReconcilesEndpoints(t *testing.T) {
	engine := &fakeEngine{
		items: []container.Summary{{ID: "stopped-container", State: container.StateExited}},
		inspections: []container.InspectResponse{
			{State: &container.State{Running: false, Pid: 0}},
			{State: &container.State{Running: true, Pid: 4343}},
		},
	}
	endpoints := &fakeEndpoints{}
	adapter := NewAdapterWithRuntime(engine, endpoints)

	if err := adapter.Start(context.Background(), domain.Node{ID: "node-stopped"}); err != nil {
		t.Fatal(err)
	}
	if engine.startCalls != 1 || engine.removeCalls != 0 {
		t.Fatalf("start=%d remove=%d", engine.startCalls, engine.removeCalls)
	}
	if !reflect.DeepEqual(endpoints.ensurePIDs, []int{4343}) {
		t.Fatalf("endpoint PIDs=%#v", endpoints.ensurePIDs)
	}
}

func TestStartWaitsForContainerNamespacePID(t *testing.T) {
	engine := &fakeEngine{
		items: []container.Summary{{ID: "stopped-container", State: container.StateExited}},
		inspections: []container.InspectResponse{
			{State: &container.State{Running: true, Pid: 0}},
			{State: &container.State{Running: true, Pid: 4545}},
		},
	}
	endpoints := &fakeEndpoints{}
	adapter := NewAdapterWithRuntime(engine, endpoints)
	if err := adapter.Start(context.Background(), domain.Node{ID: "node-wait"}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(endpoints.ensurePIDs, []int{4545}) {
		t.Fatalf("unexpected endpoint PIDs: %#v", endpoints.ensurePIDs)
	}
}

func TestStartUsesInspectInsteadOfStaleContainerListState(t *testing.T) {
	engine := &fakeEngine{
		items: []container.Summary{{ID: "stale-running", State: container.StateRunning}},
		inspections: []container.InspectResponse{
			{State: &container.State{Running: false, Pid: 0}},
			{State: &container.State{Running: true, Pid: 4747}},
		},
	}
	endpoints := &fakeEndpoints{}
	adapter := NewAdapterWithRuntime(engine, endpoints)
	if err := adapter.Start(context.Background(), domain.Node{ID: "node-stale"}); err != nil {
		t.Fatal(err)
	}
	if engine.startCalls != 1 || !reflect.DeepEqual(endpoints.ensurePIDs, []int{4747}) {
		t.Fatalf("start=%d endpoint PIDs=%#v", engine.startCalls, endpoints.ensurePIDs)
	}
}

func TestStopCleansEndpointsBeforeStoppingContainer(t *testing.T) {
	engine := &fakeEngine{items: []container.Summary{{ID: "running-container", State: container.StateRunning}}, inspection: container.InspectResponse{State: &container.State{Running: true, Pid: 4646}}}
	endpoints := &fakeEndpoints{}
	adapter := NewAdapterWithRuntime(engine, endpoints)
	if err := adapter.Stop(context.Background(), domain.Node{ID: "node-stop"}); err != nil {
		t.Fatal(err)
	}
	if endpoints.cleanupCalls != 1 || engine.stopCalls != 1 {
		t.Fatalf("cleanup=%d stop=%d", endpoints.cleanupCalls, engine.stopCalls)
	}
	endpoints.cleanupErr = errors.New("cleanup failed")
	if err := adapter.Stop(context.Background(), domain.Node{ID: "node-stop"}); err == nil || engine.stopCalls != 1 {
		t.Fatalf("cleanup failure must prevent stop: err=%v stop=%d", err, engine.stopCalls)
	}
}

func TestStartCompensatesNewContainerWhenEndpointSetupFails(t *testing.T) {
	engine := &fakeEngine{inspection: container.InspectResponse{State: &container.State{Running: true, Pid: 5252}}}
	endpoints := &fakeEndpoints{ensureErr: errors.New("endpoint failed")}
	adapter := NewAdapterWithRuntime(engine, endpoints)
	node := domain.Node{ID: "node-2", LaboratoryID: "lab-1", Config: map[string]any{
		"image":   "busybox:latest",
		"command": []any{"sleep", "86400"},
	}}

	err := adapter.Start(context.Background(), node)
	if err == nil || !errors.Is(err, endpoints.ensureErr) {
		t.Fatalf("expected endpoint failure, got %v", err)
	}
	if engine.startCalls != 1 || engine.stopCalls != 1 || engine.removeCalls != 1 || endpoints.cleanupCalls != 1 {
		t.Fatalf("unexpected compensation: start=%d stop=%d remove=%d cleanup=%d", engine.startCalls, engine.stopCalls, engine.removeCalls, endpoints.cleanupCalls)
	}
	if !reflect.DeepEqual(engine.created.Config.Cmd, []string{"sleep", "86400"}) {
		t.Fatalf("JSON command was not preserved: %#v", engine.created.Config.Cmd)
	}
}

func TestStartGrantsOnlyRequiredNetworkCapabilities(t *testing.T) {
	engine := &fakeEngine{}
	adapter := NewAdapterWithEngine(engine)
	node := domain.Node{ID: "node-network-admin", LaboratoryID: "lab-1", Config: map[string]any{
		"image":   "busybox:latest",
		"command": []any{"sleep", "86400"},
	}}

	if err := adapter.Start(context.Background(), node); err != nil {
		t.Fatal(err)
	}
	if engine.created.HostConfig == nil {
		t.Fatal("host config was not created")
	}
	want := []string{"NET_ADMIN", "NET_RAW"}
	if !reflect.DeepEqual([]string(engine.created.HostConfig.CapAdd), want) {
		t.Fatalf("cap add=%v want=%v", engine.created.HostConfig.CapAdd, want)
	}
	if engine.created.HostConfig.Privileged {
		t.Fatal("network nodes must not run privileged")
	}
}

func TestContainerCommandRejectsNonStringArguments(t *testing.T) {
	if _, err := containerCommand([]any{"sleep", 10}); err == nil {
		t.Fatal("expected invalid command argument to fail")
	}
}

func TestOpenConsoleCreatesInteractiveShell(t *testing.T) {
	client, server := net.Pipe()
	defer server.Close()
	engine := &fakeEngine{items: []container.Summary{{ID: "container-1", State: container.StateRunning}}, inspection: container.InspectResponse{State: &container.State{Running: true, Pid: 4848}}, attachConn: client}
	console, err := NewAdapterWithEngine(engine).OpenConsole(context.Background(), domain.Node{ID: "node-1"})
	if err != nil {
		t.Fatal(err)
	}
	defer console.Close()
	wantCommand := []string{"/bin/sh", "-lc", "if command -v bash >/dev/null 2>&1; then exec bash -l; else exec sh; fi"}
	if !engine.execOptions.TTY || !engine.execOptions.AttachStdin || !reflect.DeepEqual(engine.execOptions.Cmd, wantCommand) {
		t.Fatalf("exec options=%+v", engine.execOptions)
	}
	go func() { _, _ = server.Write([]byte("ready\n")) }()
	buffer := make([]byte, 6)
	if _, err = io.ReadFull(console, buffer); err != nil || string(buffer) != "ready\n" {
		t.Fatalf("read=%q err=%v", string(buffer), err)
	}
	go func() { _, _ = console.Write([]byte("whoami\n")) }()
	buffer = make([]byte, 7)
	if _, err = io.ReadFull(server, buffer); err != nil || string(buffer) != "whoami\n" {
		t.Fatalf("write=%q err=%v", string(buffer), err)
	}
}
