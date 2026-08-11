package reconcile

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/netlab/netlab/internal/app/ports"
	"github.com/netlab/netlab/internal/domain"
)

func TestParseMemAvailableUsesKernelAvailableEstimate(t *testing.T) {
	available, ok := parseMemAvailable(strings.NewReader("MemTotal: 16384 kB\nMemFree: 512 kB\nMemAvailable: 8192 kB\n"))
	if !ok || available != 8192<<10 {
		t.Fatalf("available=%d ok=%v", available, ok)
	}
}

type resourceInspectorStub struct{ actual ports.ActualNode }

func (s resourceInspectorStub) Inspect(context.Context, domain.Node) (ports.ActualNode, error) {
	return s.actual, nil
}

type cgroupRecorder struct {
	applied bool
	removed bool
	pid     int
}

func (r *cgroupRecorder) Apply(_ context.Context, _ domain.Node, pid int) error {
	r.applied = true
	r.pid = pid
	return nil
}

func (r *cgroupRecorder) Remove(domain.ID) error {
	r.removed = true
	return nil
}

func TestResourceAdmissionUsesConfiguredStorageAndLimits(t *testing.T) {
	manager := NewResourceManager(nil, nil, t.TempDir())
	node := domain.Node{ID: "node", Kind: "qemu", CPUCount: 2, MemoryMiB: 256, StorageGiB: 1, InterfaceLimit: 64, ProcessLimit: 128, Config: map[string]any{"interfaces": []any{}}}
	if err := manager.Admit(context.Background(), node, []domain.Node{node}); err != nil {
		t.Fatal(err)
	}
	node.ProcessLimit = 2000000
	if err := manager.Admit(context.Background(), node, nil); err == nil {
		t.Fatal("invalid process limit accepted")
	}
}

func TestResourceAdmissionRejectsQEMUInterfaceLimitAboveHotplugCapacity(t *testing.T) {
	manager := NewResourceManager(nil, nil, t.TempDir())
	node := domain.Node{ID: "node", Kind: "qemu", CPUCount: 2, MemoryMiB: 256, StorageGiB: 1, InterfaceLimit: 65, ProcessLimit: 128, Config: map[string]any{"interfaces": []any{}}}
	if err := manager.Admit(context.Background(), node, []domain.Node{node}); err == nil {
		t.Fatal("expected QEMU interface limit rejection")
	}
}

func TestResourceAdmissionIgnoresStaleFailedQEMUDesiredState(t *testing.T) {
	manager := NewResourceManager(nil, nil, t.TempDir())
	manager.SetMaxRunningQEMU(4)
	node := domain.Node{ID: "candidate", Kind: "qemu", CPUCount: 1, MemoryMiB: 256, StorageGiB: 1, InterfaceLimit: 4, ProcessLimit: 64, DesiredState: domain.DesiredRunning, ObservedState: domain.ObservedStopped, Config: map[string]any{"interfaces": []any{}}}
	nodes := []domain.Node{node}
	for index := 0; index < 4; index++ {
		nodes = append(nodes, domain.Node{ID: domain.ID(fmt.Sprintf("stale-%d", index)), Kind: "qemu", DesiredState: domain.DesiredRunning, ObservedState: domain.ObservedFailed})
	}
	if err := manager.Admit(context.Background(), node, nodes); err != nil {
		t.Fatal(err)
	}
}

func TestResourceAdmissionAllowsUnlimitedRunningQEMU(t *testing.T) {
	manager := NewResourceManager(nil, nil, t.TempDir())
	node := domain.Node{ID: "candidate", Kind: "qemu", CPUCount: 1, MemoryMiB: 256, StorageGiB: 1, InterfaceLimit: 4, ProcessLimit: 64, Config: map[string]any{"interfaces": []any{}}}
	nodes := []domain.Node{node}
	for index := 0; index < 12; index++ {
		nodes = append(nodes, domain.Node{ID: domain.ID(fmt.Sprintf("running-%d", index)), Kind: "qemu", ObservedState: domain.ObservedRunning})
	}
	if err := manager.Admit(context.Background(), node, nodes); err != nil {
		t.Fatal(err)
	}
}

func TestResourceAdmissionUsesConfiguredQEMULimit(t *testing.T) {
	manager := NewResourceManager(nil, nil, t.TempDir())
	manager.SetMaxRunningQEMU(2)
	node := domain.Node{ID: "candidate", Kind: "qemu", CPUCount: 1, MemoryMiB: 256, StorageGiB: 1, InterfaceLimit: 4, ProcessLimit: 64, Config: map[string]any{"interfaces": []any{}}}
	nodes := []domain.Node{node, {ID: "running-a", Kind: "qemu", ObservedState: domain.ObservedRunning}, {ID: "running-b", Kind: "qemu", ObservedState: domain.ObservedRunning}}
	if err := manager.Admit(context.Background(), node, nodes); err == nil {
		t.Fatal("configured QEMU limit was not enforced")
	}
}

func TestResourceApplicationUsesRuntimeStateAfterFreshStart(t *testing.T) {
	cgroups := &cgroupRecorder{}
	manager := NewResourceManager(resourceInspectorStub{actual: ports.ActualNode{
		State: domain.ObservedRunning,
		Owner: map[string]string{"pid": "4242"},
	}}, cgroups)
	node := domain.Node{ID: "node", Kind: "qemu", ObservedState: domain.ObservedStopped}

	if err := manager.Apply(context.Background(), node); err != nil {
		t.Fatal(err)
	}
	if !cgroups.applied || cgroups.pid != 4242 {
		t.Fatalf("resource limits were not applied to fresh process: %+v", cgroups)
	}
}

func TestResourceCleanupRemovesNodeCgroup(t *testing.T) {
	cgroups := &cgroupRecorder{}
	manager := NewResourceManager(nil, cgroups)
	if err := manager.Cleanup("node"); err != nil {
		t.Fatal(err)
	}
	if !cgroups.removed {
		t.Fatal("node cgroup cleanup was not delegated")
	}
}

func TestResourceMetricsAreNormalizedForNamespaceRuntime(t *testing.T) {
	manager := NewResourceManager(nil, nil, t.TempDir())
	manager.RegisterInspector("pc", resourceInspectorStub{actual: ports.ActualNode{State: domain.ObservedRunning, Owner: map[string]string{"netns": "nlpc"}}})
	node := domain.Node{ID: "pc", Kind: "pc", CPUCount: 1, MemoryMiB: 64, StorageGiB: 1, InterfaceLimit: 4, ProcessLimit: 8, Config: map[string]any{"interfaces": []any{map[string]any{"name": "eth0"}}}}
	snapshot, err := manager.Metrics(context.Background(), node)
	if err != nil {
		t.Fatal(err)
	}
	if !snapshot.Observed.Running || snapshot.Observed.RuntimeKind != "pc" || snapshot.Observed.InterfaceCount != 1 || snapshot.Configured.ProcessLimit != 8 {
		t.Fatalf("snapshot=%+v", snapshot)
	}
}
