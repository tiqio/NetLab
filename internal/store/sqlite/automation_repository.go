package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/netlab/netlab/internal/domain"
)

func (r *Repositories) GetIdempotency(ctx context.Context, scope, key string) (domain.IdempotencyRecord, error) {
	var record domain.IdempotencyRecord
	var createdAt, expiresAt string
	err := r.database.DB.QueryRowContext(ctx, `SELECT scope,key,request_fingerprint,state,COALESCE(status_code,0),COALESCE(response_json,''),COALESCE(error_json,''),created_at,expires_at FROM idempotency_records WHERE scope=? AND key=?`, scope, key).Scan(
		&record.Scope, &record.Key, &record.RequestFingerprint, &record.State, &record.StatusCode, &record.Response, &record.Error, &createdAt, &expiresAt,
	)
	if err == sql.ErrNoRows {
		return record, ErrNotFound
	}
	if err != nil {
		return record, err
	}
	record.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	record.ExpiresAt, _ = time.Parse(time.RFC3339Nano, expiresAt)
	return record, nil
}

func (r *Repositories) CreateIdempotency(ctx context.Context, record domain.IdempotencyRecord) error {
	_, err := r.database.DB.ExecContext(ctx, `INSERT INTO idempotency_records(scope,key,request_fingerprint,state,created_at,expires_at) VALUES(?,?,?,?,?,?)`, record.Scope, record.Key, record.RequestFingerprint, record.State, record.CreatedAt.Format(time.RFC3339Nano), record.ExpiresAt.Format(time.RFC3339Nano))
	return err
}

func (r *Repositories) CompleteIdempotency(ctx context.Context, record domain.IdempotencyRecord) error {
	_, err := r.database.DB.ExecContext(ctx, `UPDATE idempotency_records SET state=?,status_code=?,response_json=?,error_json=? WHERE scope=? AND key=?`, record.State, record.StatusCode, nullableBytes(record.Response, len(record.Response) > 0), nullableBytes(record.Error, len(record.Error) > 0), record.Scope, record.Key)
	return err
}

func (r *Repositories) DeleteIdempotency(ctx context.Context, scope, key string) error {
	_, err := r.database.DB.ExecContext(ctx, `DELETE FROM idempotency_records WHERE scope=? AND key=?`, scope, key)
	return err
}

func (r *Repositories) ListTasks(ctx context.Context, limit int) ([]domain.OperationTask, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.database.DB.QueryContext(ctx, `SELECT id FROM operation_tasks ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	var ids []domain.ID
	for rows.Next() {
		var id domain.ID
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	values := make([]domain.OperationTask, 0, len(ids))
	for _, id := range ids {
		value, getErr := r.GetTask(ctx, id)
		if getErr != nil {
			return nil, getErr
		}
		values = append(values, value)
	}
	return values, nil
}

func (r *Repositories) RequestTaskCancellation(ctx context.Context, id domain.ID, at time.Time) (domain.OperationTask, error) {
	result, err := r.database.DB.ExecContext(ctx, `UPDATE operation_tasks SET cancel_requested_at=?,state='cancelling' WHERE id=? AND state IN ('queued','running')`, at.Format(time.RFC3339Nano), id)
	if err != nil {
		return domain.OperationTask{}, err
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		if _, getErr := r.GetTask(ctx, id); getErr != nil {
			return domain.OperationTask{}, getErr
		}
	}
	return r.GetTask(ctx, id)
}

func (r *Repositories) CreateAuditEvent(ctx context.Context, event domain.AuditEvent) error {
	details, _ := json.Marshal(event.Details)
	_, err := r.database.DB.ExecContext(ctx, `INSERT INTO audit_events(id,actor_class,action,resource_type,resource_id,task_id,outcome,correlation_id,details_json,occurred_at) VALUES(?,?,?,?,?,?,?,?,?,?)`, event.ID, event.ActorClass, event.Action, event.ResourceType, event.ResourceID, nullable(string(event.TaskID)), event.Outcome, event.CorrelationID, details, event.OccurredAt.Format(time.RFC3339Nano))
	return err
}

func (r *Repositories) ListAuditEvents(ctx context.Context, limit int) ([]domain.AuditEvent, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.database.DB.QueryContext(ctx, `SELECT id,actor_class,action,resource_type,resource_id,COALESCE(task_id,''),outcome,correlation_id,COALESCE(details_json,'{}'),occurred_at FROM audit_events ORDER BY occurred_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []domain.AuditEvent
	for rows.Next() {
		var event domain.AuditEvent
		var details []byte
		var occurredAt string
		if err = rows.Scan(&event.ID, &event.ActorClass, &event.Action, &event.ResourceType, &event.ResourceID, &event.TaskID, &event.Outcome, &event.CorrelationID, &details, &occurredAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(details, &event.Details)
		event.OccurredAt, _ = time.Parse(time.RFC3339Nano, occurredAt)
		events = append(events, event)
	}
	return events, rows.Err()
}

func (r *Repositories) CreateArtifact(ctx context.Context, artifact domain.Artifact) error {
	_, err := r.database.DB.ExecContext(ctx, `INSERT INTO artifacts(id,kind,path,media_type,size_bytes,sha256,owner_type,owner_id,created_at,expires_at) VALUES(?,?,?,?,?,?,?,?,?,?)`, artifact.ID, artifact.Kind, artifact.Path, artifact.MediaType, artifact.SizeBytes, artifact.SHA256, artifact.OwnerType, artifact.OwnerID, artifact.CreatedAt.Format(time.RFC3339Nano), formatTime(artifact.ExpiresAt))
	return err
}

func (r *Repositories) GetArtifact(ctx context.Context, id domain.ID) (domain.Artifact, error) {
	var artifact domain.Artifact
	var createdAt string
	var expiresAt sql.NullString
	err := r.database.DB.QueryRowContext(ctx, `SELECT id,kind,path,media_type,size_bytes,sha256,owner_type,owner_id,created_at,expires_at FROM artifacts WHERE id=? AND deletion_state='active'`, id).Scan(&artifact.ID, &artifact.Kind, &artifact.Path, &artifact.MediaType, &artifact.SizeBytes, &artifact.SHA256, &artifact.OwnerType, &artifact.OwnerID, &createdAt, &expiresAt)
	if err == sql.ErrNoRows {
		return artifact, ErrNotFound
	}
	if err != nil {
		return artifact, err
	}
	artifact.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	artifact.ExpiresAt = parseNullTime(expiresAt)
	return artifact, nil
}
