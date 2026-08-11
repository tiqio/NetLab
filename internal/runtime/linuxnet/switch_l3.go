package linuxnet

import (
	"context"
	"encoding/json"
	"fmt"
	"net/netip"
	"os/exec"
	"strconv"
	"strings"

	"github.com/netlab/netlab/internal/domain"
)

type SwitchL3Runtime struct {
	executor CommandExecutor
	ip       string
}

func (r *SwitchL3Runtime) InspectNetworkObject(ctx context.Context, object domain.NetworkObject) (domain.RuntimeBackingObservation, error) {
	return inspectNamespaceBacking(ctx, r.executor, r.ip, SwitchL3NamespaceName(object.ID)), nil
}

func (r *SwitchL3Runtime) Diagnostics(ctx context.Context, id domain.ID) (map[string]any, error) {
	namespace := SwitchL3NamespaceName(id)
	routes, err := r.executor.Output(ctx, r.ip, "-n", namespace, "-j", "route", "show", "table", "all")
	if err != nil {
		return nil, err
	}
	forward4, _ := r.executor.Output(ctx, r.ip, "netns", "exec", namespace, "sysctl", "-n", "net.ipv4.ip_forward")
	forward6, _ := r.executor.Output(ctx, r.ip, "netns", "exec", namespace, "sysctl", "-n", "net.ipv6.conf.all.forwarding")
	return map[string]any{"routes": json.RawMessage(strings.TrimSpace(string(routes))), "forward_ipv4": strings.TrimSpace(string(forward4)) == "1", "forward_ipv6": strings.TrimSpace(string(forward6)) == "1"}, nil
}

func NewSwitchL3Runtime(executor CommandExecutor) (*SwitchL3Runtime, error) {
	if executor != nil {
		return &SwitchL3Runtime{executor: executor, ip: "ip"}, nil
	}
	ip, err := exec.LookPath("ip")
	if err != nil {
		return nil, err
	}
	return &SwitchL3Runtime{executor: SystemExecutor{}, ip: ip}, nil
}
func (r *SwitchL3Runtime) Configure(ctx context.Context, object domain.NetworkObject) error {
	var config domain.SwitchL3Config
	if err := decodeConfig(object.Config, &config); err != nil {
		return err
	}
	if err := domain.ValidateSwitchL3Config(config); err != nil {
		return err
	}
	namespace := SwitchL3NamespaceName(object.ID)
	if err := ensureNamespace(ctx, r.executor, r.ip, namespace); err != nil {
		return err
	}
	if err := r.executor.Run(ctx, r.ip, "-n", namespace, "route", "flush", "table", "main"); err != nil && !missingFIBTable(err) {
		return err
	}
	if err := r.executor.Run(ctx, r.ip, "-n", namespace, "-6", "route", "flush", "table", "main"); err != nil && !missingFIBTable(err) {
		return err
	}
	for _, iface := range config.Interfaces {
		if !hostObjectName.MatchString(iface.Name) {
			return fmt.Errorf("invalid L3 interface")
		}
		if _, err := r.executor.Output(ctx, r.ip, "-n", namespace, "link", "show", iface.Name); err != nil {
			continue
		}
		_ = r.executor.Run(ctx, r.ip, "-n", namespace, "link", "set", iface.Name, "up")
		if err := r.executor.Run(ctx, r.ip, "-n", namespace, "address", "flush", "dev", iface.Name, "scope", "global"); err != nil {
			return err
		}
		for _, address := range iface.Addresses {
			if err := r.executor.Run(ctx, r.ip, "-n", namespace, "address", "replace", address, "dev", iface.Name); err != nil {
				return err
			}
		}
	}
	for _, route := range config.Routes {
		args := []string{"-n", namespace, "route", "replace", route.Destination}
		prefix, _ := netip.ParsePrefix(route.Destination)
		if prefix.Addr().Is6() {
			args = []string{"-n", namespace, "-6", "route", "replace", route.Destination}
		}
		if route.Gateway != "" {
			args = append(args, "via", route.Gateway)
		}
		if route.Metric > 0 {
			args = append(args, "metric", strconv.Itoa(route.Metric))
		}
		if err := r.executor.Run(ctx, r.ip, args...); err != nil {
			if retryErr := r.executor.Run(ctx, r.ip, args...); retryErr != nil {
				return retryErr
			}
		}
	}
	forwardIPv4 := "0"
	if config.ForwardIPv4 {
		forwardIPv4 = "1"
	}
	forwardIPv6 := "0"
	if config.ForwardIPv6 {
		forwardIPv6 = "1"
	}
	_ = r.executor.Run(ctx, r.ip, "netns", "exec", namespace, "sysctl", "-w", "net.ipv4.ip_forward="+forwardIPv4)
	_ = r.executor.Run(ctx, r.ip, "netns", "exec", namespace, "sysctl", "-w", "net.ipv6.conf.all.forwarding="+forwardIPv6)
	return nil
}

func missingFIBTable(err error) bool {
	return err != nil && strings.Contains(err.Error(), "FIB table does not exist")
}
func (r *SwitchL3Runtime) Delete(ctx context.Context, id domain.ID) error {
	return deleteNamespace(ctx, r.executor, r.ip, SwitchL3NamespaceName(id))
}
