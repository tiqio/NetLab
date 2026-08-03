package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/netlab/netlab/internal/domain"
)

func (r *TopologyRepository) ImportTopology(ctx context.Context, lab domain.Laboratory, nodes []domain.Node, interfaces []domain.Interface, links []domain.Link, networkObjects []domain.NetworkObject) error {
	return r.database.Write(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO laboratories(id,name,description,revision,recovery_policy,lifecycle_state,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?)`, lab.ID, lab.Name, lab.Description, lab.Revision, lab.RecoveryPolicy, lab.LifecycleState, lab.CreatedAt.Format(time.RFC3339Nano), lab.UpdatedAt.Format(time.RFC3339Nano)); err != nil {
			return err
		}
		for _, node := range nodes {
			config, _ := json.Marshal(node.Config)
			if _, err := tx.ExecContext(ctx, `INSERT INTO nodes(id,laboratory_id,name,kind,template_version_id,revision,desired_state,observed_state,cpu_count,cpu_quota_micros,memory_mib,config_json,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, node.ID, node.LaboratoryID, node.Name, node.Kind, nullable(string(node.TemplateVersionID)), node.Revision, node.DesiredState, node.ObservedState, node.CPUCount, node.CPUQuotaMicros, node.MemoryMiB, config, node.CreatedAt.Format(time.RFC3339Nano), node.UpdatedAt.Format(time.RFC3339Nano)); err != nil {
				return err
			}
		}
		for _, iface := range interfaces {
			if _, err := tx.ExecContext(ctx, `INSERT INTO interfaces(id,node_id,slot,name,driver,mac_address,oper_state,revision) VALUES(?,?,?,?,?,?,?,?)`, iface.ID, iface.NodeID, iface.Slot, iface.Name, nullable(iface.Driver), iface.MACAddress, iface.OperationalState, iface.Revision); err != nil {
				return err
			}
		}
		for _, link := range links {
			if _, err := tx.ExecContext(ctx, `INSERT INTO links(id,laboratory_id,endpoint_a_id,endpoint_b_id,revision,desired_state,observed_state) VALUES(?,?,?,?,?,?,?)`, link.ID, link.LaboratoryID, link.EndpointAID, link.EndpointBID, link.Revision, link.DesiredState, link.ObservedState); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `UPDATE interfaces SET desired_link_id=? WHERE id IN (?,?)`, link.ID, link.EndpointAID, link.EndpointBID); err != nil {
				return err
			}
		}
		for _, networkObject := range networkObjects {
			config, _ := json.Marshal(networkObject.Config)
			if _, err := tx.ExecContext(ctx, `INSERT INTO network_objects(id,laboratory_id,name,kind,revision,desired_state,observed_state,config_json,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?)`, networkObject.ID, networkObject.LaboratoryID, networkObject.Name, networkObject.Kind, networkObject.Revision, networkObject.DesiredState, networkObject.ObservedState, config, networkObject.CreatedAt.Format(time.RFC3339Nano), networkObject.UpdatedAt.Format(time.RFC3339Nano)); err != nil {
				return err
			}
		}
		return appendEvent(ctx, tx, "laboratory.imported", lab.ID, "laboratory", lab.ID, lab.Revision, "", map[string]any{"nodes": len(nodes), "links": len(links), "network_objects": len(networkObjects)})
	})
}
