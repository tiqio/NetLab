package recovery

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/app/command"
	"github.com/netlab/netlab/internal/app/reconcile"
	"github.com/netlab/netlab/internal/domain"
	storesqlite "github.com/netlab/netlab/internal/store/sqlite"
)

func TestHostRecoveryPolicyAndBoundedConcurrency(t *testing.T) {
	ctx := context.Background()
	database, err := storesqlite.Open(ctx, "file:host-restart?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	repository := storesqlite.NewTopologyRepository(database)
	labs := command.NewLaboratoryService(repository)
	nodes := command.NewNodeService(repository)
	automatic, _ := labs.Create(ctx, "auto", "", domain.RecoveryAutoRestore)
	restricted, _ := labs.Create(ctx, "restricted", "", domain.RecoveryRemainStopped)
	autoNode, _, _ := nodes.Create(ctx, automatic.ID, "auto-node", "pc", 1)
	restrictedNode, _, _ := nodes.Create(ctx, restricted.ID, "restricted-node", "pc", 1)
	_, _ = nodes.SetState(ctx, autoNode.ID, autoNode.Revision, domain.DesiredRunning)
	_, _ = nodes.SetState(ctx, restrictedNode.ID, restrictedNode.Revision, domain.DesiredRunning)
	if err = repository.PrepareHostRecovery(ctx); err != nil {
		t.Fatal(err)
	}
	autoNode, _ = repository.GetNode(ctx, autoNode.ID)
	restrictedNode, _ = repository.GetNode(ctx, restrictedNode.ID)
	if autoNode.DesiredState != domain.DesiredRunning || restrictedNode.DesiredState != domain.DesiredStopped {
		t.Fatalf("auto=%s restricted=%s", autoNode.DesiredState, restrictedNode.DesiredState)
	}
	values := []domain.Node{{ID: "q1", Kind: "qemu"}, {ID: "q2", Kind: "qemu"}, {ID: "q3", Kind: "qemu"}, {ID: "p1", Kind: "pc"}, {ID: "p2", Kind: "pc"}}
	var mu sync.Mutex
	activeQ, maxQ := 0, 0
	err = reconcile.RunBounded(ctx, values, 2, 4, func(_ context.Context, node domain.Node) error {
		if node.Kind == "qemu" {
			mu.Lock()
			activeQ++
			if activeQ > maxQ {
				maxQ = activeQ
			}
			mu.Unlock()
			time.Sleep(10 * time.Millisecond)
			mu.Lock()
			activeQ--
			mu.Unlock()
		}
		return nil
	})
	if err != nil || maxQ > 2 {
		t.Fatalf("maxQ=%d err=%v", maxQ, err)
	}
}
