package linuxnet

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/netlab/netlab/internal/domain"
	"github.com/netlab/netlab/internal/runtime/ownership"
)

type PCRuntime struct {
	executor           CommandExecutor
	ip                 string
	nsenter            string
	mountNamespacePID  int
	resolvRoot         string
	helperRoot         string
	acquisitionTimeout time.Duration
	pollInterval       time.Duration
	now                func() time.Time
}

func (r *PCRuntime) InspectNetworkObject(ctx context.Context, object domain.NetworkObject) (domain.RuntimeBackingObservation, error) {
	return inspectNamespaceBacking(ctx, r.executor, r.ip, ownership.Name("nlpc", object.ID, 15)), nil
}

type PCDiagnostics struct {
	Addresses    json.RawMessage   `json:"addresses"`
	Routes       json.RawMessage   `json:"routes"`
	DHCPv4Lease  string            `json:"dhcpv4_lease,omitempty"`
	DHCPv6Lease  string            `json:"dhcpv6_lease,omitempty"`
	SLAAC        map[string]string `json:"slaac"`
	DHCPv4Status string            `json:"dhcpv4_status"`
	DHCPv6Status string            `json:"dhcpv6_status"`
	DNS          []string          `json:"dns"`
	Helpers      map[string]string `json:"helpers,omitempty"`
}

func NewPCRuntime(executor CommandExecutor) (*PCRuntime, error) {
	if executor != nil {
		return &PCRuntime{executor: executor, ip: "ip", nsenter: "nsenter", mountNamespacePID: os.Getpid(), resolvRoot: "/etc/netns", helperRoot: "/run/netlab/pc", acquisitionTimeout: 30 * time.Second, pollInterval: 250 * time.Millisecond, now: time.Now}, nil
	}
	ip, err := exec.LookPath("ip")
	if err != nil {
		return nil, err
	}
	nsenter, err := exec.LookPath("nsenter")
	if err != nil {
		return nil, err
	}
	return &PCRuntime{executor: SystemExecutor{}, ip: ip, nsenter: nsenter, mountNamespacePID: os.Getpid(), resolvRoot: "/etc/netns", helperRoot: "/run/netlab/pc", acquisitionTimeout: 30 * time.Second, pollInterval: 250 * time.Millisecond, now: time.Now}, nil
}

func (r *PCRuntime) Configure(ctx context.Context, object domain.NetworkObject) error {
	var config domain.PCConfig
	if err := decodeConfig(object.Config, &config); err != nil {
		return err
	}
	if err := domain.ValidatePCConfig(config); err != nil {
		return err
	}
	namespace := ownership.Name("nlpc", object.ID, 15)
	if err := ensureNamespace(ctx, r.executor, r.ip, namespace); err != nil {
		return err
	}
	if err := r.executor.Run(ctx, r.ip, "-n", namespace, "link", "set", "lo", "up"); err != nil {
		return err
	}
	dnsServers := make([]string, 0)
	for _, iface := range config.Interfaces {
		if !hostObjectName.MatchString(iface.Name) {
			return fmt.Errorf("invalid PC interface name")
		}
		if _, err := r.executor.Output(ctx, r.ip, "-n", namespace, "link", "show", iface.Name); err != nil {
			continue
		}
		if err := r.configureInterface(ctx, namespace, object.ID, iface); err != nil {
			return err
		}
		dnsServers = append(dnsServers, iface.DNS...)
	}
	return r.configureDNS(namespace, dnsServers)
}

func (r *PCRuntime) configureInterface(ctx context.Context, namespace string, ownerID domain.ID, iface domain.PCInterfaceConfig) error {
	if err := r.executor.Run(ctx, r.ip, "-n", namespace, "link", "set", iface.Name, "up"); err != nil {
		return err
	}
	changed, signature, err := r.interfaceConfigChanged(ownerID, iface)
	if err != nil {
		return err
	}
	if changed {
		if err = r.executor.Run(ctx, r.ip, "-n", namespace, "address", "flush", "dev", iface.Name, "scope", "global"); err != nil {
			return err
		}
		if err = r.executor.Run(ctx, r.ip, "-n", namespace, "route", "flush", "dev", iface.Name); err != nil {
			return err
		}
	}
	modes := map[domain.AddressMode]bool{}
	for _, mode := range iface.Modes {
		modes[mode] = true
	}
	for family, mode := range map[string]domain.AddressMode{"4": domain.AddressDHCPv4, "6": domain.AddressDHCPv6} {
		if !modes[mode] {
			if err := r.stopDHCPHelper(ctx, ownerID, iface.Name, family); err != nil {
				return err
			}
		}
	}
	if !modes[domain.AddressSLAAC] {
		if err := r.executor.Run(ctx, r.ip, "netns", "exec", namespace, "sysctl", "-w", "net.ipv6.conf."+iface.Name+".accept_ra=0"); err != nil {
			return err
		}
		_ = os.Remove(filepath.Join(r.pcHelperDirectory(ownerID), iface.Name+".slaac-started"))
	}
	for _, address := range iface.Addresses {
		if err := r.executor.Run(ctx, r.ip, "-n", namespace, "address", "replace", address, "dev", iface.Name); err != nil {
			return err
		}
	}
	for _, route := range iface.Routes {
		args := []string{"-n", namespace, "route", "replace", route.Destination}
		if route.Gateway != "" {
			args = append(args, "via", route.Gateway)
		}
		args = append(args, "dev", iface.Name)
		if route.Metric > 0 {
			args = append(args, "metric", fmt.Sprint(route.Metric))
		}
		if err := r.executor.Run(ctx, r.ip, args...); err != nil {
			return err
		}
	}
	for _, mode := range iface.Modes {
		switch mode {
		case domain.AddressDHCPv4:
			if err := r.ensureDHCPHelper(ctx, namespace, ownerID, iface.Name, "4"); err != nil {
				return err
			}
		case domain.AddressDHCPv6:
			if err := r.ensureDHCPHelper(ctx, namespace, ownerID, iface.Name, "6"); err != nil {
				return err
			}
		case domain.AddressSLAAC:
			if err := r.executor.Run(ctx, r.ip, "netns", "exec", namespace, "sysctl", "-w", "net.ipv6.conf."+iface.Name+".accept_ra=2"); err != nil {
				return err
			}
			if err := r.writeSLAACMarker(ownerID, iface.Name); err != nil {
				return err
			}
		}
	}
	if changed {
		if err = os.MkdirAll(r.pcHelperDirectory(ownerID), 0o755); err != nil {
			return fmt.Errorf("create PC interface state: %w", err)
		}
		if err = os.WriteFile(r.interfaceSignaturePath(ownerID, iface.Name), []byte(signature), 0o644); err != nil {
			return fmt.Errorf("persist PC interface configuration: %w", err)
		}
	}
	return nil
}

func (r *PCRuntime) interfaceConfigChanged(ownerID domain.ID, iface domain.PCInterfaceConfig) (bool, string, error) {
	body, err := json.Marshal(iface)
	if err != nil {
		return false, "", err
	}
	signature := fmt.Sprintf("%x", sha256.Sum256(body))
	current, err := os.ReadFile(r.interfaceSignaturePath(ownerID, iface.Name))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, "", fmt.Errorf("read PC interface configuration: %w", err)
	}
	return strings.TrimSpace(string(current)) != signature, signature, nil
}

func (r *PCRuntime) interfaceSignaturePath(ownerID domain.ID, interfaceName string) string {
	return filepath.Join(r.pcHelperDirectory(ownerID), interfaceName+".config.sha256")
}

func (r *PCRuntime) configureDNS(namespace string, servers []string) error {
	directory := filepath.Join(r.resolvRoot, namespace)
	path := filepath.Join(directory, "resolv.conf")
	if len(servers) == 0 {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("clear DNS: %w", err)
		}
		return nil
	}
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("configure DNS: %w", err)
	}
	seen := map[string]bool{}
	lines := make([]string, 0, len(servers))
	for _, server := range servers {
		server = strings.TrimSpace(server)
		if server != "" && !seen[server] {
			seen[server] = true
			lines = append(lines, "nameserver "+server)
		}
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		return fmt.Errorf("configure DNS: %w", err)
	}
	return nil
}

func (r *PCRuntime) stopDHCPHelper(ctx context.Context, ownerID domain.ID, interfaceName, family string) error {
	unit := pcDHCPUnit(ownerID, interfaceName, family)
	active, err := r.helperActive(ctx, unit)
	if err != nil {
		return fmt.Errorf("inspect DHCPv%s helper for %s: %w", family, interfaceName, err)
	}
	if active {
		if err = r.executor.Run(ctx, "systemctl", "stop", unit); err != nil {
			return fmt.Errorf("stop DHCPv%s helper for %s: %w", family, interfaceName, err)
		}
	}
	lease := filepath.Join(r.pcHelperDirectory(ownerID), interfaceName+".v"+family+".leases")
	if err = os.Remove(lease); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("clear DHCPv%s lease for %s: %w", family, interfaceName, err)
	}
	return nil
}

func (r *PCRuntime) ensureDHCPHelper(ctx context.Context, namespace string, ownerID domain.ID, interfaceName, family string) error {
	unit := pcDHCPUnit(ownerID, interfaceName, family)
	active, err := r.helperActive(ctx, unit)
	if err != nil {
		return fmt.Errorf("inspect DHCPv%s helper for %s: %w", family, interfaceName, err)
	}
	if active {
		current, inspectErr := r.helperUsesCurrentMountNamespace(ctx, unit)
		if inspectErr != nil {
			return fmt.Errorf("inspect DHCPv%s helper generation for %s: %w", family, interfaceName, inspectErr)
		}
		if !current {
			if err = r.stopLoadedDHCPHelper(ctx, unit); err != nil {
				return fmt.Errorf("replace stale DHCPv%s helper for %s: %w", family, interfaceName, err)
			}
			active = false
		}
	}
	started := false
	if !active {
		if err = r.stopLoadedDHCPHelper(ctx, unit); err != nil {
			return fmt.Errorf("prepare DHCPv%s helper for %s: %w", family, interfaceName, err)
		}
		directory := r.pcHelperDirectory(ownerID)
		if err = os.MkdirAll(directory, 0o755); err != nil {
			return fmt.Errorf("create DHCP helper state: %w", err)
		}
		lease := filepath.Join(directory, interfaceName+".v"+family+".leases")
		pid := filepath.Join(directory, interfaceName+".v"+family+".pid")
		args := []string{
			"--quiet", "--no-block", "--collect", "--unit=" + unit,
			"--property=BindsTo=netlab.service", "--property=After=netlab.service",
			"--property=Restart=on-failure", "--property=RestartSec=2s", "--property=KillMode=control-group",
			"--setenv=NETLAB_OWNERSHIP=network_object:" + string(ownerID), "--",
			r.nsenter, "--mount=/proc/" + fmt.Sprint(r.mountNamespacePID) + "/ns/mnt", "--",
			r.ip, "netns", "exec", namespace, "dhclient", "-d", "-v", "-" + family, "-lf", lease, "-pf", pid, interfaceName,
		}
		if err = r.executor.Run(ctx, "systemd-run", args...); err != nil {
			return fmt.Errorf("start DHCPv%s helper for %s: %w", family, interfaceName, err)
		}
		started = true
	}
	if err = r.waitForDynamicAddress(ctx, namespace, interfaceName, family); err != nil {
		if started {
			_ = r.executor.Run(context.Background(), "systemctl", "stop", unit)
		}
		return err
	}
	return nil
}

func (r *PCRuntime) helperUsesCurrentMountNamespace(ctx context.Context, unit string) (bool, error) {
	body, err := r.executor.Output(ctx, "systemctl", "show", "--property=ExecStart", "--value", unit)
	if err != nil {
		return false, err
	}
	expected := "--mount=/proc/" + fmt.Sprint(r.mountNamespacePID) + "/ns/mnt"
	return strings.Contains(string(body), expected), nil
}

func (r *PCRuntime) stopLoadedDHCPHelper(ctx context.Context, unit string) error {
	body, err := r.executor.Output(ctx, "systemctl", "show", "--property=LoadState", "--value", unit)
	if err != nil || strings.TrimSpace(string(body)) == "not-found" {
		return nil
	}
	if err = r.executor.Run(ctx, "systemctl", "stop", unit); err != nil {
		return err
	}
	for attempt := 0; attempt < 20; attempt++ {
		body, err = r.executor.Output(ctx, "systemctl", "show", "--property=LoadState", "--value", unit)
		if err != nil || strings.TrimSpace(string(body)) == "not-found" {
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

func (r *PCRuntime) helperActive(ctx context.Context, unit string) (bool, error) {
	body, err := r.executor.Output(ctx, "systemctl", "show", "--property=ActiveState", "--value", unit)
	if err != nil {
		return false, nil
	}
	return strings.TrimSpace(string(body)) == "active", nil
}

func (r *PCRuntime) waitForDynamicAddress(ctx context.Context, namespace, interfaceName, family string) error {
	waitCtx, cancel := context.WithTimeout(ctx, r.acquisitionTimeout)
	defer cancel()
	ticker := time.NewTicker(r.pollInterval)
	defer ticker.Stop()
	for {
		body, err := r.executor.Output(waitCtx, r.ip, "-n", namespace, "-j", "address", "show", "dev", interfaceName)
		if err == nil && dynamicAddressAcquired(body, family) {
			return nil
		}
		select {
		case <-waitCtx.Done():
			if errors.Is(waitCtx.Err(), context.DeadlineExceeded) {
				return fmt.Errorf("DHCPv%s timed out after %s for %s", family, r.acquisitionTimeout, interfaceName)
			}
			return fmt.Errorf("DHCPv%s cancelled for %s: %w", family, interfaceName, waitCtx.Err())
		case <-ticker.C:
		}
	}
}

func dynamicAddressAcquired(body []byte, family string) bool {
	var links []struct {
		AddressInfo []struct {
			Family    string `json:"family"`
			Scope     string `json:"scope"`
			Dynamic   bool   `json:"dynamic"`
			Tentative bool   `json:"tentative"`
		} `json:"addr_info"`
	}
	if json.Unmarshal(body, &links) != nil {
		return false
	}
	wanted := "inet"
	if family == "6" {
		wanted = "inet6"
	}
	for _, link := range links {
		for _, address := range link.AddressInfo {
			if address.Family == wanted && address.Scope != "link" && !address.Tentative && (address.Dynamic || family == "6") {
				return true
			}
		}
	}
	return false
}

func pcDHCPUnit(ownerID domain.ID, interfaceName, family string) string {
	return ownership.Name("netlab-pc-dhcp", ownerID, 0) + "-" + strings.ToLower(interfaceName) + "-v" + family + ".service"
}

func (r *PCRuntime) pcHelperDirectory(id domain.ID) string {
	return filepath.Join(r.helperRoot, ownership.Name("pc", id, 24))
}

func (r *PCRuntime) writeSLAACMarker(id domain.ID, interfaceName string) error {
	directory := r.pcHelperDirectory(id)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("create SLAAC state: %w", err)
	}
	return os.WriteFile(filepath.Join(directory, interfaceName+".slaac-started"), []byte(r.now().UTC().Format(time.RFC3339Nano)), 0o644)
}

func (r *PCRuntime) Diagnostics(ctx context.Context, id domain.ID) (PCDiagnostics, error) {
	namespace := ownership.Name("nlpc", id, 15)
	addresses, err := r.executor.Output(ctx, r.ip, "-n", namespace, "-j", "address", "show")
	if err != nil {
		return PCDiagnostics{}, err
	}
	routes, err := r.executor.Output(ctx, r.ip, "-n", namespace, "-j", "route", "show", "table", "all")
	if err != nil {
		return PCDiagnostics{}, err
	}
	diagnostics := PCDiagnostics{Addresses: json.RawMessage(strings.TrimSpace(string(addresses))), Routes: json.RawMessage(strings.TrimSpace(string(routes))), SLAAC: map[string]string{"address_status": slaacStatus(addresses), "route_status": slaacRouteStatus(routes)}, DHCPv4Status: "no_lease", DHCPv6Status: "no_lease", DNS: []string{}, Helpers: map[string]string{}}
	entries, _ := os.ReadDir(r.pcHelperDirectory(id))
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasSuffix(name, ".v4.leases") {
			body, _ := os.ReadFile(filepath.Join(r.pcHelperDirectory(id), name))
			diagnostics.DHCPv4Lease = strings.TrimSpace(string(body))
			if diagnostics.DHCPv4Lease != "" {
				diagnostics.DHCPv4Status = "acquired"
			}
		}
		if strings.HasSuffix(name, ".v6.leases") {
			body, _ := os.ReadFile(filepath.Join(r.pcHelperDirectory(id), name))
			diagnostics.DHCPv6Lease = strings.TrimSpace(string(body))
			if diagnostics.DHCPv6Lease != "" {
				diagnostics.DHCPv6Status = "acquired"
			}
		}
		if strings.HasSuffix(name, ".slaac-started") && diagnostics.SLAAC["address_status"] != "acquired" {
			body, _ := os.ReadFile(filepath.Join(r.pcHelperDirectory(id), name))
			if started, parseErr := time.Parse(time.RFC3339Nano, strings.TrimSpace(string(body))); parseErr == nil && r.now().Sub(started) >= r.acquisitionTimeout {
				diagnostics.SLAAC["address_status"] = "timeout"
				diagnostics.SLAAC["route_status"] = "timeout"
			}
		}
	}
	if body, readErr := os.ReadFile(filepath.Join(r.resolvRoot, namespace, "resolv.conf")); readErr == nil {
		for _, line := range strings.Split(string(body), "\n") {
			if fields := strings.Fields(line); len(fields) == 2 && fields[0] == "nameserver" {
				diagnostics.DNS = append(diagnostics.DNS, fields[1])
			}
		}
	}
	return diagnostics, nil
}

func slaacStatus(addresses []byte) string {
	if strings.Contains(string(addresses), `"family":"inet6"`) && !strings.Contains(string(addresses), `"scope":"link"`) {
		return "acquired"
	}
	return "waiting"
}
func slaacRouteStatus(routes []byte) string {
	if strings.Contains(string(routes), `"protocol":"ra"`) {
		return "acquired"
	}
	return "waiting"
}

func (r *PCRuntime) Delete(ctx context.Context, id domain.ID) error {
	prefix := ownership.Name("netlab-pc-dhcp", id, 0)
	if body, err := r.executor.Output(ctx, "systemctl", "list-units", "--all", "--plain", "--no-legend", prefix+"*.service"); err == nil {
		for _, line := range strings.Split(string(body), "\n") {
			unit := strings.Fields(line)
			if len(unit) > 0 && strings.HasPrefix(unit[0], prefix) && strings.HasSuffix(unit[0], ".service") {
				if stopErr := r.executor.Run(ctx, "systemctl", "stop", unit[0]); stopErr != nil {
					return stopErr
				}
			}
		}
	}
	if err := deleteNamespace(ctx, r.executor, r.ip, ownership.Name("nlpc", id, 15)); err != nil {
		return err
	}
	return os.RemoveAll(r.pcHelperDirectory(id))
}

func decodeConfig(input map[string]any, output any) error {
	body, err := json.Marshal(input)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, output)
}
