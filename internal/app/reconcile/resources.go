package reconcile

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/netlab/netlab/internal/app/ports"
	"github.com/netlab/netlab/internal/domain"
)

type ProcessResourceRuntime interface {
	Apply(context.Context, domain.Node, int) error
	Metrics(domain.ID) (any, error)
}

type NodeInspector interface {
	Inspect(context.Context, domain.Node) (ports.ActualNode, error)
}

type ResourceManager struct {
	inspectors map[string]NodeInspector
	stateDir   string
	cgroups    interface {
		Apply(context.Context, domain.Node, int) error
	}
}

type ResourceConfiguration struct {
	CPUCount       int   `json:"cpu_count"`
	CPUQuotaMicros int64 `json:"cpu_quota_micros"`
	MemoryMiB      int   `json:"memory_mib"`
	StorageGiB     int   `json:"storage_gib"`
	InterfaceLimit int   `json:"interface_limit"`
	ProcessLimit   int   `json:"process_limit"`
}

type ResourceObservation struct {
	RuntimeKind      string `json:"runtime_kind"`
	Running          bool   `json:"running"`
	MetricsAvailable bool   `json:"metrics_available"`
	CPUUsageMicros   int64  `json:"cpu_usage_micros"`
	MemoryBytes      int64  `json:"memory_bytes"`
	StorageBytes     int64  `json:"storage_bytes"`
	ProcessCount     int    `json:"process_count"`
	InterfaceCount   int    `json:"interface_count"`
}

type ResourceSnapshot struct {
	Configured ResourceConfiguration `json:"configured"`
	Observed   ResourceObservation   `json:"observed"`
}

func (m *ResourceManager) Admit(_ context.Context, node domain.Node, nodes []domain.Node) error {
	if node.CPUCount < 1 || node.CPUCount > 256 || node.MemoryMiB < 0 || len(node.Config) > 256 {
		return domain.Problem{Code: "resource_exhausted", Message: "node resource declaration is outside supported limits", ResourceType: "node", ResourceID: node.ID}
	}
	interfaces := len(extractInterfaces(node.Config))
	limit := node.InterfaceLimit
	if limit <= 0 {
		limit = maximumInterfaceLimit(node.Kind)
	}
	maximum := maximumInterfaceLimit(node.Kind)
	if interfaces > limit || limit > maximum {
		return domain.Problem{Code: "resource_exhausted", Message: fmt.Sprintf("interface limit exceeds runtime capacity of %d", maximum), ResourceType: "node", ResourceID: node.ID, Phase: "resource_admission", Cleanup: "no runtime resources created", OperatorHint: "reduce the interface limit or configured interface count", Details: map[string]any{"interface_limit": limit, "interface_count": interfaces, "runtime_capacity": maximum}}
	}
	if node.Kind == "qemu" {
		running := 0
		for _, value := range nodes {
			if value.Kind == "qemu" && (value.ID == node.ID || value.ObservedState == domain.ObservedProvisioning || value.ObservedState == domain.ObservedStarting || value.ObservedState == domain.ObservedRunning) {
				running++
			}
		}
		if running > 4 {
			return domain.Problem{Code: "resource_exhausted", Message: "first-release limit is four running QEMU nodes", ResourceType: "node", ResourceID: node.ID}
		}
	}
	if available, ok := availableMemoryBytes(); ok && node.MemoryMiB > 0 {
		if uint64(node.MemoryMiB)<<20 > available*9/10 {
			return domain.Problem{Code: "resource_exhausted", Message: "insufficient host memory", ResourceType: "node", ResourceID: node.ID, Retryable: true}
		}
	}
	if node.ProcessLimit < 1 || node.ProcessLimit > 1048576 {
		return domain.Problem{Code: "resource_exhausted", Message: "process limit is outside supported bounds", ResourceType: "node", ResourceID: node.ID}
	}
	if diskGiB := int64(node.StorageGiB); diskGiB > 0 {
		var stat syscall.Statfs_t
		storageRoot := m.stateDir
		if storageRoot == "" {
			storageRoot = "/"
		}
		if err := syscall.Statfs(storageRoot, &stat); err != nil && os.IsNotExist(err) {
			_ = syscall.Statfs("/", &stat)
		}
		available := uint64(stat.Bavail) * uint64(stat.Bsize)
		if uint64(diskGiB)<<30 > available*9/10 {
			return domain.Problem{Code: "resource_exhausted", Message: "insufficient host storage", ResourceType: "node", ResourceID: node.ID, Retryable: true}
		}
	}
	return nil
}

func availableMemoryBytes() (uint64, bool) {
	if file, err := os.Open("/proc/meminfo"); err == nil {
		defer file.Close()
		if available, ok := parseMemAvailable(file); ok {
			return available, true
		}
	}
	var info syscall.Sysinfo_t
	if err := syscall.Sysinfo(&info); err != nil {
		return 0, false
	}
	return (uint64(info.Freeram) + uint64(info.Bufferram)) * uint64(info.Unit), true
}

func parseMemAvailable(reader io.Reader) (uint64, bool) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 || fields[0] != "MemAvailable:" {
			continue
		}
		kilobytes, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return 0, false
		}
		return kilobytes << 10, true
	}
	return 0, false
}

func maximumInterfaceLimit(kind string) int {
	if kind == "qemu" {
		return 64
	}
	return 256
}

func extractInterfaces(config map[string]any) []any {
	values, _ := config["interfaces"].([]any)
	if direct, ok := config["interfaces"].([]map[string]any); ok {
		values = make([]any, len(direct))
	}
	return values
}

func numberFromConfig(value any) int64 {
	switch typed := value.(type) {
	case int:
		return int64(typed)
	case int64:
		return typed
	case float64:
		return int64(typed)
	default:
		return 0
	}
}

func NewResourceManager(inspector NodeInspector, cgroups interface {
	Apply(context.Context, domain.Node, int) error
}, stateDirs ...string) *ResourceManager {
	manager := &ResourceManager{inspectors: map[string]NodeInspector{}, cgroups: cgroups}
	if inspector != nil {
		manager.inspectors["qemu"] = inspector
	}
	if len(stateDirs) > 0 {
		manager.stateDir = stateDirs[0]
	}
	return manager
}

func (m *ResourceManager) RegisterInspector(kind string, inspector NodeInspector) {
	if inspector != nil {
		m.inspectors[kind] = inspector
	}
}

func (m *ResourceManager) Cleanup(id domain.ID) error {
	if runtime, ok := m.cgroups.(interface{ Remove(domain.ID) error }); ok {
		return runtime.Remove(id)
	}
	return nil
}

func (m *ResourceManager) Apply(ctx context.Context, node domain.Node) error {
	if node.Kind != "qemu" {
		return nil
	}
	inspector := m.inspectors[node.Kind]
	if inspector == nil || m.cgroups == nil {
		return domain.Problem{Code: "capability_unsupported", Message: "cgroup or QEMU runtime unavailable"}
	}
	actual, err := inspector.Inspect(ctx, node)
	if err != nil {
		return err
	}
	if actual.State != domain.ObservedRunning {
		return nil
	}
	pid, err := strconv.Atoi(actual.Owner["pid"])
	if err != nil || pid < 1 {
		return fmt.Errorf("QEMU process owner PID unavailable")
	}
	return m.cgroups.Apply(ctx, node, pid)
}

func (m *ResourceManager) Metrics(ctx context.Context, node domain.Node) (ResourceSnapshot, error) {
	snapshot := ResourceSnapshot{Configured: ResourceConfiguration{CPUCount: node.CPUCount, CPUQuotaMicros: node.CPUQuotaMicros, MemoryMiB: node.MemoryMiB, StorageGiB: node.StorageGiB, InterfaceLimit: node.InterfaceLimit, ProcessLimit: node.ProcessLimit}, Observed: ResourceObservation{RuntimeKind: node.Kind, InterfaceCount: len(extractInterfaces(node.Config))}}
	if inspector := m.inspectors[node.Kind]; inspector != nil {
		actual, err := inspector.Inspect(ctx, node)
		if err != nil {
			return snapshot, err
		}
		snapshot.Observed.Running = actual.State == domain.ObservedRunning
		if pid, parseErr := strconv.Atoi(actual.Owner["pid"]); parseErr == nil && pid > 0 {
			readProcessMetrics(pid, &snapshot.Observed)
		}
	}
	if node.Kind == "qemu" {
		if runtime, ok := m.cgroups.(interface {
			ResourceMetrics(domain.ID) (int64, int64, error)
		}); ok {
			cpu, memory, err := runtime.ResourceMetrics(node.ID)
			if err == nil {
				snapshot.Observed.CPUUsageMicros = cpu
				snapshot.Observed.MemoryBytes = memory
				snapshot.Observed.MetricsAvailable = true
			}
		}
	}
	snapshot.Observed.StorageBytes = directorySize(filepath.Join(m.stateDir, string(node.ID)))
	return snapshot, nil
}

func readProcessMetrics(pid int, observed *ResourceObservation) {
	if body, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid)); err == nil {
		fields := strings.Fields(string(body))
		if len(fields) > 14 {
			user, _ := strconv.ParseInt(fields[13], 10, 64)
			system, _ := strconv.ParseInt(fields[14], 10, 64)
			observed.CPUUsageMicros = (user + system) * 10000
			observed.MetricsAvailable = true
		}
	}
	if body, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid)); err == nil {
		for _, line := range strings.Split(string(body), "\n") {
			if strings.HasPrefix(line, "VmRSS:") {
				fields := strings.Fields(line)
				if len(fields) > 1 {
					value, _ := strconv.ParseInt(fields[1], 10, 64)
					observed.MemoryBytes = value << 10
				}
			}
		}
	}
	if entries, err := os.ReadDir(fmt.Sprintf("/proc/%d/task", pid)); err == nil {
		observed.ProcessCount = len(entries)
	}
}

func directorySize(root string) int64 {
	var total int64
	_ = filepath.Walk(root, func(_ string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			total += info.Size()
		}
		return nil
	})
	return total
}
