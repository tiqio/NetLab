package cgroup

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/netlab/netlab/internal/domain"
)

func TestCPUQuotaMemoryMetricsAndDockerNormalization(t *testing.T) {
	root := t.TempDir()
	manager := NewManager(root)
	node := domain.Node{ID: "node-1", CPUCount: 2, CPUQuotaMicros: 100000, MemoryMiB: 512}
	if err := manager.Apply(context.Background(), node, 1234); err != nil {
		t.Fatal(err)
	}
	body, _ := os.ReadFile(filepath.Join(root, "node-1", "cpu.max"))
	if string(body) != "100000 100000\n" {
		t.Fatalf("cpu.max=%q", body)
	}
	if DockerNanoCPUs(node.CPUQuotaMicros) != 1_000_000_000 {
		t.Fatal("docker quota mismatch")
	}
	_ = os.WriteFile(filepath.Join(root, "node-1", "cpu.stat"), []byte("usage_usec 42\n"), 0644)
	_ = os.WriteFile(filepath.Join(root, "node-1", "memory.current"), []byte("1024\n"), 0644)
	metrics, err := manager.Metrics(node.ID)
	if err != nil || metrics.CPUUsageMicros != 42 || metrics.MemoryCurrent != 1024 {
		t.Fatalf("metrics=%+v err=%v", metrics, err)
	}
}

func TestProcessLimitWritten(t *testing.T) {
	root := t.TempDir()
	manager := NewManager(root)
	node := domain.Node{ID: "node-pids", CPUCount: 1, ProcessLimit: 128}
	if err := manager.Apply(context.Background(), node, 1234); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(root, "node-pids", "pids.max"))
	if err != nil || string(body) != "128\n" {
		t.Fatalf("pids.max=%q err=%v", body, err)
	}
}

func TestDefaultRootUsesCurrentCgroupSubtree(t *testing.T) {
	root := NewManager("").Root
	if !strings.HasPrefix(root, "/sys/fs/cgroup/") || !strings.HasSuffix(root, "/nodes") {
		t.Fatalf("root=%q", root)
	}
}

func TestDelegatedManagerSubgroupUsesServiceRoot(t *testing.T) {
	root := filepath.Join("/sys/fs/cgroup/system.slice/netlab.service/manager")
	if resolved := controllerRootForPath(root); resolved != filepath.Dir(root) {
		t.Fatalf("resolved=%q", resolved)
	}
}

func TestPrepareMovesManagerAndEnablesDelegatedControllers(t *testing.T) {
	serviceRoot := t.TempDir()
	nodeRoot := filepath.Join(serviceRoot, "nodes")
	managerRoot := filepath.Join(serviceRoot, "manager")
	if err := os.WriteFile(filepath.Join(serviceRoot, "cgroup.controllers"), []byte("cpu io memory pids\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(serviceRoot, "cgroup.subtree_control"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(nodeRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nodeRoot, "cgroup.controllers"), []byte("cpu memory pids\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nodeRoot, "cgroup.subtree_control"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	manager := &Manager{Root: nodeRoot, controllerRoot: serviceRoot, managerRoot: managerRoot, pid: 4321}
	if err := manager.Prepare(context.Background()); err != nil {
		t.Fatal(err)
	}
	managerPID, err := os.ReadFile(filepath.Join(managerRoot, "cgroup.procs"))
	if err != nil || string(managerPID) != "4321\n" {
		t.Fatalf("manager cgroup.procs=%q err=%v", managerPID, err)
	}
	for _, root := range []string{serviceRoot, nodeRoot} {
		body, readErr := os.ReadFile(filepath.Join(root, "cgroup.subtree_control"))
		if readErr != nil || string(body) != "+cpu +memory +pids\n" {
			t.Fatalf("%s/cgroup.subtree_control=%q err=%v", root, body, readErr)
		}
	}
	if err := manager.Prepare(context.Background()); err != nil {
		t.Fatalf("second prepare: %v", err)
	}
}
