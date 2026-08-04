package command

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/netlab/netlab/internal/domain"
)

type interfaceMemory struct {
	node       domain.Node
	count      int
	reserved   int
	deleted    int
	iface      domain.Interface
	reserveErr error
	owned      int
	unowned    int
	ownerErr   error
	state      string
	stateErr   error
}

func (m *interfaceMemory) GetNode(context.Context, domain.ID) (domain.Node, error) {
	return m.node, nil
}

func (m *interfaceMemory) GetInterface(context.Context, domain.ID) (domain.Interface, error) {
	if m.iface.ID == "" {
		return domain.Interface{}, errors.New("not implemented")
	}
	return m.iface, nil
}

func (m *interfaceMemory) ReserveInterface(_ context.Context, value domain.Interface, limit int) (domain.Interface, error) {
	m.reserved++
	if m.reserveErr != nil {
		return domain.Interface{}, m.reserveErr
	}
	if m.count >= limit {
		return domain.Interface{}, interfaceCapacityProblem(m.node, limit)
	}
	value.Slot = m.count
	value.Name = fmt.Sprintf("eth%d", value.Slot)
	m.iface = value
	return value, nil
}

func (m *interfaceMemory) DeleteInterface(context.Context, domain.ID, domain.Revision) error {
	m.deleted++
	return nil
}

func (m *interfaceMemory) SetInterfaceOperationalState(_ context.Context, _ domain.ID, state string) error {
	m.state = state
	return m.stateErr
}

func (m *interfaceMemory) UpsertRuntimeOwnership(context.Context, string, domain.ID, string, string, map[string]string, string) error {
	m.owned++
	return m.ownerErr
}

func (m *interfaceMemory) DeleteRuntimeOwnership(context.Context, string, domain.ID, string, string) error {
	m.unowned++
	return nil
}

func TestInterfaceAddRejectsCapacityBeforePersistence(t *testing.T) {
	repository := &interfaceMemory{node: domain.Node{ID: "node", Kind: "qemu", InterfaceLimit: 64, ObservedState: domain.ObservedRunning}, count: 64}
	service := &InterfaceService{repository: repository, drivers: map[string]bool{"virtio-net-pci": true}}

	_, taskValue, err := service.Add(context.Background(), repository.node.ID, "virtio-net-pci", "capacity")
	if err == nil {
		t.Fatal("expected interface capacity rejection")
	}
	var problem domain.Problem
	if !errors.As(err, &problem) || problem.Code != "resource_exhausted" || problem.Phase != "interface_admission" {
		t.Fatalf("unexpected problem: %#v", err)
	}
	if taskValue != nil || repository.reserved != 1 {
		t.Fatalf("capacity rejection produced unexpected work: task=%v reserved=%d", taskValue, repository.reserved)
	}
}

func TestInterfaceAddRejectsUnsupportedQEMULimit(t *testing.T) {
	repository := &interfaceMemory{node: domain.Node{ID: "node", Kind: "qemu", InterfaceLimit: 65}, count: 0}
	service := &InterfaceService{repository: repository, drivers: map[string]bool{"virtio-net-pci": true}}

	_, _, err := service.Add(context.Background(), repository.node.ID, "virtio-net-pci", "capacity")
	if err == nil || repository.reserved != 0 {
		t.Fatalf("unsupported limit was not rejected before side effects: err=%v reserved=%d", err, repository.reserved)
	}
}

type failingHotplugger struct{ err error }

func (h failingHotplugger) HotAddInterface(context.Context, domain.Node, domain.Interface, string) error {
	return h.err
}
func (h failingHotplugger) HotRemoveInterface(context.Context, domain.Node, domain.Interface) error {
	return nil
}

type tapRecorder struct{ created, deleted int }

func (t *tapRecorder) CreateTap(context.Context, string, string) error { t.created++; return nil }
func (t *tapRecorder) Delete(context.Context, string) error            { t.deleted++; return nil }

func TestInterfaceHotAddFailureRollsBackPersistenceAndTap(t *testing.T) {
	iface := domain.Interface{ID: "iface", NodeID: "node", Revision: 1}
	repository := &interfaceMemory{node: domain.Node{ID: "node", Kind: "qemu", ObservedState: domain.ObservedRunning}, iface: iface}
	taps := &tapRecorder{}
	service := &InterfaceService{repository: repository, hotplugger: failingHotplugger{err: errors.New("device_add failed")}, taps: taps}
	taskValue := &domain.OperationTask{Input: map[string]any{"node_id": "node", "interface_id": "iface", "tap_name": "nltap"}}

	_, err := service.handleAdd(context.Background(), taskValue)
	problem, ok := domain.ProblemFromError(err)
	if !ok || problem.Code != "interface_hot_add_failed" || problem.Cleanup != "interface row and TAP removed" {
		t.Fatalf("problem=%+v err=%v", problem, err)
	}
	if taps.created != 1 || taps.deleted != 1 || repository.deleted != 1 {
		t.Fatalf("created=%d tap_deleted=%d row_deleted=%d", taps.created, taps.deleted, repository.deleted)
	}
	if repository.owned != 1 || repository.unowned != 1 {
		t.Fatalf("owned=%d unowned=%d", repository.owned, repository.unowned)
	}
}

func TestInterfaceOwnershipFailureRollsBackTapAndRow(t *testing.T) {
	iface := domain.Interface{ID: "iface", NodeID: "node", Revision: 1}
	repository := &interfaceMemory{node: domain.Node{ID: "node", Kind: "qemu", ObservedState: domain.ObservedRunning}, iface: iface, ownerErr: errors.New("ownership unavailable")}
	taps := &tapRecorder{}
	service := &InterfaceService{repository: repository, hotplugger: failingHotplugger{}, taps: taps}
	taskValue := &domain.OperationTask{Input: map[string]any{"node_id": "node", "interface_id": "iface", "tap_name": "nltap"}}

	_, err := service.handleAdd(context.Background(), taskValue)
	problem, ok := domain.ProblemFromError(err)
	if !ok || problem.Code != "interface_hot_add_failed" {
		t.Fatalf("problem=%+v err=%v", problem, err)
	}
	if taps.created != 1 || taps.deleted != 1 || repository.deleted != 1 || repository.owned != 1 || repository.unowned != 1 {
		t.Fatalf("taps=%+v repository=%+v", taps, repository)
	}
}

func TestInterfaceHotAddMarksUnconnectedInterfaceDown(t *testing.T) {
	iface := domain.Interface{ID: "iface", NodeID: "node", Revision: 1, OperationalState: "pending"}
	repository := &interfaceMemory{node: domain.Node{ID: "node", Kind: "qemu", ObservedState: domain.ObservedRunning}, iface: iface}
	taps := &tapRecorder{}
	service := &InterfaceService{repository: repository, hotplugger: failingHotplugger{}, taps: taps}
	taskValue := &domain.OperationTask{Input: map[string]any{"node_id": "node", "interface_id": "iface", "tap_name": "nltap"}}

	if _, err := service.handleAdd(context.Background(), taskValue); err != nil {
		t.Fatal(err)
	}
	if repository.state != "down" {
		t.Fatalf("operational state=%q", repository.state)
	}
}

func TestInterfaceHotRemoveDeletesTapOwnership(t *testing.T) {
	iface := domain.Interface{ID: "iface", NodeID: "node", Revision: 1}
	repository := &interfaceMemory{node: domain.Node{ID: "node", Kind: "qemu", ObservedState: domain.ObservedRunning}, iface: iface}
	taps := &tapRecorder{}
	service := &InterfaceService{repository: repository, hotplugger: failingHotplugger{}, taps: taps}
	taskValue := &domain.OperationTask{Input: map[string]any{"node_id": "node", "interface_id": "iface", "tap_name": "nltap", "revision": float64(1)}}

	if _, err := service.handleRemove(context.Background(), taskValue); err != nil {
		t.Fatal(err)
	}
	if taps.deleted != 1 || repository.unowned != 1 || repository.deleted != 1 {
		t.Fatalf("taps=%+v repository=%+v", taps, repository)
	}
}
