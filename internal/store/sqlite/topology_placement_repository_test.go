package sqlite

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/app/command"
	"github.com/netlab/netlab/internal/domain"
)

func TestCreateWithPlacementHandlesDenseAndConcurrentAdmissions(t *testing.T) {
	ctx := context.Background()
	database, err := Open(ctx, "file:dense-placement-create?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	repository := NewTopologyRepository(database)
	lab, err := command.NewLaboratoryService(repository).Create(ctx, "dense", "", domain.RecoveryAutoRestore)
	if err != nil {
		t.Fatal(err)
	}
	revision := lab.Revision
	x, y := 100.0, 100.0
	for index := 0; index < 20; index++ {
		now := time.Now().UTC()
		node := domain.Node{ID: domain.NewID(), LaboratoryID: lab.ID, Name: fmt.Sprintf("node-%02d", index), Kind: string(domain.RuntimeDocker), Revision: 1, DesiredState: domain.DesiredStopped, ObservedState: domain.ObservedStopped, CPUCount: 1, MemoryMiB: 64, InterfaceLimit: 1, ProcessLimit: 16, Config: map[string]any{}, CreatedAt: now, UpdatedAt: now}
		_, revision, err = repository.CreateNodeWithPlacement(ctx, node, nil, revision, &domain.PlacementIntent{PreferredX: &x, PreferredY: &y}, "test")
		if err != nil {
			t.Fatal(err)
		}
	}
	placements, err := repository.ListPlacements(ctx, lab.ID)
	if err != nil || len(placements) != 20 {
		t.Fatalf("placements=%d err=%v", len(placements), err)
	}
	seen := map[[2]float64]bool{}
	for _, placement := range placements {
		point := [2]float64{placement.X, placement.Y}
		if seen[point] {
			t.Fatalf("duplicate placement: %+v", point)
		}
		seen[point] = true
	}

	current, err := repository.GetLaboratory(ctx, lab.ID)
	if err != nil {
		t.Fatal(err)
	}
	var wait sync.WaitGroup
	results := make(chan error, 2)
	for index := 0; index < 2; index++ {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			now := time.Now().UTC()
			node := domain.Node{ID: domain.NewID(), LaboratoryID: lab.ID, Name: fmt.Sprintf("concurrent-%d", index), Kind: string(domain.RuntimeDocker), Revision: 1, DesiredState: domain.DesiredStopped, ObservedState: domain.ObservedStopped, CPUCount: 1, MemoryMiB: 64, InterfaceLimit: 1, ProcessLimit: 16, Config: map[string]any{}, CreatedAt: now, UpdatedAt: now}
			_, _, createErr := repository.CreateNodeWithPlacement(ctx, node, nil, current.Revision, &domain.PlacementIntent{PreferredX: &x, PreferredY: &y}, "test")
			results <- createErr
		}(index)
	}
	wait.Wait()
	close(results)
	succeeded := 0
	for result := range results {
		if result == nil {
			succeeded++
		}
	}
	if succeeded != 1 {
		t.Fatalf("expected one concurrent admission, got %d", succeeded)
	}
}

func TestCreateNodeWithPlacementCommitsAtomically(t *testing.T) {
	ctx := context.Background()
	database, err := Open(ctx, "file:node-placement-create?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	repository := NewTopologyRepository(database)
	lab, err := command.NewLaboratoryService(repository).Create(ctx, "atomic-node", "", domain.RecoveryAutoRestore)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	node := domain.Node{ID: "node-atomic", LaboratoryID: lab.ID, Name: "node", Kind: string(domain.RuntimeDocker), Revision: 1, DesiredState: domain.DesiredStopped, ObservedState: domain.ObservedStopped, CPUCount: 1, MemoryMiB: 128, InterfaceLimit: 1, ProcessLimit: 16, Config: map[string]any{}, CreatedAt: now, UpdatedAt: now}
	assignment, revision, err := repository.CreateNodeWithPlacement(ctx, node, nil, lab.Revision, nil, "ui")
	if err != nil || revision != lab.Revision.Next() || assignment.Placement.ResourceID != node.ID {
		t.Fatalf("assignment=%#v revision=%d err=%v", assignment, revision, err)
	}
	if _, err = repository.GetNode(ctx, node.ID); err != nil {
		t.Fatal(err)
	}
	placements, err := repository.ListPlacements(ctx, lab.ID)
	if err != nil || len(placements) != 1 {
		t.Fatalf("placements=%#v err=%v", placements, err)
	}
}

func TestCreateNetworkObjectWithPlacementRollsBackOnRevisionConflict(t *testing.T) {
	ctx := context.Background()
	database, err := Open(ctx, "file:object-placement-create?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	topology := NewTopologyRepository(database)
	lab, err := command.NewLaboratoryService(topology).Create(ctx, "atomic-object", "", domain.RecoveryAutoRestore)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	object := domain.NetworkObject{ID: "object-atomic", LaboratoryID: lab.ID, Name: "bridge", Kind: domain.NetworkBridge, Revision: 1, DesiredState: "active", ObservedState: "provisioning", Config: map[string]any{}, CreatedAt: now, UpdatedAt: now}
	_, _, err = NewRepositories(database).CreateNetworkObjectWithPlacement(ctx, object, lab.Revision.Next(), nil, "api")
	if err == nil {
		t.Fatal("expected revision conflict")
	}
	var count int
	if err = database.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM network_objects WHERE id=?`, object.ID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("count=%d err=%v", count, err)
	}
	if err = database.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM topology_placements WHERE resource_id=?`, object.ID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("placements=%d err=%v", count, err)
	}
}

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
