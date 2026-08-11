package reconcile

import "github.com/netlab/netlab/internal/domain"

type EndpointBacking struct {
	Kind         domain.RuntimeBackingKind
	ResourceType string
	ResourceID   domain.ID
	PortName     string
}

func ClassifyNetworkObjectBacking(object domain.NetworkObject) (EndpointBacking, error) {
	backing := EndpointBacking{ResourceType: "network_object", ResourceID: object.ID}
	switch object.Kind {
	case domain.NetworkBridge, domain.NetworkNAT:
		backing.Kind = domain.RuntimeBackingHostBridge
	case domain.NetworkPC, domain.NetworkSwitchL2, domain.NetworkSwitchL3:
		backing.Kind = domain.RuntimeBackingNamespace
	default:
		return EndpointBacking{}, domain.Problem{Code: "endpoint_backing_unsupported", Message: "network object backing kind is unsupported", ResourceType: "network_object", ResourceID: object.ID, Phase: "endpoint_backing_classification", Cleanup: "runtime unchanged", OperatorHint: "use a supported network object kind"}
	}
	return backing, nil
}
