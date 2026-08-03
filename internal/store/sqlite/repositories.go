package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/netlab/netlab/internal/domain"
)

var ErrNotFound = domain.ErrNotFound

type Repositories struct{ database *Database }

func NewRepositories(database *Database) *Repositories { return &Repositories{database: database} }

func (r *Repositories) CreateLaboratory(ctx context.Context, lab domain.Laboratory, event domain.OutboxEvent) error {
	return r.database.Write(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO laboratories(id,name,description,revision,recovery_policy,lifecycle_state,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?)`, lab.ID, lab.Name, lab.Description, lab.Revision, lab.RecoveryPolicy, lab.LifecycleState, lab.CreatedAt.Format(time.RFC3339Nano), lab.UpdatedAt.Format(time.RFC3339Nano))
		if err != nil {
			return err
		}
		payload, _ := json.Marshal(event.Data)
		_, err = tx.ExecContext(ctx, `INSERT INTO outbox_events(event_type,laboratory_id,resource_type,resource_id,revision,task_id,payload_json,occurred_at) VALUES(?,?,?,?,?,?,?,?)`, event.Type, event.LaboratoryID, event.ResourceType, event.ResourceID, event.Revision, event.TaskID, payload, event.OccurredAt.Format(time.RFC3339Nano))
		return err
	})
}

func (r *Repositories) GetLaboratory(ctx context.Context, id domain.ID) (domain.Laboratory, error) {
	var lab domain.Laboratory
	var created, updated string
	err := r.database.DB.QueryRowContext(ctx, `SELECT id,name,description,revision,recovery_policy,lifecycle_state,created_at,updated_at FROM laboratories WHERE id=?`, id).Scan(&lab.ID, &lab.Name, &lab.Description, &lab.Revision, &lab.RecoveryPolicy, &lab.LifecycleState, &created, &updated)
	if err == sql.ErrNoRows {
		return lab, ErrNotFound
	}
	if err != nil {
		return lab, err
	}
	lab.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	lab.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	return lab, nil
}

func (r *Repositories) CreateTask(ctx context.Context, task domain.OperationTask) error {
	input, _ := json.Marshal(task.Input)
	return r.database.Write(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO operation_tasks(id,kind,resource_type,resource_id,idempotency_key,request_fingerprint,requested_revision,state,progress_current,progress_total,input_json,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, task.ID, task.Kind, task.ResourceType, task.ResourceID, nullable(task.IdempotencyKey), nullable(task.RequestFingerprint), task.RequestedRevision, task.State, task.ProgressCurrent, task.ProgressTotal, input, task.CreatedAt.Format(time.RFC3339Nano)); err != nil {
			return err
		}
		return appendEvent(ctx, tx, "task.created", "", "operation_task", task.ID, 0, task.ID, eventData(task))
	})
}

func (r *Repositories) UpdateTask(ctx context.Context, task domain.OperationTask) error {
	input, _ := json.Marshal(task.Input)
	result, _ := json.Marshal(task.Result)
	errorJSON, _ := json.Marshal(task.Error)
	return r.database.Write(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `UPDATE operation_tasks SET state=?,progress_current=?,progress_total=?,input_json=?,result_json=?,error_json=?,started_at=?,finished_at=? WHERE id=?`, task.State, task.ProgressCurrent, task.ProgressTotal, input, nullableBytes(result, task.Result != nil), nullableBytes(errorJSON, task.Error != nil), formatTime(task.StartedAt), formatTime(task.FinishedAt), task.ID); err != nil {
			return err
		}
		return appendEvent(ctx, tx, "task.updated", "", "operation_task", task.ID, 0, task.ID, eventData(task))
	})
}

func eventData(value any) map[string]any {
	body, _ := json.Marshal(value)
	data := map[string]any{}
	_ = json.Unmarshal(body, &data)
	return data
}

func (r *Repositories) GetTask(ctx context.Context, id domain.ID) (domain.OperationTask, error) {
	var task domain.OperationTask
	var created string
	var key, fingerprint, started, finished sql.NullString
	var inputJSON, resultJSON, errorJSON []byte
	err := r.database.DB.QueryRowContext(ctx, `SELECT id,kind,resource_type,resource_id,idempotency_key,request_fingerprint,requested_revision,state,progress_current,progress_total,COALESCE(input_json,'{}'),COALESCE(result_json,'{}'),COALESCE(error_json,''),created_at,started_at,finished_at FROM operation_tasks WHERE id=?`, id).Scan(&task.ID, &task.Kind, &task.ResourceType, &task.ResourceID, &key, &fingerprint, &task.RequestedRevision, &task.State, &task.ProgressCurrent, &task.ProgressTotal, &inputJSON, &resultJSON, &errorJSON, &created, &started, &finished)
	if err == sql.ErrNoRows {
		return task, ErrNotFound
	}
	if err != nil {
		return task, err
	}
	task.IdempotencyKey = key.String
	task.RequestFingerprint = fingerprint.String
	_ = json.Unmarshal(inputJSON, &task.Input)
	if len(resultJSON) > 0 {
		_ = json.Unmarshal(resultJSON, &task.Result)
	}
	if len(errorJSON) > 0 {
		task.Error = &domain.Problem{}
		_ = json.Unmarshal(errorJSON, task.Error)
	}
	task.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	task.StartedAt = parseNullTime(started)
	task.FinishedAt = parseNullTime(finished)
	return task, nil
}

func (r *Repositories) GetTaskByIdempotency(ctx context.Context, kind, key string) (domain.OperationTask, error) {
	var id domain.ID
	err := r.database.DB.QueryRowContext(ctx, `SELECT id FROM operation_tasks WHERE kind=? AND idempotency_key=?`, kind, key).Scan(&id)
	if err == sql.ErrNoRows {
		return domain.OperationTask{}, ErrNotFound
	}
	if err != nil {
		return domain.OperationTask{}, err
	}
	return r.GetTask(ctx, id)
}

func (r *Repositories) ListRecoverableTasks(ctx context.Context, limit int) ([]domain.OperationTask, error) {
	if limit < 1 {
		limit = 256
	}
	rows, err := r.database.DB.QueryContext(ctx, `SELECT id FROM operation_tasks WHERE state IN ('queued','running','cancelling') ORDER BY created_at LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []domain.ID
	for rows.Next() {
		var id domain.ID
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
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

func (r *Repositories) OutboxAfter(ctx context.Context, after int64, limit int) ([]domain.OutboxEvent, error) {
	rows, err := r.database.DB.QueryContext(ctx, `SELECT sequence,event_type,laboratory_id,resource_type,resource_id,revision,task_id,payload_json,occurred_at FROM outbox_events WHERE sequence>? ORDER BY sequence LIMIT ?`, after, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []domain.OutboxEvent
	for rows.Next() {
		var event domain.OutboxEvent
		var lab, task sql.NullString
		var payload []byte
		var occurred string
		if err = rows.Scan(&event.Sequence, &event.Type, &lab, &event.ResourceType, &event.ResourceID, &event.Revision, &task, &payload, &occurred); err != nil {
			return nil, err
		}
		event.LaboratoryID, event.TaskID = domain.ID(lab.String), domain.ID(task.String)
		_ = json.Unmarshal(payload, &event.Data)
		event.OccurredAt, _ = time.Parse(time.RFC3339Nano, occurred)
		events = append(events, event)
	}
	return events, rows.Err()
}

func nullable(value string) any {
	if value == "" {
		return nil
	}
	return value
}
func nullableBytes(value []byte, present bool) any {
	if !present {
		return nil
	}
	return value
}
func formatTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC().Format(time.RFC3339Nano)
}
func parseNullTime(value sql.NullString) *time.Time {
	if !value.Valid {
		return nil
	}
	parsed, _ := time.Parse(time.RFC3339Nano, value.String)
	return &parsed
}
