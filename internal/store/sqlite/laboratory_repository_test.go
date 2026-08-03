package sqlite

import (
	"context"
	"errors"
	"github.com/netlab/netlab/internal/app/command"
	"github.com/netlab/netlab/internal/domain"
	"testing"
	"time"
)

func TestLaboratorySnapshotAndRevision(t *testing.T) {
	ctx := context.Background()
	db, err := Open(ctx, "file:labs?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewTopologyRepository(db)
	labs := command.NewLaboratoryService(repo)
	lab, err := labs.Create(ctx, "shared", "", domain.RecoveryAutoRestore)
	if err != nil {
		t.Fatal(err)
	}
	nodes := command.NewNodeService(repo)
	if _, _, err = nodes.Create(ctx, lab.ID, "pc1", "pc", 1); err != nil {
		t.Fatal(err)
	}
	snapshot, err := repo.Snapshot(ctx, lab.ID)
	if err != nil || len(snapshot.Nodes) != 1 || len(snapshot.Interfaces) != 1 {
		t.Fatalf("snapshot=%+v err=%v", snapshot, err)
	}
	if _, err = labs.Update(ctx, lab.ID, 99, "bad", "", domain.RecoveryAutoRestore); err == nil {
		t.Fatal("expected conflict")
	}
}

func TestFinalizeLaboratoryDeletionRemovesDeleteFailedRetry(t *testing.T) {
	ctx := context.Background()
	database, err := Open(ctx, "file:delete-retry?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	repository := NewTopologyRepository(database)
	laboratory := domain.Laboratory{ID: domain.NewID(), Name: "retry-delete", Revision: 1, RecoveryPolicy: domain.RecoveryAutoRestore, LifecycleState: "active", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if err = repository.CreateLaboratory(ctx, laboratory); err != nil {
		t.Fatal(err)
	}
	node, _, err := command.NewNodeService(repository).Create(ctx, laboratory.ID, "owned-node", "pc", 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = database.DB.ExecContext(ctx, `INSERT INTO runtime_ownership(resource_type,resource_id,object_kind,object_name,metadata_json,cleanup_state) VALUES('unknown',?,'test_object','owned-object','{}','missing_validation_required')`, node.ID); err != nil {
		t.Fatal(err)
	}
	captureID := domain.NewID()
	if _, err = database.DB.ExecContext(ctx, `INSERT INTO operation_tasks(id,kind,resource_type,resource_id,requested_revision,state,progress_current,progress_total,input_json,created_at) VALUES(?,?,?,?,0,'succeeded',2,2,?,?)`, domain.NewID(), "capture.start", "capture", captureID, `{"request":{"laboratory_id":"`+string(laboratory.ID)+`"}}`, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		t.Fatal(err)
	}
	if _, err = database.DB.ExecContext(ctx, `INSERT INTO runtime_ownership(resource_type,resource_id,object_kind,object_name,metadata_json,cleanup_state) VALUES('capture',?,'capture_worker_process','worker','{}','missing_validation_required')`, captureID); err != nil {
		t.Fatal(err)
	}
	if _, err = database.DB.ExecContext(ctx, `UPDATE laboratories SET lifecycle_state='delete_failed' WHERE id=?`, laboratory.ID); err != nil {
		t.Fatal(err)
	}
	if err = repository.FinalizeLaboratoryDeletion(ctx, laboratory.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = repository.GetLaboratory(ctx, laboratory.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected retried laboratory deletion to finalize, got %v", err)
	}
	var ownershipCount int
	if err = database.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM runtime_ownership WHERE resource_id=?`, node.ID).Scan(&ownershipCount); err != nil {
		t.Fatal(err)
	}
	if ownershipCount != 0 {
		t.Fatalf("expected owned runtime records to be deleted, got %d", ownershipCount)
	}
	if err = database.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM runtime_ownership WHERE resource_id=?`, captureID).Scan(&ownershipCount); err != nil {
		t.Fatal(err)
	}
	if ownershipCount != 0 {
		t.Fatalf("expected purged capture ownership to be deleted, got %d", ownershipCount)
	}
}
