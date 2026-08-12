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
