package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/netlab/netlab/internal/domain"
)

func (r *TopologyRepository) ImportTopology(ctx context.Context, lab domain.Laboratory, nodes []domain.Node, interfaces []domain.Interface, links []domain.Link, networkObjects []domain.NetworkObject, networkObjectLinks []domain.NetworkObjectLink, placements []domain.TopologyPlacement) error {
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
		for _, link := range networkObjectLinks {
			if _, err := tx.ExecContext(ctx, `INSERT INTO network_object_links(id,laboratory_id,object_a_id,port_a_name,object_b_id,port_b_name,revision,desired_state,observed_state,last_error_json) VALUES(?,?,?,?,?,?,?,?,?,?)`, link.ID, link.LaboratoryID, link.ObjectAID, link.PortAName, link.ObjectBID, link.PortBName, link.Revision, link.DesiredState, link.ObservedState, nil); err != nil {
				return err
			}
			if link.DesiredState == "deleted" {
				continue
			}
			for _, endpoint := range []struct {
				objectID domain.ID
				portName string
			}{{link.ObjectAID, link.PortAName}, {link.ObjectBID, link.PortBName}} {
				if _, err := tx.ExecContext(ctx, `INSERT INTO topology_endpoint_reservations(laboratory_id,owner_type,owner_id,port_name,resource_type,resource_id) VALUES(?,?,?,?,?,?)`, lab.ID, "network_object", endpoint.objectID, endpoint.portName, "network_object_link", link.ID); err != nil {
					return err
				}
			}
		}
		for _, placement := range placements {
			if placement.LaboratoryID != lab.ID {
				return domain.Problem{Code: "invalid_placement", Message: "placement laboratory does not match import", ResourceType: string(placement.ResourceType), ResourceID: placement.ResourceID}
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO topology_placements(laboratory_id,resource_id,resource_type,x,y,revision,updated_at) VALUES(?,?,?,?,?,?,?)`, placement.LaboratoryID, placement.ResourceID, placement.ResourceType, placement.X, placement.Y, placement.Revision, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
				return err
			}
		}
		return appendEvent(ctx, tx, "laboratory.imported", lab.ID, "laboratory", lab.ID, lab.Revision, "", map[string]any{"nodes": len(nodes), "links": len(links), "network_objects": len(networkObjects), "network_object_links": len(networkObjectLinks), "placements": len(placements)})
	})
}
