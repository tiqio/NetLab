package linuxnet

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/netlab/netlab/internal/domain"
)

func TestL3AddressRouteAndForwarding(t *testing.T) {
	executor := &scriptExecutor{}
	runtime, _ := NewSwitchL3Runtime(executor)
	object := domain.NetworkObject{ID: "r-1", Config: map[string]any{"interfaces": []any{map[string]any{"name": "eth0", "addresses": []any{"192.0.2.1/24", "2001:db8::1/64"}}}, "routes": []any{map[string]any{"destination": "198.51.100.0/24", "gateway": "192.0.2.2", "metric": 20}, map[string]any{"destination": "2001:db8:2::/64", "gateway": "2001:db8::2"}}, "forward_ipv4": true, "forward_ipv6": false}}
	if err := runtime.Configure(context.Background(), object); err != nil {
		t.Fatal(err)
	}
	commands := strings.Join(executor.commands, "\n")
	for _, fragment := range []string{"route flush table main", "-6 route flush table main", "address flush dev eth0 scope global", "address replace 192.0.2.1/24", "route replace 198.51.100.0/24 via 192.0.2.2 metric 20", "-6 route replace 2001:db8:2::/64 via 2001:db8::2", "net.ipv4.ip_forward=1", "net.ipv6.conf.all.forwarding=0"} {
		if !strings.Contains(commands, fragment) {
			t.Fatalf("missing %q in\n%s", fragment, commands)
		}
	}
}

func TestL3AllowsUnaddressedConnectableInterfaces(t *testing.T) {
	executor := &scriptExecutor{}
	runtime, _ := NewSwitchL3Runtime(executor)
	object := domain.NetworkObject{ID: "r-unaddressed", Config: map[string]any{
		"interfaces":   []any{map[string]any{"name": "eth0", "addresses": []any{}}},
		"forward_ipv4": true,
		"forward_ipv6": true,
	}}
	if err := runtime.Configure(context.Background(), object); err != nil {
		t.Fatal(err)
	}
	commands := strings.Join(executor.commands, "\n")
	if !strings.Contains(commands, "link set eth0 up") {
		t.Fatalf("unaddressed interface was not brought up:\n%s", commands)
	}
	if strings.Contains(commands, "address replace") {
		t.Fatalf("unaddressed interface unexpectedly received an address:\n%s", commands)
	}
}

type retryRouteExecutor struct {
	scriptExecutor
	routeAttempts int
}

func (e *retryRouteExecutor) Run(ctx context.Context, name string, args ...string) error {
	command := name + " " + strings.Join(args, " ")
	e.commands = append(e.commands, command)
	if strings.Contains(command, "route replace 198.51.100.0/24") {
		e.routeAttempts++
		if e.routeAttempts == 1 {
			return errors.New("Nexthop has invalid gateway")
		}
	}
	return nil
}

func TestL3RetriesRouteAfterTransientGatewayFailure(t *testing.T) {
	executor := &retryRouteExecutor{}
	runtime, _ := NewSwitchL3Runtime(executor)
	object := domain.NetworkObject{
		ID: "r-retry",
		Config: map[string]any{
			"interfaces": []any{map[string]any{"name": "eth0", "addresses": []any{"192.0.2.1/24"}}},
			"routes":     []any{map[string]any{"destination": "198.51.100.0/24", "gateway": "192.0.2.2"}},
		},
	}

	if err := runtime.Configure(context.Background(), object); err != nil {
		t.Fatal(err)
	}
	if executor.routeAttempts != 2 {
		t.Fatalf("route attempts=%d, want 2", executor.routeAttempts)
	}
}

type missingL3PortExecutor struct {
	scriptExecutor
}

func (e *missingL3PortExecutor) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	command := name + " " + strings.Join(args, " ")
	e.commands = append(e.commands, command)
	if strings.Contains(command, "link show eth0") {
		return nil, errors.New("Device eth0 does not exist")
	}
	return e.scriptExecutor.Output(ctx, name, args...)
}

func TestL3DefersRoutesUntilDeclaredPortsArrive(t *testing.T) {
	executor := &missingL3PortExecutor{}
	runtime, _ := NewSwitchL3Runtime(executor)
	object := domain.NetworkObject{ID: "r-late-port", Config: map[string]any{
		"interfaces":   []any{map[string]any{"name": "eth0", "addresses": []any{"192.0.2.1/24"}}},
		"routes":       []any{map[string]any{"destination": "198.51.100.0/24", "gateway": "192.0.2.2"}},
		"forward_ipv4": true,
		"forward_ipv6": true,
	}}

	if err := runtime.Configure(context.Background(), object); err != nil {
		t.Fatal(err)
	}
	commands := strings.Join(executor.commands, "\n")
	if strings.Contains(commands, "route replace 198.51.100.0/24") {
		t.Fatalf("route was applied before its port arrived:\n%s", commands)
	}
	if strings.Contains(commands, "route flush table main") {
		t.Fatalf("existing routes were flushed before every declared port arrived:\n%s", commands)
	}
	for _, fragment := range []string{"net.ipv4.ip_forward=1", "net.ipv6.conf.all.forwarding=1"} {
		if !strings.Contains(commands, fragment) {
			t.Fatalf("missing base configuration %q in\n%s", fragment, commands)
		}
	}
}

type missingFIBExecutor struct {
	scriptExecutor
}

func (e *missingFIBExecutor) Run(ctx context.Context, name string, args ...string) error {
	command := name + " " + strings.Join(args, " ")
	e.commands = append(e.commands, command)
	if strings.Contains(command, "route flush table main") {
		return errors.New("ipv4: FIB table does not exist")
	}
	return nil
}

func TestL3AllowsAnInitiallyEmptyFIBTable(t *testing.T) {
	executor := &missingFIBExecutor{}
	runtime, _ := NewSwitchL3Runtime(executor)
	object := domain.NetworkObject{ID: "r-empty", Config: map[string]any{
		"interfaces":   []any{map[string]any{"name": "eth0", "addresses": []any{"192.0.2.1/24"}}},
		"forward_ipv4": true,
	}}
	if err := runtime.Configure(context.Background(), object); err != nil {
		t.Fatal(err)
	}
}

func TestL3DiagnosticsReportsRequestedVersusObservedMismatch(t *testing.T) {
	executor := &scriptExecutor{outputFor: func(_ string, args ...string) []byte {
		command := strings.Join(args, " ")
		switch {
		case strings.Contains(command, "-j address show"):
			return []byte(`[{"ifname":"eth0","addr_info":[{"local":"192.0.2.99","prefixlen":24,"scope":"global"}]}]`)
		case strings.Contains(command, "-j route show"):
			return []byte(`[{"dst":"198.51.100.0/24","gateway":"192.0.2.2","metric":20}]`)
		case strings.Contains(command, "net.ipv4.ip_forward"):
			return []byte("0\n")
		case strings.Contains(command, "net.ipv6.conf.all.forwarding"):
			return []byte("1\n")
		default:
			return nil
		}
	}}
	runtime, _ := NewSwitchL3Runtime(executor)
	object := domain.NetworkObject{ID: "diagnostic-router", Config: map[string]any{
		"interfaces":   []any{map[string]any{"name": "eth0", "addresses": []any{"192.0.2.1/24"}}},
		"routes":       []any{map[string]any{"destination": "198.51.100.0/24", "gateway": "192.0.2.2", "metric": 20}},
		"forward_ipv4": true,
		"forward_ipv6": true,
	}}
	diagnostics, err := runtime.DiagnosticsObject(context.Background(), object)
	if err != nil {
		t.Fatal(err)
	}
	mismatches := diagnostics["mismatches"].([]string)
	joined := strings.Join(mismatches, "\n")
	if !strings.Contains(joined, "forward_ipv4 desired=true observed=false") || !strings.Contains(joined, "interface eth0 addresses") {
		t.Fatalf("mismatches=%v", mismatches)
	}
	if strings.Contains(joined, "forward_ipv6") || strings.Contains(joined, "routes desired") {
		t.Fatalf("unexpected mismatch=%v", mismatches)
	}
}

func TestL3ConfigurationConvergenceAllowsAdditionalConnectedRoutes(t *testing.T) {
	executor := &scriptExecutor{outputFor: func(_ string, args ...string) []byte {
		command := strings.Join(args, " ")
		switch {
		case strings.Contains(command, "-j address show"):
			return []byte(`[{"ifname":"eth0","addr_info":[{"local":"192.0.2.1","prefixlen":24,"scope":"global"}]}]`)
		case strings.Contains(command, "-j route show"):
			return []byte(`[{"dst":"192.0.2.0/24"},{"dst":"198.51.100.0/24","gateway":"192.0.2.2","metric":20}]`)
		case strings.Contains(command, "net.ipv4.ip_forward"):
			return []byte("1\n")
		case strings.Contains(command, "net.ipv6.conf.all.forwarding"):
			return []byte("0\n")
		default:
			return nil
		}
	}}
	runtime, _ := NewSwitchL3Runtime(executor)
	object := domain.NetworkObject{ID: "diagnostic-router", Config: map[string]any{
		"interfaces":   []any{map[string]any{"name": "eth0", "addresses": []any{"192.0.2.1/24"}}},
		"routes":       []any{map[string]any{"destination": "198.51.100.0/24", "gateway": "192.0.2.2", "metric": 20}},
		"forward_ipv4": true,
	}}
	converged, diagnostics, err := runtime.ConfigurationConverged(context.Background(), object)
	if err != nil {
		t.Fatal(err)
	}
	if !converged {
		t.Fatalf("diagnostics=%v", diagnostics)
	}
}
