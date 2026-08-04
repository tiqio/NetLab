package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type MaintenanceStats struct {
	Laboratories        int64
	Nodes               int64
	NetworkObjects      int64
	OperationTasks      int64
	AuditEvents         int64
	OutboxEvents        int64
	Captures            int64
	TrafficFilters      int64
	TrafficObservations int64
}

type PruneResult struct {
	OperationTasks      int64
	AuditEvents         int64
	OutboxEvents        int64
	TrafficObservations int64
}

func (r *Repositories) MaintenanceStats(ctx context.Context) (MaintenanceStats, error) {
	var value MaintenanceStats
	err := r.database.DB.QueryRowContext(ctx, `SELECT
    (SELECT count(*) FROM laboratories),
    (SELECT count(*) FROM nodes),
    (SELECT count(*) FROM network_objects),
    (SELECT count(*) FROM operation_tasks),
    (SELECT count(*) FROM audit_events),
    (SELECT count(*) FROM outbox_events),
    (SELECT count(*) FROM captures),
    (SELECT count(*) FROM traffic_filters),
    (SELECT count(*) FROM traffic_observations)`).Scan(
		&value.Laboratories, &value.Nodes, &value.NetworkObjects, &value.OperationTasks,
		&value.AuditEvents, &value.OutboxEvents, &value.Captures, &value.TrafficFilters,
		&value.TrafficObservations,
	)
	return value, err
}

func (r *Repositories) PruneHistory(ctx context.Context, cutoff time.Time, batch int, replayFloor int64) (PruneResult, error) {
	if batch < 1 {
		return PruneResult{}, fmt.Errorf("maintenance prune batch must be positive")
	}
	if replayFloor < 0 {
		return PruneResult{}, fmt.Errorf("outbox replay floor must not be negative")
	}
	var result PruneResult
	tx, err := r.database.DB.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return PruneResult{}, err
	}
	defer tx.Rollback()
	err = func() error {
		cutoffText := cutoff.UTC().Format(time.RFC3339Nano)
		queries := []struct {
			result *int64
			query  string
			args   []any
		}{
			{&result.TrafficObservations, `DELETE FROM traffic_observations WHERE rowid IN (SELECT rowid FROM traffic_observations WHERE last_seen_at < ? ORDER BY last_seen_at LIMIT ?)`, []any{cutoffText, batch}},
			{&result.OperationTasks, `DELETE FROM operation_tasks WHERE id IN (SELECT id FROM operation_tasks WHERE state IN ('succeeded','failed','cancelled') AND COALESCE(finished_at,created_at) < ? ORDER BY COALESCE(finished_at,created_at) LIMIT ?)`, []any{cutoffText, batch}},
			{&result.AuditEvents, `DELETE FROM audit_events WHERE id IN (SELECT id FROM audit_events WHERE occurred_at < ? ORDER BY occurred_at LIMIT ?)`, []any{cutoffText, batch}},
			{&result.OutboxEvents, `DELETE FROM outbox_events WHERE sequence IN (SELECT sequence FROM outbox_events WHERE published_at IS NOT NULL AND occurred_at < ? AND sequence <= (SELECT COALESCE(MAX(sequence),0)-? FROM outbox_events) ORDER BY sequence LIMIT ?)`, []any{cutoffText, replayFloor, batch}},
		}
		for _, item := range queries {
			execution, err := tx.ExecContext(ctx, item.query, item.args...)
			if err != nil {
				return err
			}
			*item.result, err = execution.RowsAffected()
			if err != nil {
				return err
			}
		}
		return nil
	}()
	if err != nil {
		return PruneResult{}, err
	}
	if err = tx.Commit(); err != nil {
		return PruneResult{}, err
	}
	return result, nil
}
