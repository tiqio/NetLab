package qemu

import (
	"context"
	"github.com/netlab/netlab/internal/domain"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBuildArgsAndOwnership(t *testing.T) {
	adapter := &Adapter{Root: "/var/lib/netlab", Acceleration: "kvm"}
	node := domain.Node{ID: "node", CPUCount: 2, MemoryMiB: 1024, InterfaceLimit: 8, Config: map[string]any{"console_modes": []string{"telnet", "vnc"}, "interfaces": []any{map[string]any{"id": "iface-0", "slot": 0, "name": "eth0", "driver": "virtio-net-pci", "mac_address": "02:00:00:00:00:01"}}}}
	args, manifest := adapter.BuildArgs(node, "/images/base.qcow2")
	joined := strings.Join(args, " ")
	for _, expected := range []string{"-smp 2", "-m 1024", "netlab:node", "qmp.sock", "qga.sock", "-device VGA,bus=pcie.0,addr=2", "vnc.sock", "base.qcow2", "pcie-root-port,id=netlab-rp-0", "addr=3.0", "multifunction=on", "vnet_hdr=off", "bus=netlab-rp-0"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("missing %s in %s", expected, joined)
		}
	}
	if manifest.NodeID != node.ID {
		t.Fatal("manifest owner missing")
	}
	if manifest.Disk != "/images/base.qcow2" {
		t.Fatalf("manifest disk=%q", manifest.Disk)
	}
}

func TestBuildArgsDoesNotExposeVNCWithoutDeclaredMode(t *testing.T) {
	adapter := &Adapter{Root: "/var/lib/netlab", Acceleration: "kvm"}
	args, _ := adapter.BuildArgs(domain.Node{ID: "node", Config: map[string]any{"console_modes": []string{"telnet"}}}, "/images/base.qcow2")
	joined := strings.Join(args, " ")
	if strings.Contains(joined, "-vnc") || strings.Contains(joined, "-device VGA") {
		t.Fatalf("unexpected VNC graphics arguments: %s", joined)
	}
}

func TestBuildArgsFallsBackToTCGWithoutKVM(t *testing.T) {
	adapter := &Adapter{Root: "/var/lib/netlab", Acceleration: "tcg"}
	args, manifest := adapter.BuildArgs(domain.Node{ID: "node", CPUCount: 1, MemoryMiB: 512}, "/images/base.qcow2")
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "-accel tcg,thread=multi") || !strings.Contains(joined, "-cpu max") {
		t.Fatalf("TCG fallback missing from %s", joined)
	}
	if manifest.Acceleration != "tcg" {
		t.Fatalf("manifest acceleration=%q", manifest.Acceleration)
	}
}

func TestBuildArgsSupportsLegacyPCNetworkAppliance(t *testing.T) {
	adapter := &Adapter{Root: "/tmp/netlab", Acceleration: "kvm"}
	node := domain.Node{ID: "ruijie", CPUCount: 1, MemoryMiB: 1024, InterfaceLimit: 10, Config: map[string]any{
		"machine":        "pc",
		"cpu":            "host",
		"disk_interface": "ide",
		"rtc_base":       "utc",
		"vga":            "std",
		"console_modes":  []string{"telnet", "vnc"},
		"interfaces": []any{map[string]any{
			"id": "iface-0", "slot": 0, "name": "G0/0", "driver": "e1000", "mac_address": "02:00:00:00:00:01",
		}},
	}}
	args, _ := adapter.BuildArgs(node, "/images/ruijie.qcow2")
	joined := strings.Join(args, " ")
	for _, expected := range []string{"-machine pc", "-cpu host", "if=ide", "-vga std", "-rtc base=utc", "-device e1000,netdev=net-iface-0"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("missing %q in %s", expected, joined)
		}
	}
	if strings.Contains(joined, "pcie-root-port") || strings.Contains(joined, "bus=netlab-rp-") {
		t.Fatalf("legacy pc appliance received PCIe topology: %s", joined)
	}
	if bus := hotplugBus(node, 3); bus != "" {
		t.Fatalf("legacy pc appliance hotplug bus=%q", bus)
	}
}

func TestStartEarlyExitRemovesTransientRuntimeState(t *testing.T) {
	root := t.TempDir()
	binary := filepath.Join(root, "qemu-early-exit")
	if err := os.WriteFile(binary, []byte("#!/bin/sh\necho invalid-image >&2\nexit 1\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	adapter := &Adapter{Binary: binary, Root: root, ReadinessTimeout: time.Second}
	node := provisionedTestNode(t, adapter)
	err := adapter.Start(context.Background(), node)
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "exited before qmp") {
		t.Fatalf("err=%v", err)
	}
	assertNoTransientQEMUState(t, adapter.RuntimeDir(node.ID))
}

func TestStartQMPTimeoutKillsProcessAndRemovesTransientState(t *testing.T) {
	root := t.TempDir()
	binary := filepath.Join(root, "qemu-no-qmp")
	if err := os.WriteFile(binary, []byte("#!/bin/sh\nsleep 10\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	adapter := &Adapter{Binary: binary, Root: root, ReadinessTimeout: 25 * time.Millisecond}
	node := provisionedTestNode(t, adapter)
	err := adapter.Start(context.Background(), node)
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "qmp readiness timed out") {
		t.Fatalf("err=%v", err)
	}
	assertNoTransientQEMUState(t, adapter.RuntimeDir(node.ID))
}

func provisionedTestNode(t *testing.T, adapter *Adapter) domain.Node {
	t.Helper()
	node := domain.Node{ID: "node", CPUCount: 1, MemoryMiB: 512, Config: map[string]any{"image_path": filepath.Join(adapter.Root, "base.qcow2")}}
	if err := os.MkdirAll(adapter.RuntimeDir(node.ID), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(adapter.RuntimeDir(node.ID), "disk.qcow2"), []byte("overlay"), 0o600); err != nil {
		t.Fatal(err)
	}
	return node
}

func assertNoTransientQEMUState(t *testing.T, runtimeDir string) {
	t.Helper()
	for _, name := range []string{"launch.json", "qmp.sock", "qga.sock", "serial.sock", "vnc.sock"} {
		if _, err := os.Stat(filepath.Join(runtimeDir, name)); !os.IsNotExist(err) {
			t.Fatalf("transient %s remained: %v", name, err)
		}
	}
}

func TestProvisionRejectsMissingImageBeforeLaunch(t *testing.T) {
	adapter := &Adapter{Root: t.TempDir()}
	err := adapter.Provision(context.Background(), domain.Node{ID: "node", Config: map[string]any{}})
	if err == nil || !strings.Contains(err.Error(), "image_path") {
		t.Fatalf("err=%v", err)
	}
	if _, statErr := os.Stat(filepath.Join(adapter.RuntimeDir("node"), "launch.json")); !os.IsNotExist(statErr) {
		t.Fatalf("launch manifest created during failed provisioning: %v", statErr)
	}
}
