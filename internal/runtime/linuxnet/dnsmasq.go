package linuxnet

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/netlab/netlab/internal/domain"
	"github.com/netlab/netlab/internal/runtime/ownership"
)

type DNSMasqManager struct{ root, dnsmasq, systemdRun, systemctl string }

func NewDNSMasqManager(root string) (*DNSMasqManager, error) {
	dnsmasq, err := exec.LookPath("dnsmasq")
	if err != nil {
		return nil, err
	}
	systemdRun, err := exec.LookPath("systemd-run")
	if err != nil {
		return nil, err
	}
	systemctl, err := exec.LookPath("systemctl")
	if err != nil {
		return nil, err
	}
	return &DNSMasqManager{root: root, dnsmasq: dnsmasq, systemdRun: systemdRun, systemctl: systemctl}, nil
}

func (m *DNSMasqManager) Start(ctx context.Context, object domain.NetworkObject, config domain.NATConfig) (domain.NATServiceObservation, error) {
	observation := m.paths(object.ID)
	if config.DHCPv4 == nil && config.DHCPv6 == nil && !config.RouterAdvertisements {
		observation.State = "not_required"
		observation.ObservedAt = time.Now().UTC()
		return observation, nil
	}
	if err := os.MkdirAll(filepath.Dir(observation.ConfigPath), 0o700); err != nil {
		return observation, err
	}
	var previous domain.NATServiceObservation
	manifestPath := filepath.Join(filepath.Dir(observation.ConfigPath), "observation.json")
	if body, readErr := os.ReadFile(manifestPath); readErr == nil {
		_ = json.Unmarshal(body, &previous)
	}
	body, err := BuildDNSMasqConfig(ownership.Name("nlnat", object.ID, 15), observation.LeasePath, config)
	if err != nil {
		return observation, err
	}
	if err := os.WriteFile(observation.ConfigPath, []byte(body), 0o600); err != nil {
		return observation, err
	}
	observation.ConfigDigest = domain.DigestBytes([]byte(body))
	if previous.ConfigDigest == observation.ConfigDigest && exec.CommandContext(ctx, m.systemctl, "is-active", "--quiet", observation.UnitName).Run() == nil {
		observation.State = "active"
		observation.ObservedAt = time.Now().UTC()
		observation.PID = readPID(filepath.Join(filepath.Dir(observation.ConfigPath), "dnsmasq.pid"))
		manifest, _ := json.Marshal(observation)
		_ = os.WriteFile(manifestPath, manifest, 0o600)
		return observation, nil
	}
	_ = exec.CommandContext(ctx, m.systemctl, "stop", observation.UnitName).Run()
	_ = exec.CommandContext(ctx, m.systemctl, "reset-failed", observation.UnitName).Run()
	args := []string{"--unit=" + strings.TrimSuffix(observation.UnitName, ".service"), "--collect", "--property=Restart=on-failure", "--property=RuntimeDirectory=netlab-dnsmasq", m.dnsmasq, "--keep-in-foreground", "--conf-file=" + observation.ConfigPath, "--pid-file=" + filepath.Join(filepath.Dir(observation.ConfigPath), "dnsmasq.pid")}
	if output, err := exec.CommandContext(ctx, m.systemdRun, args...).CombinedOutput(); err != nil {
		return observation, fmt.Errorf("start dnsmasq: %s: %w", strings.TrimSpace(string(output)), err)
	}
	observation.State = "active"
	observation.ObservedAt = time.Now().UTC()
	observation.PID = readPID(filepath.Join(filepath.Dir(observation.ConfigPath), "dnsmasq.pid"))
	manifest, _ := json.Marshal(observation)
	_ = os.WriteFile(manifestPath, manifest, 0o600)
	return observation, nil
}

func readPID(path string) int {
	body, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	pid, _ := strconv.Atoi(strings.TrimSpace(string(body)))
	return pid
}

func (m *DNSMasqManager) Stop(ctx context.Context, id domain.ID) error {
	observation := m.paths(id)
	_ = exec.CommandContext(ctx, m.systemctl, "stop", observation.UnitName).Run()
	return os.RemoveAll(filepath.Dir(observation.ConfigPath))
}

func (m *DNSMasqManager) paths(id domain.ID) domain.NATServiceObservation {
	directory := filepath.Join(m.root, "runtime", "nat", string(id))
	return domain.NATServiceObservation{NetworkObjectID: id, UnitName: "netlab-dnsmasq-" + ownership.Name("nat", id, 32) + ".service", ConfigPath: filepath.Join(directory, "dnsmasq.conf"), LeasePath: filepath.Join(directory, "dnsmasq.leases")}
}
