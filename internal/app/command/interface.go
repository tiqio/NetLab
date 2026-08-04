package command

import (
	"context"
	"fmt"
	"time"

	"github.com/netlab/netlab/internal/app/task"
	"github.com/netlab/netlab/internal/domain"
)

type InterfaceRepository interface {
	GetNode(context.Context, domain.ID) (domain.Node, error)
	GetInterface(context.Context, domain.ID) (domain.Interface, error)
	ReserveInterface(context.Context, domain.Interface, int) (domain.Interface, error)
	DeleteInterface(context.Context, domain.ID, domain.Revision) error
	SetInterfaceOperationalState(context.Context, domain.ID, string) error
}

type InterfaceHotplugger interface {
	InterfaceTapName(domain.Interface) string
	HotAddInterface(context.Context, domain.Node, domain.Interface, string) error
	HotRemoveInterface(context.Context, domain.Node, domain.Interface) error
}

type TapController interface {
	CreateTap(context.Context, string, string) error
	Delete(context.Context, string) error
}

type RuntimeOwnershipWriter interface {
	UpsertRuntimeOwnership(context.Context, string, domain.ID, string, string, map[string]string, string) error
	DeleteRuntimeOwnership(context.Context, string, domain.ID, string, string) error
}

type InterfaceService struct {
	repository InterfaceRepository
	runner     *task.Runner
	hotplugger InterfaceHotplugger
	taps       TapController
	drivers    map[string]bool
}

func NewInterfaceService(repository InterfaceRepository, runner *task.Runner, hotplugger InterfaceHotplugger, taps TapController, drivers []string) *InterfaceService {
	service := &InterfaceService{repository: repository, runner: runner, hotplugger: hotplugger, taps: taps, drivers: map[string]bool{}}
	for _, driver := range drivers {
		service.drivers[driver] = true
	}
	runner.Register("interface.hot_add", service.handleAdd)
	runner.Register("interface.hot_remove", service.handleRemove)
	return service
}

func (s *InterfaceService) Add(ctx context.Context, nodeID domain.ID, driver, idempotencyKey string) (domain.Interface, *domain.OperationTask, error) {
	if len(s.drivers) > 0 && !s.drivers[driver] {
		return domain.Interface{}, nil, domain.Problem{Code: "capability_unsupported", Message: "NIC driver is not supported by the template"}
	}
	node, err := s.repository.GetNode(ctx, nodeID)
	if err != nil {
		return domain.Interface{}, nil, err
	}
	if node.Kind != "qemu" {
		return domain.Interface{}, nil, domain.Problem{Code: "capability_unsupported", Message: "hot-add is supported only for QEMU nodes"}
	}
	if node.InterfaceLimit > 64 {
		return domain.Interface{}, nil, interfaceCapacityProblem(node, 64)
	}
	limit := node.InterfaceLimit
	if limit <= 0 {
		limit = 64
	}
	iface := domain.Interface{ID: domain.NewID(), NodeID: nodeID, Driver: driver, MACAddress: randomMAC(), OperationalState: "pending", Revision: 1}
	iface, err = s.repository.ReserveInterface(ctx, iface, limit)
	if err != nil {
		return domain.Interface{}, nil, err
	}
	if node.ObservedState != domain.ObservedRunning {
		return iface, nil, nil
	}
	taskValue := domain.OperationTask{ID: domain.NewID(), Kind: "interface.hot_add", ResourceType: "interface", ResourceID: iface.ID, IdempotencyKey: idempotencyKey, State: domain.TaskQueued, ProgressTotal: 3, Input: map[string]any{"node_id": string(nodeID), "interface_id": string(iface.ID), "tap_name": s.hotplugger.InterfaceTapName(iface)}, CreatedAt: time.Now().UTC()}
	if err = s.runner.Enqueue(ctx, taskValue); err != nil {
		_ = s.repository.DeleteInterface(ctx, iface.ID, iface.Revision)
		return domain.Interface{}, nil, err
	}
	return iface, &taskValue, nil
}

func interfaceCapacityProblem(node domain.Node, limit int) domain.Problem {
	return domain.Problem{
		Code:         "resource_exhausted",
		Message:      fmt.Sprintf("node interface capacity of %d has been reached", limit),
		ResourceType: "node",
		ResourceID:   node.ID,
		Phase:        "interface_admission",
		Cleanup:      "no side effects created",
		OperatorHint: "remove an interface or recreate the node with a supported interface limit",
		Details:      map[string]any{"interface_limit": limit, "runtime_kind": node.Kind},
	}
}

func (s *InterfaceService) Remove(ctx context.Context, interfaceID domain.ID, revision domain.Revision, idempotencyKey string) (domain.OperationTask, error) {
	iface, err := s.repository.GetInterface(ctx, interfaceID)
	if err != nil {
		return domain.OperationTask{}, err
	}
	node, err := s.repository.GetNode(ctx, iface.NodeID)
	if err != nil {
		return domain.OperationTask{}, err
	}
	if node.ObservedState != domain.ObservedRunning {
		return domain.OperationTask{}, s.repository.DeleteInterface(ctx, interfaceID, revision)
	}
	value := domain.OperationTask{ID: domain.NewID(), Kind: "interface.hot_remove", ResourceType: "interface", ResourceID: interfaceID, IdempotencyKey: idempotencyKey, RequestedRevision: revision, ProgressTotal: 3, Input: map[string]any{"node_id": string(node.ID), "interface_id": string(interfaceID), "revision": int64(revision), "tap_name": s.hotplugger.InterfaceTapName(iface)}, CreatedAt: time.Now().UTC()}
	return value, s.runner.Enqueue(ctx, value)
}

func (s *InterfaceService) handleAdd(ctx context.Context, value *domain.OperationTask) (map[string]any, error) {
	node, iface, tap, err := s.resolve(value)
	if err != nil {
		return nil, err
	}
	value.ProgressCurrent = 1
	if err = s.taps.CreateTap(ctx, tap, string(iface.ID)); err != nil {
		return nil, s.rollbackAdd(iface, tap, err, false)
	}
	if ownershipWriter, ok := s.repository.(RuntimeOwnershipWriter); ok {
		metadata := map[string]string{"node_id": string(node.ID), "hot_added": "true"}
		if err = ownershipWriter.UpsertRuntimeOwnership(ctx, "interface", iface.ID, "tap", tap, metadata, "active"); err != nil {
			return nil, s.rollbackAdd(iface, tap, err, true)
		}
	}
	value.ProgressCurrent = 2
	if err = s.hotplugger.HotAddInterface(ctx, node, iface, tap); err != nil {
		return nil, s.rollbackAdd(iface, tap, err, true)
	}
	if err = s.repository.SetInterfaceOperationalState(ctx, iface.ID, "down"); err != nil {
		removeErr := s.hotplugger.HotRemoveInterface(context.Background(), node, iface)
		rollbackErr := s.rollbackAdd(iface, tap, err, true)
		if removeErr != nil {
			return nil, domain.Problem{Code: "interface_hot_add_cleanup_failed", Message: err.Error(), Retryable: true, ResourceType: "interface", ResourceID: iface.ID, Phase: "hot_add", Cleanup: fmt.Sprintf("QMP rollback: %v; persistence cleanup: %v", removeErr, rollbackErr), OperatorHint: "inspect the node QMP state and remove the stale device before retrying"}
		}
		return nil, rollbackErr
	}
	value.ProgressCurrent = 3
	return map[string]any{"interface_id": iface.ID, "tap": tap}, nil
}

func (s *InterfaceService) rollbackAdd(iface domain.Interface, tap string, cause error, tapCreated bool) error {
	var tapErr error
	var ownershipErr error
	if tapCreated {
		tapErr = s.taps.Delete(context.Background(), tap)
	}
	if ownershipWriter, ok := s.repository.(RuntimeOwnershipWriter); ok {
		ownershipErr = ownershipWriter.DeleteRuntimeOwnership(context.Background(), "interface", iface.ID, "tap", tap)
	}
	deleteErr := s.repository.DeleteInterface(context.Background(), iface.ID, iface.Revision)
	if tapErr == nil && ownershipErr == nil && deleteErr == nil {
		return domain.Problem{Code: "interface_hot_add_failed", Message: cause.Error(), ResourceType: "interface", ResourceID: iface.ID, Phase: "hot_add", Cleanup: "interface row and TAP removed", OperatorHint: "verify QMP readiness and retry the operation"}
	}
	return domain.Problem{Code: "interface_hot_add_cleanup_failed", Message: cause.Error(), Retryable: true, ResourceType: "interface", ResourceID: iface.ID, Phase: "hot_add", Cleanup: fmt.Sprintf("tap cleanup: %v; ownership cleanup: %v; interface cleanup: %v", tapErr, ownershipErr, deleteErr), OperatorHint: "inspect the node QMP state and remove the owned TAP/interface record before retrying"}
}

func (s *InterfaceService) handleRemove(ctx context.Context, value *domain.OperationTask) (map[string]any, error) {
	node, iface, tap, err := s.resolve(value)
	if err != nil {
		return nil, err
	}
	value.ProgressCurrent = 1
	if err = s.hotplugger.HotRemoveInterface(ctx, node, iface); err != nil {
		return nil, err
	}
	value.ProgressCurrent = 2
	tapNames := []string{tap}
	if legacy := legacyTapName(iface.ID); legacy != tap {
		tapNames = append(tapNames, legacy)
	}
	for _, name := range tapNames {
		if err = s.taps.Delete(ctx, name); err != nil {
			return nil, domain.Problem{Code: "interface_hot_remove_cleanup_failed", Message: err.Error(), Retryable: true, ResourceType: "interface", ResourceID: iface.ID, Phase: "hot_remove", Cleanup: "QMP device removed but owned TAP cleanup failed", OperatorHint: "remove the owned TAP and retry deletion"}
		}
	}
	if ownershipWriter, ok := s.repository.(RuntimeOwnershipWriter); ok {
		for _, name := range tapNames {
			for _, kind := range []string{"tap", "linux_link"} {
				if err = ownershipWriter.DeleteRuntimeOwnership(ctx, "interface", iface.ID, kind, name); err != nil {
					return nil, domain.Problem{Code: "interface_hot_remove_cleanup_failed", Message: err.Error(), Retryable: true, ResourceType: "interface", ResourceID: iface.ID, Phase: "hot_remove", Cleanup: "QMP device and TAP removed but ownership cleanup failed", OperatorHint: "remove the stale runtime ownership record and retry deletion"}
				}
			}
		}
	}
	revision := domain.Revision(number(value.Input["revision"]))
	if err = s.repository.DeleteInterface(ctx, iface.ID, revision); err != nil {
		return nil, err
	}
	value.ProgressCurrent = 3
	return map[string]any{"interface_id": iface.ID}, nil
}

func (s *InterfaceService) resolve(value *domain.OperationTask) (domain.Node, domain.Interface, string, error) {
	node, err := s.repository.GetNode(context.Background(), domain.ID(text(value.Input["node_id"])))
	if err != nil {
		return node, domain.Interface{}, "", err
	}
	iface, err := s.repository.GetInterface(context.Background(), domain.ID(text(value.Input["interface_id"])))
	return node, iface, text(value.Input["tap_name"]), err
}

func legacyTapName(id domain.ID) string {
	value := "nlt" + string(id)
	if len(value) > 15 {
		value = value[:15]
	}
	return value
}

func text(value any) string  { result, _ := value.(string); return result }
func number(value any) int64 { result, _ := value.(float64); return int64(result) }
