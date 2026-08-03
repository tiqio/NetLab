package sqlite

import (
	"context"
	"testing"

	"github.com/netlab/netlab/internal/app/command"
	"github.com/netlab/netlab/internal/domain"
)

func TestLinkAndInterfaceRuntimeStatePublishOutboxEvents(t *testing.T) {
	ctx := context.Background()
	database, err := Open(ctx, "file:link-runtime-events?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	repository := NewTopologyRepository(database)
	lab, err := command.NewLaboratoryService(repository).Create(ctx, "link-events", "", domain.RecoveryRemainStopped)
	if err != nil {
		t.Fatal(err)
	}
	_, firstInterfaces, err := command.NewNodeService(repository).Create(ctx, lab.ID, "first", "pc", 1)
	if err != nil {
		t.Fatal(err)
	}
	_, secondInterfaces, err := command.NewNodeService(repository).Create(ctx, lab.ID, "second", "pc", 1)
	if err != nil {
		t.Fatal(err)
	}
	link, err := command.NewLinkService(repository).Connect(ctx, lab.ID, firstInterfaces[0].ID, secondInterfaces[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if err = repository.SetLinkObservedState(ctx, link.ID, "connected"); err != nil {
		t.Fatal(err)
	}
	if err = repository.SetLinkObservedState(ctx, link.ID, "connected"); err != nil {
		t.Fatal(err)
	}
	if err = repository.SetInterfaceOperationalState(ctx, firstInterfaces[0].ID, "up"); err != nil {
		t.Fatal(err)
	}
	if err = repository.SetInterfaceOperationalState(ctx, firstInterfaces[0].ID, "up"); err != nil {
		t.Fatal(err)
	}
	var linkEvents, interfaceEvents int
	if err = database.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM outbox_events WHERE event_type='link.observed_state_changed' AND resource_id=?`, link.ID).Scan(&linkEvents); err != nil {
		t.Fatal(err)
	}
	if err = database.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM outbox_events WHERE event_type='interface.operational_state_changed' AND resource_id=?`, firstInterfaces[0].ID).Scan(&interfaceEvents); err != nil {
		t.Fatal(err)
	}
	if linkEvents != 1 || interfaceEvents != 1 {
		t.Fatalf("link_events=%d interface_events=%d", linkEvents, interfaceEvents)
	}
}
