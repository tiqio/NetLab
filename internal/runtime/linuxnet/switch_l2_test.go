package linuxnet

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/netlab/netlab/internal/domain"
)

func TestL2BridgeVLANMembershipAndForwardingSetup(t *testing.T) {
	executor := &scriptExecutor{}
	runtime, _ := NewSwitchL2Runtime(executor)
	object := domain.NetworkObject{ID: "sw-1", Config: map[string]any{"vlan_filtering": true, "ports": []any{map[string]any{"name": "eth0", "pvid": 10, "tagged": []any{20, 30}}}}}
	if err := runtime.Configure(context.Background(), object); err != nil {
		t.Fatal(err)
	}
	commands := strings.Join(executor.commands, "\n")
	for _, fragment := range []string{"type bridge vlan_filtering 1", "eth0 master br0", "vid 10 pvid untagged", "vid 20", "vid 30"} {
		if !strings.Contains(commands, fragment) {
			t.Fatalf("missing %q in\n%s", fragment, commands)
		}
	}
}

type vlanRuntimeExecutor struct {
	commands []string
	outputs  map[string][]byte
	errors   map[string]error
}

func (e *vlanRuntimeExecutor) Run(_ context.Context, name string, args ...string) error {
	command := name + " " + strings.Join(args, " ")
	e.commands = append(e.commands, command)
	return e.errors[command]
}

func (e *vlanRuntimeExecutor) Output(_ context.Context, name string, args ...string) ([]byte, error) {
	command := name + " " + strings.Join(args, " ")
	e.commands = append(e.commands, command)
	if err := e.errors[command]; err != nil {
		return nil, err
	}
	return e.outputs[command], nil
}

func TestL2ConfigureSkipsDisruptiveReapplyWhenMembershipMatches(t *testing.T) {
	namespace := SwitchL2NamespaceName("sw-healthy")
	executor := &vlanRuntimeExecutor{outputs: map[string][]byte{
		"ip netns list":                                     []byte(namespace + "\n"),
		"ip -n " + namespace + " link show lo":              []byte("UP"),
		"ip -n " + namespace + " link show br0":             []byte("exists"),
		"ip -n " + namespace + " -d link show br0":          []byte("2: br0: <BROADCAST,MULTICAST,UP,LOWER_UP> vlan_filtering 1"),
		"ip -n " + namespace + " -d link show eth0":         []byte("3: eth0: <BROADCAST,MULTICAST,UP,LOWER_UP> master br0"),
		"bridge -j -n " + namespace + " vlan show dev eth0": []byte(`[{"ifname":"eth0","vlans":[{"vlan":10,"flags":["PVID","Egress Untagged"]},{"vlan":20},{"vlan":30}]}]`),
	}}
	runtime, _ := NewSwitchL2Runtime(executor)
	object := domain.NetworkObject{ID: "sw-healthy", Config: map[string]any{"vlan_filtering": true, "ports": []any{map[string]any{"name": "eth0", "pvid": 10, "tagged": []any{30, 20}}}}}
	if err := runtime.Configure(context.Background(), object); err != nil {
		t.Fatal(err)
	}
	commands := strings.Join(executor.commands, "\n")
	for _, disruptive := range []string{"eth0 master br0", "vlan del dev eth0", "vlan add dev eth0"} {
		if strings.Contains(commands, disruptive) {
			t.Fatalf("healthy VLAN port was disrupted by %q in %s", disruptive, commands)
		}
	}
}

func TestL2ConfigureClearsVLAN1AndAppliesLatePortMembership(t *testing.T) {
	namespace := SwitchL2NamespaceName("sw-late")
	executor := &vlanRuntimeExecutor{outputs: map[string][]byte{
		"ip netns list":                                       []byte(namespace + "\n"),
		"ip -n " + namespace + " link show lo":                []byte("UP"),
		"ip -n " + namespace + " link show br0":               []byte("exists"),
		"ip -n " + namespace + " -d link show br0":            []byte("2: br0: <BROADCAST,MULTICAST,UP,LOWER_UP> vlan_filtering 1"),
		"ip -n " + namespace + " -d link show trunk0":         []byte("3: trunk0: <BROADCAST,MULTICAST,UP,LOWER_UP>"),
		"bridge -j -n " + namespace + " vlan show dev trunk0": []byte(`[{"ifname":"trunk0","vlans":[{"vlan":1,"flags":["PVID","Egress Untagged"]}]}]`),
	}}
	runtime, _ := NewSwitchL2Runtime(executor)
	object := domain.NetworkObject{ID: "sw-late", Config: map[string]any{"vlan_filtering": true, "ports": []any{map[string]any{"name": "trunk0", "pvid": 10, "tagged": []any{20}}}}}
	if err := runtime.Configure(context.Background(), object); err != nil {
		t.Fatal(err)
	}
	commands := strings.Join(executor.commands, "\n")
	for _, expected := range []string{"trunk0 master br0", "vlan del dev trunk0 vid 1-4094", "vid 10 pvid untagged", "vid 20"} {
		if !strings.Contains(commands, expected) {
			t.Fatalf("missing %q in %s", expected, commands)
		}
	}
}

func TestL2ConfigurePropagatesVLANApplyFailure(t *testing.T) {
	namespace := SwitchL2NamespaceName("sw-fail")
	failedCommand := "bridge -n " + namespace + " vlan add dev eth0 vid 20"
	executor := &vlanRuntimeExecutor{outputs: map[string][]byte{
		"ip netns list":                             []byte(namespace + "\n"),
		"ip -n " + namespace + " link show lo":      []byte("UP"),
		"ip -n " + namespace + " link show br0":     []byte("exists"),
		"ip -n " + namespace + " -d link show eth0": []byte("3: eth0: <UP> master br0"),
	}, errors: map[string]error{failedCommand: errors.New("injected")}}
	runtime, _ := NewSwitchL2Runtime(executor)
	object := domain.NetworkObject{ID: "sw-fail", Config: map[string]any{"vlan_filtering": true, "ports": []any{map[string]any{"name": "eth0", "pvid": 10, "tagged": []any{20}}}}}
	if err := runtime.Configure(context.Background(), object); err == nil {
		t.Fatal("expected VLAN apply failure")
	}
}

func TestL2DiagnosticsReportsObservedMembershipMismatch(t *testing.T) {
	namespace := SwitchL2NamespaceName("sw-diag")
	executor := &vlanRuntimeExecutor{outputs: map[string][]byte{
		"ip -n " + namespace + " link show eth0":            []byte("3: eth0: <UP> master br0"),
		"bridge -j -n " + namespace + " vlan show dev eth0": []byte(`[{"ifname":"eth0","vlans":[{"vlan":1,"flags":["PVID","Egress Untagged"]}]}]`),
	}}
	runtime, _ := NewSwitchL2Runtime(executor)
	object := domain.NetworkObject{ID: "sw-diag", Config: map[string]any{"vlan_filtering": true, "ports": []any{map[string]any{"name": "eth0", "pvid": 10, "tagged": []any{20}}}}}
	diagnostics, err := runtime.DiagnosticsObject(context.Background(), object)
	if err != nil {
		t.Fatal(err)
	}
	mismatches := diagnostics["mismatches"].([]string)
	if len(mismatches) != 1 || !strings.Contains(mismatches[0], "eth0") {
		t.Fatalf("unexpected mismatches: %+v", mismatches)
	}
}

func TestL2DiagnosticsTreatsUnattachedLogicalPortsAsConverged(t *testing.T) {
	namespace := SwitchL2NamespaceName("sw-logical")
	executor := &vlanRuntimeExecutor{
		outputs: map[string][]byte{
			"ip -n " + namespace + " link show eth0":            []byte("3: eth0: <UP> master br0"),
			"bridge -j -n " + namespace + " vlan show dev eth0": []byte(`[{"ifname":"eth0","vlans":[{"vlan":30,"flags":["PVID","Egress Untagged"]}]}]`),
		},
		errors: map[string]error{
			"ip -n " + namespace + " link show eth1": errors.New("Device eth1 does not exist"),
		},
	}
	runtime, _ := NewSwitchL2Runtime(executor)
	object := domain.NetworkObject{ID: "sw-logical", Config: map[string]any{"vlan_filtering": true, "ports": []any{
		map[string]any{"name": "eth0", "pvid": 30},
		map[string]any{"name": "eth1", "pvid": 30},
	}}}

	converged, diagnostics, err := runtime.ConfigurationConverged(context.Background(), object)
	if err != nil {
		t.Fatal(err)
	}
	if !converged {
		t.Fatalf("logical unattached port prevented convergence: %+v", diagnostics)
	}
	observed := diagnostics["observed"].(map[string]any)["ports"].([]SwitchL2PortObservation)
	if len(observed) != 2 || !observed[0].Attached || observed[1].Attached {
		t.Fatalf("unexpected observations: %+v", observed)
	}
}
