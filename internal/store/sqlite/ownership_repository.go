package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/netlab/netlab/internal/domain"
	"github.com/netlab/netlab/internal/runtime/ownership"
)

func (r *Repositories) UpsertRuntimeOwnership(ctx context.Context, resourceType string, resourceID domain.ID, objectKind, objectName string, metadata map[string]string, cleanupState string) error {
	if resourceType == "" || resourceID == "" || objectKind == "" || objectName == "" {
		return fmt.Errorf("runtime ownership identity is required")
	}
	if cleanupState == "" {
		cleanupState = "active"
	}
	body, _ := json.Marshal(metadata)
	_, err := r.database.DB.ExecContext(ctx, `INSERT INTO runtime_ownership(resource_type,resource_id,object_kind,object_name,metadata_json,cleanup_state) VALUES(?,?,?,?,?,?) ON CONFLICT(resource_type,resource_id,object_kind,object_name) DO UPDATE SET metadata_json=excluded.metadata_json,cleanup_state=excluded.cleanup_state`, resourceType, resourceID, objectKind, objectName, body, cleanupState)
	return err
}

func (r *Repositories) DeleteRuntimeOwnership(ctx context.Context, resourceType string, resourceID domain.ID, objectKind, objectName string) error {
	_, err := r.database.DB.ExecContext(ctx, `DELETE FROM runtime_ownership WHERE resource_type=? AND resource_id=? AND object_kind=? AND object_name=?`, resourceType, resourceID, objectKind, objectName)
	return err
}

func (r *Repositories) ListRuntimeOwnership(ctx context.Context) ([]ownership.Record, error) {
	rows, err := r.database.DB.QueryContext(ctx, `SELECT resource_type,resource_id,object_kind,object_name,metadata_json,cleanup_state FROM runtime_ownership ORDER BY resource_type,resource_id,object_kind,object_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var values []ownership.Record
	for rows.Next() {
		var value ownership.Record
		var metadata []byte
		if err = rows.Scan(&value.ResourceType, &value.ResourceID, &value.ObjectKind, &value.ObjectName, &metadata, &value.CleanupState); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(metadata, &value.Metadata)
		values = append(values, value)
	}
	return values, rows.Err()
}

func (r *Repositories) RuntimeOwnerExists(ctx context.Context, resourceType string, resourceID domain.ID) (bool, error) {
	var query string
	switch resourceType {
	case "node":
		query = `SELECT 1 FROM nodes WHERE id=?`
	case "interface":
		query = `SELECT 1 FROM interfaces WHERE id=?`
	case "network_object":
		query = `SELECT 1 FROM network_objects WHERE id=?`
	case "link":
		query = `SELECT 1 FROM links WHERE id=?`
	case "laboratory":
		query = `SELECT 1 FROM laboratories WHERE id=?`
	case "network_attachment":
		query = `SELECT 1 FROM network_attachments WHERE id=?`
	case "capture":
		query = `SELECT 1 FROM operation_tasks WHERE resource_type='capture' AND resource_id=? LIMIT 1`
	default:
		return false, nil
	}
	var exists int
	err := r.database.DB.QueryRowContext(ctx, query, resourceID).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

func (r *TopologyRepository) UpsertRuntimeOwnership(ctx context.Context, resourceType string, resourceID domain.ID, objectKind, objectName string, metadata map[string]string, cleanupState string) error {
	return (&Repositories{database: r.database}).UpsertRuntimeOwnership(ctx, resourceType, resourceID, objectKind, objectName, metadata, cleanupState)
}

func (r *TopologyRepository) DeleteRuntimeOwnership(ctx context.Context, resourceType string, resourceID domain.ID, objectKind, objectName string) error {
	return (&Repositories{database: r.database}).DeleteRuntimeOwnership(ctx, resourceType, resourceID, objectKind, objectName)
}
