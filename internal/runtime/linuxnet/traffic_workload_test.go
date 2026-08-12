package linuxnet

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/domain"
)

type trafficExecutorFake struct {
	name   string
	args   []string
	output []byte
	err    error
	block  bool
}

func (f *trafficExecutorFake) Run(context.Context, string, ...string) error { return nil }
func (f *trafficExecutorFake) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	f.name = name
	f.args = append([]string(nil), args...)
	if f.block {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return f.output, f.err
}

type trafficGuestFake struct {
	argv   []string
	result TrafficWorkloadGuestResult
}

func (f *trafficGuestFake) Execute(_ context.Context, _ domain.Node, argv []string, _ time.Duration, _ int) (TrafficWorkloadGuestResult, error) {
	f.argv = append([]string(nil), argv...)
	return f.result, nil
}

func runtimeWorkload(protocol string) domain.TrafficWorkload {
	value := domain.TrafficWorkload{ID: "workload", LaboratoryID: "lab", Name: "probe", Revision: 1, Source: domain.TrafficWorkloadEndpoint{Kind: "node", ResourceID: "node"}, Protocol: protocol, AddressFamily: "ipv4", IntervalSeconds: 5, TimeoutSeconds: 1, DesiredState: "running", ObservedState: "running"}
	switch protocol {
	case "icmp":
		value.Destination.Address = "192.0.2.1"
	case "http":
		value.Destination.URL = "http://192.0.2.2/health"
	case "dns":
		value.Destination.Name = "example.test"
	}
	return value
}

func TestTrafficWorkloadNamespaceUsesSafeArgv(t *testing.T) {
	executor := &trafficExecutorFake{output: []byte("64 bytes")}
	runtime := NewTrafficWorkloadRuntime(executor, nil, nil)
	result, err := runtime.Execute(context.Background(), runtimeWorkload("icmp"), TrafficWorkloadTarget{Kind: "namespace", Namespace: "nlpc-safe"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"netns", "exec", "nlpc-safe", "ping", "-4", "-n", "-c", "1", "-W", "1", "192.0.2.1"}
	if executor.name != "ip" || !reflect.DeepEqual(executor.args, want) || result.MatchedBytes == 0 {
		t.Fatalf("name=%s args=%v result=%+v", executor.name, executor.args, result)
	}
}

func TestTrafficWorkloadDockerAndQGAUseAllowlists(t *testing.T) {
	docker := &trafficExecutorFake{output: []byte("128")}
	guest := &trafficGuestFake{result: TrafficWorkloadGuestResult{Stdout: []byte("203.0.113.1 STREAM")}}
	runtime := NewTrafficWorkloadRuntime(nil, docker, guest.Execute)
	result, err := runtime.Execute(context.Background(), runtimeWorkload("http"), TrafficWorkloadTarget{Kind: "docker", Container: "netlab-node"})
	if err != nil || result.MatchedBytes != 128 || docker.name != "docker" || docker.args[0] != "exec" || docker.args[2] != "curl" {
		t.Fatalf("docker args=%v result=%+v err=%v", docker.args, result, err)
	}
	if _, err = runtime.Execute(context.Background(), runtimeWorkload("dns"), TrafficWorkloadTarget{Kind: "qga", Node: domain.Node{ID: "node"}}); err != nil {
		t.Fatal(err)
	}
	if got := guest.argv; len(got) != 3 || got[0] != "getent" || got[1] != "ahostsv4" || got[2] != "example.test" {
		t.Fatalf("unexpected guest argv: %v", got)
	}
}

func TestTrafficWorkloadRejectsUnsafeDefinitionsAndBoundsOutput(t *testing.T) {
	executor := &trafficExecutorFake{output: []byte(strings.Repeat("x", TrafficWorkloadOutputLimit+10))}
	runtime := NewTrafficWorkloadRuntime(executor, nil, nil)
	unsafe := runtimeWorkload("http")
	unsafe.Destination.URL = "file:///etc/passwd"
	if _, err := runtime.Execute(context.Background(), unsafe, TrafficWorkloadTarget{Kind: "namespace", Namespace: "safe"}); err == nil || executor.name != "" {
		t.Fatalf("unsafe request executed: %v", err)
	}
	result, err := runtime.Execute(context.Background(), runtimeWorkload("dns"), TrafficWorkloadTarget{Kind: "namespace", Namespace: "safe"})
	if err != nil || !result.Truncated || len(result.Output) != TrafficWorkloadOutputLimit {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestTrafficWorkloadTimeoutIsStructured(t *testing.T) {
	executor := &trafficExecutorFake{block: true}
	runtime := NewTrafficWorkloadRuntime(executor, nil, nil)
	workload := runtimeWorkload("icmp")
	workload.TimeoutSeconds = 1
	started := time.Now()
	_, err := runtime.Execute(context.Background(), workload, TrafficWorkloadTarget{Kind: "namespace", Namespace: "safe"})
	if err == nil || time.Since(started) > 2*time.Second {
		t.Fatalf("timeout err=%v elapsed=%s", err, time.Since(started))
	}
	var problem domain.Problem
	if !errors.As(err, &problem) || problem.Code != "workload_timeout" {
		t.Fatalf("unexpected timeout: %#v", err)
	}
}
