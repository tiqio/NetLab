package testsupport

import (
	"fmt"

	"github.com/netlab/netlab/internal/domain"
)

func Laboratory(id domain.ID) domain.Laboratory {
	return domain.Laboratory{ID: id, Name: "Topology test", Revision: 1, RecoveryPolicy: domain.RecoveryAutoRestore, LifecycleState: "active"}
}

func Node(id, labID domain.ID) domain.Node {
	return domain.Node{ID: id, LaboratoryID: labID, Name: string(id), Kind: "qemu", Revision: 1, DesiredState: domain.DesiredStopped, ObservedState: domain.ObservedStopped}
}

func Interface(id, nodeID domain.ID, slot int) domain.Interface {
	return domain.Interface{ID: id, NodeID: nodeID, Slot: slot, Name: fmt.Sprintf("eth%d", slot), Driver: "virtio-net-pci", Revision: 1}
}

func Link(id, labID, endpointAID, endpointBID domain.ID) domain.Link {
	return domain.Link{ID: id, LaboratoryID: labID, EndpointAID: endpointAID, EndpointBID: endpointBID, Revision: 1, DesiredState: "connected", ObservedState: "connected"}
}
