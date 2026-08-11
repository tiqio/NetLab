package domain

import (
	"reflect"
	"testing"
)

func TestValidateSwitchConfigurations(t *testing.T) {
	if err := ValidateSwitchL2Config(SwitchL2Config{VLANFiltering: true, Ports: []VLANPort{{Name: "lan0", PVID: 10, Tagged: []int{20, 30}}}}); err != nil {
		t.Fatal(err)
	}
	for _, value := range []SwitchL2Config{
		{},
		{Ports: []VLANPort{{Name: "invalid port", PVID: 10}}},
		{Ports: []VLANPort{{Name: "lan0", PVID: 10, Tagged: []int{10}}}},
	} {
		if ValidateSwitchL2Config(value) == nil {
			t.Fatalf("expected invalid L2 config: %+v", value)
		}
	}
	if err := ValidateSwitchL3Config(SwitchL3Config{
		Interfaces:  []L3InterfaceConfig{{Name: "lan0", Addresses: []string{"192.0.2.1/24", "2001:db8::1/64"}}},
		Routes:      []RouteConfig{{Destination: "0.0.0.0/0", Gateway: "192.0.2.254", Metric: 20}},
		ForwardIPv4: true,
	}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateSwitchL3Config(SwitchL3Config{Interfaces: []L3InterfaceConfig{{Name: "lan0"}}}); err != nil {
		t.Fatalf("unaddressed L3 interface should remain configurable and connectable: %v", err)
	}
	if ValidateSwitchL3Config(SwitchL3Config{Interfaces: []L3InterfaceConfig{{Name: "lan0", Addresses: []string{"not-cidr"}}}}) == nil {
		t.Fatal("expected invalid L3 address")
	}
}

func TestSwitchL2VLANValidationRejectsInvalidAndContradictoryMembership(t *testing.T) {
	tests := []SwitchL2Config{
		{Ports: []VLANPort{{Name: "eth0", PVID: -1}}},
		{Ports: []VLANPort{{Name: "eth0", PVID: 4095}}},
		{Ports: []VLANPort{{Name: "eth0", Tagged: []int{0}}}},
		{Ports: []VLANPort{{Name: "eth0", Tagged: []int{4095}}}},
		{Ports: []VLANPort{{Name: "eth0", Tagged: []int{20, 20}}}},
		{Ports: []VLANPort{{Name: "eth0", PVID: 20, Tagged: []int{10, 20}}}},
	}
	for _, config := range tests {
		if err := ValidateSwitchL2Config(config); err == nil {
			t.Fatalf("expected VLAN validation failure for %+v", config)
		}
	}
}

func TestNormalizeSwitchL2ConfigSortsTaggedMembershipWithoutMutatingInput(t *testing.T) {
	config := SwitchL2Config{VLANFiltering: true, Ports: []VLANPort{{Name: "eth0", PVID: 10, Tagged: []int{30, 20}}}}
	normalized, err := NormalizeSwitchL2Config(config)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(normalized.Ports[0].Tagged, []int{20, 30}) {
		t.Fatalf("unexpected normalized VLANs: %+v", normalized.Ports[0].Tagged)
	}
	if !reflect.DeepEqual(config.Ports[0].Tagged, []int{30, 20}) {
		t.Fatalf("input mutated: %+v", config.Ports[0].Tagged)
	}
}
