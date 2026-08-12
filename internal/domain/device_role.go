package domain

import (
	"encoding/json"
	"fmt"
	"net/netip"
	"strings"
)

type DeviceInterfaceRole struct {
	InterfaceID   ID     `json:"interface_id"`
	Role          string `json:"role"`
	AddressFamily string `json:"address_family,omitempty"`
	Address       string `json:"address,omitempty"`
	Gateway       string `json:"gateway,omitempty"`
}

func (r *DeviceInterfaceRole) UnmarshalJSON(body []byte) error {
	type alias DeviceInterfaceRole
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return err
	}
	for _, key := range []string{"password", "secret", "token", "credential", "credentials", "private_key"} {
		if _, ok := raw[key]; ok {
			return fmt.Errorf("device role metadata must not contain %s", key)
		}
	}
	var value alias
	if err := json.Unmarshal(body, &value); err != nil {
		return err
	}
	*r = DeviceInterfaceRole(value)
	return nil
}

func ValidateDeviceInterfaceRoles(values []DeviceInterfaceRole) error {
	seen := map[ID]bool{}
	validRoles := map[string]bool{"management": true, "lan": true, "wan": true, "trunk": true, "client-facing": true}
	validFamilies := map[string]bool{"": true, "ipv4": true, "ipv6": true, "dual": true}
	for index := range values {
		value := &values[index]
		value.Role = strings.ToLower(strings.TrimSpace(value.Role))
		value.AddressFamily = strings.ToLower(strings.TrimSpace(value.AddressFamily))
		value.Address = strings.TrimSpace(value.Address)
		value.Gateway = strings.TrimSpace(value.Gateway)
		if value.InterfaceID == "" || seen[value.InterfaceID] {
			return fmt.Errorf("device role interface must be present and unique")
		}
		seen[value.InterfaceID] = true
		if !validRoles[value.Role] {
			return fmt.Errorf("unsupported device interface role %q", value.Role)
		}
		if !validFamilies[value.AddressFamily] {
			return fmt.Errorf("unsupported device role address family %q", value.AddressFamily)
		}
		if value.Address != "" {
			prefix, err := netip.ParsePrefix(value.Address)
			if err != nil {
				return fmt.Errorf("invalid device role address %q", value.Address)
			}
			if value.AddressFamily == "ipv4" && !prefix.Addr().Is4() || value.AddressFamily == "ipv6" && !prefix.Addr().Is6() {
				return fmt.Errorf("device role address family does not match %q", value.Address)
			}
		}
		if value.Gateway != "" {
			gateway, err := netip.ParseAddr(value.Gateway)
			if err != nil {
				return fmt.Errorf("invalid device role gateway %q", value.Gateway)
			}
			if value.AddressFamily == "ipv4" && !gateway.Is4() || value.AddressFamily == "ipv6" && !gateway.Is6() {
				return fmt.Errorf("device role gateway family does not match %q", value.Gateway)
			}
		}
	}
	return nil
}

type DeviceReadinessLevel struct {
	State   string   `json:"state"`
	Details []string `json:"details,omitempty"`
}

type DeviceReadiness struct {
	NodeID     ID                    `json:"node_id"`
	Roles      []DeviceInterfaceRole `json:"roles"`
	Cable      DeviceReadinessLevel  `json:"cable"`
	Guest      DeviceReadinessLevel  `json:"guest"`
	Management DeviceReadinessLevel  `json:"management"`
	DataPath   DeviceReadinessLevel  `json:"data_path"`
}
