package domain

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestValidateNodeNetworkInterfacesAcceptsDualStackRoutes(t *testing.T) {
	interfaces := []NodeNetworkInterfaceSettings{{
		Name:      "eth0",
		Addresses: []string{"10.10.1.2/24", "2001:db8:1::2/64"},
		Routes: []RouteConfig{
			{Destination: "10.30.0.0/24", Gateway: "10.10.1.1", Metric: 100},
			{Destination: "2001:db8:3::/64", Gateway: "2001:db8:1::1"},
		},
	}}
	if err := ValidateNodeNetworkInterfaces(interfaces); err != nil {
		t.Fatal(err)
	}
}

func TestValidateNodeNetworkInterfacesRejectsFamilyAndDuplicateRoutes(t *testing.T) {
	for _, interfaces := range [][]NodeNetworkInterfaceSettings{
		{{Name: "eth0", Addresses: []string{"10.10.1.2/24"}, Routes: []RouteConfig{{Destination: "10.30.0.0/24", Gateway: "2001:db8::1"}}}},
		{{Name: "eth0", Addresses: []string{"10.10.1.2/24"}, Routes: []RouteConfig{{Destination: "10.30.0.0/24", Gateway: "10.10.1.1"}, {Destination: "10.30.0.1/24", Gateway: "10.10.1.1"}}}},
	} {
		if err := ValidateNodeNetworkInterfaces(interfaces); err == nil {
			t.Fatal("expected route validation failure")
		}
	}
}

func TestValidateNodeNetworkInterfacesCanonicalizesRoutesAndAddresses(t *testing.T) {
	interfaces := []NodeNetworkInterfaceSettings{{
		Name:      "eth0",
		Addresses: []string{"2001:0db8:0001::2/64", "192.0.2.10/24"},
		Routes: []RouteConfig{
			{Destination: "198.51.100.99/24", Gateway: "192.0.2.1"},
			{Destination: "2001:0db8:0002::99/64", Gateway: "2001:0db8:0001::1"},
		},
	}}
	if err := ValidateNodeNetworkInterfaces(interfaces); err != nil {
		t.Fatal(err)
	}
	if interfaces[0].Addresses[0] != "2001:db8:1::2/64" || interfaces[0].Routes[0].Destination != "198.51.100.0/24" || interfaces[0].Routes[1].Destination != "2001:db8:2::/64" || interfaces[0].Routes[1].Gateway != "2001:db8:1::1" {
		t.Fatalf("interfaces=%+v", interfaces)
	}
}

func TestValidateNodeNetworkInterfacesUsesStableRouteErrorCodes(t *testing.T) {
	tests := []struct {
		name       string
		interfaces []NodeNetworkInterfaceSettings
		code       string
	}{
		{name: "invalid destination", interfaces: []NodeNetworkInterfaceSettings{{Name: "eth0", Routes: []RouteConfig{{Destination: "bad"}}}}, code: "invalid_route_destination"},
		{name: "family mismatch", interfaces: []NodeNetworkInterfaceSettings{{Name: "eth0", Addresses: []string{"192.0.2.10/24"}, Routes: []RouteConfig{{Destination: "0.0.0.0/0", Gateway: "2001:db8::1"}}}}, code: "route_family_mismatch"},
		{name: "unreachable gateway", interfaces: []NodeNetworkInterfaceSettings{{Name: "eth0", Addresses: []string{"192.0.2.10/24"}, Routes: []RouteConfig{{Destination: "0.0.0.0/0", Gateway: "198.51.100.1"}}}}, code: "route_gateway_unreachable"},
		{name: "negative metric", interfaces: []NodeNetworkInterfaceSettings{{Name: "eth0", Routes: []RouteConfig{{Destination: "198.51.100.0/24", Metric: -1}}}}, code: "invalid_route_metric"},
		{name: "conflicting prefix", interfaces: []NodeNetworkInterfaceSettings{{Name: "eth0", Routes: []RouteConfig{{Destination: "198.51.100.1/24"}, {Destination: "198.51.100.2/24"}}}}, code: "route_conflict"},
		{name: "connected prefix", interfaces: []NodeNetworkInterfaceSettings{{Name: "eth0", Addresses: []string{"198.51.100.2/24"}, Routes: []RouteConfig{{Destination: "198.51.100.0/24"}}}}, code: "route_conflict"},
		{name: "cross interface prefix", interfaces: []NodeNetworkInterfaceSettings{{Name: "eth0", Routes: []RouteConfig{{Destination: "198.51.100.1/24"}}}, {Name: "eth1", Routes: []RouteConfig{{Destination: "198.51.100.2/24"}}}}, code: "route_conflict"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateNodeNetworkInterfaces(test.interfaces)
			var configError NetworkConfigError
			if !errors.As(err, &configError) || configError.Code != test.code {
				t.Fatalf("err=%v code=%q", err, configError.Code)
			}
		})
	}
}

func TestInterfaceOperationalStateUsesContractProperty(t *testing.T) {
	body, err := json.Marshal(Interface{OperationalState: "up"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `"operational_state":"up"`) {
		t.Fatalf("unexpected interface payload: %s", body)
	}
	if strings.Contains(string(body), `"oper_state"`) {
		t.Fatalf("database column name leaked into API payload: %s", body)
	}
}
