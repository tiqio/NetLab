package domain

import "fmt"

var defaultLightweightPortNames = []string{"eth0", "eth1", "eth2", "eth3"}

func ApplyLightweightPortDefaultsOnCreate(kind string, config map[string]any) map[string]any {
	result := cloneStringAnyMap(config)
	switch kind {
	case NetworkSwitchL2:
		if _, explicit := result["ports"]; !explicit {
			ports := make([]any, 0, len(defaultLightweightPortNames))
			for _, name := range defaultLightweightPortNames {
				ports = append(ports, map[string]any{"name": name, "pvid": 1, "tagged": []any{}})
			}
			result["ports"] = ports
		}
		if _, exists := result["vlan_filtering"]; !exists {
			result["vlan_filtering"] = true
		}
	case NetworkSwitchL3:
		if _, explicit := result["interfaces"]; !explicit {
			interfaces := make([]any, 0, len(defaultLightweightPortNames))
			for _, name := range defaultLightweightPortNames {
				interfaces = append(interfaces, map[string]any{"name": name, "addresses": []any{}})
			}
			result["interfaces"] = interfaces
		}
		if _, exists := result["routes"]; !exists {
			result["routes"] = []any{}
		}
		if _, exists := result["forward_ipv4"]; !exists {
			result["forward_ipv4"] = true
		}
		if _, exists := result["forward_ipv6"]; !exists {
			result["forward_ipv6"] = true
		}
	}
	return result
}

func ValidateUniqueLightweightPortNames(kind string, config map[string]any) error {
	key := ""
	switch kind {
	case NetworkSwitchL2:
		key = "ports"
	case NetworkSwitchL3:
		key = "interfaces"
	default:
		return nil
	}
	values, _ := config[key].([]any)
	seen := map[string]struct{}{}
	for _, raw := range values {
		value, _ := raw.(map[string]any)
		name, _ := value["name"].(string)
		if err := ValidateNetworkObjectPortName(name); err != nil {
			return err
		}
		if _, exists := seen[name]; exists {
			return fmt.Errorf("duplicate lightweight port name %q", name)
		}
		seen[name] = struct{}{}
	}
	return nil
}

func cloneStringAnyMap(value map[string]any) map[string]any {
	result := make(map[string]any, len(value))
	for key, item := range value {
		result[key] = item
	}
	return result
}
