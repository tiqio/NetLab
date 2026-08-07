package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/domain"
)

func TestImportTopologyPersistsObjectLinksAndReservationsAtomically(t *testing.T) {
	ctx := context.Background()
	database, err := Open(ctx, "file:import-object-links?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	now := time.Now().UTC()
	laboratory := domain.Laboratory{ID: "lab-import", Name: "import", Revision: 1, RecoveryPolicy: domain.RecoveryRemainStopped, LifecycleState: "active", CreatedAt: now, UpdatedAt: now}
	objects := []domain.NetworkObject{{ID: "object-a", LaboratoryID: laboratory.ID, Name: "A", Kind: domain.NetworkSwitchL2, Revision: 1, DesiredState: "active", ObservedState: "pending", Config: map[string]any{}, CreatedAt: now, UpdatedAt: now}, {ID: "object-b", LaboratoryID: laboratory.ID, Name: "B", Kind: domain.NetworkSwitchL2, Revision: 1, DesiredState: "active", ObservedState: "pending", Config: map[string]any{}, CreatedAt: now, UpdatedAt: now}}
	links := []domain.NetworkObjectLink{{ID: "object-link", LaboratoryID: laboratory.ID, ObjectAID: objects[0].ID, PortAName: "swp1", ObjectBID: objects[1].ID, PortBName: "swp2", Revision: 1, DesiredState: "connected", ObservedState: "pending"}}
	placements := []domain.TopologyPlacement{{LaboratoryID: laboratory.ID, ResourceID: objects[0].ID, ResourceType: domain.PlacementNetworkObject, X: 220, Y: 140, Revision: 3}, {LaboratoryID: laboratory.ID, ResourceID: objects[1].ID, ResourceType: domain.PlacementNetworkObject, X: 420, Y: 140, Revision: 1}}
	if err = NewTopologyRepository(database).ImportTopology(ctx, laboratory, nil, nil, nil, objects, links, placements); err != nil {
		t.Fatal(err)
	}
	var linkCount, reservationCount int
	if err = database.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM network_object_links WHERE laboratory_id=?`, laboratory.ID).Scan(&linkCount); err != nil {
		t.Fatal(err)
	}
	if err = database.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM topology_endpoint_reservations WHERE resource_type='network_object_link' AND resource_id=?`, links[0].ID).Scan(&reservationCount); err != nil {
		t.Fatal(err)
	}
	if linkCount != 1 || reservationCount != 2 {
		t.Fatalf("links=%d reservations=%d", linkCount, reservationCount)
	}
	stored, err := NewTopologyRepository(database).ListPlacements(ctx, laboratory.ID)
	if err != nil || len(stored) != 2 || stored[0].X != 220 || stored[0].Revision != 3 {
		t.Fatalf("placements=%+v err=%v", stored, err)
	}
}

func TestImportTopologyRollsBackObjectLinksOnReservationConflict(t *testing.T) {
	ctx := context.Background()
	database, err := Open(ctx, "file:import-object-link-rollback?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	now := time.Now().UTC()
	laboratory := domain.Laboratory{ID: "lab-rollback", Name: "rollback", Revision: 1, RecoveryPolicy: domain.RecoveryRemainStopped, LifecycleState: "active", CreatedAt: now, UpdatedAt: now}
	objects := []domain.NetworkObject{{ID: "object-a", LaboratoryID: laboratory.ID, Name: "A", Kind: domain.NetworkSwitchL2, Revision: 1, DesiredState: "active", ObservedState: "pending", Config: map[string]any{}, CreatedAt: now, UpdatedAt: now}, {ID: "object-b", LaboratoryID: laboratory.ID, Name: "B", Kind: domain.NetworkSwitchL2, Revision: 1, DesiredState: "active", ObservedState: "pending", Config: map[string]any{}, CreatedAt: now, UpdatedAt: now}, {ID: "object-c", LaboratoryID: laboratory.ID, Name: "C", Kind: domain.NetworkSwitchL2, Revision: 1, DesiredState: "active", ObservedState: "pending", Config: map[string]any{}, CreatedAt: now, UpdatedAt: now}}
	links := []domain.NetworkObjectLink{{ID: "link-one", LaboratoryID: laboratory.ID, ObjectAID: objects[0].ID, PortAName: "swp1", ObjectBID: objects[1].ID, PortBName: "swp1", Revision: 1, DesiredState: "connected", ObservedState: "pending"}, {ID: "link-two", LaboratoryID: laboratory.ID, ObjectAID: objects[0].ID, PortAName: "swp1", ObjectBID: objects[2].ID, PortBName: "swp1", Revision: 1, DesiredState: "connected", ObservedState: "pending"}}
	placements := []domain.TopologyPlacement{{LaboratoryID: laboratory.ID, ResourceID: objects[0].ID, ResourceType: domain.PlacementNetworkObject, X: 10, Y: 20, Revision: 1}}
	if err = NewTopologyRepository(database).ImportTopology(ctx, laboratory, nil, nil, nil, objects, links, placements); err == nil {
		t.Fatal("expected occupied endpoint conflict")
	}
	var laboratoryCount int
	if err = database.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM laboratories WHERE id=?`, laboratory.ID).Scan(&laboratoryCount); err != nil {
		t.Fatal(err)
	}
	if laboratoryCount != 0 {
		t.Fatalf("partial import remained: laboratories=%d", laboratoryCount)
	}
	var placementCount int
	if err = database.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM topology_placements WHERE laboratory_id=?`, laboratory.ID).Scan(&placementCount); err != nil || placementCount != 0 {
		t.Fatalf("partial placements remained: count=%d err=%v", placementCount, err)
	}
}

func TestImportedPlacementRemainsOccupiedForSubsequentAuthoritativeCreation(t *testing.T) {
	ctx := context.Background()
	database, err := Open(ctx, "file:import-placement-occupancy?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	now := time.Now().UTC()
	laboratory := domain.Laboratory{ID: "lab-placement", Name: "placement", Revision: 1, RecoveryPolicy: domain.RecoveryRemainStopped, LifecycleState: "active", CreatedAt: now, UpdatedAt: now}
	node := domain.Node{ID: "node-imported", LaboratoryID: laboratory.ID, Name: "imported", Kind: "docker", Revision: 1, DesiredState: domain.DesiredStopped, ObservedState: domain.ObservedStopped, Config: map[string]any{}, CreatedAt: now, UpdatedAt: now}
	placements := []domain.TopologyPlacement{{LaboratoryID: laboratory.ID, ResourceID: node.ID, ResourceType: domain.PlacementNode, X: 100, Y: 100, Revision: 5}}
	repository := NewTopologyRepository(database)
	if err = repository.ImportTopology(ctx, laboratory, []domain.Node{node}, nil, nil, nil, nil, placements); err != nil {
		t.Fatal(err)
	}
	x, y := 100.0, 100.0
	newNode := domain.Node{ID: "node-new", LaboratoryID: laboratory.ID, Name: "new", Kind: "docker", Revision: 1, DesiredState: domain.DesiredStopped, ObservedState: domain.ObservedStopped, Config: map[string]any{}, CreatedAt: now, UpdatedAt: now}
	assignment, revision, err := repository.CreateNodeWithPlacement(ctx, newNode, nil, laboratory.Revision, &domain.PlacementIntent{PreferredX: &x, PreferredY: &y}, "test")
	if err != nil {
		t.Fatal(err)
	}
	if !assignment.Adjusted || (assignment.AssignedCenter.X == x && assignment.AssignedCenter.Y == y) || revision != 2 {
		t.Fatalf("assignment=%+v revision=%d", assignment, revision)
	}
	stored, err := repository.ListPlacements(ctx, laboratory.ID)
	if err != nil || len(stored) != 2 || stored[0].ResourceID != node.ID || stored[0].X != 100 || stored[0].Y != 100 || stored[0].Revision != 5 {
		t.Fatalf("placements=%+v err=%v", stored, err)
	}
}
