package query

import (
	"context"
	"encoding/json"

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

type DeviceReadinessService struct {
	nodes        DeviceReadinessNodeReader
	topology     DeviceReadinessTopologyReader
	capabilities DeviceReadinessCapabilityReader
}

func NewDeviceReadinessService(nodes DeviceReadinessNodeReader, topology DeviceReadinessTopologyReader, capabilities DeviceReadinessCapabilityReader) *DeviceReadinessService {
	return &DeviceReadinessService{nodes: nodes, topology: topology, capabilities: capabilities}
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
	for _, role := range roles {
		if role.Role == "management" {
			managementDeclared = true
			if role.Address == "" {
				result.Management = domain.DeviceReadinessLevel{State: "prerequisite", Details: []string{"management address requires guest configuration"}}
			} else {
				result.Management = domain.DeviceReadinessLevel{State: "unverified", Details: []string{"declared address has no successful probe evidence"}}
			}
		} else {
			dataDeclared = true
		}
	}
	if managementDeclared && result.Management.State == "not_declared" {
		result.Management.State = "unverified"
	}
	if dataDeclared {
		result.DataPath = domain.DeviceReadinessLevel{State: "unverified", Details: []string{"role declared without successful exchange evidence"}}
	}
	return result, nil
}
