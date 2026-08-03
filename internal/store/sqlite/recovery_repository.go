package sqlite

import (
	"context"
	"database/sql"
	"time"
)

func (r *TopologyRepository) PrepareHostRecovery(ctx context.Context) error {
	return r.database.Write(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `UPDATE nodes SET desired_state='stopped',revision=revision+1,updated_at=? WHERE laboratory_id IN (SELECT id FROM laboratories WHERE recovery_policy='remain_stopped') AND desired_state='running'`, time.Now().UTC().Format(time.RFC3339Nano))
		return err
	})
}
