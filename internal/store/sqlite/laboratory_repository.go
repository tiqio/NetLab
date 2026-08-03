package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/netlab/netlab/internal/domain"
)

type TopologyRepository struct{ database *Database }

func NewTopologyRepository(database *Database) *TopologyRepository {
	return &TopologyRepository{database: database}
}

func (r *TopologyRepository) CreateLaboratory(ctx context.Context, lab domain.Laboratory) error {
	return r.database.Write(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO laboratories(id,name,description,revision,recovery_policy,lifecycle_state,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?)`, lab.ID, lab.Name, lab.Description, lab.Revision, lab.RecoveryPolicy, lab.LifecycleState, lab.CreatedAt.Format(time.RFC3339Nano), lab.UpdatedAt.Format(time.RFC3339Nano)); err != nil {
			return err
		}
		return appendEvent(ctx, tx, "laboratory.created", lab.ID, "laboratory", lab.ID, lab.Revision, "", map[string]any{"name": lab.Name})
	})
}

func (r *TopologyRepository) ListLaboratories(ctx context.Context) ([]domain.Laboratory, error) {
	rows, err := r.database.DB.QueryContext(ctx, `SELECT id,name,description,revision,recovery_policy,lifecycle_state,created_at,updated_at FROM laboratories ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var labs []domain.Laboratory
	for rows.Next() {
		lab, err := scanLaboratory(rows)
		if err != nil {
			return nil, err
		}
		labs = append(labs, lab)
	}
	return labs, rows.Err()
}

func (r *TopologyRepository) GetLaboratory(ctx context.Context, id domain.ID) (domain.Laboratory, error) {
	row := r.database.DB.QueryRowContext(ctx, `SELECT id,name,description,revision,recovery_policy,lifecycle_state,created_at,updated_at FROM laboratories WHERE id=?`, id)
	lab, err := scanLaboratory(row)
	if err == sql.ErrNoRows {
		return lab, ErrNotFound
	}
	return lab, err
}

func (r *TopologyRepository) UpdateLaboratory(ctx context.Context, id domain.ID, expected domain.Revision, name, description string, policy domain.RecoveryPolicy) (domain.Laboratory, error) {
	now := time.Now().UTC()
	var updated domain.Laboratory
	err := r.database.Write(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `UPDATE laboratories SET name=?,description=?,recovery_policy=?,revision=revision+1,updated_at=? WHERE id=? AND revision=? AND lifecycle_state='active'`, name, description, policy, now.Format(time.RFC3339Nano), id, expected)
		if err != nil {
			return err
		}
		count, _ := result.RowsAffected()
		if count == 0 {
			return domain.Problem{Code: "revision_conflict", Message: "laboratory revision changed", Retryable: true, ResourceType: "laboratory", ResourceID: id}
		}
		row := tx.QueryRowContext(ctx, `SELECT id,name,description,revision,recovery_policy,lifecycle_state,created_at,updated_at FROM laboratories WHERE id=?`, id)
		updated, err = scanLaboratory(row)
		if err != nil {
			return err
		}
		return appendEvent(ctx, tx, "laboratory.updated", id, "laboratory", id, updated.Revision, "", map[string]any{"name": updated.Name})
	})
	return updated, err
}

func (r *TopologyRepository) MarkLaboratoryDeleting(ctx context.Context, id domain.ID, expected domain.Revision) error {
	return r.database.Write(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `UPDATE laboratories SET lifecycle_state='deleting',revision=revision+1,updated_at=? WHERE id=? AND revision=?`, time.Now().UTC().Format(time.RFC3339Nano), id, expected)
		if err != nil {
			return err
		}
		count, _ := result.RowsAffected()
		if count == 0 {
			return domain.Problem{Code: "revision_conflict", Message: "laboratory revision changed", Retryable: true, ResourceType: "laboratory", ResourceID: id}
		}
		return appendEvent(ctx, tx, "laboratory.deleting", id, "laboratory", id, expected.Next(), "", nil)
	})
}

func (r *TopologyRepository) FinalizeLaboratoryDeletion(ctx context.Context, id domain.ID) error {
	return r.database.Write(ctx, func(tx *sql.Tx) error {
		statements := []string{
			`DELETE FROM runtime_ownership WHERE resource_id IN (SELECT resource_id FROM operation_tasks WHERE resource_type='capture' AND COALESCE(json_extract(input_json,'$.request.LaboratoryID'),json_extract(input_json,'$.request.laboratory_id'))=?)`,
			`DELETE FROM runtime_ownership WHERE resource_id IN (SELECT resource_id FROM operation_tasks WHERE resource_type='traffic_filter' AND json_extract(input_json,'$.laboratory_id')=?)`,
			`DELETE FROM runtime_ownership WHERE resource_id IN (SELECT id FROM network_attachments WHERE network_object_id IN (SELECT id FROM network_objects WHERE laboratory_id=?))`,
			`DELETE FROM runtime_ownership WHERE resource_id IN (SELECT id FROM port_mappings WHERE node_id IN (SELECT id FROM nodes WHERE laboratory_id=?))`,
			`DELETE FROM runtime_ownership WHERE resource_id IN (SELECT id FROM interfaces WHERE node_id IN (SELECT id FROM nodes WHERE laboratory_id=?))`,
			`DELETE FROM runtime_ownership WHERE resource_id IN (SELECT id FROM links WHERE laboratory_id=?)`,
			`DELETE FROM runtime_ownership WHERE resource_id IN (SELECT id FROM network_objects WHERE laboratory_id=?)`,
			`DELETE FROM runtime_ownership WHERE resource_id IN (SELECT id FROM captures WHERE laboratory_id=?)`,
			`DELETE FROM runtime_ownership WHERE resource_id IN (SELECT id FROM nodes WHERE laboratory_id=?)`,
			`DELETE FROM runtime_ownership WHERE resource_id=?`,
			`DELETE FROM network_attachments WHERE network_object_id IN (SELECT id FROM network_objects WHERE laboratory_id=?)`,
			`DELETE FROM port_mappings WHERE node_id IN (SELECT id FROM nodes WHERE laboratory_id=?)`,
			`DELETE FROM links WHERE laboratory_id=?`,
			`DELETE FROM interfaces WHERE node_id IN (SELECT id FROM nodes WHERE laboratory_id=?)`,
			`DELETE FROM nodes WHERE laboratory_id=?`,
			`DELETE FROM network_objects WHERE laboratory_id=?`,
			`DELETE FROM artifacts WHERE owner_type='laboratory' AND owner_id=?`,
			`DELETE FROM laboratories WHERE id=? AND lifecycle_state IN ('deleting','delete_failed')`,
		}
		for _, statement := range statements {
			if _, err := tx.ExecContext(ctx, statement, id); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *TopologyRepository) MarkLaboratoryDeleteFailed(ctx context.Context, id domain.ID, problem domain.Problem) error {
	body, _ := json.Marshal(problem)
	_, err := r.database.DB.ExecContext(ctx, `UPDATE laboratories SET lifecycle_state='delete_failed',updated_at=? WHERE id=?`, time.Now().UTC().Format(time.RFC3339Nano), id)
	if err == nil {
		_, _ = r.database.DB.ExecContext(ctx, `INSERT INTO outbox_events(event_type,laboratory_id,resource_type,resource_id,revision,payload_json,occurred_at) SELECT 'laboratory.delete_failed',id,'laboratory',id,revision,?,? FROM laboratories WHERE id=?`, body, time.Now().UTC().Format(time.RFC3339Nano), id)
	}
	return err
}

func (r *TopologyRepository) ListLaboratoryPortMappings(ctx context.Context, id domain.ID) ([]domain.PortMapping, error) {
	rows, err := r.database.DB.QueryContext(ctx, `SELECT p.id,p.node_id,p.protocol,p.host_address,p.host_port,p.guest_address,p.guest_port,p.revision,p.observed_state,p.created_at FROM port_mappings p JOIN nodes n ON n.id=p.node_id WHERE n.laboratory_id=?`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var values []domain.PortMapping
	for rows.Next() {
		var value domain.PortMapping
		var created string
		if err = rows.Scan(&value.ID, &value.NodeID, &value.Protocol, &value.HostAddress, &value.HostPort, &value.GuestAddress, &value.GuestPort, &value.Revision, &value.ObservedState, &created); err != nil {
			return nil, err
		}
		value.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		values = append(values, value)
	}
	return values, rows.Err()
}

func (r *TopologyRepository) ListLaboratoryArtifacts(ctx context.Context, id domain.ID, extra []domain.ID) ([]domain.Artifact, error) {
	rows, err := r.database.DB.QueryContext(ctx, `SELECT id,kind,path,media_type,size_bytes,sha256,owner_type,owner_id,created_at,expires_at FROM artifacts WHERE (owner_type='laboratory' AND owner_id=?) OR (owner_type='node' AND owner_id IN (SELECT id FROM nodes WHERE laboratory_id=?))`, id, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var values []domain.Artifact
	for rows.Next() {
		var value domain.Artifact
		var created string
		var expires sql.NullString
		if err = rows.Scan(&value.ID, &value.Kind, &value.Path, &value.MediaType, &value.SizeBytes, &value.SHA256, &value.OwnerType, &value.OwnerID, &created, &expires); err != nil {
			return nil, err
		}
		value.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		value.ExpiresAt = parseNullTime(expires)
		values = append(values, value)
	}
	for _, artifactID := range extra {
		var value domain.Artifact
		var created string
		var expires sql.NullString
		err = r.database.DB.QueryRowContext(ctx, `SELECT id,kind,path,media_type,size_bytes,sha256,owner_type,owner_id,created_at,expires_at FROM artifacts WHERE id=?`, artifactID).Scan(&value.ID, &value.Kind, &value.Path, &value.MediaType, &value.SizeBytes, &value.SHA256, &value.OwnerType, &value.OwnerID, &created, &expires)
		if err == nil {
			value.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
			value.ExpiresAt = parseNullTime(expires)
			values = append(values, value)
		}
	}
	return values, nil
}

func (r *TopologyRepository) DeleteArtifacts(ctx context.Context, ids []domain.ID) error {
	for _, id := range ids {
		if _, err := r.database.DB.ExecContext(ctx, `DELETE FROM artifacts WHERE id=?`, id); err != nil {
			return err
		}
	}
	return nil
}

func (r *TopologyRepository) CreateNode(ctx context.Context, node domain.Node, interfaces []domain.Interface) error {
	return r.database.Write(ctx, func(tx *sql.Tx) error {
		config, _ := json.Marshal(node.Config)
		_, err := tx.ExecContext(ctx, `INSERT INTO nodes(id,laboratory_id,name,kind,template_version_id,revision,desired_state,observed_state,cpu_count,cpu_quota_micros,memory_mib,storage_gib,interface_limit,process_limit,config_json,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, node.ID, node.LaboratoryID, node.Name, node.Kind, nullable(string(node.TemplateVersionID)), node.Revision, node.DesiredState, node.ObservedState, node.CPUCount, node.CPUQuotaMicros, node.MemoryMiB, node.StorageGiB, node.InterfaceLimit, node.ProcessLimit, config, node.CreatedAt.Format(time.RFC3339Nano), node.UpdatedAt.Format(time.RFC3339Nano))
		if err != nil {
			return err
		}
		for _, iface := range interfaces {
			if _, err = tx.ExecContext(ctx, `INSERT INTO interfaces(id,node_id,slot,name,driver,mac_address,oper_state,revision) VALUES(?,?,?,?,?,?,?,?)`, iface.ID, iface.NodeID, iface.Slot, iface.Name, iface.Driver, iface.MACAddress, iface.OperationalState, iface.Revision); err != nil {
				return err
			}
		}
		data := eventData(node)
		data["interfaces"] = interfaces
		return appendEvent(ctx, tx, "node.created", node.LaboratoryID, "node", node.ID, node.Revision, "", data)
	})
}

func (r *TopologyRepository) GetNode(ctx context.Context, id domain.ID) (domain.Node, error) {
	row := r.database.DB.QueryRowContext(ctx, nodeSelect+` WHERE id=?`, id)
	node, err := scanNode(row)
	if err == sql.ErrNoRows {
		return node, ErrNotFound
	}
	return node, err
}

func (r *TopologyRepository) ListNodes(ctx context.Context, labID domain.ID) ([]domain.Node, error) {
	rows, err := r.database.DB.QueryContext(ctx, nodeSelect+` WHERE laboratory_id=? ORDER BY name`, labID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var nodes []domain.Node
	for rows.Next() {
		node, err := scanNode(rows)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}
	return nodes, rows.Err()
}

func (r *TopologyRepository) ListAllNodes(ctx context.Context) ([]domain.Node, error) {
	rows, err := r.database.DB.QueryContext(ctx, nodeSelect+` ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var nodes []domain.Node
	for rows.Next() {
		node, err := scanNode(rows)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}
	return nodes, rows.Err()
}

func (r *TopologyRepository) SetNodeDesiredState(ctx context.Context, id domain.ID, expected domain.Revision, state domain.DesiredState) (domain.Node, error) {
	var node domain.Node
	err := r.database.Write(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `UPDATE nodes SET desired_state=?,revision=revision+1,updated_at=? WHERE id=? AND revision=?`, state, time.Now().UTC().Format(time.RFC3339Nano), id, expected)
		if err != nil {
			return err
		}
		count, _ := result.RowsAffected()
		if count == 0 {
			return domain.Problem{Code: "revision_conflict", Message: "node revision changed", Retryable: true, ResourceType: "node", ResourceID: id}
		}
		node, err = scanNode(tx.QueryRowContext(ctx, nodeSelect+` WHERE id=?`, id))
		if err != nil {
			return err
		}
		return appendEvent(ctx, tx, "node.desired_state_changed", node.LaboratoryID, "node", node.ID, node.Revision, "", map[string]any{"desired_state": state})
	})
	return node, err
}

func (r *TopologyRepository) SetNodeObservedState(ctx context.Context, id domain.ID, state domain.ObservedState, problem *domain.Problem) error {
	return r.database.Write(ctx, func(tx *sql.Tx) error {
		var current domain.ObservedState
		if err := tx.QueryRowContext(ctx, `SELECT observed_state FROM nodes WHERE id=?`, id).Scan(&current); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}
		if err := domain.ValidateNodeTransition(current, state); err != nil {
			return domain.Problem{Code: "invalid_node_transition", Message: err.Error(), ResourceType: "node", ResourceID: id}
		}
		body, _ := json.Marshal(problem)
		result, err := tx.ExecContext(ctx, `UPDATE nodes SET observed_state=?,last_error_json=?,updated_at=? WHERE id=?`, state, nullableBytes(body, problem != nil), time.Now().UTC().Format(time.RFC3339Nano), id)
		if err != nil {
			return err
		}
		count, _ := result.RowsAffected()
		if count == 0 {
			return ErrNotFound
		}
		var labID domain.ID
		var revision domain.Revision
		if err = tx.QueryRowContext(ctx, `SELECT laboratory_id,revision FROM nodes WHERE id=?`, id).Scan(&labID, &revision); err != nil {
			return err
		}
		return appendEvent(ctx, tx, "node.observed_state_changed", labID, "node", id, revision, "", map[string]any{"observed_state": state})
	})
}

func (r *TopologyRepository) DeleteNode(ctx context.Context, id domain.ID, expected domain.Revision) error {
	return r.database.Write(ctx, func(tx *sql.Tx) error {
		var labID domain.ID
		if err := tx.QueryRowContext(ctx, `SELECT laboratory_id FROM nodes WHERE id=? AND revision=?`, id, expected).Scan(&labID); err != nil {
			return domain.Problem{Code: "revision_conflict", Message: "node revision changed", Retryable: true, ResourceType: "node", ResourceID: id}
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM runtime_ownership WHERE (resource_type='node' AND resource_id=?) OR (resource_type='interface' AND resource_id IN (SELECT id FROM interfaces WHERE node_id=?))`, id, id); err != nil {
			return err
		}
		rows, err := tx.QueryContext(ctx, `SELECT id,revision FROM links WHERE endpoint_a_id IN (SELECT id FROM interfaces WHERE node_id=?) OR endpoint_b_id IN (SELECT id FROM interfaces WHERE node_id=?) ORDER BY id`, id, id)
		if err != nil {
			return err
		}
		var links []domain.Link
		for rows.Next() {
			var link domain.Link
			if err = rows.Scan(&link.ID, &link.Revision); err != nil {
				rows.Close()
				return err
			}
			links = append(links, link)
		}
		if err = rows.Close(); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `UPDATE interfaces SET desired_link_id=NULL WHERE desired_link_id IN (SELECT id FROM links WHERE endpoint_a_id IN (SELECT id FROM interfaces WHERE node_id=?) OR endpoint_b_id IN (SELECT id FROM interfaces WHERE node_id=?))`, id, id); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `DELETE FROM links WHERE endpoint_a_id IN (SELECT id FROM interfaces WHERE node_id=?) OR endpoint_b_id IN (SELECT id FROM interfaces WHERE node_id=?)`, id, id); err != nil {
			return err
		}
		for _, link := range links {
			if err = appendEvent(ctx, tx, "link.deleted", labID, "link", link.ID, link.Revision.Next(), "", nil); err != nil {
				return err
			}
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM interfaces WHERE node_id=?`, id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM nodes WHERE id=?`, id); err != nil {
			return err
		}
		return appendEvent(ctx, tx, "node.deleted", labID, "node", id, expected.Next(), "", nil)
	})
}

func (r *TopologyRepository) ListNodeLinks(ctx context.Context, nodeID domain.ID) ([]domain.Link, error) {
	rows, err := r.database.DB.QueryContext(ctx, `SELECT l.id,l.laboratory_id,l.endpoint_a_id,l.endpoint_b_id,l.revision,l.desired_state,l.observed_state FROM links l JOIN interfaces a ON a.id=l.endpoint_a_id JOIN interfaces b ON b.id=l.endpoint_b_id WHERE a.node_id=? OR b.node_id=? ORDER BY l.id`, nodeID, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var values []domain.Link
	for rows.Next() {
		var value domain.Link
		if err = rows.Scan(&value.ID, &value.LaboratoryID, &value.EndpointAID, &value.EndpointBID, &value.Revision, &value.DesiredState, &value.ObservedState); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func (r *TopologyRepository) ListNodeOwnedTaps(ctx context.Context, nodeID domain.ID) ([]string, error) {
	rows, err := r.database.DB.QueryContext(ctx, `SELECT object_name FROM runtime_ownership WHERE resource_type='interface' AND object_kind='tap' AND resource_id IN (SELECT id FROM interfaces WHERE node_id=?) ORDER BY object_name`, nodeID)
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

func (r *TopologyRepository) ListInterfaces(ctx context.Context, labID domain.ID) ([]domain.Interface, error) {
	rows, err := r.database.DB.QueryContext(ctx, `SELECT i.id,i.node_id,i.slot,i.name,COALESCE(i.driver,''),i.mac_address,COALESCE(i.desired_link_id,''),i.oper_state,i.revision FROM interfaces i JOIN nodes n ON n.id=i.node_id WHERE n.laboratory_id=? ORDER BY i.node_id,i.slot`, labID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var values []domain.Interface
	for rows.Next() {
		var v domain.Interface
		if err = rows.Scan(&v.ID, &v.NodeID, &v.Slot, &v.Name, &v.Driver, &v.MACAddress, &v.DesiredLinkID, &v.OperationalState, &v.Revision); err != nil {
			return nil, err
		}
		values = append(values, v)
	}
	return values, rows.Err()
}

func (r *TopologyRepository) CreateLink(ctx context.Context, link domain.Link) error {
	return r.database.Write(ctx, func(tx *sql.Tx) error {
		var labA, labB domain.ID
		if err := tx.QueryRowContext(ctx, `SELECT n.laboratory_id FROM interfaces i JOIN nodes n ON n.id=i.node_id WHERE i.id=?`, link.EndpointAID).Scan(&labA); err != nil {
			return err
		}
		if err := tx.QueryRowContext(ctx, `SELECT n.laboratory_id FROM interfaces i JOIN nodes n ON n.id=i.node_id WHERE i.id=?`, link.EndpointBID).Scan(&labB); err != nil {
			return err
		}
		if labA != labB || labA != link.LaboratoryID {
			return fmt.Errorf("link endpoints must belong to laboratory")
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO links(id,laboratory_id,endpoint_a_id,endpoint_b_id,revision,desired_state,observed_state) VALUES(?,?,?,?,?,?,?)`, link.ID, link.LaboratoryID, link.EndpointAID, link.EndpointBID, link.Revision, link.DesiredState, link.ObservedState); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE interfaces SET desired_link_id=? WHERE id IN (?,?) AND (desired_link_id IS NULL OR desired_link_id='')`, link.ID, link.EndpointAID, link.EndpointBID); err != nil {
			return err
		}
		return appendEvent(ctx, tx, "link.created", link.LaboratoryID, "link", link.ID, link.Revision, "", eventData(link))
	})
}

func (r *TopologyRepository) DeleteLink(ctx context.Context, id domain.ID) error {
	return r.database.Write(ctx, func(tx *sql.Tx) error {
		var labID domain.ID
		var revision domain.Revision
		if err := tx.QueryRowContext(ctx, `SELECT laboratory_id,revision FROM links WHERE id=?`, id).Scan(&labID, &revision); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE interfaces SET desired_link_id=NULL WHERE desired_link_id=?`, id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM links WHERE id=?`, id); err != nil {
			return err
		}
		return appendEvent(ctx, tx, "link.deleted", labID, "link", id, revision.Next(), "", nil)
	})
}

func (r *TopologyRepository) MarkLinkDisconnected(ctx context.Context, id domain.ID) error {
	return r.database.Write(ctx, func(tx *sql.Tx) error {
		var labID domain.ID
		var revision domain.Revision
		if err := tx.QueryRowContext(ctx, `SELECT laboratory_id,revision FROM links WHERE id=?`, id).Scan(&labID, &revision); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE links SET desired_state='disconnected',observed_state='disconnecting',revision=revision+1 WHERE id=?`, id); err != nil {
			return err
		}
		return appendEvent(ctx, tx, "link.disconnecting", labID, "link", id, revision.Next(), "", nil)
	})
}

func (r *TopologyRepository) MarkLinkConnected(ctx context.Context, id domain.ID) error {
	return r.database.Write(ctx, func(tx *sql.Tx) error {
		var labID domain.ID
		var revision domain.Revision
		if err := tx.QueryRowContext(ctx, `SELECT laboratory_id,revision FROM links WHERE id=?`, id).Scan(&labID, &revision); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE links SET desired_state='connected',observed_state='pending',revision=revision+1 WHERE id=?`, id); err != nil {
			return err
		}
		return appendEvent(ctx, tx, "link.connecting", labID, "link", id, revision.Next(), "", nil)
	})
}

func (r *TopologyRepository) CommitLinkReconnect(ctx context.Context, id domain.ID, expected domain.Revision, retainedEndpointID, replacementEndpointID domain.ID) (domain.Link, error) {
	var updated domain.Link
	err := r.database.Write(ctx, func(tx *sql.Tx) error {
		var previous domain.Link
		if err := tx.QueryRowContext(ctx, `SELECT id,laboratory_id,endpoint_a_id,endpoint_b_id,revision,desired_state,observed_state FROM links WHERE id=?`, id).Scan(&previous.ID, &previous.LaboratoryID, &previous.EndpointAID, &previous.EndpointBID, &previous.Revision, &previous.DesiredState, &previous.ObservedState); err != nil {
			return err
		}
		if previous.Revision != expected {
			return domain.Problem{Code: "revision_conflict", Message: "link revision changed", Retryable: true, ResourceType: "link", ResourceID: id}
		}
		if retainedEndpointID != previous.EndpointAID && retainedEndpointID != previous.EndpointBID {
			return domain.Problem{Code: "invalid_reconnect_endpoint", Message: "retained endpoint is not attached to the link", ResourceType: "link", ResourceID: id}
		}
		var laboratoryID, desiredLinkID domain.ID
		if err := tx.QueryRowContext(ctx, `SELECT n.laboratory_id,COALESCE(i.desired_link_id,'') FROM interfaces i JOIN nodes n ON n.id=i.node_id WHERE i.id=?`, replacementEndpointID).Scan(&laboratoryID, &desiredLinkID); err != nil {
			return domain.Problem{Code: "endpoint_not_found", Message: "replacement endpoint not found", ResourceType: "interface", ResourceID: replacementEndpointID}
		}
		if laboratoryID != previous.LaboratoryID || (desiredLinkID != "" && desiredLinkID != id) {
			return domain.Problem{Code: "endpoint_unavailable", Message: "replacement endpoint is unavailable", ResourceType: "interface", ResourceID: replacementEndpointID}
		}
		oldEndpointID := previous.EndpointAID
		updated = previous
		if retainedEndpointID == previous.EndpointAID {
			oldEndpointID = previous.EndpointBID
			updated.EndpointBID = replacementEndpointID
		} else {
			updated.EndpointAID = replacementEndpointID
		}
		var duplicate int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM links WHERE id<>? AND ((endpoint_a_id=? AND endpoint_b_id=?) OR (endpoint_a_id=? AND endpoint_b_id=?))`, id, updated.EndpointAID, updated.EndpointBID, updated.EndpointBID, updated.EndpointAID).Scan(&duplicate); err != nil {
			return err
		}
		if duplicate > 0 {
			return domain.Problem{Code: "duplicate_link", Message: "replacement would duplicate an existing link", ResourceType: "link", ResourceID: id}
		}
		if _, err := tx.ExecContext(ctx, `UPDATE interfaces SET desired_link_id=NULL WHERE id=? AND desired_link_id=?`, oldEndpointID, id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE interfaces SET desired_link_id=? WHERE id=?`, id, replacementEndpointID); err != nil {
			return err
		}
		updated.Revision = previous.Revision.Next()
		updated.DesiredState, updated.ObservedState = "connected", "connected"
		if _, err := tx.ExecContext(ctx, `UPDATE interfaces SET oper_state='down' WHERE id=?`, oldEndpointID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE interfaces SET oper_state='up' WHERE id IN (?,?)`, updated.EndpointAID, updated.EndpointBID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE links SET endpoint_a_id=?,endpoint_b_id=?,revision=?,desired_state='connected',observed_state='connected' WHERE id=?`, updated.EndpointAID, updated.EndpointBID, updated.Revision, id); err != nil {
			return err
		}
		return appendEvent(ctx, tx, "link.reconnected", updated.LaboratoryID, "link", id, updated.Revision, "", map[string]any{"link": updated, "previous_endpoint_a_id": previous.EndpointAID, "previous_endpoint_b_id": previous.EndpointBID})
	})
	return updated, err
}

func (r *TopologyRepository) SetLinkObservedState(ctx context.Context, id domain.ID, state string) error {
	return r.database.Write(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `UPDATE links SET observed_state=? WHERE id=? AND observed_state<>?`, state, id, state)
		if err != nil {
			return err
		}
		changed, _ := result.RowsAffected()
		if changed == 0 {
			return nil
		}
		var laboratoryID domain.ID
		var revision domain.Revision
		if err = tx.QueryRowContext(ctx, `SELECT laboratory_id,revision FROM links WHERE id=?`, id).Scan(&laboratoryID, &revision); err != nil {
			return err
		}
		return appendEvent(ctx, tx, "link.observed_state_changed", laboratoryID, "link", id, revision, "", map[string]any{"observed_state": state})
	})
}

func (r *TopologyRepository) SetInterfaceOperationalState(ctx context.Context, id domain.ID, state string) error {
	return r.database.Write(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `UPDATE interfaces SET oper_state=?,revision=revision+1 WHERE id=? AND oper_state<>?`, state, id, state)
		if err != nil {
			return err
		}
		changed, _ := result.RowsAffected()
		if changed == 0 {
			return nil
		}
		var laboratoryID domain.ID
		var revision domain.Revision
		if err = tx.QueryRowContext(ctx, `SELECT n.laboratory_id,i.revision FROM interfaces i JOIN nodes n ON n.id=i.node_id WHERE i.id=?`, id).Scan(&laboratoryID, &revision); err != nil {
			return err
		}
		return appendEvent(ctx, tx, "interface.operational_state_changed", laboratoryID, "interface", id, revision, "", map[string]any{"operational_state": state})
	})
}

func (r *TopologyRepository) ListLinks(ctx context.Context, labID domain.ID) ([]domain.Link, error) {
	rows, err := r.database.DB.QueryContext(ctx, `SELECT id,laboratory_id,endpoint_a_id,endpoint_b_id,revision,desired_state,observed_state FROM links WHERE laboratory_id=? ORDER BY id`, labID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var values []domain.Link
	for rows.Next() {
		var v domain.Link
		if err = rows.Scan(&v.ID, &v.LaboratoryID, &v.EndpointAID, &v.EndpointBID, &v.Revision, &v.DesiredState, &v.ObservedState); err != nil {
			return nil, err
		}
		values = append(values, v)
	}
	return values, rows.Err()
}

func (r *TopologyRepository) GetLink(ctx context.Context, id domain.ID) (domain.Link, error) {
	var value domain.Link
	err := r.database.DB.QueryRowContext(ctx, `SELECT id,laboratory_id,endpoint_a_id,endpoint_b_id,revision,desired_state,observed_state FROM links WHERE id=?`, id).Scan(&value.ID, &value.LaboratoryID, &value.EndpointAID, &value.EndpointBID, &value.Revision, &value.DesiredState, &value.ObservedState)
	if err == sql.ErrNoRows {
		return value, ErrNotFound
	}
	return value, err
}

func (r *TopologyRepository) LinkEndpointsReady(ctx context.Context, link domain.Link) (bool, error) {
	var count int
	err := r.database.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM interfaces i JOIN nodes n ON n.id=i.node_id WHERE i.id IN (?,?) AND n.observed_state='running'`, link.EndpointAID, link.EndpointBID).Scan(&count)
	return count == 2, err
}

func (r *TopologyRepository) ListNetworkAttachments(ctx context.Context, laboratoryID domain.ID) ([]domain.NetworkAttachment, error) {
	return (&Repositories{database: r.database}).ListNetworkAttachments(ctx, laboratoryID)
}

func (r *TopologyRepository) SetNetworkAttachmentState(ctx context.Context, id domain.ID, state string, problem *domain.Problem) error {
	return (&Repositories{database: r.database}).SetNetworkAttachmentState(ctx, id, state, problem)
}

func (r *TopologyRepository) ListNetworkObjectLinks(ctx context.Context, laboratoryID domain.ID) ([]domain.NetworkObjectLink, error) {
	return (&Repositories{database: r.database}).ListNetworkObjectLinks(ctx, laboratoryID)
}

func (r *TopologyRepository) SetNetworkObjectLinkState(ctx context.Context, id domain.ID, state string, problem *domain.Problem) error {
	return (&Repositories{database: r.database}).SetNetworkObjectLinkState(ctx, id, state, problem)
}

func (r *TopologyRepository) DeleteNetworkObjectLink(ctx context.Context, id domain.ID) error {
	return (&Repositories{database: r.database}).DeleteNetworkObjectLink(ctx, id)
}

func (r *TopologyRepository) Snapshot(ctx context.Context, id domain.ID) (domain.TopologySnapshot, error) {
	lab, err := r.GetLaboratory(ctx, id)
	if err != nil {
		return domain.TopologySnapshot{}, err
	}
	nodes, err := r.ListNodes(ctx, id)
	if err != nil {
		return domain.TopologySnapshot{}, err
	}
	interfaces, err := r.ListInterfaces(ctx, id)
	if err != nil {
		return domain.TopologySnapshot{}, err
	}
	links, err := r.ListLinks(ctx, id)
	if err != nil {
		return domain.TopologySnapshot{}, err
	}
	networkObjects, err := (&Repositories{database: r.database}).ListNetworkObjects(ctx, id)
	if err != nil {
		return domain.TopologySnapshot{}, err
	}
	attachments, err := r.ListNetworkAttachments(ctx, id)
	if err != nil {
		return domain.TopologySnapshot{}, err
	}
	objectLinks, err := r.ListNetworkObjectLinks(ctx, id)
	if err != nil {
		return domain.TopologySnapshot{}, err
	}
	placements, err := r.ListPlacements(ctx, id)
	if err != nil {
		return domain.TopologySnapshot{}, err
	}
	var sequence int64
	_ = r.database.DB.QueryRowContext(ctx, `SELECT COALESCE(MAX(sequence),0) FROM outbox_events`).Scan(&sequence)
	return domain.TopologySnapshot{Laboratory: lab, Nodes: nodes, Interfaces: interfaces, Links: links, NetworkObjects: networkObjects, Attachments: attachments, NetworkObjectLinks: objectLinks, Placements: placements, Sequence: sequence}, nil
}

type scanner interface{ Scan(...any) error }

func scanLaboratory(row scanner) (domain.Laboratory, error) {
	var lab domain.Laboratory
	var created, updated string
	err := row.Scan(&lab.ID, &lab.Name, &lab.Description, &lab.Revision, &lab.RecoveryPolicy, &lab.LifecycleState, &created, &updated)
	if err != nil {
		return lab, err
	}
	lab.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	lab.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	return lab, nil
}
func scanNode(row scanner) (domain.Node, error) {
	var node domain.Node
	var config, lastError []byte
	var created, updated string
	err := row.Scan(&node.ID, &node.LaboratoryID, &node.Name, &node.Kind, &node.TemplateVersionID, &node.Revision, &node.DesiredState, &node.ObservedState, &node.CPUCount, &node.CPUQuotaMicros, &node.MemoryMiB, &node.StorageGiB, &node.InterfaceLimit, &node.ProcessLimit, &config, &lastError, &created, &updated)
	if err != nil {
		return node, err
	}
	_ = json.Unmarshal(config, &node.Config)
	if len(lastError) > 0 {
		node.LastError = &domain.Problem{}
		_ = json.Unmarshal(lastError, node.LastError)
	}
	node.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	node.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	return node, nil
}

const nodeSelect = `SELECT id,laboratory_id,name,kind,COALESCE(template_version_id,''),revision,desired_state,observed_state,COALESCE(cpu_count,0),COALESCE(cpu_quota_micros,0),COALESCE(memory_mib,0),COALESCE(storage_gib,0),COALESCE(interface_limit,0),COALESCE(process_limit,0),config_json,COALESCE(last_error_json,''),created_at,updated_at FROM nodes`

func appendEvent(ctx context.Context, tx *sql.Tx, eventType string, labID domain.ID, resourceType string, resourceID domain.ID, revision domain.Revision, taskID domain.ID, data map[string]any) error {
	payload, _ := json.Marshal(data)
	_, err := tx.ExecContext(ctx, `INSERT INTO outbox_events(event_type,laboratory_id,resource_type,resource_id,revision,task_id,payload_json,occurred_at) VALUES(?,?,?,?,?,?,?,?)`, eventType, nullable(string(labID)), resourceType, resourceID, revision, nullable(string(taskID)), payload, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}
