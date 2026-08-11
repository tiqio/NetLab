package linuxnet

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"slices"
	"strings"

	"github.com/netlab/netlab/internal/domain"
)

type SwitchL2Runtime struct {
	executor   CommandExecutor
	ip, bridge string
}

type SwitchL2PortObservation struct {
	Name   string `json:"name"`
	PVID   int    `json:"pvid,omitempty"`
	Tagged []int  `json:"tagged,omitempty"`
}

func (r *SwitchL2Runtime) InspectNetworkObject(ctx context.Context, object domain.NetworkObject) (domain.RuntimeBackingObservation, error) {
	return inspectNamespaceBacking(ctx, r.executor, r.ip, SwitchL2NamespaceName(object.ID)), nil
}

func NewSwitchL2Runtime(executor CommandExecutor) (*SwitchL2Runtime, error) {
	if executor != nil {
		return &SwitchL2Runtime{executor: executor, ip: "ip", bridge: "bridge"}, nil
	}
	ip, err := exec.LookPath("ip")
	if err != nil {
		return nil, err
	}
	bridge, err := exec.LookPath("bridge")
	if err != nil {
		return nil, err
	}
	return &SwitchL2Runtime{executor: SystemExecutor{}, ip: ip, bridge: bridge}, nil
}

func (r *SwitchL2Runtime) Configure(ctx context.Context, object domain.NetworkObject) error {
	var config domain.SwitchL2Config
	if err := decodeConfig(object.Config, &config); err != nil {
		return err
	}
	normalized, err := domain.NormalizeSwitchL2Config(config)
	if err != nil {
		return err
	}
	namespace := SwitchL2NamespaceName(object.ID)
	if err := ensureNamespace(ctx, r.executor, r.ip, namespace); err != nil {
		return err
	}
	args := []string{"-n", namespace, "link", "add", "br0", "type", "bridge"}
	if normalized.VLANFiltering {
		args = append(args, "vlan_filtering", "1")
	}
	if err := r.executor.Run(ctx, r.ip, args...); err != nil {
		if _, inspectErr := r.executor.Output(ctx, r.ip, "-n", namespace, "link", "show", "br0"); inspectErr != nil {
			return err
		}
	}
	bridgeState, _ := r.executor.Output(ctx, r.ip, "-n", namespace, "-d", "link", "show", "br0")
	filtering := "0"
	if normalized.VLANFiltering {
		filtering = "1"
	}
	if !strings.Contains(string(bridgeState), "vlan_filtering "+filtering) {
		if err := r.executor.Run(ctx, r.ip, "-n", namespace, "link", "set", "br0", "type", "bridge", "vlan_filtering", filtering); err != nil {
			return err
		}
	}
	if !linkIsUp(bridgeState) {
		if err := r.executor.Run(ctx, r.ip, "-n", namespace, "link", "set", "br0", "up"); err != nil {
			return err
		}
	}
	for _, port := range normalized.Ports {
		if err := r.configurePort(ctx, namespace, port, normalized.VLANFiltering); err != nil {
			return err
		}
	}
	return nil
}

func (r *SwitchL2Runtime) configurePort(ctx context.Context, namespace string, port domain.VLANPort, vlanFiltering bool) error {
	if !hostObjectName.MatchString(port.Name) {
		return fmt.Errorf("invalid L2 port name")
	}
	state, err := r.executor.Output(ctx, r.ip, "-n", namespace, "-d", "link", "show", port.Name)
	if err != nil {
		return nil
	}
	if !strings.Contains(string(state), "master br0") {
		if err = r.executor.Run(ctx, r.ip, "-n", namespace, "link", "set", port.Name, "master", "br0"); err != nil {
			return err
		}
	}
	if !linkIsUp(state) {
		if err = r.executor.Run(ctx, r.ip, "-n", namespace, "link", "set", port.Name, "up"); err != nil {
			return err
		}
	}
	if !vlanFiltering {
		return nil
	}
	observed, observeErr := r.observePort(ctx, namespace, port.Name)
	if observeErr == nil && vlanMembershipMatches(port, observed) {
		return nil
	}
	if err = r.executor.Run(ctx, r.bridge, "-n", namespace, "vlan", "del", "dev", port.Name, "vid", "1-4094"); err != nil && !missingVLANMembership(err) {
		return err
	}
	if port.PVID > 0 {
		if err = r.executor.Run(ctx, r.bridge, "-n", namespace, "vlan", "add", "dev", port.Name, "vid", fmt.Sprint(port.PVID), "pvid", "untagged"); err != nil {
			return err
		}
	}
	for _, vlan := range port.Tagged {
		if err = r.executor.Run(ctx, r.bridge, "-n", namespace, "vlan", "add", "dev", port.Name, "vid", fmt.Sprint(vlan)); err != nil {
			return err
		}
	}
	return nil
}

func missingVLANMembership(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "no such file") || strings.Contains(message, "cannot find device") || strings.Contains(message, "does not exist")
}

func (r *SwitchL2Runtime) observePort(ctx context.Context, namespace, portName string) (SwitchL2PortObservation, error) {
	body, err := r.executor.Output(ctx, r.bridge, "-j", "-n", namespace, "vlan", "show", "dev", portName)
	if err != nil {
		return SwitchL2PortObservation{Name: portName}, err
	}
	var values []struct {
		Name  string `json:"ifname"`
		VLANs []struct {
			VLAN  int      `json:"vlan"`
			Flags []string `json:"flags"`
		} `json:"vlans"`
	}
	if err = json.Unmarshal(body, &values); err != nil {
		return SwitchL2PortObservation{Name: portName}, err
	}
	observation := SwitchL2PortObservation{Name: portName}
	for _, value := range values {
		if value.Name != "" && value.Name != portName {
			continue
		}
		for _, vlan := range value.VLANs {
			pvid := false
			untagged := false
			for _, flag := range vlan.Flags {
				switch strings.ToLower(flag) {
				case "pvid":
					pvid = true
				case "egress untagged", "untagged":
					untagged = true
				}
			}
			if pvid && untagged {
				observation.PVID = vlan.VLAN
				continue
			}
			observation.Tagged = append(observation.Tagged, vlan.VLAN)
		}
	}
	slices.Sort(observation.Tagged)
	return observation, nil
}

func vlanMembershipMatches(desired domain.VLANPort, observed SwitchL2PortObservation) bool {
	return desired.PVID == observed.PVID && slices.Equal(desired.Tagged, observed.Tagged)
}

func (r *SwitchL2Runtime) DiagnosticsObject(ctx context.Context, object domain.NetworkObject) (map[string]any, error) {
	var config domain.SwitchL2Config
	if err := decodeConfig(object.Config, &config); err != nil {
		return nil, err
	}
	desired, err := domain.NormalizeSwitchL2Config(config)
	if err != nil {
		return nil, err
	}
	namespace := SwitchL2NamespaceName(object.ID)
	observed := make([]SwitchL2PortObservation, 0, len(desired.Ports))
	mismatches := make([]string, 0)
	for _, port := range desired.Ports {
		value, observeErr := r.observePort(ctx, namespace, port.Name)
		if observeErr != nil {
			mismatches = append(mismatches, port.Name+": unavailable")
		} else if !vlanMembershipMatches(port, value) {
			mismatches = append(mismatches, port.Name+": VLAN membership differs")
		}
		observed = append(observed, value)
	}
	return map[string]any{
		"desired":    map[string]any{"vlan_filtering": desired.VLANFiltering, "ports": desired.Ports},
		"observed":   map[string]any{"ports": observed},
		"mismatches": mismatches,
	}, nil
}

func (r *SwitchL2Runtime) ConfigurationConverged(ctx context.Context, object domain.NetworkObject) (bool, map[string]any, error) {
	diagnostics, err := r.DiagnosticsObject(ctx, object)
	if err != nil {
		return false, nil, err
	}
	mismatches, _ := diagnostics["mismatches"].([]string)
	return len(mismatches) == 0, diagnostics, nil
}

func (r *SwitchL2Runtime) Delete(ctx context.Context, id domain.ID) error {
	return deleteNamespace(ctx, r.executor, r.ip, SwitchL2NamespaceName(id))
}
