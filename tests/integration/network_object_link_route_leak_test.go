package integration

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/domain"
	captureRuntime "github.com/netlab/netlab/internal/runtime/capture"
	"github.com/netlab/netlab/internal/runtime/linuxnet"
)

type cycleExecutor struct {
	creates int
	deletes int
}

func (e *cycleExecutor) Run(_ context.Context, _ string, args ...string) error {
	for index := range args {
		if args[index] == "add" && index+1 < len(args) {
			e.creates++
		}
		if args[index] == "delete" && index+1 < len(args) {
			e.deletes++
		}
	}
	return nil
}

func (*cycleExecutor) Output(context.Context, string, ...string) ([]byte, error) {
	return nil, errors.New("not found")
}

func TestNetworkObjectLinksAndDockerRoutesCompleteOneHundredLeakFreeCycles(t *testing.T) {
	cycles := 100
	if configured, err := strconv.Atoi(os.Getenv("CYCLES")); err == nil && configured > 0 {
		cycles = configured
	}
	executor := &cycleExecutor{}
	dataPlane, err := linuxnet.NewDataPlane(executor)
	if err != nil {
		t.Fatal(err)
	}
	objectA := domain.NetworkObject{ID: "cycle-a", Kind: domain.NetworkSwitchL3, Config: map[string]any{"interfaces": []any{map[string]any{"name": "swp1"}}}}
	objectB := domain.NetworkObject{ID: "cycle-b", Kind: domain.NetworkSwitchL3, Config: map[string]any{"interfaces": []any{map[string]any{"name": "swp1"}}}}
	owned := t.TempDir()
	for cycle := 0; cycle < cycles; cycle++ {
		link := domain.NetworkObjectLink{ID: domain.ID("cycle-link-" + strconv.Itoa(cycle)), ObjectAID: objectA.ID, PortAName: "swp1", ObjectBID: objectB.ID, PortBName: "swp1"}
		if err = dataPlane.EnsureNetworkObjectLink(context.Background(), link, objectA, objectB); err != nil {
			t.Fatalf("cycle %d create: %v", cycle, err)
		}
		correlator := captureRuntime.NewCorrelator(10*time.Millisecond, 4)
		correlator.ObserveNetworkObjectLinkPacket("cycle", link.ID, "a_to_b", captureRuntime.PacketKey{Protocol: 17, Length: 64}, time.Now().UTC())
		if observations, _ := correlator.Snapshot(); len(observations) != 1 || observations[0].NetworkObjectLinkID != link.ID {
			t.Fatalf("cycle %d observations=%+v", cycle, observations)
		}
		routes := []domain.NodeNetworkInterfaceSettings{{Name: "eth0", Addresses: []string{"192.0.2.10/24", "2001:db8:1::10/64"}, Routes: []domain.RouteConfig{{Destination: "198.51.100.0/24", Gateway: "192.0.2.1"}, {Destination: "2001:db8:2::/64", Gateway: "2001:db8:1::1"}}}}
		if err = domain.ValidateNodeNetworkInterfaces(routes); err != nil {
			t.Fatalf("cycle %d route validation: %v", cycle, err)
		}
		cycleDirectory := filepath.Join(owned, strconv.Itoa(cycle))
		if err = os.Mkdir(cycleDirectory, 0o700); err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(cycleDirectory, "managed-routes.json"), []byte(`[]`), 0o600); err != nil {
			t.Fatal(err)
		}
		if err = dataPlane.DeleteNetworkObjectLink(context.Background(), link, objectA, objectB); err != nil {
			t.Fatalf("cycle %d delete: %v", cycle, err)
		}
		if err = os.RemoveAll(cycleDirectory); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := os.ReadDir(owned)
	if err != nil || len(entries) != 0 {
		t.Fatalf("owned route/capture state leaked: entries=%v err=%v", entries, err)
	}
	if executor.creates != cycles || executor.deletes != cycles {
		t.Fatalf("cycles=%d creates=%d deletes=%d", cycles, executor.creates, executor.deletes)
	}
}
