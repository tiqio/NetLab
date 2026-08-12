package integration

import (
	"github.com/netlab/netlab/internal/domain"
	"testing"
)

func TestVendorPathFixtureRequiresRolesWithoutSecrets(t *testing.T) {
	roles := []domain.DeviceInterfaceRole{{InterfaceID: "mgmt", Role: "management", AddressFamily: "ipv4", Address: "10.30.30.2/24"}, {InterfaceID: "lan", Role: "lan"}, {InterfaceID: "wan", Role: "wan"}, {InterfaceID: "client", Role: "client-facing"}}
	if err := domain.ValidateDeviceInterfaceRoles(roles); err != nil {
		t.Fatal(err)
	}
}
