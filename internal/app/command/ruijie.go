package command

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"

	"github.com/netlab/netlab/internal/domain"
)

type RuijieConfigRequest struct {
	Operation    string `json:"operation"`
	Interface    string `json:"interface,omitempty"`
	VLANID       int    `json:"vlan_id,omitempty"`
	VLANName     string `json:"vlan_name,omitempty"`
	AllowedVLANs string `json:"allowed_vlans,omitempty"`
	AddressCIDR  string `json:"address_cidr,omitempty"`
	AdminUp      bool   `json:"admin_up"`
	Save         bool   `json:"save"`
}

var (
	ruijieInterfacePattern   = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9./_-]{0,63}$`)
	ruijieVLANNamePattern    = regexp.MustCompile(`^[A-Za-z0-9_-]{1,32}$`)
	ruijieAllowedVLANPattern = regexp.MustCompile(`^[0-9,-]{1,128}$`)
)

func BuildRuijieCommands(node domain.Node, request RuijieConfigRequest) ([]string, error) {
	templateKey, _ := node.Config["template_key"].(string)
	isSwitch := templateKey == "ruijie-switch"
	isRouter := templateKey == "ruijie-router"
	if !isSwitch && !isRouter {
		return nil, fmt.Errorf("node is not a supported Ruijie appliance")
	}

	commands := []string{"enable", "configure terminal"}
	switch request.Operation {
	case "create_vlan":
		if !isSwitch {
			return nil, fmt.Errorf("VLAN configuration is only supported by Ruijie switches")
		}
		if err := validateRuijieVLAN(request.VLANID); err != nil {
			return nil, err
		}
		commands = append(commands, "vlan "+strconv.Itoa(request.VLANID))
		if name := strings.TrimSpace(request.VLANName); name != "" {
			if !ruijieVLANNamePattern.MatchString(name) {
				return nil, fmt.Errorf("vlan_name must contain only letters, numbers, underscores, or hyphens")
			}
			commands = append(commands, "name "+name)
		}
		commands = append(commands, "exit")
	case "l2_access":
		if !isSwitch {
			return nil, fmt.Errorf("access VLAN configuration is only supported by Ruijie switches")
		}
		iface, err := validateRuijieInterface(request.Interface)
		if err != nil {
			return nil, err
		}
		if err = validateRuijieVLAN(request.VLANID); err != nil {
			return nil, err
		}
		commands = append(commands, "interface "+iface, "switchport mode access", "switchport access vlan "+strconv.Itoa(request.VLANID))
		commands = append(commands, ruijieAdminCommand(request.AdminUp), "exit")
	case "l2_trunk":
		if !isSwitch {
			return nil, fmt.Errorf("trunk configuration is only supported by Ruijie switches")
		}
		iface, err := validateRuijieInterface(request.Interface)
		if err != nil {
			return nil, err
		}
		allowed, err := validateRuijieAllowedVLANs(request.AllowedVLANs)
		if err != nil {
			return nil, err
		}
		commands = append(commands, "interface "+iface, "switchport mode trunk", "switchport trunk allowed vlan "+allowed)
		commands = append(commands, ruijieAdminCommand(request.AdminUp), "exit")
	case "l3_address":
		if !isRouter {
			return nil, fmt.Errorf("layer-3 addressing is only supported by Ruijie routers")
		}
		iface, err := validateRuijieInterface(request.Interface)
		if err != nil {
			return nil, err
		}
		address, mask, err := ruijieIPv4Address(request.AddressCIDR)
		if err != nil {
			return nil, err
		}
		commands = append(commands, "interface "+iface, "ip address "+address+" "+mask, ruijieAdminCommand(request.AdminUp), "exit")
	case "admin_state":
		iface, err := validateRuijieInterface(request.Interface)
		if err != nil {
			return nil, err
		}
		commands = append(commands, "interface "+iface, ruijieAdminCommand(request.AdminUp), "exit")
	default:
		return nil, fmt.Errorf("unsupported Ruijie operation %q", request.Operation)
	}
	commands = append(commands, "end")
	if request.Save {
		commands = append(commands, "write")
	}
	return commands, nil
}

func validateRuijieInterface(value string) (string, error) {
	value = strings.TrimSpace(value)
	if !ruijieInterfacePattern.MatchString(value) {
		return "", fmt.Errorf("interface must be a valid Ruijie interface name")
	}
	return value, nil
}

func validateRuijieVLAN(value int) error {
	if value < 1 || value > 4094 {
		return fmt.Errorf("vlan_id must be between 1 and 4094")
	}
	return nil
}

func validateRuijieAllowedVLANs(value string) (string, error) {
	value = strings.TrimSpace(value)
	if !ruijieAllowedVLANPattern.MatchString(value) {
		return "", fmt.Errorf("allowed_vlans must use comma-separated VLAN IDs or ranges")
	}
	for _, item := range strings.Split(value, ",") {
		bounds := strings.Split(item, "-")
		if len(bounds) > 2 {
			return "", fmt.Errorf("allowed_vlans contains an invalid range")
		}
		first, err := strconv.Atoi(bounds[0])
		if err != nil || validateRuijieVLAN(first) != nil {
			return "", fmt.Errorf("allowed_vlans contains an invalid VLAN")
		}
		if len(bounds) == 2 {
			last, rangeErr := strconv.Atoi(bounds[1])
			if rangeErr != nil || validateRuijieVLAN(last) != nil || last < first {
				return "", fmt.Errorf("allowed_vlans contains an invalid range")
			}
		}
	}
	return value, nil
}

func ruijieIPv4Address(value string) (string, string, error) {
	address, network, err := net.ParseCIDR(strings.TrimSpace(value))
	if err != nil || address.To4() == nil {
		return "", "", fmt.Errorf("address_cidr must be a valid IPv4 CIDR")
	}
	return address.String(), net.IP(network.Mask).String(), nil
}

func ruijieAdminCommand(up bool) string {
	if up {
		return "no shutdown"
	}
	return "shutdown"
}
