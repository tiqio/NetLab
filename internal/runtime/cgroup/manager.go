package cgroup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/netlab/netlab/internal/domain"
)

type Manager struct {
	Root           string
	controllerRoot string
	managerRoot    string
	pid            int
	mu             sync.Mutex
	prepared       bool
}

type Metrics struct {
	CPUUsageMicros int64 `json:"cpu_usage_micros"`
	MemoryCurrent  int64 `json:"memory_current"`
}

func NewManager(root string) *Manager {
	if root != "" {
		return &Manager{Root: root}
	}
	controllerRoot := selfCgroupRoot()
	return &Manager{
		Root:           filepath.Join(controllerRoot, "nodes"),
		controllerRoot: controllerRoot,
		managerRoot:    filepath.Join(controllerRoot, "manager"),
		pid:            os.Getpid(),
	}
}

func selfCgroupRoot() string {
	body, err := os.ReadFile("/proc/self/cgroup")
	if err == nil {
		for _, line := range strings.Split(string(body), "\n") {
			parts := strings.SplitN(line, ":", 3)
			if len(parts) == 3 && parts[0] == "0" {
				root := filepath.Join("/sys/fs/cgroup", strings.TrimPrefix(parts[2], "/"))
				return controllerRootForPath(root)
			}
		}
	}
	return "/sys/fs/cgroup"
}

func controllerRootForPath(root string) string {
	if filepath.Base(root) == "manager" {
		return filepath.Dir(root)
	}
	return root
}

func (m *Manager) Prepare(ctx context.Context) error {
	if m.controllerRoot == "" {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.prepared {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.MkdirAll(m.managerRoot, 0o755); err != nil {
		return fmt.Errorf("create manager cgroup: %w", err)
	}
	if err := os.WriteFile(filepath.Join(m.managerRoot, "cgroup.procs"), []byte(strconv.Itoa(m.pid)+"\n"), 0o644); err != nil {
		return fmt.Errorf("move manager process into delegated cgroup: %w", err)
	}
	if err := enableControllers(m.controllerRoot); err != nil {
		return fmt.Errorf("enable service cgroup controllers: %w", err)
	}
	if err := os.MkdirAll(m.Root, 0o755); err != nil {
		return fmt.Errorf("create node cgroup root: %w", err)
	}
	if err := enableControllers(m.Root); err != nil {
		return fmt.Errorf("enable node cgroup controllers: %w", err)
	}
	m.prepared = true
	return nil
}

func enableControllers(root string) error {
	availableBody, err := os.ReadFile(filepath.Join(root, "cgroup.controllers"))
	if err != nil {
		return err
	}
	available := make(map[string]bool)
	for _, controller := range strings.Fields(string(availableBody)) {
		available[controller] = true
	}
	required := []string{"cpu", "memory", "pids"}
	for _, controller := range required {
		if !available[controller] {
			return fmt.Errorf("required controller %q is unavailable", controller)
		}
	}
	return os.WriteFile(filepath.Join(root, "cgroup.subtree_control"), []byte("+cpu +memory +pids\n"), 0o644)
}

func (m *Manager) Apply(ctx context.Context, node domain.Node, pid int) error {
	if err := m.Prepare(ctx); err != nil {
		return err
	}
	directory := filepath.Join(m.Root, string(node.ID))
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return err
	}
	period := int64(100000)
	quota := node.CPUQuotaMicros
	if quota <= 0 {
		quota = period * int64(max(node.CPUCount, 1))
	}
	if err := os.WriteFile(filepath.Join(directory, "cpu.max"), []byte(fmt.Sprintf("%d %d\n", quota, period)), 0o644); err != nil {
		return err
	}
	if node.MemoryMiB > 0 {
		if err := os.WriteFile(filepath.Join(directory, "memory.max"), []byte(strconv.FormatInt(int64(node.MemoryMiB)<<20, 10)+"\n"), 0o644); err != nil {
			return err
		}
	}
	processLimit := int64(node.ProcessLimit)
	if processLimit > 0 {
		if err := os.WriteFile(filepath.Join(directory, "pids.max"), []byte(strconv.FormatInt(processLimit, 10)+"\n"), 0o644); err != nil {
			return err
		}
	}
	return os.WriteFile(filepath.Join(directory, "cgroup.procs"), []byte(strconv.Itoa(pid)+"\n"), 0o644)
}

func (m *Manager) Metrics(id domain.ID) (Metrics, error) {
	directory := filepath.Join(m.Root, string(id))
	var metrics Metrics
	body, err := os.ReadFile(filepath.Join(directory, "cpu.stat"))
	if err != nil {
		return metrics, err
	}
	for _, line := range strings.Split(string(body), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "usage_usec" {
			metrics.CPUUsageMicros, _ = strconv.ParseInt(fields[1], 10, 64)
		}
	}
	if body, err = os.ReadFile(filepath.Join(directory, "memory.current")); err == nil {
		metrics.MemoryCurrent, _ = strconv.ParseInt(strings.TrimSpace(string(body)), 10, 64)
	}
	return metrics, nil
}

func (m *Manager) MetricsAny(id domain.ID) (any, error) { return m.Metrics(id) }

func (m *Manager) ResourceMetrics(id domain.ID) (int64, int64, error) {
	metrics, err := m.Metrics(id)
	return metrics.CPUUsageMicros, metrics.MemoryCurrent, err
}

func (m *Manager) Remove(id domain.ID) error {
	err := os.Remove(filepath.Join(m.Root, string(id)))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func DockerNanoCPUs(quotaMicros int64) int64 {
	if quotaMicros <= 0 {
		return 0
	}
	return quotaMicros * 10000
}
