package linuxnet

import (
	"fmt"
	"net/netip"
	"strings"

	"github.com/netlab/netlab/internal/domain"
)

func BuildDNSMasqConfig(bridge, leasePath string, config domain.NATConfig) (string, error) {
	if err := domain.ValidateNATConfig(config); err != nil {
		return "", err
	}
	if strings.TrimSpace(bridge) == "" || strings.ContainsAny(bridge, "\r\n") || strings.ContainsAny(leasePath, "\r\n") {
		return "", fmt.Errorf("invalid dnsmasq path or interface")
	}
	lines := []string{"bind-interfaces", "interface=" + bridge, "except-interface=lo", "dhcp-authoritative", "dhcp-leasefile=" + leasePath, "no-hosts", "no-resolv"}
	if config.DHCPv4 != nil {
		lines = append(lines, fmt.Sprintf("dhcp-range=%s,%s,%s", config.DHCPv4.Start, config.DHCPv4.End, leaseTime(config.DHCPv4.LeaseTime)))
	}
	if config.DHCPv6 != nil {
		lines = append(lines, fmt.Sprintf("dhcp-range=%s,%s,%s", config.DHCPv6.Start, config.DHCPv6.End, leaseTime(config.DHCPv6.LeaseTime)))
	}
	if config.RouterAdvertisements {
		prefix, _ := netip.ParsePrefix(config.IPv6Prefix)
		lines = append(lines, "enable-ra", fmt.Sprintf("dhcp-range=%s,ra-stateless,ra-names,64,12h", prefix.Masked().Addr()))
	}
	for _, server := range config.DNSServers {
		lines = append(lines, "server="+server)
	}
	if config.Domain != "" {
		if strings.ContainsAny(config.Domain, "\r\n=, ") {
			return "", fmt.Errorf("invalid DNS domain")
		}
		lines = append(lines, "domain="+config.Domain)
	}
	return strings.Join(lines, "\n") + "\n", nil
}

func leaseTime(value string) string {
	if value == "" {
		return "12h"
	}
	return value
}
