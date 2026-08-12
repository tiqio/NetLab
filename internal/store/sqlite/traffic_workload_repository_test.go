package sqlite

import (
	"context"
	"github.com/netlab/netlab/internal/domain"
	"testing"
	"time"
)

func TestTrafficWorkloadRepositoryPersists(t *testing.T) {
	ctx := context.Background()
	db, err := Open(ctx, "file:workloads?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	top := NewTopologyRepository(db)
	lab := domain.Laboratory{ID: "lab", Name: "lab", Revision: 1, RecoveryPolicy: domain.RecoveryRemainStopped, LifecycleState: "active", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err = top.CreateLaboratory(ctx, lab); err != nil {
		t.Fatal(err)
	}
	r := NewRepositories(db)
	w := domain.TrafficWorkload{ID: "w", LaboratoryID: "lab", Name: "ping", Revision: 1, Source: domain.TrafficWorkloadEndpoint{Kind: "node", ResourceID: "n"}, Protocol: "icmp", AddressFamily: "ipv4", Destination: domain.TrafficWorkloadDestination{Address: "192.0.2.1"}, IntervalSeconds: 5, TimeoutSeconds: 2, DesiredState: "stopped", ObservedState: "stopped", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err = r.CreateTrafficWorkload(ctx, w); err != nil {
		t.Fatal(err)
	}
	got, err := r.GetTrafficWorkload(ctx, "w")
	if err != nil || got.Name != "ping" {
		t.Fatalf("got=%+v err=%v", got, err)
	}
}
