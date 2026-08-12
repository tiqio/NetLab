package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/netlab/netlab/internal/domain"
	"time"
)

func (r *Repositories) CreateTrafficWorkload(ctx context.Context, w domain.TrafficWorkload) error {
	if err := w.Validate(); err != nil {
		return err
	}
	s, _ := json.Marshal(w.Source)
	d, _ := json.Marshal(w.Destination)
	_, err := r.database.DB.ExecContext(ctx, `INSERT INTO traffic_workloads(id,laboratory_id,name,revision,source_json,protocol,address_family,destination_json,interval_seconds,timeout_seconds,desired_state,observed_state,attempts,successes,failures,matched_bytes,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, w.ID, w.LaboratoryID, w.Name, w.Revision, s, w.Protocol, w.AddressFamily, d, w.IntervalSeconds, w.TimeoutSeconds, w.DesiredState, w.ObservedState, w.Attempts, w.Successes, w.Failures, w.MatchedBytes, w.CreatedAt.Format(time.RFC3339Nano), w.UpdatedAt.Format(time.RFC3339Nano))
	return err
}
func (r *Repositories) GetTrafficWorkload(ctx context.Context, id domain.ID) (domain.TrafficWorkload, error) {
	var w domain.TrafficWorkload
	var s, d, le []byte
	var created, updated, last sql.NullString
	err := r.database.DB.QueryRowContext(ctx, `SELECT id,laboratory_id,name,revision,source_json,protocol,address_family,destination_json,interval_seconds,timeout_seconds,desired_state,observed_state,attempts,successes,failures,matched_bytes,last_success_at,COALESCE(last_error_json,''),created_at,updated_at FROM traffic_workloads WHERE id=?`, id).Scan(&w.ID, &w.LaboratoryID, &w.Name, &w.Revision, &s, &w.Protocol, &w.AddressFamily, &d, &w.IntervalSeconds, &w.TimeoutSeconds, &w.DesiredState, &w.ObservedState, &w.Attempts, &w.Successes, &w.Failures, &w.MatchedBytes, &last, &le, &created, &updated)
	if err == sql.ErrNoRows {
		return w, ErrNotFound
	}
	if err != nil {
		return w, err
	}
	_ = json.Unmarshal(s, &w.Source)
	_ = json.Unmarshal(d, &w.Destination)
	w.CreatedAt, _ = time.Parse(time.RFC3339Nano, created.String)
	w.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated.String)
	if last.Valid {
		v, _ := time.Parse(time.RFC3339Nano, last.String)
		w.LastSuccessAt = &v
	}
	if len(le) > 0 {
		w.LastError = &domain.Problem{}
		_ = json.Unmarshal(le, w.LastError)
	}
	return w, nil
}
func (r *Repositories) ListTrafficWorkloads(ctx context.Context, lab domain.ID) ([]domain.TrafficWorkload, error) {
	rows, err := r.database.DB.QueryContext(ctx, `SELECT id FROM traffic_workloads WHERE laboratory_id=? ORDER BY created_at`, lab)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.TrafficWorkload{}
	for rows.Next() {
		var id domain.ID
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		w, e := r.GetTrafficWorkload(ctx, id)
		if e != nil {
			return nil, e
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (r *Repositories) UpdateTrafficWorkloadState(ctx context.Context, id domain.ID, expected domain.Revision, desired, observed string, problem *domain.Problem, taskID domain.ID) (domain.TrafficWorkload, error) {
	var updated domain.TrafficWorkload
	err := r.database.Write(ctx, func(tx *sql.Tx) error {
		problemJSON, _ := json.Marshal(problem)
		result, err := tx.ExecContext(ctx, `UPDATE traffic_workloads SET desired_state=?,observed_state=?,last_error_json=?,revision=revision+1,updated_at=? WHERE id=? AND revision=?`, desired, observed, nullableBytes(problemJSON, problem != nil), time.Now().UTC().Format(time.RFC3339Nano), id, expected)
		if err != nil {
			return err
		}
		if affected, _ := result.RowsAffected(); affected != 1 {
			return domain.Problem{Code: "revision_conflict", Message: "traffic workload revision mismatch", Retryable: true, ResourceType: "traffic_workload", ResourceID: id}
		}
		row := tx.QueryRowContext(ctx, `SELECT laboratory_id,revision FROM traffic_workloads WHERE id=?`, id)
		var lab domain.ID
		var revision domain.Revision
		if err = row.Scan(&lab, &revision); err != nil {
			return err
		}
		return appendEvent(ctx, tx, "traffic_workload.updated", lab, "traffic_workload", id, revision, taskID, map[string]any{"desired_state": desired, "observed_state": observed})
	})
	if err != nil {
		return updated, err
	}
	return r.GetTrafficWorkload(ctx, id)
}

func (r *Repositories) RecordTrafficWorkloadOutcome(ctx context.Context, id domain.ID, success bool, matchedBytes int64, problem *domain.Problem) (domain.TrafficWorkload, error) {
	err := r.database.Write(ctx, func(tx *sql.Tx) error {
		now := time.Now().UTC()
		successDelta, failureDelta := 0, 0
		var last any = nil
		if success {
			successDelta = 1
			last = now.Format(time.RFC3339Nano)
		} else {
			failureDelta = 1
		}
		problemJSON, _ := json.Marshal(problem)
		result, err := tx.ExecContext(ctx, `UPDATE traffic_workloads SET attempts=attempts+1,successes=successes+?,failures=failures+?,matched_bytes=matched_bytes+?,last_success_at=COALESCE(?,last_success_at),last_error_json=?,updated_at=? WHERE id=?`, successDelta, failureDelta, matchedBytes, last, nullableBytes(problemJSON, problem != nil), now.Format(time.RFC3339Nano), id)
		if err != nil {
			return err
		}
		if n, _ := result.RowsAffected(); n != 1 {
			return ErrNotFound
		}
		var lab domain.ID
		var revision domain.Revision
		if err = tx.QueryRowContext(ctx, `SELECT laboratory_id,revision FROM traffic_workloads WHERE id=?`, id).Scan(&lab, &revision); err != nil {
			return err
		}
		return appendEvent(ctx, tx, "traffic_workload.outcome", lab, "traffic_workload", id, revision, "", map[string]any{"success": success, "matched_bytes": matchedBytes})
	})
	if err != nil {
		return domain.TrafficWorkload{}, err
	}
	return r.GetTrafficWorkload(ctx, id)
}

func (r *Repositories) DeleteTrafficWorkload(ctx context.Context, id domain.ID, expected domain.Revision, taskID domain.ID) error {
	return r.database.Write(ctx, func(tx *sql.Tx) error {
		var lab domain.ID
		if err := tx.QueryRowContext(ctx, `SELECT laboratory_id FROM traffic_workloads WHERE id=? AND revision=?`, id, expected).Scan(&lab); err == sql.ErrNoRows {
			return domain.Problem{Code: "revision_conflict", Message: "traffic workload revision mismatch", ResourceType: "traffic_workload", ResourceID: id}
		} else if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM traffic_workloads WHERE id=? AND revision=?`, id, expected); err != nil {
			return err
		}
		return appendEvent(ctx, tx, "traffic_workload.deleted", lab, "traffic_workload", id, expected, taskID, map[string]any{})
	})
}
