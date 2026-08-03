package sqlite

import (
	"context"
	"encoding/json"
	"time"

	"github.com/netlab/netlab/internal/domain"
)

func (r *Repositories) SaveNATServiceObservation(ctx context.Context, observation domain.NATServiceObservation) error {
	problem, _ := json.Marshal(observation.Problem)
	if observation.ObservedAt.IsZero() {
		observation.ObservedAt = time.Now().UTC()
	}
	_, err := r.database.DB.ExecContext(ctx, `INSERT INTO nat_service_observations(network_object_id,config_digest,unit_name,config_path,lease_path,pid,state,problem_json,observed_at) VALUES(?,?,?,?,?,?,?,?,?) ON CONFLICT(network_object_id) DO UPDATE SET config_digest=excluded.config_digest,unit_name=excluded.unit_name,config_path=excluded.config_path,lease_path=excluded.lease_path,pid=excluded.pid,state=excluded.state,problem_json=excluded.problem_json,observed_at=excluded.observed_at`, observation.NetworkObjectID, observation.ConfigDigest, observation.UnitName, observation.ConfigPath, observation.LeasePath, nullableInt(observation.PID), observation.State, nullableBytes(problem, observation.Problem != nil), observation.ObservedAt.Format(time.RFC3339Nano))
	return err
}

func (r *Repositories) DeleteNATServiceObservation(ctx context.Context, id domain.ID) error {
	_, err := r.database.DB.ExecContext(ctx, `DELETE FROM nat_service_observations WHERE network_object_id=?`, id)
	return err
}

func (r *TopologyRepository) ListNodeNATLeasePaths(ctx context.Context, nodeID domain.ID) ([]string, error) {
	rows, err := r.database.DB.QueryContext(ctx, `SELECT DISTINCT observation.lease_path
		FROM interfaces interface
		JOIN network_attachments attachment ON attachment.interface_id=interface.id
		JOIN network_objects object ON object.id=attachment.network_object_id AND object.kind=?
		JOIN nat_service_observations observation ON observation.network_object_id=object.id
		WHERE interface.node_id=? AND observation.state='active'
		ORDER BY observation.lease_path`, domain.NetworkNAT, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var values []string
	for rows.Next() {
		var value string
		if err = rows.Scan(&value); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func nullableInt(value int) any {
	if value == 0 {
		return nil
	}
	return value
}
