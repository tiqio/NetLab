package query_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/netlab/netlab/internal/app/command"
	"github.com/netlab/netlab/internal/app/query"
	"github.com/netlab/netlab/internal/domain"
	storesqlite "github.com/netlab/netlab/internal/store/sqlite"
)

func TestLaboratorySnapshotRetainsAuthoritativePlacementAcrossRepositoryRestart(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "netlab.db")
	database, err := storesqlite.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	repository := storesqlite.NewTopologyRepository(database)
	laboratory, err := command.NewLaboratoryService(repository).Create(ctx, "restart placement", "", domain.RecoveryAutoRestore)
	if err != nil {
		t.Fatal(err)
	}
	x, y := 360.0, 240.0
	result := command.CreateNodePlacementResult{}
	node, _, err := command.NewNodeService(repository).CreateConfigured(ctx, laboratory.ID, command.CreateNodeRequest{Name: "node-a", Kind: "docker", InterfaceCount: 1, ExpectedLabRevision: laboratory.Revision, PlacementIntent: &domain.PlacementIntent{PreferredX: &x, PreferredY: &y}, PlacementResult: &result})
	if err != nil {
		t.Fatal(err)
	}
	if err = database.Close(); err != nil {
		t.Fatal(err)
	}

	database, err = storesqlite.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	snapshot, err := query.NewLaboratoryService(storesqlite.NewTopologyRepository(database)).Snapshot(ctx, laboratory.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Placements) != 1 || snapshot.Placements[0].ResourceID != node.ID || snapshot.Placements[0].X != x || snapshot.Placements[0].Y != y {
		t.Fatalf("placements=%+v", snapshot.Placements)
	}
	if snapshot.Laboratory.Revision != result.LaboratoryRevision || snapshot.Sequence < 2 {
		t.Fatalf("laboratory=%+v sequence=%d result=%+v", snapshot.Laboratory, snapshot.Sequence, result)
	}
}
