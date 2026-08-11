package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/domain"
)

func TestRecoveryFinalizesExhaustedConnectionsAndPreservesActiveWork(t *testing.T) {
	ctx := context.Background()
	database, err := Open(ctx, "file:"+string(domain.NewID())+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	repositories := NewRepositories(database)
	now := time.Now().UTC()
	laboratoryID := domain.NewID()
	if _, err = database.DB.ExecContext(ctx, `INSERT INTO laboratories(id,name,revision,recovery_policy,lifecycle_state,created_at,updated_at) VALUES(?,?,1,'auto_restore','active',?,?)`, laboratoryID, "recovery-finalization", now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano)); err != nil {
		t.Fatal(err)
	}
	objects := make([]domain.NetworkObject, 6)
	for index := range objects {
		objects[index] = domain.NetworkObject{ID: domain.NewID(), LaboratoryID: laboratoryID, Name: "switch-" + string(rune('a'+index)), Kind: domain.NetworkSwitchL2, Revision: 1, DesiredState: "active", ObservedState: "active", Config: map[string]any{"ports": []any{map[string]any{"name": "eth0"}}}, CreatedAt: now, UpdatedAt: now}
		if err = repositories.CreateNetworkObject(ctx, objects[index]); err != nil {
			t.Fatal(err)
		}
	}
	links := []domain.NetworkObjectLink{
		{ID: domain.NewID(), LaboratoryID: laboratoryID, ObjectAID: objects[0].ID, PortAName: "eth0", ObjectBID: objects[1].ID, PortBName: "eth0", Revision: 1, DesiredState: "connected", ObservedState: "pending"},
		{ID: domain.NewID(), LaboratoryID: laboratoryID, ObjectAID: objects[2].ID, PortAName: "eth0", ObjectBID: objects[3].ID, PortBName: "eth0", Revision: 1, DesiredState: "disconnected", ObservedState: "disconnecting"},
		{ID: domain.NewID(), LaboratoryID: laboratoryID, ObjectAID: objects[4].ID, PortAName: "eth0", ObjectBID: objects[5].ID, PortBName: "eth0", Revision: 1, DesiredState: "connected", ObservedState: "pending"},
	}
	for _, link := range links {
		if err = repositories.CreateNetworkObjectLink(ctx, link); err != nil {
			t.Fatal(err)
		}
	}
	for _, task := range []struct {
		id, resourceID domain.ID
		state          string
	}{
		{domain.NewID(), links[0].ID, "failed"},
		{domain.NewID(), links[1].ID, "cancelled"},
		{domain.NewID(), links[2].ID, "failed"},
		{domain.NewID(), links[2].ID, "queued"},
	} {
		if _, err = database.DB.ExecContext(ctx, `INSERT INTO operation_tasks(id,kind,resource_type,resource_id,state,progress_current,progress_total,input_json,created_at) VALUES(?,'network_object_link.reconcile','network_object_link',?, ?,0,1,'{}',?)`, task.id, task.resourceID, task.state, now.Format(time.RFC3339Nano)); err != nil {
			t.Fatal(err)
		}
	}
	outcomes, err := repositories.RecoverTopologyConnectionReservations(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(outcomes) != 2 || outcomes[0].State != "failed" || outcomes[1].State != "failed" {
		t.Fatalf("outcomes=%+v", outcomes)
	}
	for _, link := range links[:2] {
		value, getErr := repositories.GetNetworkObjectLink(ctx, link.ID)
		if getErr != nil {
			t.Fatal(getErr)
		}
		if value.ObservedState != "failed" || value.LastError == nil || value.LastError.Code != "connection_recovery_exhausted" || !value.LastError.Retryable {
			t.Fatalf("link=%+v", value)
		}
		var reservations int
		if err = database.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM topology_endpoint_reservations WHERE resource_type='network_object_link' AND resource_id=?`, link.ID).Scan(&reservations); err != nil || reservations != 2 {
			t.Fatalf("reservations=%d err=%v", reservations, err)
		}
	}
	active, err := repositories.GetNetworkObjectLink(ctx, links[2].ID)
	if err != nil || active.ObservedState != "pending" || active.LastError != nil {
		t.Fatalf("active=%+v err=%v", active, err)
	}
	rows, err := database.DB.QueryContext(ctx, `SELECT resource_id,sequence FROM outbox_events WHERE event_type='network_object_link.state_changed' AND json_extract(payload_json,'$.last_error.code')='connection_recovery_exhausted' ORDER BY sequence`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var eventIDs []domain.ID
	var previous int64
	for rows.Next() {
		var id domain.ID
		var sequence int64
		if err = rows.Scan(&id, &sequence); err != nil {
			t.Fatal(err)
		}
		if sequence <= previous {
			t.Fatalf("outbox sequence=%d previous=%d", sequence, previous)
		}
		previous = sequence
		eventIDs = append(eventIDs, id)
	}
	if len(eventIDs) != 2 || eventIDs[0] != links[0].ID || eventIDs[1] != links[1].ID {
		t.Fatalf("event ids=%v", eventIDs)
	}
	second, err := repositories.RecoverTopologyConnectionReservations(ctx)
	if err != nil || len(second) != 0 {
		t.Fatalf("second=%+v err=%v", second, err)
	}
}
