package testsupport

import "github.com/netlab/netlab/internal/domain"

type NetworkObjectLinkPathFixture struct {
	Laboratory domain.Laboratory
	Objects    []domain.NetworkObject
	Links      []domain.NetworkObjectLink
}

func ThreeObjectLinkPath() NetworkObjectLinkPathFixture {
	laboratory := Laboratory("lab-object-links")
	objects := []domain.NetworkObject{
		{ID: "switch-a", LaboratoryID: laboratory.ID, Name: "Switch A", Kind: domain.NetworkSwitchL2, Revision: 1, DesiredState: "running", ObservedState: "running"},
		{ID: "switch-b", LaboratoryID: laboratory.ID, Name: "Switch B", Kind: domain.NetworkSwitchL2, Revision: 1, DesiredState: "running", ObservedState: "running"},
		{ID: "switch-c", LaboratoryID: laboratory.ID, Name: "Switch C", Kind: domain.NetworkSwitchL2, Revision: 1, DesiredState: "running", ObservedState: "running"},
	}
	links := []domain.NetworkObjectLink{
		{ID: "object-link-ab", LaboratoryID: laboratory.ID, ObjectAID: objects[0].ID, PortAName: "swp1", ObjectBID: objects[1].ID, PortBName: "swp1", Revision: 1, DesiredState: "connected", ObservedState: "connected"},
		{ID: "object-link-bc", LaboratoryID: laboratory.ID, ObjectAID: objects[1].ID, PortAName: "swp2", ObjectBID: objects[2].ID, PortBName: "swp1", Revision: 1, DesiredState: "connected", ObservedState: "connected"},
	}
	return NetworkObjectLinkPathFixture{Laboratory: laboratory, Objects: objects, Links: links}
}

func ParallelObjectLink(path NetworkObjectLinkPathFixture) domain.NetworkObjectLink {
	return domain.NetworkObjectLink{ID: "object-link-ab-parallel", LaboratoryID: path.Laboratory.ID, ObjectAID: path.Objects[0].ID, PortAName: "swp2", ObjectBID: path.Objects[1].ID, PortBName: "swp3", Revision: 1, DesiredState: "connected", ObservedState: "connected"}
}
