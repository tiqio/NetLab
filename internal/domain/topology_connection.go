package domain

import "strings"

type ConnectionEndpointKind string

const (
	ConnectionEndpointNodeInterface       ConnectionEndpointKind = "node_interface"
	ConnectionEndpointNetworkObjectPort   ConnectionEndpointKind = "network_object_port"
	ConnectionEndpointNetworkObjectAccess ConnectionEndpointKind = "network_object_access"
)

type ConnectionEndpointAvailability string

const (
	ConnectionEndpointFree         ConnectionEndpointAvailability = "free"
	ConnectionEndpointReserved     ConnectionEndpointAvailability = "reserved"
	ConnectionEndpointOccupied     ConnectionEndpointAvailability = "occupied"
	ConnectionEndpointReconciling  ConnectionEndpointAvailability = "reconciling"
	ConnectionEndpointUnavailable  ConnectionEndpointAvailability = "unavailable"
	ConnectionEndpointIncompatible ConnectionEndpointAvailability = "incompatible"
)

type ConnectionBackingKind string

const (
	ConnectionBackingLink       ConnectionBackingKind = "link"
	ConnectionBackingAttachment ConnectionBackingKind = "network_attachment"
	ConnectionBackingObjectLink ConnectionBackingKind = "network_object_link"
)

type ConnectionEndpoint struct {
	Kind              ConnectionEndpointKind         `json:"kind"`
	LaboratoryID      ID                             `json:"laboratory_id"`
	ResourceID        ID                             `json:"resource_id"`
	ResourceKind      string                         `json:"resource_kind,omitempty"`
	PortID            ID                             `json:"port_id,omitempty"`
	PortName          string                         `json:"port_name,omitempty"`
	DisplayName       string                         `json:"display_name,omitempty"`
	Capabilities      []string                       `json:"capabilities,omitempty"`
	Availability      ConnectionEndpointAvailability `json:"availability,omitempty"`
	UnavailableReason string                         `json:"unavailable_reason,omitempty"`
}

type TopologyConnectionConfig struct {
	PVID        int   `json:"pvid,omitempty"`
	TaggedVLANs []int `json:"tagged_vlans,omitempty"`
}

type TopologyConnection struct {
	ID            ID                       `json:"id"`
	LaboratoryID  ID                       `json:"laboratory_id"`
	Source        ConnectionEndpoint       `json:"source"`
	Target        ConnectionEndpoint       `json:"target"`
	BackingKind   ConnectionBackingKind    `json:"backing_kind"`
	BackingID     ID                       `json:"backing_id"`
	Config        TopologyConnectionConfig `json:"config,omitempty"`
	Revision      Revision                 `json:"revision"`
	DesiredState  string                   `json:"desired_state"`
	ObservedState string                   `json:"observed_state"`
	Capabilities  []string                 `json:"capabilities,omitempty"`
	LastError     *Problem                 `json:"last_error,omitempty"`
}

type TopologyConnectionRecoveryOutcome struct {
	ResourceType string `json:"resource_type"`
	ResourceID   ID     `json:"resource_id"`
	Action       string `json:"action"`
}

func (e ConnectionEndpoint) Key() string {
	switch e.Kind {
	case ConnectionEndpointNodeInterface:
		return string(e.Kind) + ":" + string(e.PortID)
	case ConnectionEndpointNetworkObjectPort:
		return string(e.Kind) + ":" + string(e.ResourceID) + ":" + strings.TrimSpace(e.PortName)
	case ConnectionEndpointNetworkObjectAccess:
		return string(e.Kind) + ":" + string(e.ResourceID)
	default:
		return ""
	}
}

func ValidateConnectionEndpoint(endpoint ConnectionEndpoint) error {
	if endpoint.LaboratoryID == "" || endpoint.ResourceID == "" {
		return Problem{Code: "endpoint_missing", Message: "connection endpoint resource is required", Phase: "connection_validation"}
	}
	switch endpoint.Kind {
	case ConnectionEndpointNodeInterface:
		if endpoint.PortID == "" {
			return Problem{Code: "endpoint_missing", Message: "node interface endpoint requires port_id", ResourceType: "node", ResourceID: endpoint.ResourceID, Phase: "connection_validation"}
		}
	case ConnectionEndpointNetworkObjectPort:
		if strings.TrimSpace(endpoint.PortName) == "" {
			return Problem{Code: "endpoint_missing", Message: "network object endpoint requires port_name", ResourceType: "network_object", ResourceID: endpoint.ResourceID, Phase: "connection_validation"}
		}
		if err := ValidateNetworkObjectPortName(strings.TrimSpace(endpoint.PortName)); err != nil {
			return err
		}
	case ConnectionEndpointNetworkObjectAccess:
	default:
		return Problem{Code: "endpoint_incompatible", Message: "unsupported connection endpoint kind", Phase: "connection_validation"}
	}
	return nil
}

func ResolveTopologyConnectionBacking(source, target ConnectionEndpoint) (ConnectionBackingKind, error) {
	if err := ValidateConnectionEndpoint(source); err != nil {
		return "", err
	}
	if err := ValidateConnectionEndpoint(target); err != nil {
		return "", err
	}
	if source.LaboratoryID != target.LaboratoryID {
		return "", Problem{Code: "cross_laboratory_connection", Message: "connection endpoints must belong to the same laboratory", Phase: "connection_validation"}
	}
	if source.Key() == target.Key() || source.ResourceID == target.ResourceID {
		return "", Problem{Code: "invalid_topology", Message: "connection endpoints must belong to different resources", Phase: "connection_validation"}
	}
	sourceNode := source.Kind == ConnectionEndpointNodeInterface
	targetNode := target.Kind == ConnectionEndpointNodeInterface
	sourcePort := source.Kind == ConnectionEndpointNetworkObjectPort
	targetPort := target.Kind == ConnectionEndpointNetworkObjectPort
	sourceAccess := source.Kind == ConnectionEndpointNetworkObjectAccess
	targetAccess := target.Kind == ConnectionEndpointNetworkObjectAccess
	switch {
	case sourceNode && targetNode:
		return ConnectionBackingLink, nil
	case (sourceNode && (targetPort || targetAccess)) || (targetNode && (sourcePort || sourceAccess)):
		return ConnectionBackingAttachment, nil
	case sourcePort && targetPort:
		return ConnectionBackingObjectLink, nil
	default:
		return "", Problem{Code: "endpoint_incompatible", Message: "connection endpoint combination is not supported", Phase: "connection_validation"}
	}
}
