package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/domain"
)

func TestNATServiceObservationPersistence(t *testing.T) {
	ctx := context.Background()
	database, err := Open(ctx, "file:nat-service-observation?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	now := time.Now().UTC()
	topology := NewTopologyRepository(database)
	lab := domain.Laboratory{ID: "lab-nat-service", Name: "nat-service", Revision: 1, RecoveryPolicy: domain.RecoveryAutoRestore, LifecycleState: "active", CreatedAt: now, UpdatedAt: now}
	if err := topology.CreateLaboratory(ctx, lab); err != nil {
		t.Fatal(err)
	}
	repositories := NewRepositories(database)
	object := domain.NetworkObject{ID: "nat-service", LaboratoryID: lab.ID, Name: "nat", Kind: domain.NetworkNAT, Revision: 1, DesiredState: "active", ObservedState: "active", Config: map[string]any{"ipv4_prefix": "10.99.0.0/24", "uplink": "eth0"}, CreatedAt: now, UpdatedAt: now}
	if err := repositories.CreateNetworkObject(ctx, object); err != nil {
		t.Fatal(err)
	}
	observation := domain.NATServiceObservation{NetworkObjectID: object.ID, ConfigDigest: domain.DigestBytes([]byte("config")), UnitName: "netlab-dnsmasq.service", ConfigPath: "/run/netlab/dnsmasq.conf", LeasePath: "/run/netlab/dnsmasq.leases", State: "active", ObservedAt: now}
	if err := repositories.SaveNATServiceObservation(ctx, observation); err != nil {
		t.Fatal(err)
	}
	var state string
	if err := database.DB.QueryRowContext(ctx, `SELECT state FROM nat_service_observations WHERE network_object_id=?`, object.ID).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != "active" {
		t.Fatalf("state=%s", state)
	}
}
