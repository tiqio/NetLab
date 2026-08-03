package domain

import (
	"fmt"
	"net/netip"
	"regexp"
	"strings"
)

const (
	NetworkBridge   = "bridge"
	NetworkNAT      = "nat_bridge"
	NetworkPC       = "pc"
	NetworkSwitchL2 = "switch_l2"
	NetworkSwitchL3 = "switch_l3"
)

type AddressMode string

const (
	AddressStatic AddressMode = "static"
	AddressDHCPv4 AddressMode = "dhcpv4"
	AddressDHCPv6 AddressMode = "dhcpv6"
	AddressSLAAC  AddressMode = "slaac"
)

type RouteConfig struct {
	Destination string `json:"destination" yaml:"destination"`
	Gateway     string `json:"gateway,omitempty" yaml:"gateway,omitempty"`
	Metric      int    `json:"metric,omitempty" yaml:"metric,omitempty"`
}

type NetworkConfigError struct {
	Code    string
	Message string
}

func (e NetworkConfigError) Error() string { return e.Message }

func networkConfigError(code, format string, args ...any) error {
	return NetworkConfigError{Code: code, Message: fmt.Sprintf(format, args...)}
}

type PCInterfaceConfig struct {
	Name      string        `json:"name" yaml:"name"`
	Modes     []AddressMode `json:"modes" yaml:"modes"`
	Addresses []string      `json:"addresses,omitempty" yaml:"addresses,omitempty"`
	Routes    []RouteConfig `json:"routes,omitempty" yaml:"routes,omitempty"`
	DNS       []string      `json:"dns,omitempty" yaml:"dns,omitempty"`
}

type PCConfig struct {
	Interfaces []PCInterfaceConfig `json:"interfaces" yaml:"interfaces"`
	Hostname   string              `json:"hostname,omitempty" yaml:"hostname,omitempty"`
}

type BridgeConfig struct {
	MTU int  `json:"mtu,omitempty" yaml:"mtu,omitempty"`
	STP bool `json:"stp,omitempty" yaml:"stp,omitempty"`
}

type NATConfig struct {
	IPv4Prefix           string     `json:"ipv4_prefix" yaml:"ipv4_prefix"`
	IPv6Prefix           string     `json:"ipv6_prefix,omitempty" yaml:"ipv6_prefix,omitempty"`
	Uplink               string     `json:"uplink" yaml:"uplink"`
	DHCPv4               *DHCPRange `json:"dhcpv4,omitempty" yaml:"dhcpv4,omitempty"`
	DHCPv6               *DHCPRange `json:"dhcpv6,omitempty" yaml:"dhcpv6,omitempty"`
	RouterAdvertisements bool       `json:"router_advertisements,omitempty" yaml:"router_advertisements,omitempty"`
	DNSServers           []string   `json:"dns_servers,omitempty" yaml:"dns_servers,omitempty"`
	Domain               string     `json:"domain,omitempty" yaml:"domain,omitempty"`
}

type DHCPRange struct {
	Start     string `json:"start" yaml:"start"`
	End       string `json:"end" yaml:"end"`
	LeaseTime string `json:"lease_time,omitempty" yaml:"lease_time,omitempty"`
}

type VLANPort struct {
	Name   string `json:"name" yaml:"name"`
	PVID   int    `json:"pvid,omitempty" yaml:"pvid,omitempty"`
	Tagged []int  `json:"tagged,omitempty" yaml:"tagged,omitempty"`
}

type SwitchL2Config struct {
	VLANFiltering bool       `json:"vlan_filtering" yaml:"vlan_filtering"`
	Ports         []VLANPort `json:"ports" yaml:"ports"`
}

type L3InterfaceConfig struct {
	Name      string   `json:"name" yaml:"name"`
	Addresses []string `json:"addresses" yaml:"addresses"`
}

type SwitchL3Config struct {
	Interfaces  []L3InterfaceConfig `json:"interfaces" yaml:"interfaces"`
	Routes      []RouteConfig       `json:"routes" yaml:"routes"`
	ForwardIPv4 bool                `json:"forward_ipv4" yaml:"forward_ipv4"`
	ForwardIPv6 bool                `json:"forward_ipv6" yaml:"forward_ipv6"`
}

var networkInterfaceNamePattern = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,15}$`)

func ValidateNetworkKind(kind string) error {
	switch kind {
	case NetworkBridge, NetworkNAT, NetworkPC, NetworkSwitchL2, NetworkSwitchL3:
		return nil
	default:
		return fmt.Errorf("unsupported network object kind %q", kind)
	}
}

func ValidatePCConfig(config PCConfig) error {
	if len(config.Interfaces) == 0 {
		return fmt.Errorf("PC requires at least one interface")
	}
	for _, iface := range config.Interfaces {
		if strings.TrimSpace(iface.Name) == "" {
			return fmt.Errorf("PC interface name required")
		}
		modes := map[AddressMode]bool{}
		for _, mode := range iface.Modes {
			switch mode {
			case AddressStatic, AddressDHCPv4, AddressDHCPv6, AddressSLAAC:
				modes[mode] = true
			default:
				return fmt.Errorf("unsupported address mode %q", mode)
			}
		}
		if modes[AddressStatic] && len(iface.Addresses) == 0 {
			return fmt.Errorf("static mode requires addresses")
		}
		for _, address := range iface.Addresses {
			if _, err := netip.ParsePrefix(address); err != nil {
				return fmt.Errorf("invalid address %q", address)
			}
		}
		for _, route := range iface.Routes {
			if _, err := netip.ParsePrefix(route.Destination); err != nil {
				return fmt.Errorf("invalid route destination %q", route.Destination)
			}
			if route.Gateway != "" {
				if _, err := netip.ParseAddr(route.Gateway); err != nil {
					return fmt.Errorf("invalid route gateway %q", route.Gateway)
				}
			}
		}
		for _, server := range iface.DNS {
			if _, err := netip.ParseAddr(server); err != nil {
				return fmt.Errorf("invalid DNS server %q", server)
			}
		}
	}
	return nil
}

func ValidateNodeNetworkInterfaces(interfaces []NodeNetworkInterfaceSettings) error {
	seenNames := map[string]bool{}
	connectedPrefixes := map[string]string{}
	declaredRoutes := map[string]string{}
	interfacePrefixes := make(map[string][]netip.Prefix, len(interfaces))
	for interfaceIndex := range interfaces {
		iface := &interfaces[interfaceIndex]
		if !networkInterfaceNamePattern.MatchString(iface.Name) {
			return networkConfigError("invalid_interface_name", "invalid network interface name %q", iface.Name)
		}
		if seenNames[iface.Name] {
			return networkConfigError("duplicate_interface", "duplicate network interface %q", iface.Name)
		}
		seenNames[iface.Name] = true
		prefixes := make([]netip.Prefix, 0, len(iface.Addresses))
		for addressIndex, raw := range iface.Addresses {
			prefix, err := netip.ParsePrefix(raw)
			if err != nil {
				return networkConfigError("invalid_interface_address", "invalid address %q on %s", raw, iface.Name)
			}
			iface.Addresses[addressIndex] = prefix.String()
			prefix = prefix.Masked()
			prefixes = append(prefixes, prefix)
			connectedPrefixes[prefix.String()] = iface.Name
		}
		interfacePrefixes[iface.Name] = prefixes
	}
	for interfaceIndex := range interfaces {
		iface := &interfaces[interfaceIndex]
		prefixes := interfacePrefixes[iface.Name]
		for routeIndex := range iface.Routes {
			route := &iface.Routes[routeIndex]
			destination, err := netip.ParsePrefix(route.Destination)
			if err != nil {
				return networkConfigError("invalid_route_destination", "invalid route destination %q on %s", route.Destination, iface.Name)
			}
			destination = destination.Masked()
			route.Destination = destination.String()
			if route.Metric < 0 {
				return networkConfigError("invalid_route_metric", "invalid route metric %d on %s", route.Metric, iface.Name)
			}
			gateway := netip.Addr{}
			if route.Gateway != "" {
				gateway, err = netip.ParseAddr(route.Gateway)
				if err != nil || gateway.Is4() != destination.Addr().Is4() {
					return networkConfigError("route_family_mismatch", "route gateway %q does not match destination family on %s", route.Gateway, iface.Name)
				}
				route.Gateway = gateway.String()
				reachable := false
				for _, prefix := range prefixes {
					if prefix.Addr().Is4() == gateway.Is4() && prefix.Contains(gateway) {
						reachable = true
						break
					}
				}
				if !reachable {
					return networkConfigError("route_gateway_unreachable", "route gateway %q is unreachable through %s", route.Gateway, iface.Name)
				}
			}
			key := destination.String()
			if connectedInterface, exists := connectedPrefixes[key]; exists {
				return networkConfigError("route_conflict", "route %q on %s conflicts with connected prefix on %s", key, iface.Name, connectedInterface)
			}
			if existingInterface, exists := declaredRoutes[key]; exists {
				return networkConfigError("route_conflict", "duplicate or conflicting route %q on %s and %s", key, existingInterface, iface.Name)
			}
			declaredRoutes[key] = iface.Name
		}
	}
	return nil
}

func ValidateNATConfig(config NATConfig) error {
	ipv4Prefix, err := netip.ParsePrefix(config.IPv4Prefix)
	if err != nil || !ipv4Prefix.Addr().Is4() || ipv4Prefix.Bits() > 30 {
		return fmt.Errorf("valid IPv4 NAT prefix of /30 or larger required")
	}
	var ipv6Prefix netip.Prefix
	if config.IPv6Prefix != "" {
		ipv6Prefix, err = netip.ParsePrefix(config.IPv6Prefix)
		if err != nil || !ipv6Prefix.Addr().Is6() || ipv6Prefix.Bits() > 126 {
			return fmt.Errorf("invalid IPv6 NAT prefix")
		}
	}
	if strings.TrimSpace(config.Uplink) == "" {
		return fmt.Errorf("NAT uplink required")
	}
	for label, item := range map[string]*DHCPRange{"dhcpv4": config.DHCPv4, "dhcpv6": config.DHCPv6} {
		if item == nil {
			continue
		}
		start, startErr := netip.ParseAddr(item.Start)
		end, endErr := netip.ParseAddr(item.End)
		if startErr != nil || endErr != nil || start.Compare(end) > 0 {
			return fmt.Errorf("invalid %s range", label)
		}
		parent := ipv4Prefix
		if label == "dhcpv6" {
			parent = ipv6Prefix
		}
		if !parent.IsValid() || !parent.Contains(start) || !parent.Contains(end) {
			return fmt.Errorf("%s range must be inside its NAT prefix", label)
		}
	}
	if (config.DHCPv6 != nil || config.RouterAdvertisements) && config.IPv6Prefix == "" {
		return fmt.Errorf("IPv6 prefix required for DHCPv6 or router advertisements")
	}
	for _, server := range config.DNSServers {
		if _, err := netip.ParseAddr(server); err != nil {
			return fmt.Errorf("invalid DNS server %q", server)
		}
	}
	return nil
}

func ValidateSwitchL2Config(config SwitchL2Config) error {
	if len(config.Ports) == 0 {
		return fmt.Errorf("layer-2 switch requires at least one port")
	}
	seen := map[string]struct{}{}
	for _, port := range config.Ports {
		if !networkInterfaceNamePattern.MatchString(port.Name) {
			return fmt.Errorf("invalid layer-2 port name %q", port.Name)
		}
		if _, exists := seen[port.Name]; exists {
			return fmt.Errorf("duplicate layer-2 port %q", port.Name)
		}
		seen[port.Name] = struct{}{}
		if port.PVID < 0 || port.PVID > 4094 {
			return fmt.Errorf("invalid PVID %d", port.PVID)
		}
		tagged := map[int]struct{}{}
		for _, vlan := range port.Tagged {
			if vlan < 1 || vlan > 4094 {
				return fmt.Errorf("invalid tagged VLAN %d", vlan)
			}
			if vlan == port.PVID {
				return fmt.Errorf("PVID %d cannot also be tagged", vlan)
			}
			if _, exists := tagged[vlan]; exists {
				return fmt.Errorf("duplicate tagged VLAN %d", vlan)
			}
			tagged[vlan] = struct{}{}
		}
	}
	return nil
}

func ValidateSwitchL3Config(config SwitchL3Config) error {
	if len(config.Interfaces) == 0 {
		return fmt.Errorf("layer-3 switch requires at least one interface")
	}
	seen := map[string]struct{}{}
	for _, iface := range config.Interfaces {
		if !networkInterfaceNamePattern.MatchString(iface.Name) {
			return fmt.Errorf("invalid layer-3 interface name %q", iface.Name)
		}
		if _, exists := seen[iface.Name]; exists {
			return fmt.Errorf("duplicate layer-3 interface %q", iface.Name)
		}
		seen[iface.Name] = struct{}{}
		if len(iface.Addresses) == 0 {
			return fmt.Errorf("layer-3 interface %q requires an address", iface.Name)
		}
		for _, address := range iface.Addresses {
			if _, err := netip.ParsePrefix(address); err != nil {
				return fmt.Errorf("invalid layer-3 address %q", address)
			}
		}
	}
	for _, route := range config.Routes {
		destination, err := netip.ParsePrefix(route.Destination)
		if err != nil {
			return fmt.Errorf("invalid route destination %q", route.Destination)
		}
		if route.Gateway == "" {
			continue
		}
		gateway, err := netip.ParseAddr(route.Gateway)
		if err != nil || gateway.BitLen() != destination.Addr().BitLen() {
			return fmt.Errorf("invalid route gateway %q", route.Gateway)
		}
	}
	return nil
}

func PrefixesOverlap(left, right string) bool {
	a, errA := netip.ParsePrefix(left)
	b, errB := netip.ParsePrefix(right)
	if errA != nil || errB != nil || a.Addr().BitLen() != b.Addr().BitLen() {
		return false
	}
	return a.Contains(b.Addr()) || b.Contains(a.Addr())
}
