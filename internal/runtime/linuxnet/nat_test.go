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

func TestNATReconfigureReplacesOwnedRulesWithCurrentUplink(t *testing.T) {
	executor := &scriptExecutor{outputFor: func(name string, args ...string) []byte {
		command := name + " " + strings.Join(args, " ")
		switch {
		case strings.Contains(command, "ip route show default"):
			return []byte("default via 10.72.1.231 dev pnet0\n")
		case strings.Contains(command, "nft -a list chain inet netlab_nat postrouting"):
			return []byte(`ip saddr 10.20.0.0/24 oifname "eth0" masquerade comment "netlab:nat-auto" # handle 10`)
		case strings.Contains(command, "nft -a list chain ip filter FORWARD"):
			return []byte("iifname nlnat-old oifname eth0 accept comment \"netlab-forward-out:nat-auto\" # handle 11\niifname eth0 oifname nlnat-old accept comment \"netlab-forward-in:nat-auto\" # handle 12\n")
		default:
			return nil
		}
	}}
	runtime, _ := NewNATRuntime(executor)
	object := domain.NetworkObject{ID: "nat-auto", Kind: domain.NetworkNAT, Config: map[string]any{"ipv4_prefix": "10.21.0.0/24", "uplink": "auto"}}
	if err := runtime.Configure(context.Background(), object); err != nil {
		t.Fatal(err)
	}
	commands := strings.Join(executor.commands, "\n")
	for _, fragment := range []string{
		"delete rule inet netlab_nat postrouting handle 10",
		"delete rule ip filter FORWARD handle 11",
		"delete rule ip filter FORWARD handle 12",
		"ip saddr 10.21.0.0/24 oifname pnet0 masquerade",
		"iifname nlnat",
		"oifname pnet0 accept",
		"iifname pnet0 oifname nlnat",
	} {
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
