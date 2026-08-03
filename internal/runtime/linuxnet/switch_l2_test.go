package linuxnet

import (
	"context"
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
