package integration

import (
	"context"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/app/command"
	"github.com/netlab/netlab/internal/app/query"
	"github.com/netlab/netlab/internal/app/reconcile"
	"github.com/netlab/netlab/internal/app/task"
	"github.com/netlab/netlab/internal/domain"
	storesqlite "github.com/netlab/netlab/internal/store/sqlite"
)

type frontendNetworkRuntime struct{}

func (frontendNetworkRuntime) Configure(context.Context, domain.NetworkObject) error { return nil }
func (frontendNetworkRuntime) Delete(context.Context, domain.ID) error               { return nil }

func TestFrontendTopologyWorkflowUsesAuthoritativeServices(t *testing.T) {
	ctx := context.Background()
	database, err := storesqlite.Open(ctx, "file:frontend-topology-workflow?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	repositories := storesqlite.NewRepositories(database)
	topology := storesqlite.NewTopologyRepository(database)
	labs := command.NewLaboratoryService(topology)
	nodes := command.NewNodeService(topology)
	links := command.NewLinkService(topology)

	lab, err := labs.Create(ctx, "frontend workflow", "shared topology", domain.RecoveryRemainStopped)
	if err != nil {
		t.Fatal(err)
	}
	left, leftInterfaces, err := nodes.Create(ctx, lab.ID, "left", "pc", 2)
	if err != nil {
		t.Fatal(err)
	}
	_, rightInterfaces, err := nodes.Create(ctx, lab.ID, "right", "pc", 1)
	if err != nil {
		t.Fatal(err)
	}
	link, err := links.Connect(ctx, lab.ID, leftInterfaces[0].ID, rightInterfaces[0].ID)
	if err != nil {
		t.Fatal(err)
	}

	networks := reconcile.NewNetworkObjectService(repositories, reconcile.NetworkRuntimeDispatch{Bridge: frontendNetworkRuntime{}})
	bridge, err := networks.Create(ctx, lab.ID, "shared bridge", "bridge", map[string]any{"stp": true})
	if err != nil {
		t.Fatal(err)
	}
	if err = networks.Attach(ctx, bridge.ID, leftInterfaces[1].ID, "port-1", nil); err != nil {
		t.Fatal(err)
	}

	snapshot, err := query.NewLaboratoryService(topology).Snapshot(ctx, lab.ID)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Sequence == 0 || len(snapshot.Nodes) != 2 || len(snapshot.Links) != 1 || len(snapshot.NetworkObjects) != 1 {
		t.Fatalf("unexpected authoritative snapshot: %+v", snapshot)
	}
	if snapshot.Nodes[0].ID != left.ID && snapshot.Nodes[1].ID != left.ID {
		t.Fatalf("left node missing from snapshot: %+v", snapshot.Nodes)
	}

	if err = links.Disconnect(ctx, link.ID); err != nil {
		t.Fatal(err)
	}
	rewired, err := links.Connect(ctx, lab.ID, leftInterfaces[0].ID, rightInterfaces[0].ID)
	if err != nil || rewired.ID == link.ID {
		t.Fatalf("live rewire failed: link=%+v err=%v", rewired, err)
	}

	runner := task.NewRunner(repositories, 1, 4)
	defer runner.Close()
	runner.Register("frontend.topology.verify", func(_ context.Context, value *domain.OperationTask) (map[string]any, error) {
		value.ProgressCurrent = value.ProgressTotal
		return map[string]any{"laboratory_id": lab.ID}, nil
	})
	operation := domain.OperationTask{ID: domain.NewID(), Kind: "frontend.topology.verify", ResourceType: "laboratory", ResourceID: lab.ID, ProgressTotal: 1}
	if err = runner.Enqueue(ctx, operation); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		current, getErr := repositories.GetTask(ctx, operation.ID)
		if getErr == nil && current.State == domain.TaskSucceeded {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("topology workflow task did not complete")
}
