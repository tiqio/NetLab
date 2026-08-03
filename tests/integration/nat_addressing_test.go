package integration_test

import (
	"strings"
	"testing"

	"github.com/netlab/netlab/internal/domain"
	"github.com/netlab/netlab/internal/runtime/linuxnet"
)

func TestDNSMasqConfigurationCoversDHCPv4DHCPv6AndRA(t *testing.T) {
	config := domain.NATConfig{IPv4Prefix: "10.44.0.0/24", IPv6Prefix: "fd44::/64", Uplink: "eth0", DHCPv4: &domain.DHCPRange{Start: "10.44.0.10", End: "10.44.0.200"}, DHCPv6: &domain.DHCPRange{Start: "fd44::10", End: "fd44::ff"}, RouterAdvertisements: true, DNSServers: []string{"1.1.1.1", "2606:4700:4700::1111"}, Domain: "lab.internal"}
	body, err := linuxnet.BuildDNSMasqConfig("nlnat-test", "/run/netlab/leases", config)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"interface=nlnat-test", "10.44.0.10,10.44.0.200", "fd44::10,fd44::ff", "enable-ra", "server=1.1.1.1", "domain=lab.internal"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("missing %q in %s", expected, body)
		}
	}
}

func TestNATRangeMustStayInsideDeclaredPrefix(t *testing.T) {
	config := domain.NATConfig{IPv4Prefix: "10.44.0.0/24", Uplink: "eth0", DHCPv4: &domain.DHCPRange{Start: "10.45.0.10", End: "10.45.0.20"}}
	if domain.ValidateNATConfig(config) == nil {
		t.Fatal("expected out-of-prefix range rejection")
	}
}
