package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/domain"
)

func TestCapabilityObservationRevisionAndOutboxAreTransactional(t *testing.T) {
	ctx := context.Background()
	database, err := Open(ctx, "file:capability-observation?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	topology := NewTopologyRepository(database)
	now := time.Now().UTC()
	lab := domain.Laboratory{ID: "lab-cap", Name: "cap", Revision: 1, RecoveryPolicy: domain.RecoveryAutoRestore, LifecycleState: "active", CreatedAt: now, UpdatedAt: now}
	if err := topology.CreateLaboratory(ctx, lab); err != nil {
		t.Fatal(err)
	}
	node := domain.Node{ID: "node-cap", LaboratoryID: lab.ID, Name: "node", Kind: "qemu", Revision: 1, DesiredState: domain.DesiredStopped, ObservedState: domain.ObservedStopped, Config: map[string]any{}, CreatedAt: now, UpdatedAt: now}
	if err := topology.CreateNode(ctx, node, nil); err != nil {
		t.Fatal(err)
	}
	repositories := NewRepositories(database)
	first, err := repositories.ObserveRuntimeCapability(ctx, domain.RuntimeCapabilityObservation{NodeID: node.ID, Capability: domain.CapabilityQMP, State: domain.CapabilityReady, Required: true})
	if err != nil {
		t.Fatal(err)
	}
	second, err := repositories.ObserveRuntimeCapability(ctx, domain.RuntimeCapabilityObservation{NodeID: node.ID, Capability: domain.CapabilityQMP, State: domain.CapabilityUnavailable, Required: true})
	if err != nil {
		t.Fatal(err)
	}
	if first.Revision != 1 || second.Revision != 2 {
		t.Fatalf("revisions %d %d", first.Revision, second.Revision)
	}
	var count int
	if err := database.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM outbox_events WHERE event_type='node.capability_changed' AND resource_id=?`, node.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("events=%d", count)
	}
}
