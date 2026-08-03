package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/netlab/netlab/internal/domain"
)

func (r *Repositories) ListAllNodes(ctx context.Context) ([]domain.Node, error) {
	return NewTopologyRepository(r.database).ListAllNodes(ctx)
}

func (r *Repositories) ObserveRuntimeCapability(ctx context.Context, observation domain.RuntimeCapabilityObservation) (domain.RuntimeCapabilityObservation, error) {
	return observation, r.database.Write(ctx, func(tx *sql.Tx) error {
		var laboratoryID domain.ID
		if err := tx.QueryRowContext(ctx, `SELECT laboratory_id FROM nodes WHERE id=?`, observation.NodeID).Scan(&laboratoryID); err != nil {
			if err == sql.ErrNoRows {
				return ErrNotFound
			}
			return err
		}
		var current int64
		err := tx.QueryRowContext(ctx, `SELECT revision FROM node_runtime_capabilities WHERE node_id=? AND capability=?`, observation.NodeID, observation.Capability).Scan(&current)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		observation.Revision = domain.Revision(current + 1)
		if observation.ObservedAt.IsZero() {
			observation.ObservedAt = time.Now().UTC()
		}
		if err := observation.Validate(); err != nil {
			return err
		}
		details, _ := json.Marshal(observation.Details)
		problem, _ := json.Marshal(observation.Problem)
		_, err = tx.ExecContext(ctx, `INSERT INTO node_runtime_capabilities(node_id,capability,revision,state,required,details_json,problem_json,observed_at) VALUES(?,?,?,?,?,?,?,?) ON CONFLICT(node_id,capability) DO UPDATE SET revision=excluded.revision,state=excluded.state,required=excluded.required,details_json=excluded.details_json,problem_json=excluded.problem_json,observed_at=excluded.observed_at`, observation.NodeID, observation.Capability, observation.Revision, observation.State, observation.Required, details, nullableBytes(problem, observation.Problem != nil), observation.ObservedAt.Format(time.RFC3339Nano))
		if err != nil {
			return err
		}
		return appendEvent(ctx, tx, "node.capability_changed", laboratoryID, "node", observation.NodeID, observation.Revision, "", eventData(observation))
	})
}

func (r *Repositories) ListRuntimeCapabilities(ctx context.Context, nodeID domain.ID) ([]domain.RuntimeCapabilityObservation, error) {
	var exists int
	if err := r.database.DB.QueryRowContext(ctx, `SELECT 1 FROM nodes WHERE id=?`, nodeID).Scan(&exists); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	rows, err := r.database.DB.QueryContext(ctx, `SELECT capability,revision,state,required,details_json,COALESCE(problem_json,''),observed_at FROM node_runtime_capabilities WHERE node_id=? ORDER BY capability`, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	observations := []domain.RuntimeCapabilityObservation{}
	for rows.Next() {
		observation := domain.RuntimeCapabilityObservation{NodeID: nodeID}
		var required bool
		var details, problem []byte
		var observedAt string
		if err := rows.Scan(&observation.Capability, &observation.Revision, &observation.State, &required, &details, &problem, &observedAt); err != nil {
			return nil, err
		}
		observation.Required = required
		_ = json.Unmarshal(details, &observation.Details)
		if len(problem) > 0 {
			observation.Problem = &domain.Problem{}
			_ = json.Unmarshal(problem, observation.Problem)
		}
		observation.ObservedAt, _ = time.Parse(time.RFC3339Nano, observedAt)
		observations = append(observations, observation)
	}
	return observations, rows.Err()
}
