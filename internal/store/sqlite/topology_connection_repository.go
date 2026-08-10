package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/netlab/netlab/internal/domain"
)

func (r *Repositories) ResolveConnectionEndpoint(ctx context.Context, reference domain.ConnectionEndpoint) (domain.ConnectionEndpoint, error) {
	switch reference.Kind {
	case domain.ConnectionEndpointNodeInterface:
		var value domain.ConnectionEndpoint
		var nodeName, interfaceName string
		err := r.database.DB.QueryRowContext(ctx, `SELECT n.laboratory_id,n.id,n.kind,n.name,i.id,i.name FROM interfaces i JOIN nodes n ON n.id=i.node_id WHERE i.id=?`, reference.PortID).Scan(&value.LaboratoryID, &value.ResourceID, &value.ResourceKind, &nodeName, &value.PortID, &interfaceName)
		if err == sql.ErrNoRows {
			return value, ErrNotFound
		}
		if err != nil {
			return value, err
		}
		if reference.LaboratoryID != "" && reference.LaboratoryID != value.LaboratoryID {
			return value, domain.Problem{Code: "cross_laboratory_connection", Message: "node interface belongs to another laboratory", ResourceType: "node", ResourceID: value.ResourceID, Phase: "connection_validation"}
		}
		if reference.ResourceID != "" && reference.ResourceID != value.ResourceID {
			return value, domain.Problem{Code: "endpoint_missing", Message: "node interface does not belong to the requested resource", ResourceType: "node", ResourceID: reference.ResourceID, Phase: "connection_validation"}
		}
		value.Kind = domain.ConnectionEndpointNodeInterface
		value.PortName = interfaceName
		value.DisplayName = nodeName + ":" + interfaceName
		value.Capabilities = []string{"connect", "capture", "traffic_filter"}
		value.Availability, err = r.connectionEndpointAvailability(ctx, value.LaboratoryID, "node_interface", value.PortID, interfaceName)
		return value, err
	case domain.ConnectionEndpointNetworkObjectPort, domain.ConnectionEndpointNetworkObjectAccess:
		var value domain.ConnectionEndpoint
		var objectName string
		var configJSON []byte
		err := r.database.DB.QueryRowContext(ctx, `SELECT laboratory_id,id,kind,name,config_json FROM network_objects WHERE id=?`, reference.ResourceID).Scan(&value.LaboratoryID, &value.ResourceID, &value.ResourceKind, &objectName, &configJSON)
		if err == sql.ErrNoRows {
			return value, ErrNotFound
		}
		if err != nil {
			return value, err
		}
		if reference.LaboratoryID != "" && reference.LaboratoryID != value.LaboratoryID {
			return value, domain.Problem{Code: "cross_laboratory_connection", Message: "network object belongs to another laboratory", ResourceType: "network_object", ResourceID: value.ResourceID, Phase: "connection_validation"}
		}
		value.Kind = reference.Kind
		value.Capabilities = []string{"connect", "capture", "traffic_filter"}
		if reference.Kind == domain.ConnectionEndpointNetworkObjectAccess {
			if value.ResourceKind != domain.NetworkBridge && value.ResourceKind != domain.NetworkNAT {
				return value, domain.Problem{Code: "endpoint_incompatible", Message: "network object does not expose a logical access endpoint", ResourceType: "network_object", ResourceID: value.ResourceID, Phase: "connection_validation"}
			}
			value.DisplayName = objectName + ":逻辑接入口"
			value.Availability = domain.ConnectionEndpointFree
			return value, nil
		}
		value.PortName = strings.TrimSpace(reference.PortName)
		if !containsString(networkObjectPortNames(value.ResourceKind, configJSON), value.PortName) {
			return value, domain.Problem{Code: "endpoint_missing", Message: "network object port does not exist", ResourceType: "network_object", ResourceID: value.ResourceID, Phase: "connection_validation"}
		}
		value.DisplayName = objectName + ":" + value.PortName
		value.Availability, err = r.connectionEndpointAvailability(ctx, value.LaboratoryID, "network_object", value.ResourceID, value.PortName)
		return value, err
	default:
		return domain.ConnectionEndpoint{}, domain.Problem{Code: "endpoint_incompatible", Message: "unsupported connection endpoint kind", Phase: "connection_validation"}
	}
}

func (r *Repositories) connectionEndpointAvailability(ctx context.Context, laboratoryID domain.ID, ownerType string, ownerID domain.ID, portName string) (domain.ConnectionEndpointAvailability, error) {
	var count int
	if err := r.database.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM topology_endpoint_reservations WHERE laboratory_id=? AND owner_type=? AND owner_id=? AND port_name=?`, laboratoryID, ownerType, ownerID, portName).Scan(&count); err != nil {
		return "", err
	}
	if count > 0 {
		return domain.ConnectionEndpointOccupied, nil
	}
	return domain.ConnectionEndpointFree, nil
}

func (r *Repositories) ListConnectionEndpoints(ctx context.Context, laboratoryID domain.ID) ([]domain.ConnectionEndpoint, error) {
	rows, err := r.database.DB.QueryContext(ctx, `SELECT i.id FROM interfaces i JOIN nodes n ON n.id=i.node_id WHERE n.laboratory_id=? ORDER BY n.id,i.slot`, laboratoryID)
	if err != nil {
		return nil, err
	}
	var interfaceIDs []domain.ID
	for rows.Next() {
		var id domain.ID
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		interfaceIDs = append(interfaceIDs, id)
	}
	if err = rows.Close(); err != nil {
		return nil, err
	}
	objectRows, err := r.database.DB.QueryContext(ctx, `SELECT id,kind,config_json FROM network_objects WHERE laboratory_id=? ORDER BY id`, laboratoryID)
	if err != nil {
		return nil, err
	}
	type objectPorts struct {
		id    domain.ID
		kind  string
		ports []string
	}
	var objects []objectPorts
	for objectRows.Next() {
		var object objectPorts
		var configJSON []byte
		if err = objectRows.Scan(&object.id, &object.kind, &configJSON); err != nil {
			objectRows.Close()
			return nil, err
		}
		object.ports = networkObjectPortNames(object.kind, configJSON)
		objects = append(objects, object)
	}
	if err = objectRows.Close(); err != nil {
		return nil, err
	}
	values := make([]domain.ConnectionEndpoint, 0, len(interfaceIDs)+len(objects)*2)
	for _, id := range interfaceIDs {
		value, resolveErr := r.ResolveConnectionEndpoint(ctx, domain.ConnectionEndpoint{Kind: domain.ConnectionEndpointNodeInterface, LaboratoryID: laboratoryID, PortID: id})
		if resolveErr != nil {
			return nil, resolveErr
		}
		values = append(values, value)
	}
	for _, object := range objects {
		if object.kind == domain.NetworkBridge || object.kind == domain.NetworkNAT {
			value, resolveErr := r.ResolveConnectionEndpoint(ctx, domain.ConnectionEndpoint{Kind: domain.ConnectionEndpointNetworkObjectAccess, LaboratoryID: laboratoryID, ResourceID: object.id})
			if resolveErr != nil {
				return nil, resolveErr
			}
			values = append(values, value)
			continue
		}
		for _, portName := range object.ports {
			value, resolveErr := r.ResolveConnectionEndpoint(ctx, domain.ConnectionEndpoint{Kind: domain.ConnectionEndpointNetworkObjectPort, LaboratoryID: laboratoryID, ResourceID: object.id, PortName: portName})
			if resolveErr != nil {
				return nil, resolveErr
			}
			values = append(values, value)
		}
	}
	return values, nil
}

func (r *Repositories) ListTopologyConnections(ctx context.Context, laboratoryID domain.ID) ([]domain.TopologyConnection, error) {
	topology := NewTopologyRepository(r.database)
	links, err := topology.ListLinks(ctx, laboratoryID)
	if err != nil {
		return nil, err
	}
	attachments, err := r.ListNetworkAttachments(ctx, laboratoryID)
	if err != nil {
		return nil, err
	}
	objectLinks, err := r.ListNetworkObjectLinks(ctx, laboratoryID)
	if err != nil {
		return nil, err
	}
	values := make([]domain.TopologyConnection, 0, len(links)+len(attachments)+len(objectLinks))
	for _, link := range links {
		source, resolveErr := r.ResolveConnectionEndpoint(ctx, domain.ConnectionEndpoint{Kind: domain.ConnectionEndpointNodeInterface, LaboratoryID: laboratoryID, PortID: link.EndpointAID})
		if resolveErr != nil {
			return nil, resolveErr
		}
		target, resolveErr := r.ResolveConnectionEndpoint(ctx, domain.ConnectionEndpoint{Kind: domain.ConnectionEndpointNodeInterface, LaboratoryID: laboratoryID, PortID: link.EndpointBID})
		if resolveErr != nil {
			return nil, resolveErr
		}
		values = append(values, topologyConnectionFromLink(link, source, target))
	}
	for _, attachment := range attachments {
		source, resolveErr := r.ResolveConnectionEndpoint(ctx, domain.ConnectionEndpoint{Kind: domain.ConnectionEndpointNodeInterface, LaboratoryID: laboratoryID, PortID: attachment.InterfaceID})
		if resolveErr != nil {
			return nil, resolveErr
		}
		objectKind := domain.ConnectionEndpointNetworkObjectPort
		if attachment.PortName == "" {
			objectKind = domain.ConnectionEndpointNetworkObjectAccess
		}
		target, resolveErr := r.ResolveConnectionEndpoint(ctx, domain.ConnectionEndpoint{Kind: objectKind, LaboratoryID: laboratoryID, ResourceID: attachment.NetworkObjectID, PortName: attachment.PortName})
		if resolveErr != nil {
			return nil, resolveErr
		}
		values = append(values, topologyConnectionFromAttachment(laboratoryID, attachment, source, target))
	}
	for _, link := range objectLinks {
		source, resolveErr := r.ResolveConnectionEndpoint(ctx, domain.ConnectionEndpoint{Kind: domain.ConnectionEndpointNetworkObjectPort, LaboratoryID: laboratoryID, ResourceID: link.ObjectAID, PortName: link.PortAName})
		if resolveErr != nil {
			return nil, resolveErr
		}
		target, resolveErr := r.ResolveConnectionEndpoint(ctx, domain.ConnectionEndpoint{Kind: domain.ConnectionEndpointNetworkObjectPort, LaboratoryID: laboratoryID, ResourceID: link.ObjectBID, PortName: link.PortBName})
		if resolveErr != nil {
			return nil, resolveErr
		}
		values = append(values, topologyConnectionFromObjectLink(link, source, target))
	}
	sort.Slice(values, func(i, j int) bool { return values[i].ID < values[j].ID })
	return values, nil
}

func (r *Repositories) GetTopologyConnection(ctx context.Context, id domain.ID) (domain.TopologyConnection, error) {
	topology := NewTopologyRepository(r.database)
	if link, err := topology.GetLink(ctx, id); err == nil {
		source, sourceErr := r.ResolveConnectionEndpoint(ctx, domain.ConnectionEndpoint{Kind: domain.ConnectionEndpointNodeInterface, LaboratoryID: link.LaboratoryID, PortID: link.EndpointAID})
		if sourceErr != nil {
			return domain.TopologyConnection{}, sourceErr
		}
		target, targetErr := r.ResolveConnectionEndpoint(ctx, domain.ConnectionEndpoint{Kind: domain.ConnectionEndpointNodeInterface, LaboratoryID: link.LaboratoryID, PortID: link.EndpointBID})
		if targetErr != nil {
			return domain.TopologyConnection{}, targetErr
		}
		return topologyConnectionFromLink(link, source, target), nil
	}
	if attachment, err := r.GetNetworkAttachment(ctx, id); err == nil {
		var laboratoryID domain.ID
		if err = r.database.DB.QueryRowContext(ctx, `SELECT laboratory_id FROM network_objects WHERE id=?`, attachment.NetworkObjectID).Scan(&laboratoryID); err != nil {
			return domain.TopologyConnection{}, err
		}
		source, sourceErr := r.ResolveConnectionEndpoint(ctx, domain.ConnectionEndpoint{Kind: domain.ConnectionEndpointNodeInterface, LaboratoryID: laboratoryID, PortID: attachment.InterfaceID})
		if sourceErr != nil {
			return domain.TopologyConnection{}, sourceErr
		}
		objectKind := domain.ConnectionEndpointNetworkObjectPort
		if attachment.PortName == "" {
			objectKind = domain.ConnectionEndpointNetworkObjectAccess
		}
		target, targetErr := r.ResolveConnectionEndpoint(ctx, domain.ConnectionEndpoint{Kind: objectKind, LaboratoryID: laboratoryID, ResourceID: attachment.NetworkObjectID, PortName: attachment.PortName})
		if targetErr != nil {
			return domain.TopologyConnection{}, targetErr
		}
		return topologyConnectionFromAttachment(laboratoryID, attachment, source, target), nil
	}
	if link, err := r.GetNetworkObjectLink(ctx, id); err == nil {
		source, sourceErr := r.ResolveConnectionEndpoint(ctx, domain.ConnectionEndpoint{Kind: domain.ConnectionEndpointNetworkObjectPort, LaboratoryID: link.LaboratoryID, ResourceID: link.ObjectAID, PortName: link.PortAName})
		if sourceErr != nil {
			return domain.TopologyConnection{}, sourceErr
		}
		target, targetErr := r.ResolveConnectionEndpoint(ctx, domain.ConnectionEndpoint{Kind: domain.ConnectionEndpointNetworkObjectPort, LaboratoryID: link.LaboratoryID, ResourceID: link.ObjectBID, PortName: link.PortBName})
		if targetErr != nil {
			return domain.TopologyConnection{}, targetErr
		}
		return topologyConnectionFromObjectLink(link, source, target), nil
	}
	return domain.TopologyConnection{}, ErrNotFound
}

func networkObjectPortNames(kind string, configJSON []byte) []string {
	var config map[string]any
	_ = json.Unmarshal(configJSON, &config)
	key := ""
	switch kind {
	case domain.NetworkPC, domain.NetworkSwitchL3:
		key = "interfaces"
	case domain.NetworkSwitchL2:
		key = "ports"
	default:
		return nil
	}
	rows, _ := config[key].([]any)
	values := make([]string, 0, len(rows))
	for _, row := range rows {
		entry, _ := row.(map[string]any)
		name := strings.TrimSpace(anyText(entry["name"]))
		if name != "" {
			values = append(values, name)
		}
	}
	return values
}

func anyText(value any) string {
	text, _ := value.(string)
	return text
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func topologyConnectionCapabilities() []string {
	return []string{"select", "delete", "capture", "wireshark", "traffic_filter"}
}

func topologyConnectionFromLink(value domain.Link, source, target domain.ConnectionEndpoint) domain.TopologyConnection {
	return domain.TopologyConnection{ID: value.ID, LaboratoryID: value.LaboratoryID, Source: source, Target: target, BackingKind: domain.ConnectionBackingLink, BackingID: value.ID, Revision: value.Revision, DesiredState: value.DesiredState, ObservedState: value.ObservedState, Capabilities: topologyConnectionCapabilities()}
}

func topologyConnectionFromAttachment(laboratoryID domain.ID, value domain.NetworkAttachment, source, target domain.ConnectionEndpoint) domain.TopologyConnection {
	config := domain.TopologyConnectionConfig{}
	if raw, ok := value.Config["pvid"].(float64); ok {
		config.PVID = int(raw)
	}
	if tagged, ok := value.Config["tagged"].([]any); ok {
		for _, item := range tagged {
			if raw, ok := item.(float64); ok {
				config.TaggedVLANs = append(config.TaggedVLANs, int(raw))
			}
		}
	}
	return domain.TopologyConnection{ID: value.ID, LaboratoryID: laboratoryID, Source: source, Target: target, BackingKind: domain.ConnectionBackingAttachment, BackingID: value.ID, Config: config, Revision: value.Revision, DesiredState: "connected", ObservedState: value.ObservedState, Capabilities: topologyConnectionCapabilities(), LastError: value.LastError}
}

func topologyConnectionFromObjectLink(value domain.NetworkObjectLink, source, target domain.ConnectionEndpoint) domain.TopologyConnection {
	return domain.TopologyConnection{ID: value.ID, LaboratoryID: value.LaboratoryID, Source: source, Target: target, BackingKind: domain.ConnectionBackingObjectLink, BackingID: value.ID, Revision: value.Revision, DesiredState: value.DesiredState, ObservedState: value.ObservedState, Capabilities: topologyConnectionCapabilities(), LastError: value.LastError}
}

func reserveTopologyEndpointTx(ctx context.Context, tx *sql.Tx, laboratoryID domain.ID, ownerType string, ownerID domain.ID, portName string, resourceType string, resourceID, operationID domain.ID) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO topology_endpoint_reservations(laboratory_id,owner_type,owner_id,port_name,resource_type,resource_id,operation_id,state,created_at) VALUES(?,?,?,?,?,?,?,'occupied',?)`, laboratoryID, ownerType, ownerID, portName, resourceType, resourceID, operationID, time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return domain.Problem{Code: "port_in_use", Message: "topology endpoint is already occupied", ResourceType: resourceType, ResourceID: resourceID, Phase: "endpoint_reservation", Cleanup: "no endpoint reservation was committed", OperatorHint: "refresh the topology and choose another free endpoint"}
	}
	return nil
}

func releaseTopologyConnectionReservationsTx(ctx context.Context, tx *sql.Tx, resourceType string, resourceID domain.ID) error {
	_, err := tx.ExecContext(ctx, `DELETE FROM topology_endpoint_reservations WHERE resource_type=? AND resource_id=?`, resourceType, resourceID)
	return err
}

func (r *Repositories) CreateTopologyNetworkAttachment(ctx context.Context, objectID, interfaceID domain.ID, portName string, config map[string]any, operationID domain.ID) (domain.NetworkAttachment, error) {
	return r.CreateTopologyNetworkAttachmentAs(ctx, domain.NewID(), objectID, interfaceID, portName, config, operationID)
}

func (r *Repositories) CreateTopologyNetworkAttachmentAs(ctx context.Context, id, objectID, interfaceID domain.ID, portName string, config map[string]any, operationID domain.ID) (domain.NetworkAttachment, error) {
	value := domain.NetworkAttachment{ID: id, NetworkObjectID: objectID, InterfaceID: interfaceID, PortName: portName, Config: config, Revision: 1, ObservedState: "pending"}
	body, _ := json.Marshal(config)
	err := r.database.Write(ctx, func(tx *sql.Tx) error {
		var objectLaboratoryID, interfaceLaboratoryID domain.ID
		var interfaceName string
		if err := tx.QueryRowContext(ctx, `SELECT laboratory_id FROM network_objects WHERE id=?`, objectID).Scan(&objectLaboratoryID); err != nil {
			if err == sql.ErrNoRows {
				return ErrNotFound
			}
			return err
		}
		if err := tx.QueryRowContext(ctx, `SELECT n.laboratory_id,i.name FROM interfaces i JOIN nodes n ON n.id=i.node_id WHERE i.id=?`, interfaceID).Scan(&interfaceLaboratoryID, &interfaceName); err != nil {
			if err == sql.ErrNoRows {
				return ErrNotFound
			}
			return err
		}
		if objectLaboratoryID != interfaceLaboratoryID {
			return domain.Problem{Code: "cross_laboratory_connection", Message: "attachment endpoints must belong to the same laboratory", ResourceType: "network_attachment", ResourceID: value.ID, Phase: "connection_validation"}
		}
		if err := reserveTopologyEndpointTx(ctx, tx, objectLaboratoryID, "node_interface", interfaceID, interfaceName, "network_attachment", value.ID, operationID); err != nil {
			return err
		}
		if portName != "" {
			if err := reserveTopologyEndpointTx(ctx, tx, objectLaboratoryID, "network_object", objectID, portName, "network_attachment", value.ID, operationID); err != nil {
				return err
			}
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO network_attachments(id,network_object_id,interface_id,port_name,config_json,observed_state) VALUES(?,?,?,?,?,'pending')`, value.ID, objectID, nullable(string(interfaceID)), portName, body); err != nil {
			return err
		}
		return appendEvent(ctx, tx, "network_attachment.created", objectLaboratoryID, "network_attachment", value.ID, 1, operationID, eventData(value))
	})
	return value, err
}

func (r *Repositories) GetNetworkAttachment(ctx context.Context, id domain.ID) (domain.NetworkAttachment, error) {
	var value domain.NetworkAttachment
	var config, problem []byte
	err := r.database.DB.QueryRowContext(ctx, `SELECT id,network_object_id,COALESCE(interface_id,''),port_name,config_json,revision,observed_state,COALESCE(last_error_json,'') FROM network_attachments WHERE id=?`, id).Scan(&value.ID, &value.NetworkObjectID, &value.InterfaceID, &value.PortName, &config, &value.Revision, &value.ObservedState, &problem)
	if err == sql.ErrNoRows {
		return value, ErrNotFound
	}
	if err != nil {
		return value, err
	}
	_ = json.Unmarshal(config, &value.Config)
	if len(problem) > 0 {
		value.LastError = &domain.Problem{}
		_ = json.Unmarshal(problem, value.LastError)
	}
	return value, nil
}

func (r *Repositories) DeleteTopologyNetworkAttachment(ctx context.Context, id domain.ID, expected domain.Revision, operationID domain.ID) error {
	return r.database.Write(ctx, func(tx *sql.Tx) error {
		var laboratoryID domain.ID
		var revision domain.Revision
		if err := tx.QueryRowContext(ctx, `SELECT o.laboratory_id,a.revision FROM network_attachments a JOIN network_objects o ON o.id=a.network_object_id WHERE a.id=?`, id).Scan(&laboratoryID, &revision); err != nil {
			if err == sql.ErrNoRows {
				return ErrNotFound
			}
			return err
		}
		if revision != expected {
			return domain.Problem{Code: "revision_conflict", Message: "network attachment revision mismatch", ResourceType: "network_attachment", ResourceID: id, Phase: "delete_admission"}
		}
		if err := releaseTopologyConnectionReservationsTx(ctx, tx, "network_attachment", id); err != nil {
			return err
		}
		result, err := tx.ExecContext(ctx, `DELETE FROM network_attachments WHERE id=? AND revision=?`, id, expected)
		if err != nil {
			return err
		}
		if affected, _ := result.RowsAffected(); affected != 1 {
			return domain.Problem{Code: "revision_conflict", Message: "network attachment revision changed during deletion", ResourceType: "network_attachment", ResourceID: id, Phase: "delete_commit"}
		}
		return appendEvent(ctx, tx, "network_attachment.deleted", laboratoryID, "network_attachment", id, expected.Next(), operationID, nil)
	})
}
