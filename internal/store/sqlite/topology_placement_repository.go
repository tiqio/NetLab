package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/netlab/netlab/internal/domain"
)

func (r *TopologyRepository) ListPlacements(ctx context.Context, laboratoryID domain.ID) ([]domain.TopologyPlacement, error) {
	rows, err := r.database.DB.QueryContext(ctx, `SELECT laboratory_id,resource_id,resource_type,x,y,revision FROM topology_placements WHERE laboratory_id=? ORDER BY resource_id`, laboratoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var values []domain.TopologyPlacement
	for rows.Next() {
		var value domain.TopologyPlacement
		if err = rows.Scan(&value.LaboratoryID, &value.ResourceID, &value.ResourceType, &value.X, &value.Y, &value.Revision); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func (r *TopologyRepository) UpdatePlacements(ctx context.Context, laboratoryID domain.ID, expected domain.Revision, updates []domain.PlacementUpdate) (domain.Revision, []domain.TopologyPlacement, error) {
	if err := domain.ValidatePlacementBatch(updates); err != nil {
		return 0, nil, err
	}
	var revision domain.Revision
	var placements []domain.TopologyPlacement
	err := r.database.Write(ctx, func(tx *sql.Tx) error {
		var current domain.Revision
		if err := tx.QueryRowContext(ctx, `SELECT revision FROM laboratories WHERE id=?`, laboratoryID).Scan(&current); err != nil {
			if err == sql.ErrNoRows {
				return ErrNotFound
			}
			return err
		}
		if current != expected {
			return domain.Problem{Code: "revision_conflict", Message: "laboratory revision changed", Retryable: true, ResourceType: "laboratory", ResourceID: laboratoryID}
		}
		for _, update := range updates {
			table := "nodes"
			if update.ResourceType == domain.PlacementNetworkObject {
				table = "network_objects"
			}
			var owner domain.ID
			if err := tx.QueryRowContext(ctx, `SELECT laboratory_id FROM `+table+` WHERE id=?`, update.ResourceID).Scan(&owner); err != nil || owner != laboratoryID {
				return domain.Problem{Code: "resource_not_found", Message: "placement resource not found in laboratory", ResourceType: string(update.ResourceType), ResourceID: update.ResourceID}
			}
			var existing domain.Revision
			err := tx.QueryRowContext(ctx, `SELECT revision FROM topology_placements WHERE laboratory_id=? AND resource_id=?`, laboratoryID, update.ResourceID).Scan(&existing)
			if err != nil && err != sql.ErrNoRows {
				return err
			}
			if err == nil && update.Revision != existing {
				return domain.Problem{Code: "revision_conflict", Message: "placement revision changed", Retryable: true, ResourceType: string(update.ResourceType), ResourceID: update.ResourceID}
			}
			next := domain.Revision(1)
			if err == nil {
				next = existing.Next()
			}
			if _, err = tx.ExecContext(ctx, `INSERT INTO topology_placements(laboratory_id,resource_id,resource_type,x,y,revision,updated_at) VALUES(?,?,?,?,?,?,?) ON CONFLICT(laboratory_id,resource_id) DO UPDATE SET resource_type=excluded.resource_type,x=excluded.x,y=excluded.y,revision=excluded.revision,updated_at=excluded.updated_at`, laboratoryID, update.ResourceID, update.ResourceType, update.X, update.Y, next, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
				return err
			}
			placements = append(placements, domain.TopologyPlacement{LaboratoryID: laboratoryID, ResourceID: update.ResourceID, ResourceType: update.ResourceType, X: update.X, Y: update.Y, Revision: next})
		}
		revision = current.Next()
		if _, err := tx.ExecContext(ctx, `UPDATE laboratories SET revision=?,updated_at=? WHERE id=? AND revision=?`, revision, time.Now().UTC().Format(time.RFC3339Nano), laboratoryID, current); err != nil {
			return err
		}
		return appendEvent(ctx, tx, "topology.placements_changed", laboratoryID, "laboratory", laboratoryID, revision, "", map[string]any{"placements": placements})
	})
	return revision, placements, err
}
