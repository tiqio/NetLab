package query

import (
	"context"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/domain"
)

type deviceReadinessFixture struct {
	node         domain.Node
	snapshot     domain.TopologySnapshot
	capabilities []domain.RuntimeCapabilityObservation
	filters      []domain.TrafficFilter
}

func (f deviceReadinessFixture) GetNode(context.Context, domain.ID) (domain.Node, error) {
	return f.node, nil
}
func (f deviceReadinessFixture) Snapshot(context.Context, domain.ID) (domain.TopologySnapshot, error) {
	return f.snapshot, nil
}
func (f deviceReadinessFixture) ListRuntimeCapabilities(context.Context, domain.ID) ([]domain.RuntimeCapabilityObservation, error) {
	return f.capabilities, nil
}
func (f deviceReadinessFixture) ListTrafficFilterObservations(context.Context, domain.ID) ([]domain.TrafficFilter, error) {
	return f.filters, nil
}

func TestDeviceReadinessUsesSuccessfulTrafficEvidence(t *testing.T) {
	now := time.Now().UTC()
	fixture := deviceReadinessFixture{
		node: domain.Node{ID: "vendor", LaboratoryID: "lab", Config: map[string]any{"device_roles": []any{
			map[string]any{"interface_id": "mgmt", "role": "management", "address_family": "ipv4", "address": "10.30.30.2/24"},
			map[string]any{"interface_id": "wan", "role": "wan"},
		}}},
		snapshot: domain.TopologySnapshot{
			Interfaces:  []domain.Interface{{ID: "mgmt", NodeID: "vendor"}, {ID: "wan", NodeID: "vendor"}, {ID: "peer", NodeID: "peer"}},
			Attachments: []domain.NetworkAttachment{{InterfaceID: "mgmt", ObservedState: "active"}},
			Links:       []domain.Link{{ID: "vendor-link", EndpointAID: "wan", EndpointBID: "peer", ObservedState: "connected"}},
		},
		capabilities: []domain.RuntimeCapabilityObservation{{Capability: domain.CapabilitySerial, State: domain.CapabilityReady}},
		filters: []domain.TrafficFilter{{ID: "evidence", LaboratoryID: "lab", MatchedPackets: 4, Observations: []domain.TrafficObservation{
			{InterfaceID: "mgmt", SourceAddress: "10.30.30.2", DestinationAddress: "10.30.30.10", Direction: "egress", Count: 1, LastSeen: now},
			{InterfaceID: "mgmt", SourceAddress: "10.30.30.10", DestinationAddress: "10.30.30.2", Direction: "ingress", Count: 1, LastSeen: now},
			{LinkID: "vendor-link", Direction: "a_to_b", Count: 1, LastSeen: now},
			{LinkID: "vendor-link", Direction: "b_to_a", Count: 1, LastSeen: now},
		}}},
	}
	service := NewDeviceReadinessService(fixture, fixture, fixture, fixture)
	value, err := service.Get(context.Background(), "vendor")
	if err != nil {
		t.Fatal(err)
	}
	if value.Management.State != "ready" || value.DataPath.State != "ready" {
		t.Fatalf("readiness=%+v", value)
	}
}

func TestDeviceReadinessDoesNotTreatOneWayTrafficAsProvenDataPath(t *testing.T) {
	fixture := deviceReadinessFixture{
		node:     domain.Node{ID: "vendor", LaboratoryID: "lab", Config: map[string]any{"device_roles": []any{map[string]any{"interface_id": "wan", "role": "wan"}}}},
		snapshot: domain.TopologySnapshot{Interfaces: []domain.Interface{{ID: "wan", NodeID: "vendor"}, {ID: "peer", NodeID: "peer"}}, Links: []domain.Link{{ID: "vendor-link", EndpointAID: "wan", EndpointBID: "peer", ObservedState: "connected"}}},
		filters:  []domain.TrafficFilter{{MatchedPackets: 1, Observations: []domain.TrafficObservation{{LinkID: "vendor-link", Direction: "a_to_b", Count: 1}}}},
	}
	service := NewDeviceReadinessService(fixture, fixture, fixture, fixture)
	value, err := service.Get(context.Background(), "vendor")
	if err != nil {
		t.Fatal(err)
	}
	if value.DataPath.State != "unverified" {
		t.Fatalf("readiness=%+v", value)
	}
}
