package linuxnet

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

var hostObjectName = regexp.MustCompile(`^[a-zA-Z0-9_.:-]{1,15}$`)

type Runner interface {
	Run(context.Context, string, ...string) error
}

type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, name string, args ...string) error {
	output, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

type LinkRuntime struct {
	ip     string
	runner Runner
}

func NewLinkRuntime(runner Runner) (*LinkRuntime, error) {
	if runner != nil {
		return &LinkRuntime{ip: "ip", runner: runner}, nil
	}
	ip, err := exec.LookPath("ip")
	if err != nil {
		return nil, err
	}
	return &LinkRuntime{ip: ip, runner: ExecRunner{}}, nil
}

func (r *LinkRuntime) CreateTap(ctx context.Context, name, owner string) error {
	if !hostObjectName.MatchString(name) || owner == "" {
		return fmt.Errorf("invalid tap name or owner")
	}
	if err := r.runner.Run(ctx, r.ip, "tuntap", "add", "dev", name, "mode", "tap"); err != nil {
		return err
	}
	if err := r.runner.Run(ctx, r.ip, "link", "set", "dev", name, "alias", "netlab:"+owner); err != nil {
		_ = r.runner.Run(ctx, r.ip, "link", "delete", "dev", name)
		return err
	}
	return r.runner.Run(ctx, r.ip, "link", "set", "dev", name, "up")
}

func (r *LinkRuntime) CreateVeth(ctx context.Context, host, peer, owner string) error {
	if !hostObjectName.MatchString(host) || !hostObjectName.MatchString(peer) || owner == "" {
		return fmt.Errorf("invalid veth name or owner")
	}
	if err := r.runner.Run(ctx, r.ip, "link", "add", host, "type", "veth", "peer", "name", peer); err != nil {
		return err
	}
	if err := r.runner.Run(ctx, r.ip, "link", "set", "dev", host, "alias", "netlab:"+owner); err != nil {
		_ = r.runner.Run(ctx, r.ip, "link", "delete", "dev", host)
		return err
	}
	return nil
}

func (r *LinkRuntime) MoveToBridge(ctx context.Context, interfaceName, fromBridge, toBridge string) error {
	if !hostObjectName.MatchString(interfaceName) || !hostObjectName.MatchString(toBridge) {
		return fmt.Errorf("invalid interface or bridge name")
	}
	if err := r.runner.Run(ctx, r.ip, "link", "set", "dev", interfaceName, "nomaster"); err != nil {
		return err
	}
	if err := r.runner.Run(ctx, r.ip, "link", "set", "dev", interfaceName, "master", toBridge); err != nil {
		if fromBridge != "" && hostObjectName.MatchString(fromBridge) {
			_ = r.runner.Run(ctx, r.ip, "link", "set", "dev", interfaceName, "master", fromBridge)
		}
		return fmt.Errorf("move %s to %s: %w", interfaceName, toBridge, err)
	}
	return nil
}

func (r *LinkRuntime) Delete(ctx context.Context, name string) error {
	if !hostObjectName.MatchString(name) {
		return fmt.Errorf("invalid interface name")
	}
	err := r.runner.Run(ctx, r.ip, "link", "delete", "dev", name)
	if err != nil && (strings.Contains(err.Error(), "Cannot find device") || strings.Contains(err.Error(), "does not exist")) {
		return nil
	}
	return err
}
