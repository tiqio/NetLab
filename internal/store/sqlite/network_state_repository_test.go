package sqlite

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/domain"
)

func TestNetworkObservedStateAndEventCommitTogether(t *testing.T) {
	ctx := context.Background()
	database, err := Open(ctx, "file:network-observed-state?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	now := time.Now().UTC()
	topology := NewTopologyRepository(database)
	lab := domain.Laboratory{ID: "lab-net", Name: "net", Revision: 1, RecoveryPolicy: domain.RecoveryAutoRestore, LifecycleState: "active", CreatedAt: now, UpdatedAt: now}
	if err := topology.CreateLaboratory(ctx, lab); err != nil {
		t.Fatal(err)
	}
	repositories := NewRepositories(database)
	object := domain.NetworkObject{ID: "net-1", LaboratoryID: lab.ID, Name: "bridge", Kind: domain.NetworkBridge, Revision: 1, DesiredState: "active", ObservedState: "provisioning", Config: map[string]any{}, CreatedAt: now, UpdatedAt: now}
	if err := repositories.CreateNetworkObject(ctx, object); err != nil {
		t.Fatal(err)
	}
	if err := repositories.SetNetworkObjectState(ctx, object.ID, "active", nil); err != nil {
		t.Fatal(err)
	}
	stored, err := repositories.GetNetworkObject(ctx, object.ID)
	if err != nil || stored.ObservedState != "active" {
		t.Fatalf("stored=%#v err=%v", stored, err)
	}
	var count int
	if err := database.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM outbox_events WHERE event_type='network_object.observed_state_changed' AND resource_id=?`, object.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("events=%d", count)
	}
	if err := repositories.DeleteNetworkObject(ctx, object.ID, object.Revision); err != nil {
		t.Fatal(err)
	}
	if err := database.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM network_object_tombstones WHERE id=? AND revision=?`, object.ID, object.Revision.Next()).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("tombstones=%d", count)
	}
}

func TestNetworkObjectLinkCreatePublishesRevisionedEvent(t *testing.T) {
	ctx := context.Background()
	database, err := Open(ctx, "file:network-object-link-event?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	now := time.Now().UTC()
	topology := NewTopologyRepository(database)
	lab := domain.Laboratory{ID: "lab-link-event", Name: "links", Revision: 1, RecoveryPolicy: domain.RecoveryRemainStopped, LifecycleState: "active", CreatedAt: now, UpdatedAt: now}
	if err = topology.CreateLaboratory(ctx, lab); err != nil {
		t.Fatal(err)
	}
	repositories := NewRepositories(database)
	for _, object := range []domain.NetworkObject{{ID: "a", LaboratoryID: lab.ID, Name: "A", Kind: domain.NetworkSwitchL2, Revision: 1, DesiredState: "active", ObservedState: "active", Config: map[string]any{}, CreatedAt: now, UpdatedAt: now}, {ID: "b", LaboratoryID: lab.ID, Name: "B", Kind: domain.NetworkSwitchL2, Revision: 1, DesiredState: "active", ObservedState: "active", Config: map[string]any{}, CreatedAt: now, UpdatedAt: now}} {
		if err = repositories.CreateNetworkObject(ctx, object); err != nil {
			t.Fatal(err)
		}
	}
	link := domain.NetworkObjectLink{ID: "link-event", LaboratoryID: lab.ID, ObjectAID: "a", PortAName: "swp1", ObjectBID: "b", PortBName: "swp1", Revision: 1, DesiredState: "connected", ObservedState: "pending"}
	if err = repositories.CreateNetworkObjectLink(ctx, link); err != nil {
		t.Fatal(err)
	}
	var revision int
	if err = database.DB.QueryRowContext(ctx, `SELECT revision FROM outbox_events WHERE event_type='network_object_link.created' AND resource_id=?`, link.ID).Scan(&revision); err != nil {
		t.Fatal(err)
	}
	if revision != 1 {
		t.Fatalf("revision=%d", revision)
	}
	problem := &domain.Problem{Code: "runtime_failed", Message: "veth endpoint missing"}
	if err = repositories.SetNetworkObjectLinkState(ctx, link.ID, "failed", problem); err != nil {
		t.Fatal(err)
	}
	var payload string
	if err = database.DB.QueryRowContext(ctx, `SELECT payload_json FROM outbox_events WHERE event_type='network_object_link.state_changed' AND resource_id=?`, link.ID).Scan(&payload); err != nil {
		t.Fatal(err)
	}
	if payload == "" || !strings.Contains(payload, `"observed_state":"failed"`) || !strings.Contains(payload, `"last_error"`) {
		t.Fatalf("payload=%s", payload)
	}
}
