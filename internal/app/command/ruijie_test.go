package command

import (
	"reflect"
	"testing"

	"github.com/netlab/netlab/internal/domain"
)

func TestBuildRuijieAccessCommands(t *testing.T) {
	node := domain.Node{Config: map[string]any{"template_key": "ruijie-switch"}}
	commands, err := BuildRuijieCommands(node, RuijieConfigRequest{Operation: "l2_access", Interface: "G0/0", VLANID: 10, AdminUp: true, Save: true})
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"enable", "configure terminal", "interface G0/0", "switchport mode access", "switchport access vlan 10", "no shutdown", "exit", "end", "write"}
	if !reflect.DeepEqual(commands, expected) {
		t.Fatalf("commands=%v expected=%v", commands, expected)
	}
}

func TestBuildRuijieRouterAddressCommands(t *testing.T) {
	node := domain.Node{Config: map[string]any{"template_key": "ruijie-router"}}
	commands, err := BuildRuijieCommands(node, RuijieConfigRequest{Operation: "l3_address", Interface: "G0/0", AddressCIDR: "192.0.2.1/24", AdminUp: true})
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"enable", "configure terminal", "interface G0/0", "ip address 192.0.2.1 255.255.255.0", "no shutdown", "exit", "end"}
	if !reflect.DeepEqual(commands, expected) {
		t.Fatalf("commands=%v expected=%v", commands, expected)
	}
}

func TestBuildRuijieRejectsUnsafeValues(t *testing.T) {
	node := domain.Node{Config: map[string]any{"template_key": "ruijie-switch"}}
	for _, request := range []RuijieConfigRequest{
		{Operation: "l2_access", Interface: "G0/0;reload", VLANID: 10},
		{Operation: "l2_access", Interface: "G0/0", VLANID: 4095},
		{Operation: "l2_trunk", Interface: "G0/0", AllowedVLANs: "10,20;reload"},
	} {
		if _, err := BuildRuijieCommands(node, request); err == nil {
			t.Fatalf("request %+v was accepted", request)
		}
	}
}
