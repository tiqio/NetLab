package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/domain"
)

func TestOpenMigrationAndAtomicOutbox(t *testing.T) {
	ctx := context.Background()
	db, err := Open(ctx, "file:testdb?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var foreignKeys int
	if err = db.DB.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&foreignKeys); err != nil || foreignKeys != 1 {
		t.Fatalf("foreign keys=%d err=%v", foreignKeys, err)
	}
	repo := NewRepositories(db)
	now := time.Now().UTC()
	lab := domain.Laboratory{ID: domain.NewID(), Name: "lab", Revision: 1, RecoveryPolicy: domain.RecoveryAutoRestore, LifecycleState: "active", CreatedAt: now, UpdatedAt: now}
	event := domain.OutboxEvent{Type: "laboratory.created", LaboratoryID: lab.ID, ResourceType: "laboratory", ResourceID: lab.ID, Revision: 1, OccurredAt: now}
	if err = repo.CreateLaboratory(ctx, lab, event); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.GetLaboratory(ctx, lab.ID); err != nil {
		t.Fatal(err)
	}
	events, err := repo.OutboxAfter(ctx, 0, 10)
	if err != nil || len(events) != 1 {
		t.Fatalf("events=%d err=%v", len(events), err)
	}
}

func TestMigrationIsIdempotent(t *testing.T) {
	db, err := Open(context.Background(), "file:migrate?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = db.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
}
