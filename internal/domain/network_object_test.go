package domain

import "testing"

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
	if ValidateSwitchL3Config(SwitchL3Config{Interfaces: []L3InterfaceConfig{{Name: "lan0", Addresses: []string{"not-cidr"}}}}) == nil {
		t.Fatal("expected invalid L3 address")
	}
}
