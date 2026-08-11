package linuxnet

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/netlab/netlab/internal/domain"
	"github.com/netlab/netlab/internal/runtime/ownership"
)

type dataPlaneExecutor struct {
	commands     []string
	bridgeExists bool
	failMaster   string
	existing     map[string]bool
	outputs      map[string]string
}

type missingFirstEndpointExecutor struct {
	commands []string
	first    string
}

func (e *missingFirstEndpointExecutor) Run(_ context.Context, name string, args ...string) error {
	command := strings.Join(append([]string{name}, args...), " ")
	e.commands = append(e.commands, command)
	if strings.Contains(command, e.first) {
		return errors.New("Cannot find device")
	}
	return nil
}

func (e *missingFirstEndpointExecutor) Output(context.Context, string, ...string) ([]byte, error) {
	return nil, errors.New("missing")
}

func (e *dataPlaneExecutor) Run(_ context.Context, name string, args ...string) error {
	command := strings.Join(append([]string{name}, args...), " ")
	e.commands = append(e.commands, command)
	if strings.Contains(command, "link add") && e.bridgeExists {
		return errors.New("exists")
	}
	if e.failMaster != "" && strings.Contains(command, e.failMaster+" master") {
		return errors.New("attach failed")
	}
	return nil
}
func (e *dataPlaneExecutor) Output(_ context.Context, name string, args ...string) ([]byte, error) {
	command := strings.Join(append([]string{name}, args...), " ")
	if output, ok := e.outputs[command]; ok {
		return []byte(output), nil
	}
	if e.existing != nil && e.existing[command] {
		return []byte("exists"), nil
	}
	if e.bridgeExists {
		return []byte("bridge"), nil
	}
	return nil, errors.New("missing")
}

func TestNetworkObjectLinkDoesNotReattachHealthyEndpoints(t *testing.T) {
	link := domain.NetworkObjectLink{ID: "healthy-link", ObjectAID: "l2", PortAName: "uplink0", ObjectBID: "bridge", PortBName: "uplink0"}
	l2 := domain.NetworkObject{ID: "l2", Kind: domain.NetworkSwitchL2, Config: map[string]any{
		"ports": []any{map[string]any{"name": "uplink0", "pvid": 10}},
	}}
	bridge := domain.NetworkObject{ID: "bridge", Kind: domain.NetworkBridge}
	namespace := SwitchL2NamespaceName(l2.ID)
	hostEndpoint := ownership.Name("nvb", link.ID, 15)
	bridgeName, err := NetworkBridgeName(bridge)
	if err != nil {
		t.Fatal(err)
	}
	executor := &dataPlaneExecutor{outputs: map[string]string{
		"ip -n " + namespace + " link show uplink0":    "exists",
		"ip link show " + hostEndpoint:                 "exists",
		"ip -n " + namespace + " -d link show uplink0": "1: uplink0: <BROADCAST,MULTICAST,UP,LOWER_UP> master br0",
		"ip -d link show " + hostEndpoint:              "2: " + hostEndpoint + ": <BROADCAST,MULTICAST,UP,LOWER_UP> master " + bridgeName,
	}}
	runtime, _ := NewDataPlane(executor)
	if err := runtime.EnsureNetworkObjectLink(context.Background(), link, l2, bridge); err != nil {
		t.Fatal(err)
	}
	commands := strings.Join(executor.commands, "\n")
	for _, disruptive := range []string{
		"uplink0 master br0",
		hostEndpoint + " master " + bridgeName,
		"link set uplink0 up",
		"link set " + hostEndpoint + " up",
	} {
		if strings.Contains(commands, disruptive) {
			t.Fatalf("healthy endpoint was reattached with %q in %s", disruptive, commands)
		}
	}
}

func TestNetworkObjectLinkReplacesPartialNamespacePair(t *testing.T) {
	objectA := domain.NetworkObject{ID: "a", Kind: domain.NetworkSwitchL2}
	objectB := domain.NetworkObject{ID: "b", Kind: domain.NetworkSwitchL2}
	link := domain.NetworkObjectLink{ID: "partial", ObjectAID: objectA.ID, PortAName: "swp1", ObjectBID: objectB.ID, PortBName: "swp2"}
	executor := &dataPlaneExecutor{existing: map[string]bool{
		"ip -n " + SwitchL2NamespaceName(objectA.ID) + " link show swp1": true,
	}}
	runtime, _ := NewDataPlane(executor)
	if err := runtime.EnsureNetworkObjectLink(context.Background(), link, objectA, objectB); err != nil {
		t.Fatal(err)
	}
	commands := strings.Join(executor.commands, "\n")
	deleteCommand := "ip -n " + SwitchL2NamespaceName(objectA.ID) + " link delete swp1"
	if !strings.Contains(commands, deleteCommand) || !strings.Contains(commands, "type veth peer name") {
		t.Fatalf("partial pair was not replaced: %s", commands)
	}
}

func TestDataPlaneIdempotencyAndRollback(t *testing.T) {
	executor := &dataPlaneExecutor{bridgeExists: true}
	runtime, _ := NewDataPlane(executor)
	link := domain.Link{ID: "link"}
	a := domain.Interface{ID: "a"}
	b := domain.Interface{ID: "b"}
	if err := runtime.EnsureLink(context.Background(), link, a, b); err != nil {
		t.Fatal(err)
	}
	executor = &dataPlaneExecutor{failMaster: HostInterfaceName("b")}
	runtime, _ = NewDataPlane(executor)
	if err := runtime.EnsureLink(context.Background(), link, a, b); err == nil {
		t.Fatal("expected attach failure")
	}
	if !strings.Contains(strings.Join(executor.commands, "\n"), "link delete "+LinkBridgeName("link")) {
		t.Fatal("bridge rollback missing")
	}
}

func TestNamespaceAttachmentCreatesTransitBridgeAndPeer(t *testing.T) {
	executor := &dataPlaneExecutor{}
	runtime, _ := NewDataPlane(executor)
	attachment := domain.NetworkAttachment{ID: "attach", InterfaceID: "if-a", PortName: "lan0", Config: map[string]any{"pvid": 10, "tagged": []any{20, 30}}}
	object := domain.NetworkObject{ID: "switch", Kind: domain.NetworkSwitchL2}
	if err := runtime.AttachNamespace(context.Background(), attachment, domain.Interface{ID: "if-a"}, object); err != nil {
		t.Fatal(err)
	}
	commands := strings.Join(executor.commands, "\n")
	for _, expected := range []string{"type bridge", HostInterfaceName("if-a") + " master", "netns " + SwitchL2NamespaceName("switch"), "lan0 master br0", "vid 10 pvid untagged", "vid 20", "vid 30"} {
		if !strings.Contains(commands, expected) {
			t.Fatalf("missing %q in %s", expected, commands)
		}
	}
}

func TestNamespaceAttachmentCleanupOwnsTransitBridge(t *testing.T) {
	executor := &dataPlaneExecutor{}
	runtime, _ := NewDataPlane(executor)
	attachment := domain.NetworkAttachment{ID: "attach"}
	if err := runtime.DeleteAttachment(context.Background(), attachment); err != nil {
		t.Fatal(err)
	}
	expected := "link delete " + ownership.Name("nla", attachment.ID, 15)
	if !strings.Contains(strings.Join(executor.commands, "\n"), expected) {
		t.Fatalf("missing %q", expected)
	}
}

func TestNamespaceAttachmentAppliesPCAndL3AddressesAfterPortArrival(t *testing.T) {
	for _, object := range []domain.NetworkObject{
		{
			ID:   "pc",
			Kind: domain.NetworkPC,
			Config: map[string]any{"interfaces": []any{
				map[string]any{"name": "eth0", "modes": []any{"static", "slaac"}, "addresses": []any{"192.0.2.10/24", "2001:db8::10/64"}},
			}},
		},
		{
			ID:   "router",
			Kind: domain.NetworkSwitchL3,
			Config: map[string]any{"interfaces": []any{
				map[string]any{"name": "eth0", "addresses": []any{"192.0.2.1/24", "2001:db8::1/64"}},
			}},
		},
	} {
		executor := &dataPlaneExecutor{}
		runtime, _ := NewDataPlane(executor)
		attachment := domain.NetworkAttachment{ID: domain.ID("attach-" + object.Kind), InterfaceID: "if-a", PortName: "eth0"}
		if err := runtime.AttachNamespace(context.Background(), attachment, domain.Interface{ID: "if-a"}, object); err != nil {
			t.Fatal(err)
		}
		commands := strings.Join(executor.commands, "\n")
		for _, address := range []string{"192.0.2.", "2001:db8::"} {
			if !strings.Contains(commands, address) {
				t.Fatalf("%s attachment missing %s in %s", object.Kind, address, commands)
			}
		}
	}
}

func TestNetworkObjectLinkCreatesDirectNamespaceVethPair(t *testing.T) {
	executor := &dataPlaneExecutor{}
	runtime, _ := NewDataPlane(executor)
	link := domain.NetworkObjectLink{ID: "object-link", ObjectAID: "l2-a", PortAName: "uplink0", ObjectBID: "l3-b", PortBName: "eth9"}
	l2 := domain.NetworkObject{ID: "l2-a", Kind: domain.NetworkSwitchL2, Config: map[string]any{"vlan_filtering": true, "ports": []any{map[string]any{"name": "uplink0", "pvid": 10, "tagged": []any{20}}}}}
	l3 := domain.NetworkObject{ID: "l3-b", Kind: domain.NetworkSwitchL3, Config: map[string]any{
		"interfaces":   []any{map[string]any{"name": "eth9", "addresses": []any{"192.0.2.1/24"}}},
		"forward_ipv4": true,
		"forward_ipv6": true,
	}}
	if err := runtime.EnsureNetworkObjectLink(context.Background(), link, l2, l3); err != nil {
		t.Fatal(err)
	}
	commands := strings.Join(executor.commands, "\n")
	for _, expected := range []string{"type veth peer name", "netns " + SwitchL2NamespaceName(l2.ID), "uplink0 master br0", "vid 10 pvid untagged", "vid 20", "netns " + SwitchL3NamespaceName(l3.ID), "192.0.2.1/24 dev eth9", "net/ipv4/conf/eth9/forwarding=1", "net/ipv6/conf/eth9/forwarding=1"} {
		if !strings.Contains(commands, expected) {
			t.Fatalf("missing %q in %s", expected, commands)
		}
	}
	if strings.Contains(commands, LinkBridgeName(link.ID)+" type bridge") {
		t.Fatalf("object link unexpectedly created a host bridge: %s", commands)
	}
}

func TestNetworkObjectLinkConnectsHostBridgeToNamespace(t *testing.T) {
	executor := &dataPlaneExecutor{}
	runtime, _ := NewDataPlane(executor)
	link := domain.NetworkObjectLink{ID: "bridge-l3", ObjectAID: "bridge", PortAName: "uplink0", ObjectBID: "l3", PortBName: "eth2"}
	bridge := domain.NetworkObject{ID: "bridge", Kind: domain.NetworkBridge}
	bridgeName, err := NetworkBridgeName(bridge)
	if err != nil {
		t.Fatal(err)
	}
	l3 := domain.NetworkObject{ID: "l3", Kind: domain.NetworkSwitchL3, Config: map[string]any{
		"interfaces": []any{map[string]any{"name": "eth2", "addresses": []any{}}},
	}}
	if err := runtime.EnsureNetworkObjectLink(context.Background(), link, bridge, l3); err != nil {
		t.Fatal(err)
	}
	commands := strings.Join(executor.commands, "\n")
	hostEnd := ownership.Name("nva", link.ID, 15)
	for _, expected := range []string{
		"type veth peer name",
		hostEnd + " master " + bridgeName,
		hostEnd + " up",
		"netns " + SwitchL3NamespaceName(l3.ID),
		"name " + link.PortBName,
	} {
		if !strings.Contains(commands, expected) {
			t.Fatalf("missing %q in %s", expected, commands)
		}
	}
	if strings.Contains(commands, "-n "+bridgeName) {
		t.Fatalf("host bridge was treated as a namespace: %s", commands)
	}
}

func TestNetworkObjectLinkPreparesBothEndsBeforeApplyingPCConfiguration(t *testing.T) {
	executor := &dataPlaneExecutor{}
	runtime, _ := NewDataPlane(executor)
	link := domain.NetworkObjectLink{ID: "pc-link", ObjectAID: "pc-a", PortAName: "eth0", ObjectBID: "l2-b", PortBName: "access0"}
	pc := domain.NetworkObject{ID: "pc-a", Kind: domain.NetworkPC, Config: map[string]any{
		"interfaces": []any{map[string]any{"name": "eth0", "modes": []any{"static"}, "addresses": []any{"192.0.2.10/24"}}},
	}}
	l2 := domain.NetworkObject{ID: "l2-b", Kind: domain.NetworkSwitchL2, Config: map[string]any{
		"vlan_filtering": true, "ports": []any{map[string]any{"name": "access0", "pvid": 1, "tagged": []any{}}},
	}}
	if err := runtime.EnsureNetworkObjectLink(context.Background(), link, pc, l2); err != nil {
		t.Fatal(err)
	}
	commands := strings.Join(executor.commands, "\n")
	prepared := strings.Index(commands, "access0 master br0")
	configured := strings.Index(commands, "address replace 192.0.2.10/24 dev eth0")
	if prepared < 0 || configured < 0 || prepared > configured {
		t.Fatalf("L2 peer was not prepared before PC configuration:\n%s", commands)
	}
}

func TestDeleteNetworkObjectLinkDeletesNamespaceEndpoint(t *testing.T) {
	executor := &dataPlaneExecutor{}
	runtime, _ := NewDataPlane(executor)
	link := domain.NetworkObjectLink{ID: "object-link", PortAName: "swp1", PortBName: "swp2"}
	objectA := domain.NetworkObject{ID: "a", Kind: domain.NetworkSwitchL2}
	objectB := domain.NetworkObject{ID: "b", Kind: domain.NetworkSwitchL2}
	if err := runtime.DeleteNetworkObjectLink(context.Background(), link, objectA, objectB); err != nil {
		t.Fatal(err)
	}
	commands := strings.Join(executor.commands, "\n")
	if !strings.Contains(commands, "-n "+SwitchL2NamespaceName(objectA.ID)+" link delete swp1") {
		t.Fatalf("missing direct pair cleanup in %s", commands)
	}
}

func TestDeleteNetworkObjectLinkFallsBackToSurvivingEndpoint(t *testing.T) {
	objectA := domain.NetworkObject{ID: "a", Kind: domain.NetworkSwitchL2}
	objectB := domain.NetworkObject{ID: "b", Kind: domain.NetworkSwitchL2}
	link := domain.NetworkObjectLink{ID: "partial-delete", PortAName: "swp1", PortBName: "swp2"}
	first := "-n " + SwitchL2NamespaceName(objectA.ID) + " link delete swp1"
	executor := &missingFirstEndpointExecutor{first: first}
	runtime, _ := NewDataPlane(executor)
	if err := runtime.DeleteNetworkObjectLink(context.Background(), link, objectA, objectB); err != nil {
		t.Fatal(err)
	}
	commands := strings.Join(executor.commands, "\n")
	if !strings.Contains(commands, first) || !strings.Contains(commands, "-n "+SwitchL2NamespaceName(objectB.ID)+" link delete swp2") {
		t.Fatalf("missing fallback cleanup: %s", commands)
	}
}

func TestDeleteNetworkObjectLinkDoesNotTouchUnrelatedInterfaces(t *testing.T) {
	objectA := domain.NetworkObject{ID: "a", Kind: domain.NetworkSwitchL2}
	objectB := domain.NetworkObject{ID: "b", Kind: domain.NetworkSwitchL2}
	link := domain.NetworkObjectLink{ID: "owned-link", ObjectAID: objectA.ID, PortAName: "swp1", ObjectBID: objectB.ID, PortBName: "swp2"}
	executor := &dataPlaneExecutor{}
	runtime, _ := NewDataPlane(executor)
	if err := runtime.DeleteNetworkObjectLink(context.Background(), link, objectA, objectB); err != nil {
		t.Fatal(err)
	}
	commands := strings.Join(executor.commands, "\n")
	if strings.Contains(commands, "unrelated0") || strings.Contains(commands, "link delete swp") && !strings.Contains(commands, "link delete swp1") {
		t.Fatalf("cleanup escaped owned endpoint identity: %s", commands)
	}
	if !strings.Contains(commands, "ip -n "+SwitchL2NamespaceName(objectA.ID)+" link delete swp1") {
		t.Fatalf("owned endpoint was not targeted exactly: %s", commands)
	}
}

func TestDeleteNetworkObjectLinkSkipsNamespaceCleanupForPlainBridge(t *testing.T) {
	bridge := domain.NetworkObject{ID: "bridge", Kind: domain.NetworkBridge}
	l3 := domain.NetworkObject{ID: "l3", Kind: domain.NetworkSwitchL3}
	link := domain.NetworkObjectLink{ID: "link", ObjectAID: bridge.ID, PortAName: "uplink0", ObjectBID: l3.ID, PortBName: "eth2"}
	executor := &dataPlaneExecutor{}
	runtime, _ := NewDataPlane(executor)
	if err := runtime.DeleteNetworkObjectLink(context.Background(), link, bridge, l3); err != nil {
		t.Fatal(err)
	}
	commands := strings.Join(executor.commands, "\n")
	if strings.Contains(commands, "nlbr") {
		t.Fatalf("plain bridge was treated as namespace-backed: %s", commands)
	}
	if !strings.Contains(commands, "ip link delete "+ownership.Name("nva", link.ID, 15)) {
		t.Fatalf("owned host bridge endpoint was not cleaned: %s", commands)
	}
	if strings.Contains(commands, "ip -n "+SwitchL3NamespaceName(l3.ID)+" link delete eth2") {
		t.Fatalf("peer cleanup continued after deleting the host veth endpoint: %s", commands)
	}
}
