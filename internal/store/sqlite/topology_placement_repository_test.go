package sqlite

import (
	"context"
	"testing"

	"github.com/netlab/netlab/internal/app/command"
	"github.com/netlab/netlab/internal/domain"
)

func TestPlacementBatchRevisionAtomicityAndCleanup(t *testing.T) {
	ctx := context.Background()
	database, err := Open(ctx, "file:placements?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	repository := NewTopologyRepository(database)
	lab, err := command.NewLaboratoryService(repository).Create(ctx, "placements", "", domain.RecoveryAutoRestore)
	if err != nil {
		t.Fatal(err)
	}
	node, _, err := command.NewNodeService(repository).Create(ctx, lab.ID, "node", "pc", 1)
	if err != nil {
		t.Fatal(err)
	}
	revision, placements, err := repository.UpdatePlacements(ctx, lab.ID, lab.Revision, []domain.PlacementUpdate{{ResourceID: node.ID, ResourceType: domain.PlacementNode, X: 12, Y: 34}})
	if err != nil || revision != lab.Revision.Next() || len(placements) != 1 {
		t.Fatalf("revision=%d placements=%+v err=%v", revision, placements, err)
	}
	if _, _, err = repository.UpdatePlacements(ctx, lab.ID, lab.Revision, []domain.PlacementUpdate{{ResourceID: node.ID, ResourceType: domain.PlacementNode, X: 99, Y: 99, Revision: 1}}); err == nil {
		t.Fatal("expected laboratory revision conflict")
	}
	if _, _, err = repository.UpdatePlacements(ctx, lab.ID, revision, []domain.PlacementUpdate{{ResourceID: node.ID, ResourceType: domain.PlacementNode, X: 50, Y: 60, Revision: 1}, {ResourceID: "missing", ResourceType: domain.PlacementNode}}); err == nil {
		t.Fatal("expected atomic validation failure")
	}
	values, err := repository.ListPlacements(ctx, lab.ID)
	if err != nil || len(values) != 1 || values[0].X != 12 {
		t.Fatalf("values=%+v err=%v", values, err)
	}
	if err = repository.DeleteNode(ctx, node.ID, node.Revision); err != nil {
		t.Fatal(err)
	}
	values, err = repository.ListPlacements(ctx, lab.ID)
	if err != nil || len(values) != 0 {
		t.Fatalf("expected cascade cleanup: %+v %v", values, err)
	}
}

func TestDeleteNodeCascadesConnectedLinks(t *testing.T) {
	ctx := context.Background()
	database, err := Open(ctx, "file:node-delete-links?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	repository := NewTopologyRepository(database)
	lab, err := command.NewLaboratoryService(repository).Create(ctx, "node-delete-links", "", domain.RecoveryAutoRestore)
	if err != nil {
		t.Fatal(err)
	}
	nodeA, interfacesA, err := command.NewNodeService(repository).Create(ctx, lab.ID, "node-a", "pc", 1)
	if err != nil {
		t.Fatal(err)
	}
	_, interfacesB, err := command.NewNodeService(repository).Create(ctx, lab.ID, "node-b", "pc", 1)
	if err != nil {
		t.Fatal(err)
	}
	link, err := command.NewLinkService(repository).Connect(ctx, lab.ID, interfacesA[0].ID, interfacesB[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if err = repository.DeleteNode(ctx, nodeA.ID, nodeA.Revision); err != nil {
		t.Fatal(err)
	}
	if _, err = repository.GetLink(ctx, link.ID); err == nil {
		t.Fatal("connected link remained after node deletion")
	}
	remaining, err := repository.GetInterface(ctx, interfacesB[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if remaining.DesiredLinkID != "" {
		t.Fatalf("remaining endpoint still references deleted link: %+v", remaining)
	}
}
