package reconcile

import (
	"context"
	"github.com/netlab/netlab/internal/app/ports"
	"github.com/netlab/netlab/internal/domain"
)

type TopologyStore interface {
	ListAllNodes(context.Context) ([]domain.Node, error)
	SetNodeObservedState(context.Context, domain.ID, domain.ObservedState, *domain.Problem) error
}
type TopologyReconciler struct {
	store   TopologyStore
	runtime ports.NodeRuntime
}

func NewTopologyReconciler(store TopologyStore, runtime ports.NodeRuntime) *TopologyReconciler {
	return &TopologyReconciler{store: store, runtime: runtime}
}
func (r *TopologyReconciler) Name() string { return "topology" }
func (r *TopologyReconciler) Reconcile(ctx context.Context) error {
	nodes, err := r.store.ListAllNodes(ctx)
	if err != nil {
		return err
	}
	for _, node := range nodes {
		actual, inspectErr := r.runtime.Inspect(ctx, node)
		if inspectErr != nil {
			problem := structuredProblem(inspectErr, nodeProblem(node, "inspect_failed", "inspecting", "runtime state left unchanged", "inspect the owned runtime and retry reconciliation", 2))
			_ = r.store.SetNodeObservedState(ctx, node.ID, domain.ObservedFailed, problem)
			continue
		}
		switch node.DesiredState {
		case domain.DesiredRunning:
			if actual.State != domain.ObservedRunning {
				_ = r.store.SetNodeObservedState(ctx, node.ID, domain.ObservedStarting, nil)
				if err = r.runtime.Start(ctx, node); err != nil {
					problem := structuredProblem(err, nodeProblem(node, "start_failed", "starting", "runtime adapter attempted partial-start cleanup", "inspect runtime logs and owned resources before retrying", 3))
					_ = r.store.SetNodeObservedState(ctx, node.ID, domain.ObservedFailed, problem)
					continue
				}
				actual.State = domain.ObservedRunning
			}
		case domain.DesiredStopped:
			if actual.State == domain.ObservedRunning {
				_ = r.store.SetNodeObservedState(ctx, node.ID, domain.ObservedStopping, nil)
				if err = r.runtime.Stop(ctx, node); err != nil {
					problem := structuredProblem(err, nodeProblem(node, "stop_failed", "stopping", "runtime and owned resources retained for retry", "inspect the runtime process and retry stopping", 3))
					_ = r.store.SetNodeObservedState(ctx, node.ID, domain.ObservedFailed, problem)
					continue
				}
				actual.State = domain.ObservedStopped
			}
		}
		if actual.State != node.ObservedState {
			_ = r.store.SetNodeObservedState(ctx, node.ID, actual.State, nil)
		}
	}
	return nil
}
