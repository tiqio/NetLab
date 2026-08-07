package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/netlab/netlab/internal/domain"
)

func (r *Repositories) CreateNetworkObject(ctx context.Context, value domain.NetworkObject) error {
	if err := domain.ValidateNetworkKind(value.Kind); err != nil {
		return err
	}
	config, _ := json.Marshal(value.Config)
	return r.database.Write(ctx, func(tx *sql.Tx) error { return insertNetworkObjectTx(ctx, tx, value, config) })
}

func (r *Repositories) CreateNetworkObjectWithPlacement(ctx context.Context, value domain.NetworkObject, expected domain.Revision, intent *domain.PlacementIntent, entry string) (domain.PlacementAssignment, domain.Revision, error) {
	if err := domain.ValidateNetworkKind(value.Kind); err != nil {
		return domain.PlacementAssignment{}, 0, err
	}
	config, _ := json.Marshal(value.Config)
	var assignment domain.PlacementAssignment
	var laboratoryRevision domain.Revision
	err := r.database.Write(ctx, func(tx *sql.Tx) error {
		var err error
		if laboratoryRevision, err = advanceLaboratoryRevisionTx(ctx, tx, value.LaboratoryID, expected); err != nil {
			return err
		}
		if err = insertNetworkObjectTx(ctx, tx, value, config); err != nil {
			return err
		}
		if assignment, err = allocateInitialPlacementTx(ctx, tx, value.LaboratoryID, value.ID, domain.PlacementNetworkObject, intent); err != nil {
			return err
		}
		if err = appendEvent(ctx, tx, "topology.placements_changed", value.LaboratoryID, "laboratory", value.LaboratoryID, laboratoryRevision, "", map[string]any{"placements": []domain.TopologyPlacement{assignment.Placement}, "entry": entry, "placement_assignment": assignment}); err != nil {
			return err
		}
		return appendEvent(ctx, tx, "network_object.created", value.LaboratoryID, "network_object", value.ID, value.Revision, "", eventData(value))
	})
	return assignment, laboratoryRevision, err
}

func insertNetworkObjectTx(ctx context.Context, tx *sql.Tx, value domain.NetworkObject, config []byte) error {
	if value.Kind == domain.NetworkNAT {
		var requested domain.NATConfig
		_ = json.Unmarshal(config, &requested)
		rows, err := tx.QueryContext(ctx, `SELECT config_json FROM network_objects WHERE kind='nat_bridge'`)
		if err != nil {
			return err
		}
		for rows.Next() {
			var raw []byte
			if err = rows.Scan(&raw); err != nil {
				rows.Close()
				return err
			}
			var existing domain.NATConfig
			_ = json.Unmarshal(raw, &existing)
			if domain.PrefixesOverlap(requested.IPv4Prefix, existing.IPv4Prefix) || (requested.IPv6Prefix != "" && domain.PrefixesOverlap(requested.IPv6Prefix, existing.IPv6Prefix)) {
				rows.Close()
				return fmt.Errorf("NAT prefix overlaps an existing network object")
			}
		}
		rows.Close()
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO network_objects(id,laboratory_id,name,kind,revision,desired_state,observed_state,config_json,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?)`, value.ID, value.LaboratoryID, value.Name, value.Kind, value.Revision, value.DesiredState, value.ObservedState, config, value.CreatedAt.Format(time.RFC3339Nano), value.UpdatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return err
	}
	return appendEvent(ctx, tx, "network_object.created", value.LaboratoryID, "network_object", value.ID, value.Revision, "", eventData(value))
}

func (r *Repositories) UpdateNetworkObject(ctx context.Context, value domain.NetworkObject, expected domain.Revision) (domain.NetworkObject, error) {
	config, err := json.Marshal(value.Config)
	if err != nil {
		return domain.NetworkObject{}, err
	}
	updated := value
	updated.Revision = expected + 1
	updated.ObservedState = "provisioning"
	updated.LastError = nil
	updated.UpdatedAt = time.Now().UTC()
	err = r.database.Write(ctx, func(tx *sql.Tx) error {
		result, updateErr := tx.ExecContext(ctx, `UPDATE network_objects SET name=?,revision=?,observed_state='provisioning',config_json=?,last_error_json=NULL,updated_at=? WHERE id=? AND revision=?`, updated.Name, updated.Revision, config, updated.UpdatedAt.Format(time.RFC3339Nano), updated.ID, expected)
		if updateErr != nil {
			return updateErr
		}
		if affected, _ := result.RowsAffected(); affected != 1 {
			return domain.Problem{Code: "revision_conflict", Message: "network object revision mismatch", ResourceType: "network_object", ResourceID: updated.ID}
		}
		return appendEvent(ctx, tx, "network_object.updated", updated.LaboratoryID, "network_object", updated.ID, updated.Revision, "", eventData(updated))
	})
	return updated, err
}

func (r *Repositories) GetNetworkObject(ctx context.Context, id domain.ID) (domain.NetworkObject, error) {
	var value domain.NetworkObject
	var config []byte
	var createdAt, updatedAt string
	err := r.database.DB.QueryRowContext(ctx, `SELECT id,laboratory_id,name,kind,revision,desired_state,observed_state,config_json,created_at,updated_at FROM network_objects WHERE id=?`, id).Scan(&value.ID, &value.LaboratoryID, &value.Name, &value.Kind, &value.Revision, &value.DesiredState, &value.ObservedState, &config, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return value, ErrNotFound
	}
	_ = json.Unmarshal(config, &value.Config)
	value.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	value.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)
	return value, err
}

func (r *Repositories) ListNetworkObjects(ctx context.Context, labID domain.ID) ([]domain.NetworkObject, error) {
	rows, err := r.database.DB.QueryContext(ctx, `SELECT id FROM network_objects WHERE laboratory_id=? ORDER BY name`, labID)
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
	values := make([]domain.NetworkObject, 0, len(ids))
	for _, id := range ids {
		value, getErr := r.GetNetworkObject(ctx, id)
		if getErr != nil {
			return nil, getErr
		}
		values = append(values, value)
	}
	return values, nil
}

func (r *Repositories) SetNetworkObjectState(ctx context.Context, id domain.ID, state string, problem *domain.Problem) error {
	body, _ := json.Marshal(problem)
	return r.database.Write(ctx, func(tx *sql.Tx) error {
		var laboratoryID domain.ID
		var revision domain.Revision
		if err := tx.QueryRowContext(ctx, `SELECT laboratory_id,revision FROM network_objects WHERE id=?`, id).Scan(&laboratoryID, &revision); err != nil {
			if err == sql.ErrNoRows {
				return ErrNotFound
			}
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE network_objects SET observed_state=?,last_error_json=?,updated_at=? WHERE id=?`, state, nullableBytes(body, problem != nil), time.Now().UTC().Format(time.RFC3339Nano), id); err != nil {
			return err
		}
		return appendEvent(ctx, tx, "network_object.observed_state_changed", laboratoryID, "network_object", id, revision, "", map[string]any{"observed_state": state, "problem": problem})
	})
}

func (r *Repositories) DeleteNetworkObject(ctx context.Context, id domain.ID, expected domain.Revision) error {
	return r.database.Write(ctx, func(tx *sql.Tx) error {
		var laboratoryID domain.ID
		if err := tx.QueryRowContext(ctx, `SELECT laboratory_id FROM network_objects WHERE id=?`, id).Scan(&laboratoryID); err != nil {
			if err == sql.ErrNoRows {
				return ErrNotFound
			}
			return err
		}
		result, err := tx.ExecContext(ctx, `DELETE FROM network_objects WHERE id=? AND revision=?`, id, expected)
		if err != nil {
			return err
		}
		if affected, _ := result.RowsAffected(); affected != 1 {
			return domain.Problem{Code: "revision_conflict", Message: "network object revision mismatch", ResourceType: "network_object", ResourceID: id}
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO network_object_tombstones(id,laboratory_id,revision,deleted_at) VALUES(?,?,?,?) ON CONFLICT(id) DO UPDATE SET revision=excluded.revision,deleted_at=excluded.deleted_at`, id, laboratoryID, expected.Next(), time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
			return err
		}
		return appendEvent(ctx, tx, "network_object.deleted", laboratoryID, "network_object", id, expected.Next(), "", nil)
	})
}

func (r *Repositories) CreateNetworkAttachment(ctx context.Context, objectID, interfaceID domain.ID, portName string, config map[string]any) error {
	var existingObjectID domain.ID
	err := r.database.DB.QueryRowContext(ctx, `SELECT network_object_id FROM network_attachments WHERE interface_id=? ORDER BY id LIMIT 1`, interfaceID).Scan(&existingObjectID)
	if err == nil {
		if existingObjectID == objectID {
			return nil
		}
		return fmt.Errorf("interface is already attached to network object %s", existingObjectID)
	}
	if err != sql.ErrNoRows {
		return err
	}
	body, _ := json.Marshal(config)
	attachmentID := domain.NewID()
	return r.database.Write(ctx, func(tx *sql.Tx) error {
		var laboratoryID domain.ID
		if err := tx.QueryRowContext(ctx, `SELECT laboratory_id FROM network_objects WHERE id=?`, objectID).Scan(&laboratoryID); err != nil {
			return err
		}
		if portName != "" {
			if _, err := tx.ExecContext(ctx, `INSERT INTO topology_endpoint_reservations(laboratory_id,owner_type,owner_id,port_name,resource_type,resource_id) VALUES(?,'network_object',?,?,'network_attachment',?)`, laboratoryID, objectID, portName, attachmentID); err != nil {
				return domain.Problem{Code: "port_in_use", Message: "network object port is already occupied", ResourceType: "network_attachment", ResourceID: attachmentID}
			}
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO network_attachments(id,network_object_id,interface_id,port_name,config_json,observed_state) VALUES(?,?,?,?,?,'pending')`, attachmentID, objectID, nullable(string(interfaceID)), portName, body)
		return err
	})
}

func (r *Repositories) ListNetworkAttachments(ctx context.Context, laboratoryID domain.ID) ([]domain.NetworkAttachment, error) {
	rows, err := r.database.DB.QueryContext(ctx, `SELECT a.id,a.network_object_id,COALESCE(a.interface_id,''),a.port_name,a.config_json,a.observed_state,COALESCE(a.last_error_json,'') FROM network_attachments a JOIN network_objects o ON o.id=a.network_object_id WHERE o.laboratory_id=? ORDER BY a.id`, laboratoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var values []domain.NetworkAttachment
	seenInterfaces := map[domain.ID]bool{}
	for rows.Next() {
		var value domain.NetworkAttachment
		var config, problem []byte
		if err = rows.Scan(&value.ID, &value.NetworkObjectID, &value.InterfaceID, &value.PortName, &config, &value.ObservedState, &problem); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(config, &value.Config)
		if len(problem) > 0 {
			value.LastError = &domain.Problem{}
			_ = json.Unmarshal(problem, value.LastError)
		}
		if value.InterfaceID != "" && seenInterfaces[value.InterfaceID] {
			continue
		}
		seenInterfaces[value.InterfaceID] = true
		values = append(values, value)
	}
	return values, rows.Err()
}

func (r *Repositories) ListNetworkObjectAttachments(ctx context.Context, objectID domain.ID) ([]domain.NetworkAttachment, error) {
	var laboratoryID domain.ID
	if err := r.database.DB.QueryRowContext(ctx, `SELECT laboratory_id FROM network_objects WHERE id=?`, objectID).Scan(&laboratoryID); err != nil {
		return nil, err
	}
	values, err := r.ListNetworkAttachments(ctx, laboratoryID)
	if err != nil {
		return nil, err
	}
	result := make([]domain.NetworkAttachment, 0, len(values))
	for _, value := range values {
		if value.NetworkObjectID == objectID {
			result = append(result, value)
		}
	}
	return result, nil
}

func (r *Repositories) SetNetworkAttachmentState(ctx context.Context, id domain.ID, state string, problem *domain.Problem) error {
	body, _ := json.Marshal(problem)
	_, err := r.database.DB.ExecContext(ctx, `UPDATE network_attachments SET observed_state=?,last_error_json=? WHERE id=?`, state, nullableBytes(body, problem != nil), id)
	return err
}

func (r *Repositories) CreateNetworkObjectLink(ctx context.Context, value domain.NetworkObjectLink) error {
	return r.database.Write(ctx, func(tx *sql.Tx) error {
		var labA, labB domain.ID
		if err := tx.QueryRowContext(ctx, `SELECT laboratory_id FROM network_objects WHERE id=?`, value.ObjectAID).Scan(&labA); err != nil {
			return err
		}
		if err := tx.QueryRowContext(ctx, `SELECT laboratory_id FROM network_objects WHERE id=?`, value.ObjectBID).Scan(&labB); err != nil {
			return err
		}
		if labA != value.LaboratoryID || labB != value.LaboratoryID {
			return domain.Problem{Code: "invalid_topology", Message: "network objects must belong to the same laboratory", ResourceType: "network_object_link", ResourceID: value.ID}
		}
		for _, endpoint := range []struct {
			objectID domain.ID
			port     string
		}{{value.ObjectAID, value.PortAName}, {value.ObjectBID, value.PortBName}} {
			if _, err := tx.ExecContext(ctx, `INSERT INTO topology_endpoint_reservations(laboratory_id,owner_type,owner_id,port_name,resource_type,resource_id) VALUES(?,'network_object',?,?,'network_object_link',?)`, value.LaboratoryID, endpoint.objectID, endpoint.port, value.ID); err != nil {
				return domain.Problem{Code: "port_in_use", Message: "one or more network object ports are already occupied", ResourceType: "network_object_link", ResourceID: value.ID}
			}
		}
		var occupied int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM network_object_links WHERE (object_a_id=? AND port_a_name=?) OR (object_b_id=? AND port_b_name=?) OR (object_a_id=? AND port_a_name=?) OR (object_b_id=? AND port_b_name=?)`, value.ObjectAID, value.PortAName, value.ObjectAID, value.PortAName, value.ObjectBID, value.PortBName, value.ObjectBID, value.PortBName).Scan(&occupied); err != nil {
			return err
		}
		if occupied > 0 {
			return domain.Problem{Code: "port_in_use", Message: "one or more network object ports are already linked", ResourceType: "network_object_link", ResourceID: value.ID}
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO network_object_links(id,laboratory_id,object_a_id,port_a_name,object_b_id,port_b_name,revision,desired_state,observed_state) VALUES(?,?,?,?,?,?,?,?,?)`, value.ID, value.LaboratoryID, value.ObjectAID, value.PortAName, value.ObjectBID, value.PortBName, value.Revision, value.DesiredState, value.ObservedState)
		if err != nil {
			return err
		}
		return appendEvent(ctx, tx, "network_object_link.created", value.LaboratoryID, "network_object_link", value.ID, value.Revision, "", eventData(value))
	})
}

func (r *Repositories) ValidateNetworkObjectLinkEndpoint(ctx context.Context, laboratoryID, objectID domain.ID, portName string) error {
	var objectLaboratoryID domain.ID
	if err := r.database.DB.QueryRowContext(ctx, `SELECT laboratory_id FROM network_objects WHERE id=?`, objectID).Scan(&objectLaboratoryID); err != nil {
		if err == sql.ErrNoRows {
			return domain.Problem{Code: "invalid_topology", Message: "network object endpoint does not exist", ResourceType: "network_object", ResourceID: objectID}
		}
		return err
	}
	if objectLaboratoryID != laboratoryID {
		return domain.Problem{Code: "invalid_topology", Message: "network object endpoint belongs to another laboratory", ResourceType: "network_object", ResourceID: objectID}
	}
	var occupied int
	if err := r.database.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM topology_endpoint_reservations WHERE laboratory_id=? AND owner_type='network_object' AND owner_id=? AND port_name=?`, laboratoryID, objectID, portName).Scan(&occupied); err != nil {
		return err
	}
	if occupied > 0 {
		return domain.Problem{Code: "port_in_use", Message: "network object port is already occupied", ResourceType: "network_object", ResourceID: objectID}
	}
	return nil
}

func (r *Repositories) ListNetworkObjectLinks(ctx context.Context, laboratoryID domain.ID) ([]domain.NetworkObjectLink, error) {
	query := `SELECT id,laboratory_id,object_a_id,port_a_name,object_b_id,port_b_name,revision,desired_state,observed_state,COALESCE(last_error_json,'') FROM network_object_links`
	args := []any{}
	if laboratoryID != "" {
		query += ` WHERE laboratory_id=?`
		args = append(args, laboratoryID)
	}
	query += ` ORDER BY id`
	rows, err := r.database.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var values []domain.NetworkObjectLink
	for rows.Next() {
		var value domain.NetworkObjectLink
		var problem []byte
		if err = rows.Scan(&value.ID, &value.LaboratoryID, &value.ObjectAID, &value.PortAName, &value.ObjectBID, &value.PortBName, &value.Revision, &value.DesiredState, &value.ObservedState, &problem); err != nil {
			return nil, err
		}
		if len(problem) > 0 {
			value.LastError = &domain.Problem{}
			_ = json.Unmarshal(problem, value.LastError)
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func (r *Repositories) GetNetworkObjectLink(ctx context.Context, id domain.ID) (domain.NetworkObjectLink, error) {
	var value domain.NetworkObjectLink
	var problem []byte
	err := r.database.DB.QueryRowContext(ctx, `SELECT id,laboratory_id,object_a_id,port_a_name,object_b_id,port_b_name,revision,desired_state,observed_state,COALESCE(last_error_json,'') FROM network_object_links WHERE id=?`, id).Scan(&value.ID, &value.LaboratoryID, &value.ObjectAID, &value.PortAName, &value.ObjectBID, &value.PortBName, &value.Revision, &value.DesiredState, &value.ObservedState, &problem)
	if err == sql.ErrNoRows {
		return value, ErrNotFound
	}
	if len(problem) > 0 {
		value.LastError = &domain.Problem{}
		_ = json.Unmarshal(problem, value.LastError)
	}
	return value, err
}

func (r *Repositories) ListNetworkObjectLinksByObject(ctx context.Context, objectID domain.ID) ([]domain.NetworkObjectLink, error) {
	var laboratoryID domain.ID
	if err := r.database.DB.QueryRowContext(ctx, `SELECT laboratory_id FROM network_objects WHERE id=?`, objectID).Scan(&laboratoryID); err != nil {
		return nil, err
	}
	values, err := r.ListNetworkObjectLinks(ctx, laboratoryID)
	if err != nil {
		return nil, err
	}
	result := make([]domain.NetworkObjectLink, 0, len(values))
	for _, value := range values {
		if value.ObjectAID == objectID || value.ObjectBID == objectID {
			result = append(result, value)
		}
	}
	return result, nil
}

func (r *Repositories) SetNetworkObjectLinkState(ctx context.Context, id domain.ID, state string, problem *domain.Problem) error {
	body, _ := json.Marshal(problem)
	return r.database.Write(ctx, func(tx *sql.Tx) error {
		var laboratoryID domain.ID
		var revision domain.Revision
		if err := tx.QueryRowContext(ctx, `SELECT laboratory_id,revision FROM network_object_links WHERE id=?`, id).Scan(&laboratoryID, &revision); err != nil {
			if err == sql.ErrNoRows {
				return ErrNotFound
			}
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE network_object_links SET observed_state=?,last_error_json=? WHERE id=?`, state, nullableBytes(body, problem != nil), id); err != nil {
			return err
		}
		return appendEvent(ctx, tx, "network_object_link.state_changed", laboratoryID, "network_object_link", id, revision, "", map[string]any{"observed_state": state, "last_error": problem})
	})
}

func (r *Repositories) PublishNetworkObjectLinkRecovered(ctx context.Context, id, taskID domain.ID) error {
	return r.database.Write(ctx, func(tx *sql.Tx) error {
		var laboratoryID domain.ID
		var revision domain.Revision
		var desiredState, observedState string
		var lastError []byte
		if err := tx.QueryRowContext(ctx, `SELECT laboratory_id,revision,desired_state,observed_state,COALESCE(last_error_json,'') FROM network_object_links WHERE id=?`, id).Scan(&laboratoryID, &revision, &desiredState, &observedState, &lastError); err != nil {
			if err == sql.ErrNoRows {
				return ErrNotFound
			}
			return err
		}
		var problem *domain.Problem
		if len(lastError) > 0 {
			problem = &domain.Problem{}
			if err := json.Unmarshal(lastError, problem); err != nil {
				return err
			}
		}
		return appendEvent(ctx, tx, "network_object_link.recovered", laboratoryID, "network_object_link", id, revision, taskID, map[string]any{"id": id, "desired_state": desiredState, "observed_state": observedState, "last_error": problem, "recovery_action": "adopted_or_recreated"})
	})
}

func (r *Repositories) DeleteNetworkObjectLink(ctx context.Context, id domain.ID) error {
	return r.DeleteNetworkObjectLinkRevision(ctx, id, 0, "")
}

func (r *Repositories) DeleteNetworkObjectLinkRevision(ctx context.Context, id domain.ID, expectedRevision domain.Revision, taskID domain.ID) error {
	return r.database.Write(ctx, func(tx *sql.Tx) error {
		var laboratoryID domain.ID
		var revision domain.Revision
		if err := tx.QueryRowContext(ctx, `SELECT laboratory_id,revision FROM network_object_links WHERE id=?`, id).Scan(&laboratoryID, &revision); err != nil {
			if err == sql.ErrNoRows {
				return nil
			}
			return err
		}
		if expectedRevision > 0 && revision != expectedRevision {
			return domain.Problem{Code: "revision_conflict", Message: fmt.Sprintf("expected revision %d, current revision is %d", expectedRevision, revision), ResourceType: "network_object_link", ResourceID: id, TaskID: taskID, Phase: "delete_commit"}
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM network_object_links WHERE id=?`, id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM topology_endpoint_reservations WHERE resource_type='network_object_link' AND resource_id=?`, id); err != nil {
			return err
		}
		return appendEvent(ctx, tx, "network_object_link.deleted", laboratoryID, "network_object_link", id, revision.Next(), taskID, nil)
	})
}
