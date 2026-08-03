package linuxnet

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/domain"
)

type scriptExecutor struct {
	commands     []string
	output       []byte
	failContains string
	activeUnits  map[string]bool
}

func (e *scriptExecutor) Run(_ context.Context, name string, args ...string) error {
	command := name + " " + strings.Join(args, " ")
	e.commands = append(e.commands, command)
	if e.failContains != "" && strings.Contains(command, e.failContains) {
		return errors.New("injected")
	}
	if name == "systemd-run" {
		if e.activeUnits == nil {
			e.activeUnits = map[string]bool{}
		}
		for _, argument := range args {
			if strings.HasPrefix(argument, "--unit=") {
				e.activeUnits[strings.TrimPrefix(argument, "--unit=")] = true
			}
		}
	}
	if name == "systemctl" && len(args) == 2 && args[0] == "stop" && e.activeUnits != nil {
		delete(e.activeUnits, args[1])
	}
	return nil
}

func (e *scriptExecutor) Output(_ context.Context, name string, args ...string) ([]byte, error) {
	e.commands = append(e.commands, name+" "+strings.Join(args, " "))
	if name == "systemctl" && len(args) > 0 && args[0] == "show" {
		unit := args[len(args)-1]
		if e.activeUnits != nil && e.activeUnits[unit] {
			return []byte("active\n"), nil
		}
		return []byte("inactive\n"), nil
	}
	return e.output, nil
}

func TestPCStaticDHCPv4DHCPv6SLAACRoutesAndDiagnostics(t *testing.T) {
	executor := &scriptExecutor{output: []byte(`[{"addr_info":[{"family":"inet","scope":"global","dynamic":true},{"family":"inet6","scope":"global","dynamic":true}]}]`)}
	runtime, _ := NewPCRuntime(executor)
	runtime.resolvRoot = t.TempDir()
	runtime.helperRoot = t.TempDir()
	object := domain.NetworkObject{
		ID:   "pc-1",
		Kind: domain.NetworkPC,
		Config: map[string]any{
			"interfaces": []any{
				map[string]any{
					"name":      "eth0",
					"modes":     []any{"static", "dhcpv4", "dhcpv6", "slaac"},
					"addresses": []any{"192.0.2.10/24", "2001:db8::10/64"},
					"routes":    []any{map[string]any{"destination": "0.0.0.0/0", "gateway": "192.0.2.1"}},
					"dns":       []any{"192.0.2.53", "2001:db8::53"},
				},
			},
		},
	}
	if err := runtime.Configure(context.Background(), object); err != nil {
		t.Fatal(err)
	}
	commands := strings.Join(executor.commands, "\n")
	for _, fragment := range []string{"address replace 192.0.2.10/24", "systemd-run --quiet --no-block --collect", "dhclient -d -v -4", "dhclient -d -v -6", "accept_ra=2", "route replace 0.0.0.0/0 via 192.0.2.1"} {
		if !strings.Contains(commands, fragment) {
			t.Fatalf("missing %q in\n%s", fragment, commands)
		}
	}
	diagnostics, err := runtime.Diagnostics(context.Background(), object.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics.DNS) != 2 {
		t.Fatalf("dns=%v", diagnostics.DNS)
	}
}

func TestPCReportsDHCPFailure(t *testing.T) {
	executor := &scriptExecutor{failContains: "dhclient -d -v -4"}
	runtime, _ := NewPCRuntime(executor)
	runtime.helperRoot = t.TempDir()
	object := domain.NetworkObject{ID: "pc-2", Config: map[string]any{"interfaces": []any{map[string]any{"name": "eth0", "modes": []any{"dhcpv4"}}}}}
	if err := runtime.Configure(context.Background(), object); err == nil || !strings.Contains(err.Error(), "start DHCPv4 helper") {
		t.Fatalf("err=%v", err)
	}
}

func TestPCAdoptsActiveDHCPHelperAndReportsSLAACTimeout(t *testing.T) {
	executor := &scriptExecutor{output: []byte(`[{"addr_info":[]}]`), activeUnits: map[string]bool{pcDHCPUnit("pc-3", "eth0", "4"): true}}
	runtime, _ := NewPCRuntime(executor)
	runtime.helperRoot = t.TempDir()
	runtime.resolvRoot = t.TempDir()
	runtime.acquisitionTimeout = 5 * time.Millisecond
	runtime.pollInterval = time.Millisecond
	runtime.now = func() time.Time { return time.Unix(100, 0).UTC() }
	if err := runtime.writeSLAACMarker("pc-3", "eth0"); err != nil {
		t.Fatal(err)
	}
	runtime.now = func() time.Time { return time.Unix(101, 0).UTC() }
	diagnostics, err := runtime.Diagnostics(context.Background(), "pc-3")
	if err != nil {
		t.Fatal(err)
	}
	if diagnostics.SLAAC["address_status"] != "timeout" || diagnostics.SLAAC["route_status"] != "timeout" {
		t.Fatalf("slaac=%v", diagnostics.SLAAC)
	}
	if active, err := runtime.helperActive(context.Background(), pcDHCPUnit("pc-3", "eth0", "4")); err != nil || !active {
		t.Fatalf("active=%v err=%v", active, err)
	}
}
