package integration

import (
	"context"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/app/command"
	"github.com/netlab/netlab/internal/domain"
	storesqlite "github.com/netlab/netlab/internal/store/sqlite"
)

func TestAcceptanceScaleAndConvergenceBudget(t *testing.T) {
	ctx := context.Background()
	database, err := storesqlite.Open(ctx, "file:scale?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	repository := storesqlite.NewTopologyRepository(database)
	lab, _ := command.NewLaboratoryService(repository).Create(ctx, "scale", "", domain.RecoveryRemainStopped)
	nodes := command.NewNodeService(repository)
	started := time.Now()
	var interfaces []domain.Interface
	for index := 0; index < 10; index++ {
		kind := "pc"
		if index < 4 {
			kind = "qemu"
		}
		_, created, createErr := nodes.Create(ctx, lab.ID, "node-"+string(rune('a'+index)), kind, 2)
		if createErr != nil {
			t.Fatal(createErr)
		}
		interfaces = append(interfaces, created...)
	}
	if time.Since(started) > 3*time.Second {
		t.Fatalf("10-node convergence exceeded budget: %s", time.Since(started))
	}
	linkStarted := time.Now()
	if _, err = command.NewLinkService(repository).Connect(ctx, lab.ID, interfaces[0].ID, interfaces[2].ID); err != nil {
		t.Fatal(err)
	}
	if time.Since(linkStarted) > 10*time.Second {
		t.Fatal("link change exceeded budget")
	}
	snapshot, err := repository.Snapshot(ctx, lab.ID)
	if err != nil || len(snapshot.Nodes) != 10 || len(snapshot.Links) != 1 {
		t.Fatalf("nodes=%d links=%d err=%v", len(snapshot.Nodes), len(snapshot.Links), err)
	}
}
