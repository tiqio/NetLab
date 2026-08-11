package integration

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/netlab/netlab/internal/domain"
	"github.com/netlab/netlab/internal/runtime/linuxnet"
	"github.com/netlab/netlab/internal/runtime/ownership"
)

type bridgeL3CleanupExecutor struct {
	commands []string
}

func (e *bridgeL3CleanupExecutor) Run(_ context.Context, name string, args ...string) error {
	e.commands = append(e.commands, name+" "+strings.Join(args, " "))
	return nil
}

func (*bridgeL3CleanupExecutor) Output(context.Context, string, ...string) ([]byte, error) {
	return nil, errors.New("not found")
}

func TestBridgeToL3CleanupCompletesTwentyCyclesWithoutNamespaceLeak(t *testing.T) {
	executor := &bridgeL3CleanupExecutor{}
	runtime, err := linuxnet.NewDataPlane(executor)
	if err != nil {
		t.Fatal(err)
	}
	bridge := domain.NetworkObject{ID: "plain-bridge", Kind: domain.NetworkBridge, Config: map[string]any{"mtu": 1500}}
	l3 := domain.NetworkObject{ID: "routed-endpoint", Kind: domain.NetworkSwitchL3, Config: map[string]any{"interfaces": []any{map[string]any{"name": "eth0", "addresses": []any{}}}}}
	for cycle := 0; cycle < 20; cycle++ {
		link := domain.NetworkObjectLink{ID: domain.NewID(), ObjectAID: bridge.ID, PortAName: "access0", ObjectBID: l3.ID, PortBName: "eth0"}
		if err = runtime.DeleteNetworkObjectLink(context.Background(), link, bridge, l3); err != nil {
			t.Fatalf("cycle %d: %v", cycle, err)
		}
		want := "ip link delete " + ownership.Name("nva", link.ID, 15)
		if executor.commands[len(executor.commands)-1] != want {
			t.Fatalf("cycle %d cleanup=%q want=%q", cycle, executor.commands[len(executor.commands)-1], want)
		}
	}
	if len(executor.commands) != 20 {
		t.Fatalf("cleanup commands=%d want=20: %v", len(executor.commands), executor.commands)
	}
	for _, command := range executor.commands {
		if strings.Contains(command, "plain-bridge") || strings.Contains(command, "netns delete") {
			t.Fatalf("plain bridge entered namespace cleanup: %s", command)
		}
	}
}
