package linuxnet

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/netlab/netlab/internal/domain"
	"github.com/netlab/netlab/internal/runtime/ownership"
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
	executor            CommandExecutor
	ip                  string
	nsenter             string
	dhclient            string
	helperRoot          string
	routes              managedDockerRouteStore
	addressReadyTimeout time.Duration
	pollInterval        time.Duration
}

type DockerForwardingObservation struct {
	IPv4 bool `json:"forward_ipv4"`
	IPv6 bool `json:"forward_ipv6"`
}

func NewDockerEndpointRuntime(executor CommandExecutor) (*DockerEndpointRuntime, error) {
	if executor != nil {
		return &DockerEndpointRuntime{executor: executor, ip: "ip", nsenter: "nsenter", dhclient: "dhclient", helperRoot: "/run/netlab/docker", routes: &procManagedDockerRouteStore{root: "/proc"}, addressReadyTimeout: 5 * time.Second, pollInterval: 50 * time.Millisecond}, nil
	}
	ip, err := lookup("ip")
	if err != nil {
		return nil, err
	}
	nsenter, err := lookup("nsenter")
	if err != nil {
		return nil, err
	}
	dhclient, err := lookup("dhclient")
	if err != nil {
		return nil, err
	}
	return &DockerEndpointRuntime{executor: SystemExecutor{}, ip: ip, nsenter: nsenter, dhclient: dhclient, helperRoot: "/run/netlab/docker", routes: &procManagedDockerRouteStore{root: "/proc"}, addressReadyTimeout: 30 * time.Second, pollInterval: 250 * time.Millisecond}, nil
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
	if err := r.applyForwarding(ctx, node, pid); err != nil {
		return err
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
			_ = r.executor.Run(ctx, r.ip, "link", "set", host, "alias", ownership.Marker("netlab", string(node.ID)))
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
			currentMAC, inspectErr := r.interfaceMAC(ctx, pid, iface.Name)
			if inspectErr != nil || !strings.EqualFold(currentMAC, iface.MACAddress) {
				if err := r.executor.Run(ctx, r.nsenter, "-t", strconv.Itoa(pid), "-n", r.ip, "link", "set", iface.Name, "address", iface.MACAddress); err != nil {
					r.rollback(ctx, created)
					return err
				}
			}
		}
		if err := r.executor.Run(ctx, r.nsenter, "-t", strconv.Itoa(pid), "-n", r.ip, "link", "set", iface.Name, "up"); err != nil {
			r.rollback(ctx, created)
			return err
		}
		if err := r.configureInterface(ctx, node.ID, pid, iface.Name, dockerInterfaceConfigs(node)[iface.Name]); err != nil {
			r.rollback(ctx, created)
			return err
		}
	}
	return r.executor.Run(ctx, r.nsenter, "-t", strconv.Itoa(pid), "-n", r.ip, "link", "set", "lo", "up")
}

func (r *DockerEndpointRuntime) interfaceMAC(ctx context.Context, pid int, interfaceName string) (string, error) {
	body, err := r.executor.Output(ctx, r.nsenter, "-t", strconv.Itoa(pid), "-n", r.ip, "-j", "link", "show", "dev", interfaceName)
	if err != nil {
		return "", err
	}
	var links []struct {
		Address string `json:"address"`
	}
	if err = json.Unmarshal(body, &links); err != nil {
		return "", err
	}
	if len(links) == 0 || links[0].Address == "" {
		return "", fmt.Errorf("MAC address unavailable for %s", interfaceName)
	}
	return links[0].Address, nil
}

func (r *DockerEndpointRuntime) applyForwarding(ctx context.Context, node domain.Node, pid int) error {
	for _, setting := range []struct {
		key    string
		sysctl string
	}{
		{key: "forward_ipv4", sysctl: "net.ipv4.ip_forward"},
		{key: "forward_ipv6", sysctl: "net.ipv6.conf.all.forwarding"},
	} {
		value, configured := node.Config[setting.key]
		if !configured {
			continue
		}
		enabled, ok := value.(bool)
		if !ok {
			return fmt.Errorf("%s must be boolean", setting.key)
		}
		desired := "0"
		if enabled {
			desired = "1"
		}
		if err := r.executor.Run(ctx, r.nsenter, "-t", strconv.Itoa(pid), "-n", "sysctl", "-w", setting.sysctl+"="+desired); err != nil {
			return fmt.Errorf("apply Docker %s: %w", setting.key, err)
		}
	}
	return nil
}

func (r *DockerEndpointRuntime) InspectForwarding(ctx context.Context, pid int) (DockerForwardingObservation, error) {
	read := func(sysctl string) (bool, error) {
		body, err := r.executor.Output(ctx, r.nsenter, "-t", strconv.Itoa(pid), "-n", "sysctl", "-n", sysctl)
		if err != nil {
			return false, err
		}
		return strings.TrimSpace(string(body)) == "1", nil
	}
	ipv4, err := read("net.ipv4.ip_forward")
	if err != nil {
		return DockerForwardingObservation{}, err
	}
	ipv6, err := read("net.ipv6.conf.all.forwarding")
	if err != nil {
		return DockerForwardingObservation{}, err
	}
	return DockerForwardingObservation{IPv4: ipv4, IPv6: ipv6}, nil
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

func (r *DockerEndpointRuntime) configureInterface(ctx context.Context, ownerID domain.ID, pid int, interfaceName string, config dockerInterfaceConfig) error {
	addressBody, err := r.executor.Output(ctx, r.nsenter, "-t", strconv.Itoa(pid), "-n", r.ip, "-j", "address", "show", "dev", interfaceName)
	if err != nil {
		return fmt.Errorf("inspect addresses for %s: %w", interfaceName, err)
	}
	observedAddresses := dockerInterfaceAddresses(addressBody)
	hasIPv6 := false
	for _, address := range config.Addresses {
		ipAddress, _, err := net.ParseCIDR(address)
		if err != nil {
			return fmt.Errorf("invalid address %q for %s", address, interfaceName)
		}
		if !observedAddresses[address] {
			if err := r.executor.Run(ctx, r.nsenter, "-t", strconv.Itoa(pid), "-n", r.ip, "address", "replace", address, "dev", interfaceName); err != nil {
				return err
			}
		}
		hasIPv6 = hasIPv6 || ipAddress.To4() == nil
	}
	if hasIPv6 {
		if err := r.waitForIPv6AddressReady(ctx, pid, interfaceName); err != nil {
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
		if previousRouteKeys[dockerRouteKey(route)] {
			continue
		}
		if err := r.replaceManagedRoute(ctx, pid, interfaceName, route); err != nil {
			if rollbackErr := r.rollbackManagedRoutes(ctx, pid, interfaceName, appliedRoutes, deletedRoutes); rollbackErr != nil {
				return fmt.Errorf("%w; rollback managed routes: %v", err, rollbackErr)
			}
			return err
		}
		appliedRoutes = append(appliedRoutes, route)
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
		if err := r.startDHCP(ctx, ownerID, pid, interfaceName, "4"); err != nil {
			return err
		}
	} else if err := r.stopDHCP(ctx, ownerID, interfaceName, "4"); err != nil {
		return err
	}
	if config.Modes["dhcpv6"] {
		if err := r.startDHCP(ctx, ownerID, pid, interfaceName, "6"); err != nil {
			return err
		}
	} else if err := r.stopDHCP(ctx, ownerID, interfaceName, "6"); err != nil {
		return err
	}
	return nil
}

func dockerInterfaceAddresses(body []byte) map[string]bool {
	var links []struct {
		AddressInfo []struct {
			Local     string `json:"local"`
			PrefixLen int    `json:"prefixlen"`
		} `json:"addr_info"`
	}
	result := map[string]bool{}
	if json.Unmarshal(body, &links) != nil {
		return result
	}
	for _, link := range links {
		for _, address := range link.AddressInfo {
			if address.Local != "" && address.PrefixLen > 0 {
				result[fmt.Sprintf("%s/%d", address.Local, address.PrefixLen)] = true
			}
		}
	}
	return result
}

func (r *DockerEndpointRuntime) waitForIPv6AddressReady(ctx context.Context, pid int, interfaceName string) error {
	waitCtx, cancel := context.WithTimeout(ctx, r.addressReadyTimeout)
	defer cancel()
	ticker := time.NewTicker(r.pollInterval)
	defer ticker.Stop()
	for {
		body, err := r.executor.Output(waitCtx, r.nsenter, "-t", strconv.Itoa(pid), "-n", r.ip, "-j", "address", "show", "dev", interfaceName)
		if err == nil {
			ready, failed := dockerIPv6AddressState(body)
			if failed {
				return fmt.Errorf("IPv6 duplicate address detection failed for %s", interfaceName)
			}
			if ready {
				return nil
			}
		}
		select {
		case <-waitCtx.Done():
			if errors.Is(waitCtx.Err(), context.DeadlineExceeded) {
				return fmt.Errorf("IPv6 address readiness timed out after %s for %s", r.addressReadyTimeout, interfaceName)
			}
			return fmt.Errorf("IPv6 address readiness cancelled for %s: %w", interfaceName, waitCtx.Err())
		case <-ticker.C:
		}
	}
}

func dockerIPv6AddressState(body []byte) (ready bool, failed bool) {
	var links []struct {
		AddressInfo []struct {
			Family    string `json:"family"`
			Scope     string `json:"scope"`
			Tentative bool   `json:"tentative"`
			DADFailed bool   `json:"dadfailed"`
		} `json:"addr_info"`
	}
	if json.Unmarshal(body, &links) != nil {
		return false, false
	}
	for _, link := range links {
		for _, address := range link.AddressInfo {
			if address.Family != "inet6" || address.Scope == "link" {
				continue
			}
			if address.DADFailed {
				failed = true
			}
			if !address.Tentative && !address.DADFailed {
				ready = true
			}
		}
	}
	return ready, failed
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

func (r *DockerEndpointRuntime) startDHCP(ctx context.Context, ownerID domain.ID, pid int, interfaceName, family string) error {
	unit := dockerDHCPUnit(ownerID, interfaceName, family)
	active, err := r.dhcpHelperActive(ctx, unit)
	if err != nil {
		return fmt.Errorf("inspect Docker DHCPv%s helper for %s: %w", family, interfaceName, err)
	}
	if active {
		body, inspectErr := r.executor.Output(ctx, "systemctl", "show", "--property=ExecStart", "--value", unit)
		if inspectErr != nil {
			return fmt.Errorf("inspect Docker DHCPv%s helper process for %s: %w", family, interfaceName, inspectErr)
		}
		if !strings.Contains(string(body), "-t "+strconv.Itoa(pid)+" -n") {
			if err = r.stopLoadedDHCPHelper(ctx, unit); err != nil {
				return fmt.Errorf("replace stale Docker DHCPv%s helper for %s: %w", family, interfaceName, err)
			}
			active = false
		}
	}
	if active {
		return nil
	}
	if err = r.stopLoadedDHCPHelper(ctx, unit); err != nil {
		return fmt.Errorf("prepare Docker DHCPv%s helper for %s: %w", family, interfaceName, err)
	}
	directory := r.dhcpHelperDirectory(ownerID)
	emptyHooks := filepath.Join(r.helperRoot, "empty-dhclient-hooks")
	if err = os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create Docker DHCP helper state: %w", err)
	}
	if err = os.MkdirAll(emptyHooks, 0o700); err != nil {
		return fmt.Errorf("create Docker DHCP hook isolation directory: %w", err)
	}
	lease := filepath.Join(directory, interfaceName+".v"+family+".leases")
	pidFile := filepath.Join(directory, interfaceName+".v"+family+".pid")
	containerResolver := filepath.Join("/proc", strconv.Itoa(pid), "root", "etc", "resolv.conf")
	hookIsolation := emptyHooks + ":/etc/dhcp/dhclient-enter-hooks.d " + emptyHooks + ":/etc/dhcp/dhclient-exit-hooks.d"
	args := []string{
		"--quiet", "--no-block", "--collect", "--unit=" + unit,
		"--property=BindsTo=netlab.service", "--property=After=netlab.service",
		"--property=Restart=on-failure", "--property=RestartSec=2s", "--property=KillMode=control-group",
		"--property=PrivateMounts=yes", "--property=BindPaths=" + containerResolver + ":/etc/resolv.conf",
		"--property=BindReadOnlyPaths=" + hookIsolation,
		"--setenv=NETLAB_OWNERSHIP=node:" + string(ownerID), "--",
		r.nsenter, "-t", strconv.Itoa(pid), "-n", "--",
		r.dhclient, "-d", "-v", "-" + family, "-lf", lease, "-pf", pidFile, interfaceName,
	}
	if err = r.executor.Run(ctx, "systemd-run", args...); err != nil {
		return fmt.Errorf("start Docker DHCPv%s helper for %s: %w", family, interfaceName, err)
	}
	return nil
}

func (r *DockerEndpointRuntime) Cleanup(ctx context.Context, node domain.Node) error {
	for _, iface := range InterfaceDescriptors(node) {
		for _, family := range []string{"4", "6"} {
			if err := r.stopDHCP(ctx, node.ID, iface.Name, family); err != nil {
				return err
			}
		}
		if err := r.executor.Run(ctx, r.ip, "link", "delete", HostInterfaceName(iface.ID)); err != nil && !strings.Contains(err.Error(), "Cannot find device") {
			return err
		}
	}
	return nil
}

func (r *DockerEndpointRuntime) stopDHCP(ctx context.Context, ownerID domain.ID, interfaceName, family string) error {
	unit := dockerDHCPUnit(ownerID, interfaceName, family)
	active, err := r.dhcpHelperActive(ctx, unit)
	if err != nil {
		return err
	}
	if active {
		if err = r.executor.Run(ctx, "systemctl", "stop", unit); err != nil {
			return fmt.Errorf("stop Docker DHCPv%s helper for %s: %w", family, interfaceName, err)
		}
	}
	directory := r.dhcpHelperDirectory(ownerID)
	for _, suffix := range []string{".leases", ".pid"} {
		path := filepath.Join(directory, interfaceName+".v"+family+suffix)
		if err = os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("clear Docker DHCPv%s helper state for %s: %w", family, interfaceName, err)
		}
	}
	return nil
}

func (r *DockerEndpointRuntime) dhcpHelperActive(ctx context.Context, unit string) (bool, error) {
	body, err := r.executor.Output(ctx, "systemctl", "show", "--property=ActiveState", "--value", unit)
	if err != nil {
		return false, nil
	}
	return strings.TrimSpace(string(body)) == "active", nil
}

func (r *DockerEndpointRuntime) stopLoadedDHCPHelper(ctx context.Context, unit string) error {
	body, err := r.executor.Output(ctx, "systemctl", "show", "--property=LoadState", "--value", unit)
	if err != nil || strings.TrimSpace(string(body)) == "not-found" || strings.TrimSpace(string(body)) == "" {
		return nil
	}
	if err = r.executor.Run(ctx, "systemctl", "stop", unit); err != nil {
		return err
	}
	for attempt := 0; attempt < 20; attempt++ {
		body, err = r.executor.Output(ctx, "systemctl", "show", "--property=LoadState", "--value", unit)
		if err != nil || strings.TrimSpace(string(body)) == "not-found" || strings.TrimSpace(string(body)) == "" {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
	return fmt.Errorf("transient unit %s remained loaded after stop", unit)
}

func dockerDHCPUnit(ownerID domain.ID, interfaceName, family string) string {
	return ownership.Name("netlab-docker-dhcp", ownerID, 0) + "-" + strings.ToLower(interfaceName) + "-v" + family + ".service"
}

func (r *DockerEndpointRuntime) dhcpHelperDirectory(ownerID domain.ID) string {
	return filepath.Join(r.helperRoot, ownership.Name("node", ownerID, 24))
}

func (r *DockerEndpointRuntime) rollback(ctx context.Context, names []string) {
	for _, name := range names {
		_ = r.executor.Run(ctx, r.ip, "link", "delete", name)
	}
}
