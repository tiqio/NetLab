package linuxnet

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/domain"
)

type recordingExecutor struct {
	commands       []string
	failOn         string
	addressOutputs [][]byte
}

type memoryManagedRouteStore struct {
	values  map[string][]dockerRoute
	saveErr error
}

func (s *memoryManagedRouteStore) key(pid int, interfaceName string) string {
	return strconv.Itoa(pid) + "/" + interfaceName
}

func (s *memoryManagedRouteStore) Load(pid int, interfaceName string) ([]dockerRoute, error) {
	return append([]dockerRoute(nil), s.values[s.key(pid, interfaceName)]...), nil
}

func (s *memoryManagedRouteStore) Save(pid int, interfaceName string, routes []dockerRoute) error {
	if s.saveErr != nil {
		return s.saveErr
	}
	if s.values == nil {
		s.values = map[string][]dockerRoute{}
	}
	s.values[s.key(pid, interfaceName)] = append([]dockerRoute(nil), routes...)
	return nil
}

func newTestDockerEndpointRuntime(t *testing.T, executor CommandExecutor, routes managedDockerRouteStore) *DockerEndpointRuntime {
	t.Helper()
	runtime, err := NewDockerEndpointRuntime(executor)
	if err != nil {
		t.Fatal(err)
	}
	if routes == nil {
		routes = &memoryManagedRouteStore{}
	}
	runtime.routes = routes
	return runtime
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
	command := strings.Join(append([]string{name}, args...), " ")
	if strings.Contains(command, "-j address show") {
		if len(e.addressOutputs) > 0 {
			body := e.addressOutputs[0]
			e.addressOutputs = e.addressOutputs[1:]
			return body, nil
		}
		return []byte(`[{"addr_info":[{"family":"inet","scope":"global","dynamic":true},{"family":"inet6","scope":"global","dynamic":true,"tentative":false}]}]`), nil
	}
	return nil, e.Run(ctx, name, args...)
}

func TestDockerEndpointWaitsForIPv6DADBeforeRoutes(t *testing.T) {
	node := domain.Node{ID: "node", Config: map[string]any{
		"interfaces":         []map[string]any{{"id": "if-1", "name": "eth0"}},
		"network_interfaces": []map[string]any{{"name": "eth0", "addresses": []any{"2001:db8::10/64"}, "routes": []any{map[string]any{"destination": "2001:db8:2::/64", "gateway": "2001:db8::1"}}}},
	}}
	executor := &recordingExecutor{addressOutputs: [][]byte{
		[]byte(`[{"addr_info":[{"family":"inet6","scope":"global","tentative":true}]}]`),
		[]byte(`[{"addr_info":[{"family":"inet6","scope":"global","tentative":false}]}]`),
	}}
	runtime := newTestDockerEndpointRuntime(t, executor, nil)
	runtime.pollInterval = time.Millisecond
	if err := runtime.Ensure(context.Background(), node, 42); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(executor.commands, "\n")
	if !strings.Contains(joined, "route replace 2001:db8:2::/64") {
		t.Fatalf("route was not applied after DAD: %s", joined)
	}
}

func TestDockerEndpointRejectsIPv6DADFailure(t *testing.T) {
	node := domain.Node{ID: "node", Config: map[string]any{
		"interfaces":         []map[string]any{{"id": "if-1", "name": "eth0"}},
		"network_interfaces": []map[string]any{{"name": "eth0", "addresses": []any{"2001:db8::10/64"}}},
	}}
	executor := &recordingExecutor{addressOutputs: [][]byte{
		[]byte(`[{"addr_info":[{"family":"inet6","scope":"global","dadfailed":true}]}]`),
	}}
	runtime := newTestDockerEndpointRuntime(t, executor, nil)
	if err := runtime.Ensure(context.Background(), node, 42); err == nil || !strings.Contains(err.Error(), "duplicate address detection failed") {
		t.Fatalf("err=%v", err)
	}
}

func TestDockerEndpointEnsureAndRollback(t *testing.T) {
	node := domain.Node{ID: "node", Config: map[string]any{
		"interfaces":         []map[string]any{{"id": "if-1", "name": "eth0", "mac_address": "02:00:00:00:00:01"}},
		"network_interfaces": []map[string]any{{"name": "eth0", "modes": []any{"static", "dhcpv4", "dhcpv6", "slaac"}, "addresses": []any{"192.0.2.10/24", "2001:db8::10/64"}}},
	}}
	executor := &recordingExecutor{}
	runtime := newTestDockerEndpointRuntime(t, executor, nil)
	if err := runtime.Ensure(context.Background(), node, 42); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(executor.commands, "\n")
	for _, expected := range []string{"link add", "netns 42", "nsenter -t 42 -n", "eth0 up", "address replace 192.0.2.10/24 dev eth0", "address replace 2001:db8::10/64 dev eth0", "accept_ra=2", "systemd-run", "dhclient -d -v -4", "dhclient -d -v -6", "BindPaths=/proc/42/root/etc/resolv.conf:/etc/resolv.conf"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("missing %q in %s", expected, joined)
		}
	}
	failing := &recordingExecutor{failOn: "netns 42"}
	runtime = newTestDockerEndpointRuntime(t, failing, nil)
	if err := runtime.Ensure(context.Background(), node, 42); err == nil {
		t.Fatal("expected failure")
	}
	if !strings.Contains(strings.Join(failing.commands, "\n"), "link delete "+HostInterfaceName("if-1")) {
		t.Fatal("rollback did not delete host endpoint")
	}
}

func TestDockerEndpointDHCPUsesHostClientWithoutContainerMountNamespace(t *testing.T) {
	node := domain.Node{ID: "node-1", Config: map[string]any{
		"interfaces":         []map[string]any{{"id": "if-1", "name": "eth0"}},
		"network_interfaces": []map[string]any{{"name": "eth0", "modes": []any{"dhcpv4"}}},
	}}
	executor := &recordingExecutor{}
	runtime := newTestDockerEndpointRuntime(t, executor, nil)
	runtime.helperRoot = t.TempDir()
	runtime.pollInterval = time.Millisecond
	if err := runtime.Ensure(context.Background(), node, 42); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(executor.commands, "\n")
	for _, forbidden := range []string{"-n -m", "/sbin/dhclient", "/sbin/udhcpc"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("DHCP must not depend on the container mount namespace or image tools (%q):\n%s", forbidden, joined)
		}
	}
	for _, expected := range []string{"nsenter -t 42 -n -- dhclient", "--setenv=NETLAB_OWNERSHIP=node:node-1", "--property=KillMode=control-group"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("missing %q:\n%s", expected, joined)
		}
	}
	for _, expected := range []string{"--property=PrivateMounts=yes", "BindPaths=/proc/42/root/etc/resolv.conf:/etc/resolv.conf", "BindReadOnlyPaths=", "/etc/dhcp/dhclient-enter-hooks.d", "/etc/dhcp/dhclient-exit-hooks.d"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("missing DHCP mount isolation %q:\n%s", expected, joined)
		}
	}
	if strings.Contains(joined, " -sf ") {
		t.Fatalf("host dhclient must use its AppArmor-approved system script:\n%s", joined)
	}
}

func TestDockerEndpointCleanupStopsDHCPHelpersBeforeDeletingLink(t *testing.T) {
	node := domain.Node{ID: "node-1", Config: map[string]any{
		"interfaces": []map[string]any{{"id": "if-1", "name": "eth0"}},
	}}
	executor := &recordingExecutor{}
	runtime := newTestDockerEndpointRuntime(t, executor, nil)
	if err := runtime.Cleanup(context.Background(), node); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(executor.commands, "\n")
	if strings.Index(joined, "systemctl show --property=ActiveState") > strings.Index(joined, "link delete ") {
		t.Fatalf("DHCP helpers must be inspected/stopped before endpoint deletion:\n%s", joined)
	}
}

func TestDockerEndpointRejectsInvalidStaticAddress(t *testing.T) {
	node := domain.Node{ID: "node", Config: map[string]any{
		"interfaces":         []map[string]any{{"id": "if-1", "name": "eth0"}},
		"network_interfaces": []map[string]any{{"name": "eth0", "addresses": []any{"not-a-cidr"}}},
	}}
	runtime := newTestDockerEndpointRuntime(t, &recordingExecutor{}, nil)
	if err := runtime.Ensure(context.Background(), node, 42); err == nil || !strings.Contains(err.Error(), "invalid address") {
		t.Fatalf("err=%v", err)
	}
}

func TestDockerEndpointReconcilesExactManagedRouteSet(t *testing.T) {
	node := domain.Node{ID: "node", Config: map[string]any{
		"interfaces": []map[string]any{{"id": "if-1", "name": "eth0"}},
		"network_interfaces": []map[string]any{{
			"name": "eth0",
			"routes": []any{
				map[string]any{"destination": "198.51.100.0/24", "gateway": "192.0.2.1", "metric": 20},
				map[string]any{"destination": "2001:db8:2::/64", "gateway": "2001:db8::1"},
			},
		}},
	}}
	executor := &recordingExecutor{}
	routes := &memoryManagedRouteStore{values: map[string][]dockerRoute{
		"42/eth0": {{Destination: "203.0.113.0/24", Gateway: "192.0.2.254", Metric: 30}},
	}}
	runtime := newTestDockerEndpointRuntime(t, executor, routes)
	if err := runtime.Ensure(context.Background(), node, 42); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(executor.commands, "\n")
	for _, expected := range []string{
		"nsenter -t 42 -n ip -4 route delete 203.0.113.0/24 via 192.0.2.254 dev eth0 metric 30",
		"nsenter -t 42 -n ip -4 route replace 198.51.100.0/24 via 192.0.2.1 dev eth0 metric 20",
		"nsenter -t 42 -n ip -6 route replace 2001:db8:2::/64 via 2001:db8::1 dev eth0",
	} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("missing %q in %s", expected, joined)
		}
	}
	if strings.Contains(joined, "route flush") || strings.Contains(joined, "203.0.114.0/24") {
		t.Fatalf("unmanaged routes may be removed:\n%s", joined)
	}
	if len(routes.values["42/eth0"]) != 2 {
		t.Fatalf("persisted routes=%+v", routes.values["42/eth0"])
	}
	executor.commands = nil
	if err := runtime.Ensure(context.Background(), node, 42); err != nil {
		t.Fatal(err)
	}
	joined = strings.Join(executor.commands, "\n")
	if strings.Contains(joined, "route delete") {
		t.Fatalf("idempotent ensure deleted an owned route:\n%s", joined)
	}
}

func TestDockerEndpointRemovesStaleManagedRoutesWhenDeclarationIsEmpty(t *testing.T) {
	node := domain.Node{ID: "node", Config: map[string]any{
		"interfaces":         []map[string]any{{"id": "if-1", "name": "eth0"}},
		"network_interfaces": []map[string]any{{"name": "eth0"}},
	}}
	executor := &recordingExecutor{}
	routes := &memoryManagedRouteStore{values: map[string][]dockerRoute{
		"42/eth0": {{Destination: "198.51.100.0/24", Gateway: "192.0.2.1"}},
	}}
	runtime := newTestDockerEndpointRuntime(t, executor, routes)
	if err := runtime.Ensure(context.Background(), node, 42); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(executor.commands, "\n")
	if !strings.Contains(joined, "-4 route delete 198.51.100.0/24 via 192.0.2.1 dev eth0") {
		t.Fatalf("managed route cleanup missing:\n%s", joined)
	}
	if strings.Contains(joined, "route replace") {
		t.Fatalf("unexpected declared route:\n%s", joined)
	}
}

func TestDockerEndpointReportsRouteSpecificFailure(t *testing.T) {
	node := domain.Node{ID: "node", Config: map[string]any{
		"interfaces": []map[string]any{{"id": "if-1", "name": "eth0"}},
		"network_interfaces": []map[string]any{{
			"name":   "eth0",
			"routes": []any{map[string]any{"destination": "198.51.100.0/24", "gateway": "192.0.2.1"}},
		}},
	}}
	executor := &recordingExecutor{failOn: "route replace 198.51.100.0/24"}
	routes := &memoryManagedRouteStore{values: map[string][]dockerRoute{
		"42/eth0": {{Destination: "203.0.113.0/24", Gateway: "192.0.2.254"}},
	}}
	runtime := newTestDockerEndpointRuntime(t, executor, routes)
	err := runtime.Ensure(context.Background(), node, 42)
	if err == nil || !strings.Contains(err.Error(), "apply managed route 198.51.100.0/24 on eth0") {
		t.Fatalf("err=%v", err)
	}
	if len(routes.values["42/eth0"]) != 1 || routes.values["42/eth0"][0].Destination != "203.0.113.0/24" {
		t.Fatalf("failed route set must preserve prior ownership: %+v", routes.values)
	}
	joined := strings.Join(executor.commands, "\n")
	if !strings.Contains(joined, "route replace 203.0.113.0/24 via 192.0.2.254 dev eth0") {
		t.Fatalf("previous route was not restored:\n%s", joined)
	}
}

func TestDockerEndpointRollsBackWhenOwnershipPersistenceFails(t *testing.T) {
	node := domain.Node{ID: "node", Config: map[string]any{
		"interfaces": []map[string]any{{"id": "if-1", "name": "eth0"}},
		"network_interfaces": []map[string]any{{
			"name":   "eth0",
			"routes": []any{map[string]any{"destination": "198.51.100.0/24", "gateway": "192.0.2.1"}},
		}},
	}}
	executor := &recordingExecutor{}
	routes := &memoryManagedRouteStore{
		values:  map[string][]dockerRoute{"42/eth0": {{Destination: "203.0.113.0/24", Gateway: "192.0.2.254"}}},
		saveErr: errors.New("disk full"),
	}
	runtime := newTestDockerEndpointRuntime(t, executor, routes)
	err := runtime.Ensure(context.Background(), node, 42)
	if err == nil || !strings.Contains(err.Error(), "persist managed routes for eth0") {
		t.Fatalf("err=%v", err)
	}
	joined := strings.Join(executor.commands, "\n")
	for _, expected := range []string{
		"route delete 198.51.100.0/24 via 192.0.2.1 dev eth0",
		"route replace 203.0.113.0/24 via 192.0.2.254 dev eth0",
	} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("missing rollback command %q:\n%s", expected, joined)
		}
	}
}

func TestProcManagedDockerRouteStorePersistsInsideContainerRoot(t *testing.T) {
	root := t.TempDir()
	store := &procManagedDockerRouteStore{root: root}
	containerRoot := filepath.Join(root, "42", "root")
	if err := os.MkdirAll(containerRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	want := []dockerRoute{{Destination: "198.51.100.0/24", Gateway: "192.0.2.1", Metric: 20}}
	if err := store.Save(42, "eth0", want); err != nil {
		t.Fatal(err)
	}
	got, err := store.Load(42, "eth0")
	if err != nil || len(got) != 1 || dockerRouteKey(got[0]) != dockerRouteKey(want[0]) {
		t.Fatalf("routes=%+v err=%v", got, err)
	}
}
