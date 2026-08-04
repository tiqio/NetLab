package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestMaintenanceStatsAndPruneKeepActiveAndReplayFloor(t *testing.T) {
	ctx := context.Background()
	database, err := Open(ctx, filepath.Join(t.TempDir(), "netlab.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	repositories := NewRepositories(database)
	old := time.Now().UTC().Add(-60 * 24 * time.Hour).Format(time.RFC3339Nano)
	_, err = database.DB.ExecContext(ctx, `
INSERT INTO operation_tasks(id,kind,resource_type,resource_id,state,progress_current,progress_total,created_at,finished_at) VALUES
('finished','test','laboratory','lab','succeeded',1,1,?,?),
('running','test','laboratory','lab','running',0,1,?,NULL);
INSERT INTO audit_events(id,actor_class,action,resource_type,resource_id,outcome,correlation_id,occurred_at) VALUES('audit','system','test','laboratory','lab','ok','correlation',?);
INSERT INTO outbox_events(event_type,resource_type,resource_id,revision,payload_json,occurred_at,published_at) VALUES
('old','laboratory','lab',1,'{}',?,?),
('floor','laboratory','lab',1,'{}',?,?);`, old, old, old, old, old, old, old, old, old)
	if err != nil {
		t.Fatal(err)
	}
	stats, err := repositories.MaintenanceStats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if stats.OperationTasks != 2 || stats.AuditEvents != 1 || stats.OutboxEvents != 2 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
	pruned, err := repositories.PruneHistory(ctx, time.Now().UTC().Add(-30*24*time.Hour), 100, 1)
	if err != nil {
		t.Fatal(err)
	}
	if pruned.OperationTasks != 1 || pruned.AuditEvents != 1 || pruned.OutboxEvents != 1 {
		t.Fatalf("unexpected prune result: %+v", pruned)
	}
	stats, err = repositories.MaintenanceStats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if stats.OperationTasks != 1 || stats.OutboxEvents != 1 {
		t.Fatalf("active task or replay floor was removed: %+v", stats)
	}
}
