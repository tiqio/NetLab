package linuxnet

import (
	"context"
	"strings"
	"testing"

	"github.com/netlab/netlab/internal/domain"
)

func TestNATPrefixGatewayMasqueradeUplinkAndCleanup(t *testing.T) {
	executor := &scriptExecutor{}
	runtime, _ := NewNATRuntime(executor)
	object := domain.NetworkObject{ID: "nat-1", Kind: domain.NetworkNAT, Config: map[string]any{"ipv4_prefix": "10.10.0.0/24", "ipv6_prefix": "2001:db8:1::/64", "uplink": "eth0"}}
	if err := runtime.Configure(context.Background(), object); err != nil {
		t.Fatal(err)
	}
	commands := strings.Join(executor.commands, "\n")
	for _, fragment := range []string{
		"address replace 10.10.0.1/24",
		"ip saddr 10.10.0.0/24 oifname eth0 masquerade",
		`comment "netlab:nat-1"`,
		"insert rule ip filter FORWARD iifname nlnat",
		`oifname eth0 accept comment "netlab-forward-out:nat-1"`,
		`iifname eth0 oifname nlnat`,
		`ct state established,related accept comment "netlab-forward-in:nat-1"`,
	} {
		if !strings.Contains(commands, fragment) {
			t.Fatalf("missing %q in\n%s", fragment, commands)
		}
	}
	if err := runtime.Delete(context.Background(), object.ID); err != nil {
		t.Fatal(err)
	}
	commands = strings.Join(executor.commands, "\n")
	if !strings.Contains(commands, "link delete nlnat") {
		t.Fatal("bridge cleanup missing")
	}
}

func TestNATAutoUplinkUsesDefaultRouteDevice(t *testing.T) {
	executor := &scriptExecutor{output: []byte("default via 192.0.2.1 dev ens33 proto dhcp\n")}
	runtime, _ := NewNATRuntime(executor)
	object := domain.NetworkObject{ID: "nat-auto", Kind: domain.NetworkNAT, Config: map[string]any{"ipv4_prefix": "10.20.0.0/24", "uplink": "auto"}}
	if err := runtime.Configure(context.Background(), object); err != nil {
		t.Fatal(err)
	}
	commands := strings.Join(executor.commands, "\n")
	for _, fragment := range []string{"ip route show default", "oifname ens33 masquerade", "oifname ens33 accept", "iifname ens33"} {
		if !strings.Contains(commands, fragment) {
			t.Fatalf("missing %q in\n%s", fragment, commands)
		}
	}
}

func TestNATOverlapAndValidation(t *testing.T) {
	if !domain.PrefixesOverlap("10.0.0.0/24", "10.0.0.128/25") {
		t.Fatal("overlap not detected")
	}
	if domain.PrefixesOverlap("10.0.0.0/24", "10.0.1.0/24") {
		t.Fatal("false overlap")
	}
	if err := domain.ValidateNATConfig(domain.NATConfig{IPv4Prefix: "10.0.0.0/24"}); err == nil {
		t.Fatal("missing uplink accepted")
	}
}
