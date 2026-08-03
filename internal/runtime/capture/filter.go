package capture

import (
	"fmt"
	"net/netip"
	"strconv"
	"strings"
)

type Match struct {
	SourceAddress      string  `json:"source_address,omitempty"`
	DestinationAddress string  `json:"destination_address,omitempty"`
	Protocol           string  `json:"protocol,omitempty"`
	SourcePort         int     `json:"source_port,omitempty"`
	DestinationPort    int     `json:"destination_port,omitempty"`
	And                []Match `json:"and,omitempty"`
	Or                 []Match `json:"or,omitempty"`
	Not                *Match  `json:"not,omitempty"`
}

func Compile(match Match) (string, error) {
	return compile(match, 0)
}

func compile(match Match, depth int) (string, error) {
	if depth > 32 {
		return "", fmt.Errorf("traffic filter expression exceeds maximum depth")
	}
	logical := 0
	if len(match.And) > 0 {
		logical++
	}
	if len(match.Or) > 0 {
		logical++
	}
	if match.Not != nil {
		logical++
	}
	if logical > 1 || (logical > 0 && hasPredicate(match)) {
		return "", fmt.Errorf("expression must contain either one logical operator or leaf predicates")
	}
	if len(match.And) > 0 {
		return compileGroup(match.And, "and", depth)
	}
	if len(match.Or) > 0 {
		return compileGroup(match.Or, "or", depth)
	}
	if match.Not != nil {
		expression, err := compile(*match.Not, depth+1)
		if err != nil {
			return "", err
		}
		if expression == "" {
			return "", fmt.Errorf("not expression cannot be empty")
		}
		return "not (" + expression + ")", nil
	}
	parts := make([]string, 0, 5)
	if match.SourceAddress != "" {
		prefix, err := parseAddressOrPrefix(match.SourceAddress)
		if err != nil {
			return "", fmt.Errorf("source address: %w", err)
		}
		parts = append(parts, "src "+prefix)
	}
	if match.DestinationAddress != "" {
		prefix, err := parseAddressOrPrefix(match.DestinationAddress)
		if err != nil {
			return "", fmt.Errorf("destination address: %w", err)
		}
		parts = append(parts, "dst "+prefix)
	}
	protocol := strings.ToLower(match.Protocol)
	if protocol != "" {
		switch protocol {
		case "tcp", "udp", "icmp", "icmp6", "arp", "ip", "ip6":
			parts = append(parts, protocol)
		default:
			return "", fmt.Errorf("unsupported protocol %q", protocol)
		}
	}
	if match.SourcePort > 0 || match.DestinationPort > 0 {
		if protocol != "tcp" && protocol != "udp" {
			return "", fmt.Errorf("ports require tcp or udp protocol")
		}
	}
	if match.SourcePort > 0 {
		if err := validatePort(match.SourcePort); err != nil {
			return "", err
		}
		parts = append(parts, "src port "+strconv.Itoa(match.SourcePort))
	}
	if match.DestinationPort > 0 {
		if err := validatePort(match.DestinationPort); err != nil {
			return "", err
		}
		parts = append(parts, "dst port "+strconv.Itoa(match.DestinationPort))
	}
	return strings.Join(parts, " and "), nil
}

func compileGroup(values []Match, operator string, depth int) (string, error) {
	if len(values) < 2 {
		return "", fmt.Errorf("%s expression requires at least two operands", operator)
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		expression, err := compile(value, depth+1)
		if err != nil {
			return "", err
		}
		if expression == "" {
			return "", fmt.Errorf("%s expression contains an empty operand", operator)
		}
		parts = append(parts, "("+expression+")")
	}
	return strings.Join(parts, " "+operator+" "), nil
}

func hasPredicate(match Match) bool {
	return match.SourceAddress != "" || match.DestinationAddress != "" || match.Protocol != "" || match.SourcePort != 0 || match.DestinationPort != 0
}

func parseAddressOrPrefix(value string) (string, error) {
	if prefix, err := netip.ParsePrefix(value); err == nil {
		return "net " + prefix.String(), nil
	}
	address, err := netip.ParseAddr(value)
	if err != nil {
		return "", fmt.Errorf("invalid IP address or prefix")
	}
	return "host " + address.String(), nil
}

func validatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port %d is out of range", port)
	}
	return nil
}
