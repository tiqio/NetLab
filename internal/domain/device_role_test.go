package domain

import (
	"encoding/json"
	"testing"
)

func TestValidateDeviceInterfaceRoles(t *testing.T) {
	values := []DeviceInterfaceRole{{InterfaceID: "if-mgmt", Role: "management", AddressFamily: "ipv4", Address: "10.30.30.2/24", Gateway: "10.30.30.1"}, {InterfaceID: "if-trunk", Role: "trunk", AddressFamily: "dual"}, {InterfaceID: "if-client", Role: "client-facing"}}
	if err := ValidateDeviceInterfaceRoles(values); err != nil {
		t.Fatal(err)
	}
	for _, invalid := range [][]DeviceInterfaceRole{
		{{InterfaceID: "if-1", Role: "secret"}},
		{{InterfaceID: "if-1", Role: "lan"}, {InterfaceID: "if-1", Role: "wan"}},
		{{InterfaceID: "if-1", Role: "management", AddressFamily: "ipv4", Address: "fd30::2/64"}},
	} {
		if ValidateDeviceInterfaceRoles(invalid) == nil {
			t.Fatalf("invalid roles accepted: %+v", invalid)
		}
	}
}

func TestDeviceInterfaceRoleRejectsSecrets(t *testing.T) {
	var role DeviceInterfaceRole
	if json.Unmarshal([]byte(`{"interface_id":"if-1","role":"management","password":"nope"}`), &role) == nil {
		t.Fatal("secret-bearing role metadata was accepted")
	}
}
