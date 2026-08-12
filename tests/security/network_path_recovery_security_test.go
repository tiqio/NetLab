package security_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/netlab/netlab/internal/domain"
	"github.com/netlab/netlab/internal/runtime/linuxnet"
)

type workloadSecurityExecutor struct {
	calls int
	args  []string
}

type workloadSecuritySourceRepository struct{}

func (workloadSecuritySourceRepository) GetNode(context.Context, domain.ID) (domain.Node, error) {
	return domain.Node{}, domain.ErrNotFound
}

func (workloadSecuritySourceRepository) GetNetworkObject(context.Context, domain.ID) (domain.NetworkObject, error) {
	return domain.NetworkObject{ID: "bridge", Kind: domain.NetworkBridge}, nil
}

func (e *workloadSecurityExecutor) Run(context.Context, string, ...string) error { return nil }
func (e *workloadSecurityExecutor) Output(_ context.Context, _ string, args ...string) ([]byte, error) {
	e.calls++
	e.args = append([]string(nil), args...)
	return []byte("64"), nil
}

func TestNetworkPathRecoveryRejectsCommandInjectionAndSecrets(t *testing.T) {
	executor := &workloadSecurityExecutor{}
	runtime := linuxnet.NewTrafficWorkloadRuntime(executor, executor, nil)
	unsafe := domain.TrafficWorkload{ID: "w", LaboratoryID: "lab", Name: "http", Revision: 1, Source: domain.TrafficWorkloadEndpoint{Kind: "node", ResourceID: "node"}, Protocol: "http", AddressFamily: "ipv4", Destination: domain.TrafficWorkloadDestination{URL: "file:///etc/passwd;id"}, IntervalSeconds: 5, TimeoutSeconds: 2, DesiredState: "running", ObservedState: "running"}
	if _, err := runtime.Execute(context.Background(), unsafe, linuxnet.TrafficWorkloadTarget{Kind: "namespace", Namespace: "safe"}); err == nil || executor.calls != 0 {
		t.Fatalf("unsafe workload executed: %v", err)
	}
	safe := unsafe
	safe.Protocol = "icmp"
	safe.Destination = domain.TrafficWorkloadDestination{Address: "192.0.2.1"}
	if _, err := runtime.Execute(context.Background(), safe, linuxnet.TrafficWorkloadTarget{Kind: "namespace", Namespace: "safe;id"}); err != nil {
		t.Fatal(err)
	}
	if strings.Join(executor.args, " ") != "netns exec safe;id ping -4 -n -c 1 -W 2 192.0.2.1" {
		t.Fatalf("argv boundary changed: %v", executor.args)
	}
	var role domain.DeviceInterfaceRole
	if err := json.Unmarshal([]byte(`{"interface_id":"eth0","role":"management","password":"secret"}`), &role); err == nil {
		t.Fatal("secret-bearing role accepted")
	}
	resolver := linuxnet.NewTrafficWorkloadTargetResolver(workloadSecuritySourceRepository{}, workloadSecuritySourceRepository{})
	unsafe.Source = domain.TrafficWorkloadEndpoint{Kind: "network_object", ResourceID: "bridge"}
	if _, err := resolver.ResolveTrafficWorkloadTarget(context.Background(), unsafe); err == nil {
		t.Fatal("unowned host bridge was accepted as a namespace workload source")
	}
}
