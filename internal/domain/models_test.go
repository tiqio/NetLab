package domain

import (
	"encoding/json"
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
