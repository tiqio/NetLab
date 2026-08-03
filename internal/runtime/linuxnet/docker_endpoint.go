package linuxnet

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"

	"github.com/netlab/netlab/internal/domain"
)

var dockerInterfaceNamePattern = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,15}$`)

type dockerInterfaceConfig struct {
	Modes     map[string]bool
	Addresses []string
	Routes    []dockerRoute
}

type dockerRoute struct {
	Destination string
	Gateway     string
	Metric      int
}

type DockerEndpointRuntime struct {
	executor CommandExecutor
	ip       string
	nsenter  string
}

func NewDockerEndpointRuntime(executor CommandExecutor) (*DockerEndpointRuntime, error) {
	if executor != nil {
		return &DockerEndpointRuntime{executor: executor, ip: "ip", nsenter: "nsenter"}, nil
	}
	ip, err := lookup("ip")
	if err != nil {
		return nil, err
	}
	nsenter, err := lookup("nsenter")
	if err != nil {
		return nil, err
	}
	return &DockerEndpointRuntime{executor: SystemExecutor{}, ip: ip, nsenter: nsenter}, nil
}

func (r *DockerEndpointRuntime) Ensure(ctx context.Context, node domain.Node, pid int) error {
	if pid <= 0 {
		return fmt.Errorf("container PID is unavailable")
	}
	created := make([]string, 0)
	for _, iface := range InterfaceDescriptors(node) {
		host, peer := HostInterfaceName(iface.ID), PeerInterfaceName(iface.ID)
		if err := r.executor.Run(ctx, r.ip, "link", "show", host); err != nil {
			if err = r.executor.Run(ctx, r.ip, "link", "add", host, "type", "veth", "peer", "name", peer); err != nil {
				r.rollback(ctx, created)
				return err
			}
			created = append(created, host)
			_ = r.executor.Run(ctx, r.ip, "link", "set", host, "alias", "netlab:"+string(node.ID))
			if err = r.executor.Run(ctx, r.ip, "link", "set", peer, "netns", strconv.Itoa(pid)); err != nil {
				r.rollback(ctx, created)
				return err
			}
		}
		_ = r.executor.Run(ctx, r.ip, "link", "set", host, "up")
		if err := r.executor.Run(ctx, r.nsenter, "-t", strconv.Itoa(pid), "-n", r.ip, "link", "set", peer, "name", iface.Name); err != nil && !strings.Contains(err.Error(), "Cannot find device") {
			r.rollback(ctx, created)
			return err
		}
		if iface.MACAddress != "" {
			_ = r.executor.Run(ctx, r.nsenter, "-t", strconv.Itoa(pid), "-n", r.ip, "link", "set", iface.Name, "address", iface.MACAddress)
		}
		if err := r.executor.Run(ctx, r.nsenter, "-t", strconv.Itoa(pid), "-n", r.ip, "link", "set", iface.Name, "up"); err != nil {
			r.rollback(ctx, created)
			return err
		}
		if err := r.configureInterface(ctx, pid, iface.Name, dockerInterfaceConfigs(node)[iface.Name]); err != nil {
			r.rollback(ctx, created)
			return err
		}
	}
	return r.executor.Run(ctx, r.nsenter, "-t", strconv.Itoa(pid), "-n", r.ip, "link", "set", "lo", "up")
}

func dockerInterfaceConfigs(node domain.Node) map[string]dockerInterfaceConfig {
	values, _ := node.Config["network_interfaces"].([]any)
	if direct, ok := node.Config["network_interfaces"].([]map[string]any); ok {
		values = make([]any, len(direct))
		for index := range direct {
			values[index] = direct[index]
		}
	}
	result := make(map[string]dockerInterfaceConfig, len(values))
	for _, raw := range values {
		value, _ := raw.(map[string]any)
		name, _ := value["name"].(string)
		if !dockerInterfaceNamePattern.MatchString(name) {
			continue
		}
		config := dockerInterfaceConfig{Modes: map[string]bool{}}
		for _, rawMode := range anyStrings(value["modes"]) {
			config.Modes[strings.ToLower(rawMode)] = true
		}
		config.Addresses = anyStrings(value["addresses"])
		for _, rawRoute := range anyMaps(value["routes"]) {
			config.Routes = append(config.Routes, dockerRoute{
				Destination: textValue(rawRoute["destination"]),
				Gateway:     textValue(rawRoute["gateway"]),
				Metric:      intValue(rawRoute["metric"]),
			})
		}
		result[name] = config
	}
	return result
}

func anyMaps(value any) []map[string]any {
	var result []map[string]any
	switch values := value.(type) {
	case []map[string]any:
		result = append(result, values...)
	case []any:
		for _, item := range values {
			if mapped, ok := item.(map[string]any); ok {
				result = append(result, mapped)
			}
		}
	}
	return result
}

func textValue(value any) string {
	text, _ := value.(string)
	return text
}

func intValue(value any) int {
	switch number := value.(type) {
	case int:
		return number
	case float64:
		return int(number)
	default:
		return 0
	}
}

func anyStrings(value any) []string {
	var result []string
	switch values := value.(type) {
	case []string:
		result = append(result, values...)
	case []any:
		for _, item := range values {
			if text, ok := item.(string); ok {
				result = append(result, text)
			}
		}
	}
	return result
}

func (r *DockerEndpointRuntime) configureInterface(ctx context.Context, pid int, interfaceName string, config dockerInterfaceConfig) error {
	for _, address := range config.Addresses {
		if _, _, err := net.ParseCIDR(address); err != nil {
			return fmt.Errorf("invalid address %q for %s", address, interfaceName)
		}
		if err := r.executor.Run(ctx, r.nsenter, "-t", strconv.Itoa(pid), "-n", r.ip, "address", "replace", address, "dev", interfaceName); err != nil {
			return err
		}
	}
	for _, route := range config.Routes {
		if _, _, err := net.ParseCIDR(route.Destination); err != nil {
			return fmt.Errorf("invalid route destination %q for %s", route.Destination, interfaceName)
		}
		if route.Gateway != "" && net.ParseIP(route.Gateway) == nil {
			return fmt.Errorf("invalid route gateway %q for %s", route.Gateway, interfaceName)
		}
		args := []string{"route", "replace", route.Destination}
		if route.Gateway != "" {
			args = append(args, "via", route.Gateway)
		}
		args = append(args, "dev", interfaceName)
		if route.Metric > 0 {
			args = append(args, "metric", strconv.Itoa(route.Metric))
		}
		if err := r.executor.Run(ctx, r.nsenter, append([]string{"-t", strconv.Itoa(pid), "-n", r.ip}, args...)...); err != nil {
			return err
		}
	}
	if config.Modes["slaac"] {
		if err := r.executor.Run(ctx, r.nsenter, "-t", strconv.Itoa(pid), "-n", "sysctl", "-w", "net.ipv6.conf."+interfaceName+".accept_ra=2"); err != nil {
			return err
		}
	}
	if config.Modes["dhcpv4"] {
		if err := r.startDHCP(ctx, pid, interfaceName, "4"); err != nil {
			return err
		}
	}
	if config.Modes["dhcpv6"] {
		if err := r.startDHCP(ctx, pid, interfaceName, "6"); err != nil {
			return err
		}
	}
	return nil
}

func (r *DockerEndpointRuntime) startDHCP(ctx context.Context, pid int, interfaceName, family string) error {
	prefix := "/run/netlab-dhclient" + family + "-" + interfaceName
	return r.executor.Run(ctx, r.nsenter, "-t", strconv.Itoa(pid), "-n", "-m", "--", "/sbin/dhclient", "-"+family, "-nw", "-pf", prefix+".pid", "-lf", prefix+".leases", interfaceName)
}

func (r *DockerEndpointRuntime) Cleanup(ctx context.Context, node domain.Node) error {
	for _, iface := range InterfaceDescriptors(node) {
		if err := r.executor.Run(ctx, r.ip, "link", "delete", HostInterfaceName(iface.ID)); err != nil && !strings.Contains(err.Error(), "Cannot find device") {
			return err
		}
	}
	return nil
}

func (r *DockerEndpointRuntime) rollback(ctx context.Context, names []string) {
	for _, name := range names {
		_ = r.executor.Run(ctx, r.ip, "link", "delete", name)
	}
}
