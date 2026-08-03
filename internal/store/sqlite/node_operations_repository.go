package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/netlab/netlab/internal/domain"
)

func (r *TopologyRepository) GetInterface(ctx context.Context, id domain.ID) (domain.Interface, error) {
	var value domain.Interface
	var driver, desiredLink sql.NullString
	err := r.database.DB.QueryRowContext(ctx, `SELECT id,node_id,slot,name,driver,mac_address,desired_link_id,oper_state,revision FROM interfaces WHERE id=?`, id).Scan(&value.ID, &value.NodeID, &value.Slot, &value.Name, &driver, &value.MACAddress, &desiredLink, &value.OperationalState, &value.Revision)
	if err == sql.ErrNoRows {
		return value, ErrNotFound
	}
	value.Driver = driver.String
	value.DesiredLinkID = domain.ID(desiredLink.String)
	return value, err
}

func (r *TopologyRepository) AddInterface(ctx context.Context, value domain.Interface) error {
	return r.database.Write(ctx, func(tx *sql.Tx) error {
		var labID domain.ID
		if err := tx.QueryRowContext(ctx, `SELECT laboratory_id FROM nodes WHERE id=?`, value.NodeID).Scan(&labID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO interfaces(id,node_id,slot,name,driver,mac_address,oper_state,revision) VALUES(?,?,?,?,?,?,?,?)`, value.ID, value.NodeID, value.Slot, value.Name, nullable(value.Driver), value.MACAddress, value.OperationalState, value.Revision); err != nil {
			return err
		}
		return appendEvent(ctx, tx, "interface.created", labID, "interface", value.ID, value.Revision, "", eventData(value))
	})
}

func (r *TopologyRepository) ReserveInterface(ctx context.Context, value domain.Interface, limit int) (domain.Interface, error) {
	if limit < 1 {
		return domain.Interface{}, domain.Problem{Code: "resource_exhausted", Message: "interface capacity must be positive", ResourceType: "node", ResourceID: value.NodeID, Phase: "interface_admission", Cleanup: "no side effects created"}
	}
	err := r.database.Write(ctx, func(tx *sql.Tx) error {
		var labID domain.ID
		if err := tx.QueryRowContext(ctx, `SELECT laboratory_id FROM nodes WHERE id=?`, value.NodeID).Scan(&labID); err != nil {
			return err
		}
		rows, err := tx.QueryContext(ctx, `SELECT slot FROM interfaces WHERE node_id=? AND slot>=0 AND slot<? ORDER BY slot`, value.NodeID, limit)
		if err != nil {
			return err
		}
		occupied := make([]bool, limit)
		for rows.Next() {
			var slot int
			if err = rows.Scan(&slot); err != nil {
				rows.Close()
				return err
			}
			occupied[slot] = true
		}
		if err = rows.Close(); err != nil {
			return err
		}
		value.Slot = -1
		for slot, used := range occupied {
			if !used {
				value.Slot = slot
				break
			}
		}
		if value.Slot < 0 {
			return domain.Problem{Code: "resource_exhausted", Message: fmt.Sprintf("node interface capacity of %d has been reached", limit), ResourceType: "node", ResourceID: value.NodeID, Phase: "interface_admission", Cleanup: "no side effects created", OperatorHint: "remove an interface before adding another", Details: map[string]any{"interface_limit": limit}}
		}
		value.Name = fmt.Sprintf("eth%d", value.Slot)
		if _, err = tx.ExecContext(ctx, `INSERT INTO interfaces(id,node_id,slot,name,driver,mac_address,oper_state,revision) VALUES(?,?,?,?,?,?,?,?)`, value.ID, value.NodeID, value.Slot, value.Name, nullable(value.Driver), value.MACAddress, value.OperationalState, value.Revision); err != nil {
			return err
		}
		return appendEvent(ctx, tx, "interface.created", labID, "interface", value.ID, value.Revision, "", eventData(value))
	})
	return value, err
}

func (r *TopologyRepository) DeleteInterface(ctx context.Context, id domain.ID, expected domain.Revision) error {
	return r.database.Write(ctx, func(tx *sql.Tx) error {
		var revision domain.Revision
		var labID domain.ID
		var desiredLink sql.NullString
		if err := tx.QueryRowContext(ctx, `SELECT i.revision,i.desired_link_id,n.laboratory_id FROM interfaces i JOIN nodes n ON n.id=i.node_id WHERE i.id=?`, id).Scan(&revision, &desiredLink, &labID); err != nil {
			return err
		}
		if revision != expected {
			return domain.Problem{Code: "revision_conflict", Message: "interface revision mismatch", ResourceType: "interface", ResourceID: id}
		}
		if desiredLink.Valid && desiredLink.String != "" {
			return fmt.Errorf("disconnect interface before removing it")
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM interfaces WHERE id=?`, id); err != nil {
			return err
		}
		return appendEvent(ctx, tx, "interface.deleted", labID, "interface", id, revision.Next(), "", nil)
	})
}

func (r *TopologyRepository) UpdateNodeResources(ctx context.Context, id domain.ID, expected domain.Revision, cpuCount int, quota int64, memoryMiB int) (domain.Node, error) {
	result, err := r.database.DB.ExecContext(ctx, `UPDATE nodes SET cpu_count=?,cpu_quota_micros=?,memory_mib=?,revision=revision+1,updated_at=? WHERE id=? AND revision=?`, cpuCount, quota, memoryMiB, time.Now().UTC().Format(time.RFC3339Nano), id, expected)
	if err != nil {
		return domain.Node{}, err
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return domain.Node{}, domain.Problem{Code: "revision_conflict", Message: "node revision mismatch", ResourceType: "node", ResourceID: id}
	}
	return r.GetNode(ctx, id)
}

func (r *TopologyRepository) UpdateNodeSettings(ctx context.Context, id domain.ID, expected domain.Revision, settings domain.NodeSettings) (domain.Node, error) {
	settings.Name = strings.TrimSpace(settings.Name)
	if settings.Name == "" {
		return domain.Node{}, domain.Problem{Code: "invalid_node_settings", Message: "node name is required", ResourceType: "node", ResourceID: id}
	}
	if settings.CPUCount < 1 || settings.CPUCount > 256 || settings.CPUQuotaMicros < 0 || (settings.CPUQuotaMicros > 0 && settings.CPUQuotaMicros < 1000) || settings.MemoryMiB < 64 || settings.InterfaceLimit < 1 || settings.ProcessLimit < 1 || settings.ProcessLimit > 1048576 {
		return domain.Node{}, domain.Problem{Code: "invalid_node_settings", Message: "node settings are outside supported limits", ResourceType: "node", ResourceID: id}
	}
	if err := domain.ValidateNodeNetworkInterfaces(settings.NetworkInterfaces); err != nil {
		return domain.Node{}, domain.Problem{Code: "invalid_node_network", Message: err.Error(), ResourceType: "node", ResourceID: id}
	}
	var updated domain.Node
	err := r.database.Write(ctx, func(tx *sql.Tx) error {
		var laboratoryID domain.ID
		var revision domain.Revision
		var desired domain.DesiredState
		var observed domain.ObservedState
		var configBody []byte
		if err := tx.QueryRowContext(ctx, `SELECT laboratory_id,revision,desired_state,observed_state,config_json FROM nodes WHERE id=?`, id).Scan(&laboratoryID, &revision, &desired, &observed, &configBody); err != nil {
			if err == sql.ErrNoRows {
				return ErrNotFound
			}
			return err
		}
		if revision != expected {
			return domain.Problem{Code: "revision_conflict", Message: "node revision mismatch", Retryable: true, ResourceType: "node", ResourceID: id, Details: map[string]any{"current_revision": revision}}
		}
		if desired != domain.DesiredStopped || observed != domain.ObservedStopped {
			return domain.Problem{Code: "node_not_stopped", Message: "node settings can only be changed while the node is stopped", ResourceType: "node", ResourceID: id}
		}
		var interfaceCount int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM interfaces WHERE node_id=?`, id).Scan(&interfaceCount); err != nil {
			return err
		}
		if settings.InterfaceLimit < interfaceCount {
			return domain.Problem{Code: "invalid_node_settings", Message: "interface limit cannot be lower than the current interface count", ResourceType: "node", ResourceID: id, Details: map[string]any{"interface_count": interfaceCount}}
		}
		var duplicateCount int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM nodes WHERE laboratory_id=? AND name=? AND id<>?`, laboratoryID, settings.Name, id).Scan(&duplicateCount); err != nil {
			return err
		}
		if duplicateCount > 0 {
			return domain.Problem{Code: "node_name_conflict", Message: "a node with this name already exists in the laboratory", ResourceType: "node", ResourceID: id}
		}
		var config map[string]any
		_ = json.Unmarshal(configBody, &config)
		if config == nil {
			config = map[string]any{}
		}
		if len(settings.NetworkInterfaces) > 0 {
			rows, queryErr := tx.QueryContext(ctx, `SELECT id,slot,name,COALESCE(driver,''),mac_address FROM interfaces WHERE node_id=? ORDER BY slot`, id)
			if queryErr != nil {
				return queryErr
			}
			var interfaces []domain.Interface
			for rows.Next() {
				var iface domain.Interface
				iface.NodeID = id
				if queryErr = rows.Scan(&iface.ID, &iface.Slot, &iface.Name, &iface.Driver, &iface.MACAddress); queryErr != nil {
					rows.Close()
					return queryErr
				}
				interfaces = append(interfaces, iface)
			}
			rows.Close()
			if len(interfaces) != len(settings.NetworkInterfaces) {
				return domain.Problem{Code: "invalid_node_settings", Message: "network settings must include every current interface", ResourceType: "node", ResourceID: id}
			}
			requested := make(map[domain.ID]domain.NodeNetworkInterfaceSettings, len(settings.NetworkInterfaces))
			for _, value := range settings.NetworkInterfaces {
				requested[value.ID] = value
			}
			descriptors := make([]map[string]any, 0, len(interfaces))
			network := make([]map[string]any, 0, len(interfaces))
			for _, iface := range interfaces {
				value, ok := requested[iface.ID]
				if !ok || value.Name != iface.Name {
					return domain.Problem{Code: "invalid_node_settings", Message: "interface identity does not match the node", ResourceType: "interface", ResourceID: iface.ID}
				}
				if _, updateErr := tx.ExecContext(ctx, `UPDATE interfaces SET driver=?,revision=revision+1 WHERE id=? AND node_id=?`, value.Driver, iface.ID, id); updateErr != nil {
					return updateErr
				}
				descriptors = append(descriptors, map[string]any{"id": string(iface.ID), "slot": iface.Slot, "name": iface.Name, "driver": value.Driver, "mac_address": iface.MACAddress})
				network = append(network, map[string]any{"name": iface.Name, "modes": value.Modes, "addresses": value.Addresses, "routes": value.Routes})
			}
			config["interfaces"] = descriptors
			config["network_interfaces"] = network
		}
		updatedConfig, err := json.Marshal(config)
		if err != nil {
			return err
		}
		result, err := tx.ExecContext(ctx, `UPDATE nodes SET name=?,cpu_count=?,cpu_quota_micros=?,memory_mib=?,interface_limit=?,process_limit=?,config_json=?,revision=revision+1,updated_at=? WHERE id=? AND revision=? AND desired_state='stopped' AND observed_state='stopped'`, settings.Name, settings.CPUCount, settings.CPUQuotaMicros, settings.MemoryMiB, settings.InterfaceLimit, settings.ProcessLimit, updatedConfig, time.Now().UTC().Format(time.RFC3339Nano), id, expected)
		if err != nil {
			return err
		}
		if affected, _ := result.RowsAffected(); affected != 1 {
			return domain.Problem{Code: "revision_conflict", Message: "node revision mismatch", Retryable: true, ResourceType: "node", ResourceID: id, Details: map[string]any{"current_revision": revision}}
		}
		updated, err = scanNode(tx.QueryRowContext(ctx, nodeSelect+` WHERE id=?`, id))
		if err != nil {
			return err
		}
		return appendEvent(ctx, tx, "node.updated", updated.LaboratoryID, "node", updated.ID, updated.Revision, "", eventData(updated))
	})
	return updated, err
}

func (r *Repositories) CreatePortMapping(ctx context.Context, value domain.PortMapping) error {
	_, err := r.database.DB.ExecContext(ctx, `INSERT INTO port_mappings(id,node_id,protocol,host_address,host_port,guest_address,guest_port,revision,observed_state,created_at) VALUES(?,?,?,?,?,?,?,?,?,?)`, value.ID, value.NodeID, value.Protocol, value.HostAddress, value.HostPort, value.GuestAddress, value.GuestPort, value.Revision, value.ObservedState, value.CreatedAt.Format(time.RFC3339Nano))
	return err
}

func (r *Repositories) GetPortMapping(ctx context.Context, id domain.ID) (domain.PortMapping, error) {
	var value domain.PortMapping
	var createdAt string
	var errorJSON []byte
	err := r.database.DB.QueryRowContext(ctx, `SELECT id,node_id,protocol,host_address,host_port,guest_address,guest_port,revision,observed_state,COALESCE(last_error_json,''),created_at FROM port_mappings WHERE id=?`, id).Scan(&value.ID, &value.NodeID, &value.Protocol, &value.HostAddress, &value.HostPort, &value.GuestAddress, &value.GuestPort, &value.Revision, &value.ObservedState, &errorJSON, &createdAt)
	if err == sql.ErrNoRows {
		return value, ErrNotFound
	}
	if len(errorJSON) > 0 {
		value.LastError = &domain.Problem{}
		_ = json.Unmarshal(errorJSON, value.LastError)
	}
	value.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	return value, err
}

func (r *Repositories) ListNodePortMappings(ctx context.Context, nodeID domain.ID) ([]domain.PortMapping, error) {
	rows, err := r.database.DB.QueryContext(ctx, `SELECT id,node_id,protocol,host_address,host_port,guest_address,guest_port,revision,observed_state,COALESCE(last_error_json,''),created_at FROM port_mappings WHERE node_id=? ORDER BY created_at DESC`, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var values []domain.PortMapping
	for rows.Next() {
		var value domain.PortMapping
		var createdAt string
		var errorJSON []byte
		if err = rows.Scan(&value.ID, &value.NodeID, &value.Protocol, &value.HostAddress, &value.HostPort, &value.GuestAddress, &value.GuestPort, &value.Revision, &value.ObservedState, &errorJSON, &createdAt); err != nil {
			return nil, err
		}
		if len(errorJSON) > 0 {
			value.LastError = &domain.Problem{}
			_ = json.Unmarshal(errorJSON, value.LastError)
		}
		value.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		values = append(values, value)
	}
	return values, rows.Err()
}

func (r *Repositories) ListAllPortMappings(ctx context.Context) ([]domain.PortMapping, error) {
	rows, err := r.database.DB.QueryContext(ctx, `SELECT id FROM port_mappings ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []domain.ID
	for rows.Next() {
		var id domain.ID
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	values := make([]domain.PortMapping, 0, len(ids))
	for _, id := range ids {
		value, getErr := r.GetPortMapping(ctx, id)
		if getErr != nil {
			return nil, getErr
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func (r *Repositories) SetPortMappingState(ctx context.Context, id domain.ID, state string, problem *domain.Problem) error {
	body, _ := json.Marshal(problem)
	_, err := r.database.DB.ExecContext(ctx, `UPDATE port_mappings SET observed_state=?,last_error_json=? WHERE id=?`, state, nullableBytes(body, problem != nil), id)
	return err
}

func (r *Repositories) DeletePortMapping(ctx context.Context, id domain.ID) error {
	_, err := r.database.DB.ExecContext(ctx, `DELETE FROM port_mappings WHERE id=?`, id)
	return err
}
