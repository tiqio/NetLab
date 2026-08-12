package query

import (
	"context"
	"encoding/json"
	"fmt"
	"net/netip"

	"github.com/netlab/netlab/internal/domain"
)

type DeviceReadinessNodeReader interface {
	GetNode(context.Context, domain.ID) (domain.Node, error)
}

type DeviceReadinessTopologyReader interface {
	Snapshot(context.Context, domain.ID) (domain.TopologySnapshot, error)
}

type DeviceReadinessCapabilityReader interface {
	ListRuntimeCapabilities(context.Context, domain.ID) ([]domain.RuntimeCapabilityObservation, error)
}

type DeviceReadinessEvidenceReader interface {
	ListTrafficFilterObservations(context.Context, domain.ID) ([]domain.TrafficFilter, error)
}

type DeviceReadinessService struct {
	nodes        DeviceReadinessNodeReader
	topology     DeviceReadinessTopologyReader
	capabilities DeviceReadinessCapabilityReader
	evidence     DeviceReadinessEvidenceReader
}

func NewDeviceReadinessService(nodes DeviceReadinessNodeReader, topology DeviceReadinessTopologyReader, capabilities DeviceReadinessCapabilityReader, evidence ...DeviceReadinessEvidenceReader) *DeviceReadinessService {
	service := &DeviceReadinessService{nodes: nodes, topology: topology, capabilities: capabilities}
	if len(evidence) > 0 {
		service.evidence = evidence[0]
	}
	return service
}

func (s *DeviceReadinessService) Get(ctx context.Context, nodeID domain.ID) (domain.DeviceReadiness, error) {
	node, err := s.nodes.GetNode(ctx, nodeID)
	if err != nil {
		return domain.DeviceReadiness{}, err
	}
	roles := []domain.DeviceInterfaceRole{}
	if raw, ok := node.Config["device_roles"]; ok {
		body, _ := json.Marshal(raw)
		_ = json.Unmarshal(body, &roles)
	}
	result := domain.DeviceReadiness{NodeID: node.ID, Roles: roles, Cable: domain.DeviceReadinessLevel{State: "unavailable"}, Guest: domain.DeviceReadinessLevel{State: "unavailable"}, Management: domain.DeviceReadinessLevel{State: "not_declared"}, DataPath: domain.DeviceReadinessLevel{State: "not_declared"}}
	snapshot, snapshotErr := s.topology.Snapshot(ctx, node.LaboratoryID)
	if snapshotErr == nil {
		connected := 0
		for _, iface := range snapshot.Interfaces {
			if iface.NodeID != node.ID {
				continue
			}
			for _, link := range snapshot.Links {
				if (link.EndpointAID == iface.ID || link.EndpointBID == iface.ID) && link.ObservedState == "connected" {
					connected++
				}
			}
			for _, attachment := range snapshot.Attachments {
				if attachment.InterfaceID == iface.ID && attachment.ObservedState == "active" {
					connected++
				}
			}
		}
		if connected > 0 {
			result.Cable = domain.DeviceReadinessLevel{State: "ready", Details: []string{"NetLab-owned connected attachments observed"}}
		}
	}
	capabilities, _ := s.capabilities.ListRuntimeCapabilities(ctx, node.ID)
	for _, observation := range capabilities {
		if observation.State == domain.CapabilityReady && (observation.Capability == domain.CapabilityQGA || observation.Capability == domain.CapabilitySerial) {
			result.Guest = domain.DeviceReadinessLevel{State: "ready", Details: []string{"authorized guest control channel available"}}
			break
		}
	}
	managementDeclared, dataDeclared := false, false
	managementAddresses := map[string]bool{}
	dataInterfaces := map[domain.ID]bool{}
	for _, role := range roles {
		if role.Role == "management" {
			managementDeclared = true
			if role.Address == "" {
				result.Management = domain.DeviceReadinessLevel{State: "prerequisite", Details: []string{"management address requires guest configuration"}}
			} else {
				result.Management = domain.DeviceReadinessLevel{State: "unverified", Details: []string{"declared address has no successful probe evidence"}}
				if prefix, parseErr := netip.ParsePrefix(role.Address); parseErr == nil {
					managementAddresses[prefix.Addr().String()] = true
				}
			}
		} else {
			dataDeclared = true
			dataInterfaces[role.InterfaceID] = true
		}
	}
	if managementDeclared && result.Management.State == "not_declared" {
		result.Management.State = "unverified"
	}
	if dataDeclared {
		result.DataPath = domain.DeviceReadinessLevel{State: "unverified", Details: []string{"role declared without successful exchange evidence"}}
	}
	if s.evidence != nil {
		filters, evidenceErr := s.evidence.ListTrafficFilterObservations(ctx, node.LaboratoryID)
		if evidenceErr == nil {
			applyDeviceTrafficEvidence(&result, snapshot, node.ID, managementAddresses, dataInterfaces, filters)
		}
	}
	return result, nil
}

func applyDeviceTrafficEvidence(result *domain.DeviceReadiness, snapshot domain.TopologySnapshot, nodeID domain.ID, managementAddresses map[string]bool, dataInterfaces map[domain.ID]bool, filters []domain.TrafficFilter) {
	managementDirections := map[string]bool{}
	dataDirections := map[string]bool{}
	dataLinks := map[domain.ID]bool{}
	for _, link := range snapshot.Links {
		for _, iface := range snapshot.Interfaces {
			if iface.NodeID == nodeID && dataInterfaces[iface.ID] && (link.EndpointAID == iface.ID || link.EndpointBID == iface.ID) {
				dataLinks[link.ID] = true
			}
		}
	}
	var managementPackets, dataPackets int64
	for _, filter := range filters {
		for _, observation := range filter.Observations {
			if managementAddresses[observation.SourceAddress] {
				managementDirections["from_device"] = true
				managementPackets += observation.Count
			}
			if managementAddresses[observation.DestinationAddress] {
				managementDirections["to_device"] = true
				managementPackets += observation.Count
			}
			if dataInterfaces[observation.InterfaceID] || dataLinks[observation.LinkID] {
				direction := normalizedEvidenceDirection(observation.Direction)
				if direction != "" {
					dataDirections[direction] = true
					dataPackets += observation.Count
				}
			}
		}
	}
	if managementDirections["from_device"] && managementDirections["to_device"] {
		result.Management = domain.DeviceReadinessLevel{State: "ready", Details: []string{fmt.Sprintf("bidirectional management traffic observed (%d packets)", managementPackets)}}
	}
	if dataDirections["forward"] && dataDirections["reverse"] {
		result.DataPath = domain.DeviceReadinessLevel{State: "ready", Details: []string{fmt.Sprintf("bidirectional data traffic observed (%d packets)", dataPackets)}}
	}
}

func normalizedEvidenceDirection(value string) string {
	switch value {
	case "egress", "a_to_b":
		return "forward"
	case "ingress", "b_to_a":
		return "reverse"
	default:
		return ""
	}
}
