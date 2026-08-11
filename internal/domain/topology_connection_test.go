package domain

import "testing"

func TestConnectionEndpointBackingKindDoesNotPermitNamespaceForHostBridge(t *testing.T) {
	if RuntimeBackingHostBridge == RuntimeBackingNamespace {
		t.Fatal("host bridge and namespace backing must remain distinct")
	}
}

func TestResolveTopologyConnectionBacking(t *testing.T) {
	tests := []struct {
		name   string
		source ConnectionEndpoint
		target ConnectionEndpoint
		want   ConnectionBackingKind
	}{
		{name: "node link", source: nodeEndpoint("lab", "node-a", "if-a"), target: nodeEndpoint("lab", "node-b", "if-b"), want: ConnectionBackingLink},
		{name: "attachment", source: nodeEndpoint("lab", "node-a", "if-a"), target: objectPortEndpoint("lab", "switch", "eth0"), want: ConnectionBackingAttachment},
		{name: "reversed attachment", source: objectPortEndpoint("lab", "switch", "eth0"), target: nodeEndpoint("lab", "node-a", "if-a"), want: ConnectionBackingAttachment},
		{name: "logical access", source: nodeEndpoint("lab", "node-a", "if-a"), target: ConnectionEndpoint{Kind: ConnectionEndpointNetworkObjectAccess, LaboratoryID: "lab", ResourceID: "nat"}, want: ConnectionBackingAttachment},
		{name: "object link", source: objectPortEndpoint("lab", "switch-a", "eth0"), target: objectPortEndpoint("lab", "switch-b", "eth1"), want: ConnectionBackingObjectLink},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ResolveTopologyConnectionBacking(test.source, test.target)
			if err != nil {
				t.Fatalf("resolve: %v", err)
			}
			if got != test.want {
				t.Fatalf("backing=%q want %q", got, test.want)
			}
		})
	}
}

func TestResolveTopologyConnectionBackingRejectsInvalidEndpoints(t *testing.T) {
	tests := []struct {
		name   string
		source ConnectionEndpoint
		target ConnectionEndpoint
		code   string
	}{
		{name: "same endpoint", source: nodeEndpoint("lab", "node", "if-a"), target: nodeEndpoint("lab", "node", "if-a"), code: "invalid_topology"},
		{name: "same resource", source: nodeEndpoint("lab", "node", "if-a"), target: ConnectionEndpoint{Kind: ConnectionEndpointNodeInterface, LaboratoryID: "lab", ResourceID: "node", PortID: "if-b", PortName: "eth1"}, code: "invalid_topology"},
		{name: "cross laboratory", source: nodeEndpoint("lab-a", "node-a", "if-a"), target: nodeEndpoint("lab-b", "node-b", "if-b"), code: "cross_laboratory_connection"},
		{name: "missing interface", source: ConnectionEndpoint{Kind: ConnectionEndpointNodeInterface, LaboratoryID: "lab", ResourceID: "node-a"}, target: nodeEndpoint("lab", "node-b", "if-b"), code: "endpoint_missing"},
		{name: "access to access", source: ConnectionEndpoint{Kind: ConnectionEndpointNetworkObjectAccess, LaboratoryID: "lab", ResourceID: "bridge-a"}, target: ConnectionEndpoint{Kind: ConnectionEndpointNetworkObjectAccess, LaboratoryID: "lab", ResourceID: "bridge-b"}, code: "endpoint_incompatible"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ResolveTopologyConnectionBacking(test.source, test.target)
			problem := NormalizeProblem(err, Problem{})
			if problem.Code != test.code {
				t.Fatalf("problem=%+v want code %q", problem, test.code)
			}
		})
	}
}

func nodeEndpoint(lab, resource, port ID) ConnectionEndpoint {
	return ConnectionEndpoint{Kind: ConnectionEndpointNodeInterface, LaboratoryID: lab, ResourceID: resource, PortID: port, PortName: "eth0"}
}

func objectPortEndpoint(lab, resource ID, port string) ConnectionEndpoint {
	return ConnectionEndpoint{Kind: ConnectionEndpointNetworkObjectPort, LaboratoryID: lab, ResourceID: resource, PortName: port}
}
