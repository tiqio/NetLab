package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/domain"
)

func TestRuntimeOwnerExistsRecognizesNetworkObjectLink(t *testing.T) {
	ctx := context.Background()
	database, err := Open(ctx, "file:ownership-object-link?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	topology := NewTopologyRepository(database)
	repositories := NewRepositories(database)
	now := time.Now().UTC()
	laboratory := domain.Laboratory{ID: "lab", Name: "lab", Revision: 1, RecoveryPolicy: domain.RecoveryRemainStopped, LifecycleState: "active", CreatedAt: now, UpdatedAt: now}
	if err = topology.CreateLaboratory(ctx, laboratory); err != nil {
		t.Fatal(err)
	}
	for _, object := range []domain.NetworkObject{
		{ID: "left", LaboratoryID: laboratory.ID, Name: "left", Kind: domain.NetworkSwitchL2, Revision: 1, DesiredState: "active", ObservedState: "active", Config: map[string]any{}, CreatedAt: now, UpdatedAt: now},
		{ID: "right", LaboratoryID: laboratory.ID, Name: "right", Kind: domain.NetworkSwitchL3, Revision: 1, DesiredState: "active", ObservedState: "active", Config: map[string]any{}, CreatedAt: now, UpdatedAt: now},
	} {
		if err = repositories.CreateNetworkObject(ctx, object); err != nil {
			t.Fatal(err)
		}
	}
	link := domain.NetworkObjectLink{ID: "object-link", LaboratoryID: laboratory.ID, ObjectAID: "left", PortAName: "eth0", ObjectBID: "right", PortBName: "eth0", Revision: 1, DesiredState: "connected", ObservedState: "connected"}
	if err = repositories.CreateNetworkObjectLink(ctx, link); err != nil {
		t.Fatal(err)
	}

	exists, err := repositories.RuntimeOwnerExists(ctx, "network_object_link", link.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatal("persisted network object link was not recognized as a runtime owner")
	}
}
