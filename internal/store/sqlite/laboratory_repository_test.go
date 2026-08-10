package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/app/command"
	"github.com/netlab/netlab/internal/domain"
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

func TestCreateNodeReturnsStructuredNameConflict(t *testing.T) {
	ctx := context.Background()
	database, err := Open(ctx, "file:node-name-conflict?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	repository := NewTopologyRepository(database)
	laboratory, err := command.NewLaboratoryService(repository).Create(ctx, "names", "", domain.RecoveryAutoRestore)
	if err != nil {
		t.Fatal(err)
	}
	nodes := command.NewNodeService(repository)
	if _, _, err = nodes.Create(ctx, laboratory.ID, "Ubuntu", "pc", 1); err != nil {
		t.Fatal(err)
	}
	_, _, err = nodes.Create(ctx, laboratory.ID, "Ubuntu", "pc", 1)
	var problem domain.Problem
	if !errors.As(err, &problem) {
		t.Fatalf("expected structured problem, got %v", err)
	}
	if problem.Code != "node_name_conflict" {
		t.Fatalf("expected node_name_conflict, got %+v", problem)
	}
	fields, ok := problem.Details["fields"].(map[string]string)
	if !ok || fields["name"] == "" {
		t.Fatalf("expected name field guidance, got %+v", problem.Details)
	}
}

func TestCreateNodeWithPlacementRollsBackEveryRecord(t *testing.T) {
	ctx := context.Background()
	database, err := Open(ctx, "file:node-placement-rollback?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	repository := NewTopologyRepository(database)
	laboratory, err := command.NewLaboratoryService(repository).Create(ctx, "rollback", "", domain.RecoveryAutoRestore)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	node := domain.Node{ID: "rollback-node", LaboratoryID: laboratory.ID, Name: "rollback", Kind: string(domain.RuntimeDocker), Revision: 1, DesiredState: domain.DesiredStopped, ObservedState: domain.ObservedStopped, CPUCount: 1, MemoryMiB: 64, InterfaceLimit: 1, ProcessLimit: 16, Config: map[string]any{}, CreatedAt: now, UpdatedAt: now}
	interfaces := []domain.Interface{{ID: "rollback-if", NodeID: node.ID, Slot: 0, Name: "eth0", Revision: 1}}
	_, _, err = repository.CreateNodeWithPlacement(ctx, node, interfaces, laboratory.Revision, &domain.PlacementIntent{FootprintClass: domain.FootprintNetworkObjectStandard}, "test")
	if err == nil {
		t.Fatal("expected invalid footprint failure")
	}
	for table, id := range map[string]string{"nodes": string(node.ID), "interfaces": string(interfaces[0].ID), "topology_placements": string(node.ID)} {
		var count int
		column := "id"
		if table == "topology_placements" {
			column = "resource_id"
		}
		if err = database.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+table+` WHERE `+column+`=?`, id).Scan(&count); err != nil || count != 0 {
			t.Fatalf("table=%s count=%d err=%v", table, count, err)
		}
	}
	stored, err := repository.GetLaboratory(ctx, laboratory.ID)
	if err != nil || stored.Revision != laboratory.Revision {
		t.Fatalf("laboratory=%+v err=%v", stored, err)
	}
	var events int
	if err = database.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM outbox_events WHERE resource_id IN (?,?)`, node.ID, interfaces[0].ID).Scan(&events); err != nil || events != 0 {
		t.Fatalf("events=%d err=%v", events, err)
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
	repositories := NewRepositories(database)
	now := time.Now().UTC()
	objectA := domain.NetworkObject{ID: domain.NewID(), LaboratoryID: laboratory.ID, Name: "switch-a", Kind: domain.NetworkSwitchL2, Revision: 1, DesiredState: "active", ObservedState: "active", Config: map[string]any{"ports": []any{map[string]any{"name": "eth0"}}}, CreatedAt: now, UpdatedAt: now}
	objectB := domain.NetworkObject{ID: domain.NewID(), LaboratoryID: laboratory.ID, Name: "switch-b", Kind: domain.NetworkSwitchL2, Revision: 1, DesiredState: "active", ObservedState: "active", Config: map[string]any{"ports": []any{map[string]any{"name": "eth0"}}}, CreatedAt: now, UpdatedAt: now}
	if err = repositories.CreateNetworkObject(ctx, objectA); err != nil {
		t.Fatal(err)
	}
	if err = repositories.CreateNetworkObject(ctx, objectB); err != nil {
		t.Fatal(err)
	}
	objectLink := domain.NetworkObjectLink{ID: domain.NewID(), LaboratoryID: laboratory.ID, ObjectAID: objectA.ID, PortAName: "eth0", ObjectBID: objectB.ID, PortBName: "eth0", Revision: 1, DesiredState: "connected", ObservedState: "connected"}
	if err = repositories.CreateNetworkObjectLink(ctx, objectLink); err != nil {
		t.Fatal(err)
	}
	for _, endpoint := range []string{"namespace-a:eth0", "namespace-b:eth0"} {
		if err = repositories.UpsertRuntimeOwnership(ctx, "network_object_link", objectLink.ID, "network_object_link_endpoint", endpoint, nil, "active"); err != nil {
			t.Fatal(err)
		}
	}
	objectLinkTaskID := domain.NewID()
	if _, err = database.DB.ExecContext(ctx, `INSERT INTO operation_tasks(id,kind,resource_type,resource_id,state,progress_current,progress_total,input_json,created_at) VALUES(?,'network_object_link.create','network_object_link',?,'failed',1,2,'{}',?)`, objectLinkTaskID, objectLink.ID, now.Format(time.RFC3339Nano)); err != nil {
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
	if err = database.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM runtime_ownership WHERE resource_type='network_object_link' AND resource_id=?`, objectLink.ID).Scan(&ownershipCount); err != nil {
		t.Fatal(err)
	}
	if ownershipCount != 0 {
		t.Fatalf("expected purged network object link ownership to be deleted, got %d", ownershipCount)
	}
	for table, target := range map[string][2]string{
		"network_object_links":           {"id", string(objectLink.ID)},
		"topology_endpoint_reservations": {"resource_id", string(objectLink.ID)},
		"operation_tasks":                {"id", string(objectLinkTaskID)},
	} {
		var count int
		if err = database.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+table+` WHERE `+target[0]+`=?`, target[1]).Scan(&count); err != nil || count != 0 {
			t.Fatalf("table=%s count=%d err=%v", table, count, err)
		}
	}
}
