package linuxnet

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/netlab/netlab/internal/domain"
)

type SwitchL2Runtime struct {
	executor   CommandExecutor
	ip, bridge string
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
	if err := domain.ValidateSwitchL2Config(config); err != nil {
		return err
	}
	namespace := SwitchL2NamespaceName(object.ID)
	if err := ensureNamespace(ctx, r.executor, r.ip, namespace); err != nil {
		return err
	}
	args := []string{"-n", namespace, "link", "add", "br0", "type", "bridge"}
	if config.VLANFiltering {
		args = append(args, "vlan_filtering", "1")
	}
	if err := r.executor.Run(ctx, r.ip, args...); err != nil {
		if _, inspectErr := r.executor.Output(ctx, r.ip, "-n", namespace, "link", "show", "br0"); inspectErr != nil {
			return err
		}
	}
	if err := r.executor.Run(ctx, r.ip, "-n", namespace, "link", "set", "br0", "up"); err != nil {
		return err
	}
	for _, port := range config.Ports {
		if !hostObjectName.MatchString(port.Name) {
			return fmt.Errorf("invalid L2 port name")
		}
		if _, err := r.executor.Output(ctx, r.ip, "-n", namespace, "link", "show", port.Name); err != nil {
			continue
		}
		if err := r.executor.Run(ctx, r.ip, "-n", namespace, "link", "set", port.Name, "master", "br0"); err != nil {
			return err
		}
		_ = r.executor.Run(ctx, r.ip, "-n", namespace, "link", "set", port.Name, "up")
		_ = r.executor.Run(ctx, r.ip, "netns", "exec", namespace, r.bridge, "vlan", "del", "dev", port.Name, "vid", "1-4094")
		if port.PVID > 0 {
			_ = r.executor.Run(ctx, r.ip, "netns", "exec", namespace, r.bridge, "vlan", "add", "dev", port.Name, "vid", fmt.Sprint(port.PVID), "pvid", "untagged")
		}
		for _, vlan := range port.Tagged {
			_ = r.executor.Run(ctx, r.ip, "netns", "exec", namespace, r.bridge, "vlan", "add", "dev", port.Name, "vid", fmt.Sprint(vlan))
		}
	}
	return nil
}

func (r *SwitchL2Runtime) Delete(ctx context.Context, id domain.ID) error {
	return deleteNamespace(ctx, r.executor, r.ip, SwitchL2NamespaceName(id))
}
