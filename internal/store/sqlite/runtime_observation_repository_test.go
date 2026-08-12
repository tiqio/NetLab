package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/domain"
)

func TestRuntimeObservationRepositoryPersistsObjectLinkAttribution(t *testing.T) {
	ctx := context.Background()
	database, err := Open(ctx, "file:runtime-observations?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	repositories := NewRepositories(database)
	now := time.Now().UTC().Truncate(time.Microsecond)
	createObservationLaboratory(t, ctx, repositories, now)
	capture := domain.Capture{ID: "capture-1", LaboratoryID: "lab-1", SourceType: "network_object_link", SourceID: "object-link-1", Purpose: "traffic_filter", ParentResourceID: "filter-1", Filter: "icmp", Format: "pcap", State: "completed", Retain: true, MaxBytes: 4096, BytesWritten: 512, Packets: 4, ArtifactID: "artifact-1", ArtifactURL: "/api/v1/artifacts/artifact-1", CompletionReason: "link_deleted", CreatedAt: now, StartedAt: &now, FinishedAt: &now}
	if err = repositories.SaveCaptureObservation(ctx, capture); err != nil {
		t.Fatal(err)
	}
	loaded, err := repositories.GetCaptureObservation(ctx, capture.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.SourceType != "network_object_link" || loaded.SourceID != "object-link-1" || loaded.CompletionReason != "link_deleted" || loaded.ArtifactURL != capture.ArtifactURL || loaded.Packets != 4 {
		t.Fatalf("capture=%+v", loaded)
	}

	lastMatch := now.Add(time.Second)
	filter := domain.TrafficFilter{ID: "filter-1", LaboratoryID: "lab-1", Expression: "icmp", Color: "#22c55e", State: "running", MaxObservations: 100, NetworkObjectLinkIDs: []domain.ID{"object-link-1", "object-link-2"}, FingerprintCount: 2, MatchedPackets: 3, MatchedBytes: 192, FirstMatchAt: &now, LastMatchAt: &lastMatch, CreatedAt: now, Observations: []domain.TrafficObservation{{Fingerprint: "icmp:a", ResourceType: "network_object_link", ResourceID: "object-link-1", NetworkObjectLinkID: "object-link-1", Direction: "a_to_b", FirstSeen: now, LastSeen: now, Count: 2, Bytes: 128}, {Fingerprint: "icmp:b", ResourceType: "network_object_link", ResourceID: "object-link-2", NetworkObjectLinkID: "object-link-2", Direction: "ambiguous", FirstSeen: now, LastSeen: now, Count: 1, Bytes: 64}}}
	if err = repositories.SaveTrafficFilterObservation(ctx, filter); err != nil {
		t.Fatal(err)
	}
	loadedFilter, err := repositories.GetTrafficFilterObservation(ctx, filter.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loadedFilter.Color != filter.Color || len(loadedFilter.NetworkObjectLinkIDs) != 2 || len(loadedFilter.Observations) != 2 || loadedFilter.Observations[0].ResourceType != "network_object_link" || loadedFilter.FingerprintCount != 2 || loadedFilter.MatchedPackets != 3 || loadedFilter.MatchedBytes != 192 || loadedFilter.FirstMatchAt == nil || loadedFilter.LastMatchAt == nil {
		t.Fatalf("filter=%+v", loadedFilter)
	}
	listed, err := repositories.ListTrafficFilterObservations(ctx, "lab-1")
	if err != nil || len(listed) != 1 || listed[0].ID != filter.ID || len(listed[0].Observations) != 2 {
		t.Fatalf("listed=%+v err=%v", listed, err)
	}
}

func TestCompleteLinkDeletedObservationCommitsTaskAuditAndOutbox(t *testing.T) {
	ctx := context.Background()
	database, err := Open(ctx, "file:runtime-observation-completion?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	repositories := NewRepositories(database)
	now := time.Now().UTC()
	createObservationLaboratory(t, ctx, repositories, now)
	task := domain.OperationTask{ID: "task-1", Kind: "capture.stop", ResourceType: "capture", ResourceID: "capture-1", State: domain.TaskRunning, ProgressTotal: 1, CreatedAt: now, StartedAt: &now}
	if err = repositories.CreateTask(ctx, task); err != nil {
		t.Fatal(err)
	}
	finished := now.Add(time.Second)
	task.State, task.ProgressCurrent, task.FinishedAt = domain.TaskSucceeded, 1, &finished
	capture := domain.Capture{ID: "capture-1", LaboratoryID: "lab-1", SourceType: "network_object_link", SourceID: "object-link-1", Format: "pcap", State: "completed", Retain: true, MaxBytes: 4096, CompletionReason: "link_deleted", CreatedAt: now, FinishedAt: &finished}
	audit := domain.AuditEvent{ID: "audit-1", ActorClass: "system", Action: "capture.completed", ResourceType: "capture", ResourceID: capture.ID, TaskID: task.ID, Outcome: "succeeded", CorrelationID: "object-link-1", Details: map[string]any{"completion_reason": "link_deleted"}, OccurredAt: finished}
	event := domain.OutboxEvent{Type: "capture.completed", LaboratoryID: capture.LaboratoryID, ResourceType: "capture", ResourceID: capture.ID, TaskID: task.ID, Data: map[string]any{"completion_reason": "link_deleted", "source_type": "network_object_link", "source_id": "object-link-1"}, OccurredAt: finished}
	if err = repositories.CompleteLinkDeletedCapture(ctx, capture, task, audit, event); err != nil {
		t.Fatal(err)
	}
	loadedTask, err := repositories.GetTask(ctx, task.ID)
	if err != nil || loadedTask.State != domain.TaskSucceeded || loadedTask.ProgressCurrent != 1 {
		t.Fatalf("task=%+v err=%v", loadedTask, err)
	}
	loadedCapture, err := repositories.GetCaptureObservation(ctx, capture.ID)
	if err != nil || loadedCapture.CompletionReason != "link_deleted" {
		t.Fatalf("capture=%+v err=%v", loadedCapture, err)
	}
	var auditCount, eventCount int
	if err = database.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_events WHERE id='audit-1' AND json_extract(details_json,'$.completion_reason')='link_deleted'`).Scan(&auditCount); err != nil {
		t.Fatal(err)
	}
	if err = database.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM outbox_events WHERE event_type='capture.completed' AND json_extract(payload_json,'$.completion_reason')='link_deleted'`).Scan(&eventCount); err != nil {
		t.Fatal(err)
	}
	if auditCount != 1 || eventCount != 1 {
		t.Fatalf("audit=%d event=%d", auditCount, eventCount)
	}
}

func TestCompleteLinkDeletedObservationRollsBackWhenTaskIsMissing(t *testing.T) {
	ctx := context.Background()
	database, err := Open(ctx, "file:runtime-observation-rollback?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	repositories := NewRepositories(database)
	now := time.Now().UTC()
	createObservationLaboratory(t, ctx, repositories, now)
	capture := domain.Capture{ID: "capture-missing-task", LaboratoryID: "lab-1", SourceType: "network_object_link", SourceID: "object-link-1", Format: "pcap", State: "completed", MaxBytes: 1024, CompletionReason: "link_deleted", CreatedAt: now, FinishedAt: &now}
	task := domain.OperationTask{ID: "missing-task", State: domain.TaskSucceeded, FinishedAt: &now}
	audit := domain.AuditEvent{ID: "missing-task-audit", ActorClass: "system", Action: "capture.completed", ResourceType: "capture", ResourceID: capture.ID, TaskID: task.ID, Outcome: "succeeded", CorrelationID: "object-link-1", OccurredAt: now}
	event := domain.OutboxEvent{Type: "capture.completed", LaboratoryID: capture.LaboratoryID, ResourceType: "capture", ResourceID: capture.ID, TaskID: task.ID, OccurredAt: now}
	if err = repositories.CompleteLinkDeletedCapture(ctx, capture, task, audit, event); err != ErrNotFound {
		t.Fatalf("err=%v", err)
	}
	var captureCount int
	if err = database.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM captures WHERE id=?`, capture.ID).Scan(&captureCount); err != nil {
		t.Fatal(err)
	}
	if captureCount != 0 {
		t.Fatalf("partial capture persisted: %d", captureCount)
	}
}

func createObservationLaboratory(t *testing.T, ctx context.Context, repositories *Repositories, now time.Time) {
	t.Helper()
	laboratory := domain.Laboratory{ID: "lab-1", Name: "runtime observations", Revision: 1, RecoveryPolicy: domain.RecoveryRemainStopped, LifecycleState: "active", CreatedAt: now, UpdatedAt: now}
	event := domain.OutboxEvent{Type: "laboratory.created", LaboratoryID: laboratory.ID, ResourceType: "laboratory", ResourceID: laboratory.ID, Revision: laboratory.Revision, OccurredAt: now}
	if err := repositories.CreateLaboratory(ctx, laboratory, event); err != nil {
		t.Fatal(err)
	}
}
