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
	if err = NewTopologyRepository(database).ImportTopology(ctx, laboratory, nil, nil, nil, objects, links); err != nil {
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
	if err = NewTopologyRepository(database).ImportTopology(ctx, laboratory, nil, nil, nil, objects, links); err == nil {
		t.Fatal("expected occupied endpoint conflict")
	}
	var laboratoryCount int
	if err = database.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM laboratories WHERE id=?`, laboratory.ID).Scan(&laboratoryCount); err != nil {
		t.Fatal(err)
	}
	if laboratoryCount != 0 {
		t.Fatalf("partial import remained: laboratories=%d", laboratoryCount)
	}
}
