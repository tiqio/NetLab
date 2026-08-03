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
func (e *dataPlaneExecutor) Output(_ context.Context, _ string, _ ...string) ([]byte, error) {
	if e.bridgeExists {
		return []byte("bridge"), nil
	}
	return nil, errors.New("missing")
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
	l3 := domain.NetworkObject{ID: "l3-b", Kind: domain.NetworkSwitchL3, Config: map[string]any{"interfaces": []any{map[string]any{"name": "eth9", "addresses": []any{"192.0.2.1/24"}}}}}
	if err := runtime.EnsureNetworkObjectLink(context.Background(), link, l2, l3); err != nil {
		t.Fatal(err)
	}
	commands := strings.Join(executor.commands, "\n")
	for _, expected := range []string{"type veth peer name", "netns " + SwitchL2NamespaceName(l2.ID), "uplink0 master br0", "vid 10 pvid untagged", "vid 20", "netns " + SwitchL3NamespaceName(l3.ID), "192.0.2.1/24 dev eth9"} {
		if !strings.Contains(commands, expected) {
			t.Fatalf("missing %q in %s", expected, commands)
		}
	}
	if strings.Contains(commands, LinkBridgeName(link.ID)+" type bridge") {
		t.Fatalf("object link unexpectedly created a host bridge: %s", commands)
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
