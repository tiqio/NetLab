package linuxnet

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/netlab/netlab/internal/domain"
)

type recordingExecutor struct {
	commands []string
	failOn   string
}

func (e *recordingExecutor) Run(_ context.Context, name string, args ...string) error {
	command := strings.Join(append([]string{name}, args...), " ")
	e.commands = append(e.commands, command)
	if e.failOn != "" && strings.Contains(command, e.failOn) {
		return errors.New("injected failure")
	}
	if strings.Contains(command, "link show") {
		return errors.New("missing")
	}
	return nil
}

func (e *recordingExecutor) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	return nil, e.Run(ctx, name, args...)
}

func TestDockerEndpointEnsureAndRollback(t *testing.T) {
	node := domain.Node{ID: "node", Config: map[string]any{
		"interfaces":         []map[string]any{{"id": "if-1", "name": "eth0", "mac_address": "02:00:00:00:00:01"}},
		"network_interfaces": []map[string]any{{"name": "eth0", "modes": []any{"static", "dhcpv4", "dhcpv6", "slaac"}, "addresses": []any{"192.0.2.10/24", "2001:db8::10/64"}}},
	}}
	executor := &recordingExecutor{}
	runtime, _ := NewDockerEndpointRuntime(executor)
	if err := runtime.Ensure(context.Background(), node, 42); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(executor.commands, "\n")
	for _, expected := range []string{"link add", "netns 42", "nsenter -t 42 -n", "eth0 up", "address replace 192.0.2.10/24 dev eth0", "address replace 2001:db8::10/64 dev eth0", "accept_ra=2", "/sbin/dhclient -4 -nw", "/sbin/dhclient -6 -nw"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("missing %q in %s", expected, joined)
		}
	}
	failing := &recordingExecutor{failOn: "netns 42"}
	runtime, _ = NewDockerEndpointRuntime(failing)
	if err := runtime.Ensure(context.Background(), node, 42); err == nil {
		t.Fatal("expected failure")
	}
	if !strings.Contains(strings.Join(failing.commands, "\n"), "link delete "+HostInterfaceName("if-1")) {
		t.Fatal("rollback did not delete host endpoint")
	}
}

func TestDockerEndpointRejectsInvalidStaticAddress(t *testing.T) {
	node := domain.Node{ID: "node", Config: map[string]any{
		"interfaces":         []map[string]any{{"id": "if-1", "name": "eth0"}},
		"network_interfaces": []map[string]any{{"name": "eth0", "addresses": []any{"not-a-cidr"}}},
	}}
	runtime, _ := NewDockerEndpointRuntime(&recordingExecutor{})
	if err := runtime.Ensure(context.Background(), node, 42); err == nil || !strings.Contains(err.Error(), "invalid address") {
		t.Fatalf("err=%v", err)
	}
}
