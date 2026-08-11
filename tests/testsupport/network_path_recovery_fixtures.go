package testsupport

import (
	"fmt"
	"net/url"

	"github.com/netlab/netlab/internal/domain"
)

const (
	ComponentMatrixLaboratoryID domain.ID = "019feee704ee-c5b2bd57159bebe86bed"
	KnownStuckObjectLinkID      domain.ID = "019fef7b284d-280a689d6233f257dd08"
	ComponentMatrixL3ObjectID   domain.ID = "019feee71261-536b824c99d9083627f8"
)

type NetworkPathRecoveryFixture struct {
	LaboratoryID     domain.ID
	L3ObjectID       domain.ID
	StuckObjectLink  domain.ID
	ServiceIPv4Peers []string
	ServiceIPv6Peers []string
	TransitIPv4Peers []string
	TransitIPv6Peers []string
}

func ComponentMatrixNetworkPaths() NetworkPathRecoveryFixture {
	return NetworkPathRecoveryFixture{
		LaboratoryID:     ComponentMatrixLaboratoryID,
		L3ObjectID:       ComponentMatrixL3ObjectID,
		StuckObjectLink:  KnownStuckObjectLinkID,
		ServiceIPv4Peers: []string{"10.40.40.1", "10.40.40.11", "10.40.40.20"},
		ServiceIPv6Peers: []string{"fd40::1", "fd40::11", "fd40::20"},
		TransitIPv4Peers: []string{"172.16.0.1", "172.16.0.2"},
		TransitIPv6Peers: []string{"fd16::1", "fd16::2"},
	}
}

func TargetAPIURL(baseURL, path string) (string, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	reference, err := url.Parse(path)
	if err != nil {
		return "", err
	}
	if reference.IsAbs() {
		return "", fmt.Errorf("target API path must be relative")
	}
	return base.ResolveReference(reference).String(), nil
}
