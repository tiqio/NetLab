package linuxnet

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
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
	Destination string `json:"destination"`
	Gateway     string `json:"gateway,omitempty"`
	Metric      int    `json:"metric,omitempty"`
}

type managedDockerRouteStore interface {
	Load(int, string) ([]dockerRoute, error)
	Save(int, string, []dockerRoute) error
}

type procManagedDockerRouteStore struct {
	root string
}

type DockerEndpointRuntime struct {
	executor CommandExecutor
	ip       string
	nsenter  string
	routes   managedDockerRouteStore
}

func NewDockerEndpointRuntime(executor CommandExecutor) (*DockerEndpointRuntime, error) {
	if executor != nil {
		return &DockerEndpointRuntime{executor: executor, ip: "ip", nsenter: "nsenter", routes: &procManagedDockerRouteStore{root: "/proc"}}, nil
	}
	ip, err := lookup("ip")
	if err != nil {
		return nil, err
	}
	nsenter, err := lookup("nsenter")
	if err != nil {
		return nil, err
	}
	return &DockerEndpointRuntime{executor: SystemExecutor{}, ip: ip, nsenter: nsenter, routes: &procManagedDockerRouteStore{root: "/proc"}}, nil
}

func (s *procManagedDockerRouteStore) path(pid int, interfaceName string) (string, error) {
	if pid <= 0 || !dockerInterfaceNamePattern.MatchString(interfaceName) {
		return "", fmt.Errorf("invalid managed route owner")
	}
	return filepath.Join(s.root, strconv.Itoa(pid), "root", "run", "netlab", "managed-routes", interfaceName+".json"), nil
}

func (s *procManagedDockerRouteStore) Load(pid int, interfaceName string) ([]dockerRoute, error) {
	path, err := s.path(pid, interfaceName)
	if err != nil {
		return nil, err
	}
	body, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var routes []dockerRoute
	if err = json.Unmarshal(body, &routes); err != nil {
		return nil, fmt.Errorf("decode managed route ownership for %s: %w", interfaceName, err)
	}
	return routes, nil
}

func (s *procManagedDockerRouteStore) Save(pid int, interfaceName string, routes []dockerRoute) error {
	path, err := s.path(pid, interfaceName)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	body, err := json.Marshal(routes)
	if err != nil {
		return err
	}
	temporary := path + ".tmp"
	if err = os.WriteFile(temporary, body, 0o600); err != nil {
		return err
	}
	return os.Rename(temporary, path)
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
		if err := validateDockerRoute(interfaceName, route); err != nil {
			return err
		}
	}
	previousRoutes, err := r.routes.Load(pid, interfaceName)
	if err != nil {
		return fmt.Errorf("load managed routes for %s: %w", interfaceName, err)
	}
	desiredRoutes := make(map[string]bool, len(config.Routes))
	for _, route := range config.Routes {
		desiredRoutes[dockerRouteKey(route)] = true
	}
	deletedRoutes := make([]dockerRoute, 0)
	for _, route := range previousRoutes {
		if desiredRoutes[dockerRouteKey(route)] {
			continue
		}
		if err := r.deleteManagedRoute(ctx, pid, interfaceName, route); err != nil {
			return err
		}
		deletedRoutes = append(deletedRoutes, route)
	}
	previousRouteKeys := make(map[string]bool, len(previousRoutes))
	for _, route := range previousRoutes {
		previousRouteKeys[dockerRouteKey(route)] = true
	}
	appliedRoutes := make([]dockerRoute, 0)
	for _, route := range config.Routes {
		if err := r.replaceManagedRoute(ctx, pid, interfaceName, route); err != nil {
			if rollbackErr := r.rollbackManagedRoutes(ctx, pid, interfaceName, appliedRoutes, deletedRoutes); rollbackErr != nil {
				return fmt.Errorf("%w; rollback managed routes: %v", err, rollbackErr)
			}
			return err
		}
		if !previousRouteKeys[dockerRouteKey(route)] {
			appliedRoutes = append(appliedRoutes, route)
		}
	}
	if err := r.routes.Save(pid, interfaceName, config.Routes); err != nil {
		if rollbackErr := r.rollbackManagedRoutes(ctx, pid, interfaceName, appliedRoutes, deletedRoutes); rollbackErr != nil {
			return fmt.Errorf("persist managed routes for %s: %w; rollback managed routes: %v", interfaceName, err, rollbackErr)
		}
		return fmt.Errorf("persist managed routes for %s: %w", interfaceName, err)
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

func (r *DockerEndpointRuntime) rollbackManagedRoutes(ctx context.Context, pid int, interfaceName string, appliedRoutes, deletedRoutes []dockerRoute) error {
	var failures []string
	for _, route := range appliedRoutes {
		if err := r.deleteManagedRoute(ctx, pid, interfaceName, route); err != nil {
			failures = append(failures, err.Error())
		}
	}
	for _, route := range deletedRoutes {
		if err := r.replaceManagedRoute(ctx, pid, interfaceName, route); err != nil {
			failures = append(failures, err.Error())
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("%s", strings.Join(failures, "; "))
	}
	return nil
}

func validateDockerRoute(interfaceName string, route dockerRoute) error {
	if _, _, err := net.ParseCIDR(route.Destination); err != nil {
		return fmt.Errorf("invalid route destination %q for %s", route.Destination, interfaceName)
	}
	if route.Gateway != "" && net.ParseIP(route.Gateway) == nil {
		return fmt.Errorf("invalid route gateway %q for %s", route.Gateway, interfaceName)
	}
	return nil
}

func dockerRouteKey(route dockerRoute) string {
	return route.Destination + "\x00" + route.Gateway + "\x00" + strconv.Itoa(route.Metric)
}

func (r *DockerEndpointRuntime) replaceManagedRoute(ctx context.Context, pid int, interfaceName string, route dockerRoute) error {
	args := dockerRouteArgs("replace", interfaceName, route)
	if err := r.executor.Run(ctx, r.nsenter, append([]string{"-t", strconv.Itoa(pid), "-n", r.ip}, args...)...); err != nil {
		return fmt.Errorf("apply managed route %s on %s: %w", route.Destination, interfaceName, err)
	}
	return nil
}

func (r *DockerEndpointRuntime) deleteManagedRoute(ctx context.Context, pid int, interfaceName string, route dockerRoute) error {
	args := dockerRouteArgs("delete", interfaceName, route)
	if err := r.executor.Run(ctx, r.nsenter, append([]string{"-t", strconv.Itoa(pid), "-n", r.ip}, args...)...); err != nil && !isMissingManagedRouteError(err) {
		return fmt.Errorf("remove stale managed route %s on %s: %w", route.Destination, interfaceName, err)
	}
	return nil
}

func dockerRouteArgs(operation, interfaceName string, route dockerRoute) []string {
	family := "-4"
	if strings.Contains(route.Destination, ":") {
		family = "-6"
	}
	args := []string{family, "route", operation, route.Destination}
	if route.Gateway != "" {
		args = append(args, "via", route.Gateway)
	}
	args = append(args, "dev", interfaceName)
	if route.Metric > 0 {
		args = append(args, "metric", strconv.Itoa(route.Metric))
	}
	return args
}

func isMissingManagedRouteError(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "no such process") || strings.Contains(message, "not found")
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
