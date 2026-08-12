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

func (r *SwitchL3Runtime) DiagnosticsObject(ctx context.Context, object domain.NetworkObject) (map[string]any, error) {
	var desired domain.SwitchL3Config
	if err := decodeConfig(object.Config, &desired); err != nil {
		return nil, err
	}
	namespace := SwitchL3NamespaceName(object.ID)
	addressesBody, err := r.executor.Output(ctx, r.ip, "-n", namespace, "-j", "address", "show")
	if err != nil {
		return nil, err
	}
	routesBody, err := r.executor.Output(ctx, r.ip, "-n", namespace, "-j", "route", "show", "table", "all")
	if err != nil {
		return nil, err
	}
	forward4, err := r.executor.Output(ctx, r.ip, "netns", "exec", namespace, "sysctl", "-n", "net.ipv4.ip_forward")
	if err != nil {
		return nil, err
	}
	forward6, err := r.executor.Output(ctx, r.ip, "netns", "exec", namespace, "sysctl", "-n", "net.ipv6.conf.all.forwarding")
	if err != nil {
		return nil, err
	}
	observedInterfaces := observedL3Interfaces(addressesBody)
	observedRoutes := observedL3Routes(routesBody)
	observedForward4 := strings.TrimSpace(string(forward4)) == "1"
	observedForward6 := strings.TrimSpace(string(forward6)) == "1"
	mismatches := make([]string, 0)
	if desired.ForwardIPv4 != observedForward4 {
		mismatches = append(mismatches, fmt.Sprintf("forward_ipv4 desired=%t observed=%t", desired.ForwardIPv4, observedForward4))
	}
	if desired.ForwardIPv6 != observedForward6 {
		mismatches = append(mismatches, fmt.Sprintf("forward_ipv6 desired=%t observed=%t", desired.ForwardIPv6, observedForward6))
	}
	for _, iface := range desired.Interfaces {
		if strings.Join(iface.Addresses, ",") != strings.Join(observedInterfaces[iface.Name], ",") {
			mismatches = append(mismatches, fmt.Sprintf("interface %s addresses desired=%v observed=%v", iface.Name, iface.Addresses, observedInterfaces[iface.Name]))
		}
	}
	desiredRoutes := routeKeys(desired.Routes)
	if !containsAllRouteKeys(observedRoutes, desiredRoutes) {
		mismatches = append(mismatches, fmt.Sprintf("routes desired=%v observed=%v", desiredRoutes, observedRoutes))
	}
	return map[string]any{
		"desired":    map[string]any{"forward_ipv4": desired.ForwardIPv4, "forward_ipv6": desired.ForwardIPv6, "interfaces": desired.Interfaces, "routes": desired.Routes},
		"observed":   map[string]any{"forward_ipv4": observedForward4, "forward_ipv6": observedForward6, "interfaces": observedInterfaces, "routes": observedRoutes},
		"mismatches": mismatches,
	}, nil
}

func (r *SwitchL3Runtime) ConfigurationConverged(ctx context.Context, object domain.NetworkObject) (bool, map[string]any, error) {
	diagnostics, err := r.DiagnosticsObject(ctx, object)
	if err != nil {
		return false, nil, err
	}
	mismatches, _ := diagnostics["mismatches"].([]string)
	return len(mismatches) == 0, diagnostics, nil
}

func observedL3Interfaces(body []byte) map[string][]string {
	var links []struct {
		Name      string `json:"ifname"`
		Addresses []struct {
			Local     string `json:"local"`
			PrefixLen int    `json:"prefixlen"`
			Scope     string `json:"scope"`
		} `json:"addr_info"`
	}
	_ = json.Unmarshal(body, &links)
	result := make(map[string][]string, len(links))
	for _, link := range links {
		for _, address := range link.Addresses {
			if address.Scope == "link" || address.Local == "" {
				continue
			}
			result[link.Name] = append(result[link.Name], fmt.Sprintf("%s/%d", address.Local, address.PrefixLen))
		}
	}
	return result
}

func observedL3Routes(body []byte) []string {
	var routes []struct {
		Destination string `json:"dst"`
		Gateway     string `json:"gateway"`
		Metric      int    `json:"metric"`
	}
	_ = json.Unmarshal(body, &routes)
	values := make([]domain.RouteConfig, 0, len(routes))
	for _, route := range routes {
		if route.Destination == "" || route.Destination == "default" {
			continue
		}
		values = append(values, domain.RouteConfig{Destination: route.Destination, Gateway: route.Gateway, Metric: route.Metric})
	}
	return routeKeys(values)
}

func routeKeys(routes []domain.RouteConfig) []string {
	result := make([]string, 0, len(routes))
	for _, route := range routes {
		result = append(result, fmt.Sprintf("%s|%s|%d", route.Destination, route.Gateway, route.Metric))
	}
	return result
}

func containsAllRouteKeys(observed, desired []string) bool {
	values := make(map[string]struct{}, len(observed))
	for _, route := range observed {
		values[route] = struct{}{}
	}
	for _, route := range desired {
		if _, ok := values[route]; !ok {
			return false
		}
	}
	return true
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
	availableInterfaces := make(map[string]bool, len(config.Interfaces))
	for _, iface := range config.Interfaces {
		if !hostObjectName.MatchString(iface.Name) {
			return fmt.Errorf("invalid L3 interface")
		}
		if _, err := r.executor.Output(ctx, r.ip, "-n", namespace, "link", "show", iface.Name); err != nil {
			continue
		}
		availableInterfaces[iface.Name] = true
	}
	allInterfacesAvailable := len(availableInterfaces) == len(config.Interfaces)
	if allInterfacesAvailable {
		if err := r.executor.Run(ctx, r.ip, "-n", namespace, "route", "flush", "table", "main"); err != nil && !missingFIBTable(err) {
			return err
		}
		if err := r.executor.Run(ctx, r.ip, "-n", namespace, "-6", "route", "flush", "table", "main"); err != nil && !missingFIBTable(err) {
			return err
		}
	}
	for _, iface := range config.Interfaces {
		if !availableInterfaces[iface.Name] {
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
	if allInterfacesAvailable {
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
