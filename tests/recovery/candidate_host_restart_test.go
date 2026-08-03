package recovery_test

import (
	"context"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/app/command"
	"github.com/netlab/netlab/internal/domain"
	storesqlite "github.com/netlab/netlab/internal/store/sqlite"
)

func TestCandidateHostRestartRetainsNATHelperOwnershipObservation(t *testing.T) {
	db, err := storesqlite.Open(context.Background(), "file:candidate-host-restart?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repository := storesqlite.NewRepositories(db)
	laboratory, err := command.NewLaboratoryService(storesqlite.NewTopologyRepository(db)).Create(context.Background(), "restart", "", domain.RecoveryAutoRestore)
	if err != nil {
		t.Fatal(err)
	}
	object := domain.NetworkObject{ID: domain.NewID(), LaboratoryID: laboratory.ID, Name: "nat", Kind: domain.NetworkNAT, Revision: 1, DesiredState: "active", ObservedState: "active", Config: map[string]any{"ipv4_prefix": "10.0.0.0/24", "uplink": "eth0"}}
	if err = repository.CreateNetworkObject(context.Background(), object); err != nil {
		t.Fatal(err)
	}
	observation := domain.NATServiceObservation{NetworkObjectID: object.ID, ConfigDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", UnitName: "netlab-dnsmasq.service", ConfigPath: "/run/netlab/dnsmasq.conf", LeasePath: "/run/netlab/dnsmasq.leases", State: "active", ObservedAt: time.Now().UTC()}
	if err = repository.SaveNATServiceObservation(context.Background(), observation); err != nil {
		t.Fatal(err)
	}
	var unitName, state string
	err = db.DB.QueryRowContext(context.Background(), `SELECT unit_name,state FROM nat_service_observations WHERE network_object_id=?`, observation.NetworkObjectID).Scan(&unitName, &state)
	if err != nil || unitName != observation.UnitName || state != "active" {
		t.Fatalf("unit=%s state=%s err=%v", unitName, state, err)
	}
}
