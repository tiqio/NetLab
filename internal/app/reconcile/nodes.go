package reconcile

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/netlab/netlab/internal/app/ports"
	"github.com/netlab/netlab/internal/domain"
)

type NodeProvisioner interface {
	Provision(context.Context, domain.Node) error
}

type NodePhaseTimeouts struct {
	Inspect   time.Duration
	Provision time.Duration
	Start     time.Duration
	Apply     time.Duration
	Stop      time.Duration
}

func defaultNodePhaseTimeouts() NodePhaseTimeouts {
	return NodePhaseTimeouts{Inspect: 10 * time.Second, Provision: 2 * time.Minute, Start: 30 * time.Second, Apply: 15 * time.Second, Stop: 30 * time.Second}
}

type RuntimeDispatch struct {
	QEMU        ports.NodeRuntime
	Docker      ports.NodeRuntime
	Lightweight ports.NodeRuntime
}

func (d RuntimeDispatch) For(node domain.Node) (ports.NodeRuntime, error) {
	switch node.Kind {
	case "qemu":
		if runtimeUnavailable(d.QEMU) {
			return nil, fmt.Errorf("qemu runtime unavailable")
		}
		return d.QEMU, nil
	case "docker":
		if runtimeUnavailable(d.Docker) {
			return nil, fmt.Errorf("docker runtime unavailable")
		}
		return d.Docker, nil
	default:
		if runtimeUnavailable(d.Lightweight) {
			return nil, fmt.Errorf("lightweight runtime unavailable")
		}
		return d.Lightweight, nil
	}
}

func runtimeUnavailable(runtime ports.NodeRuntime) bool {
	if runtime == nil {
		return true
	}
	value := reflect.ValueOf(runtime)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

type NodeReconciler struct {
	store      TopologyStore
	dispatch   RuntimeDispatch
	qemuLimit  int
	otherLimit int
	resources  interface {
		Admit(context.Context, domain.Node, []domain.Node) error
		Apply(context.Context, domain.Node) error
	}
	timeouts NodePhaseTimeouts
	nodes    []domain.Node
}

func NewNodeReconciler(store TopologyStore, dispatch RuntimeDispatch) *NodeReconciler {
	return &NodeReconciler{store: store, dispatch: dispatch, qemuLimit: 2, otherLimit: 4, timeouts: defaultNodePhaseTimeouts()}
}

func (r *NodeReconciler) SetConcurrency(qemuLimit, otherLimit int) {
	r.qemuLimit, r.otherLimit = qemuLimit, otherLimit
}
func (r *NodeReconciler) SetPhaseTimeouts(timeouts NodePhaseTimeouts) {
	defaults := defaultNodePhaseTimeouts()
	if timeouts.Inspect <= 0 {
		timeouts.Inspect = defaults.Inspect
	}
	if timeouts.Provision <= 0 {
		timeouts.Provision = defaults.Provision
	}
	if timeouts.Start <= 0 {
		timeouts.Start = defaults.Start
	}
	if timeouts.Apply <= 0 {
		timeouts.Apply = defaults.Apply
	}
	if timeouts.Stop <= 0 {
		timeouts.Stop = defaults.Stop
	}
	r.timeouts = timeouts
}
func (r *NodeReconciler) SetResources(resources interface {
	Admit(context.Context, domain.Node, []domain.Node) error
	Apply(context.Context, domain.Node) error
}) {
	r.resources = resources
}
func (r *NodeReconciler) Name() string { return "nodes" }
func (r *NodeReconciler) Reconcile(ctx context.Context) (err error) {
	defer normalizeTerminalError(&err, terminalProblem("reconciler", domain.ID(r.Name()), "node_reconcile"))
	nodes, err := r.store.ListAllNodes(ctx)
	if err != nil {
		return err
	}
	r.nodes = nodes
	return RunBounded(ctx, nodes, r.qemuLimit, r.otherLimit, r.reconcileNode)
}

func (r *NodeReconciler) ReconcileWithCheckpoints(ctx context.Context, checkpoint func(RecoveryResourceOutcome) error) (err error) {
	defer normalizeTerminalError(&err, terminalProblem("reconciler", domain.ID(r.Name()), "node_recovery"))
	nodes, err := r.store.ListAllNodes(ctx)
	if err != nil {
		return err
	}
	r.nodes = nodes
	var failures []string
	for _, node := range nodes {
		reconcileErr := r.reconcileNodeWithRecovery(ctx, node, true)
		outcome := RecoveryResourceOutcome{ResourceType: "node", ResourceID: node.ID, State: "recovered"}
		if reconcileErr != nil {
			outcome.State = "failed"
			outcome.Error = reconcileErr.Error()
			failures = append(failures, string(node.ID)+": "+reconcileErr.Error())
		} else if runtime, runtimeErr := r.dispatch.For(node); runtimeErr == nil {
			if actual, inspectErr := runtime.Inspect(ctx, node); inspectErr == nil {
				outcome.Details = cloneMetadata(actual.Owner)
				outcome.RuntimeID = stableRuntimeID(actual.Owner)
			}
		}
		if err = checkpoint(outcome); err != nil {
			return err
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("node recovery failures: %s", strings.Join(failures, "; "))
	}
	return nil
}

func stableRuntimeID(owner map[string]string) string {
	for _, key := range []string{"container_id", "pid", "netns", "namespace"} {
		if owner[key] != "" {
			return owner[key]
		}
	}
	return ""
}

func (r *NodeReconciler) reconcileNode(ctx context.Context, node domain.Node) error {
	return r.reconcileNodeWithRecovery(ctx, node, false)
}

func (r *NodeReconciler) reconcileNodeWithRecovery(ctx context.Context, node domain.Node, restoreMissingRuntime bool) error {
	runtime, err := r.dispatch.For(node)
	if err != nil {
		problem := structuredProblem(err, nodeProblem(node, "runtime_unavailable", "runtime_selection", "no runtime resources created", "enable the required runtime and retry", 5))
		_ = r.transition(ctx, &node, domain.ObservedFailed, problem)
		return nil
	}
	var actual ports.ActualNode
	startedNow := false
	err = runNodePhase(ctx, r.timeouts.Inspect, func(phaseCtx context.Context) error {
		var inspectErr error
		actual, inspectErr = runtime.Inspect(phaseCtx, node)
		return inspectErr
	})
	if err != nil {
		code := "inspect_failed"
		if errors.Is(err, context.DeadlineExceeded) {
			code = "inspect_timeout"
		}
		problem := structuredProblem(err, nodeProblem(node, code, "inspecting", "runtime state left unchanged", "inspect the owned runtime and retry reconciliation", 2))
		_ = r.transition(ctx, &node, domain.ObservedFailed, problem)
		return nil
	}
	if node.DesiredState == domain.DesiredRunning && actual.State != domain.ObservedRunning {
		if node.ObservedState == domain.ObservedRunning {
			problem := nodeProblem(node, "runtime_exited", "monitoring", "runtime already exited; owned resources retained for reconciliation", "inspect runtime logs and retry or stop the node", 2)
			problem.Message = "managed runtime is no longer running"
			if err = r.transition(ctx, &node, domain.ObservedFailed, &problem); err != nil || !restoreMissingRuntime {
				return err
			}
		}
		if r.resources != nil {
			if err = r.resources.Admit(ctx, node, r.nodes); err != nil {
				problem := structuredProblem(err, nodeProblem(node, "resource_exhausted", "resource_admission", "no runtime resources created", "free host capacity or reduce node resources and retry", 5))
				_ = r.transition(ctx, &node, domain.ObservedFailed, problem)
				return nil
			}
		}
		if err = r.transition(ctx, &node, domain.ObservedProvisioning, nil); err != nil {
			return err
		}
		if provisioner, ok := runtime.(NodeProvisioner); ok {
			err = runNodePhase(ctx, r.timeouts.Provision, func(phaseCtx context.Context) error { return provisioner.Provision(phaseCtx, node) })
			if err != nil {
				code := "provision_failed"
				if errors.Is(err, context.DeadlineExceeded) {
					code = "provision_timeout"
				}
				problem := structuredProblem(err, nodeProblem(node, code, "provisioning", "partial provisioning is retained only when ownership is durable; temporary files are removed", "inspect image provenance, storage capacity, and runtime ownership before retrying", 3))
				_ = r.transition(ctx, &node, domain.ObservedFailed, problem)
				return nil
			}
		}
		if err = r.transition(ctx, &node, domain.ObservedStarting, nil); err != nil {
			return err
		}
		err = runNodePhase(ctx, r.timeouts.Start, func(phaseCtx context.Context) error { return runtime.Start(phaseCtx, node) })
		if err != nil {
			problem := structuredProblem(err, startProblem(node, err))
			_ = r.transition(ctx, &node, domain.ObservedFailed, problem)
			return nil
		}
		actual.State = domain.ObservedRunning
		startedNow = true
	}
	if !startedNow && node.Kind == string(domain.RuntimeDocker) && node.DesiredState == domain.DesiredRunning && actual.State == domain.ObservedRunning {
		err = runNodePhase(ctx, r.timeouts.Apply, func(phaseCtx context.Context) error { return runtime.Start(phaseCtx, node) })
		if err != nil {
			code := "runtime_configuration_failed"
			if errors.Is(err, context.DeadlineExceeded) {
				code = "runtime_configuration_timeout"
			}
			problem := structuredProblem(err, nodeProblem(node, code, "runtime_configuration", "running container retained; endpoint and route ownership remain available for retry", "inspect the container namespace and retry reconciliation", 3))
			_ = r.transition(ctx, &node, domain.ObservedFailed, problem)
			return nil
		}
	}
	if node.DesiredState == domain.DesiredRunning && actual.State == domain.ObservedRunning && node.ObservedState != domain.ObservedRunning {
		if node.ObservedState != domain.ObservedStarting && node.ObservedState != domain.ObservedUnknown {
			if err = r.transition(ctx, &node, domain.ObservedStarting, nil); err != nil {
				return err
			}
		}
		if err = r.transition(ctx, &node, domain.ObservedRunning, nil); err != nil {
			return err
		}
	}
	if node.DesiredState == domain.DesiredRunning && actual.State == domain.ObservedRunning && r.resources != nil {
		if err = runNodePhase(ctx, r.timeouts.Apply, func(phaseCtx context.Context) error { return r.resources.Apply(phaseCtx, node) }); err != nil {
			code := "resource_apply_failed"
			if errors.Is(err, context.DeadlineExceeded) {
				code = "resource_apply_timeout"
			}
			problem := structuredProblem(err, nodeProblem(node, code, "resource_apply", "runtime remains running with its previous limits", "inspect cgroup ownership and retry reconciliation", 3))
			_ = r.transition(ctx, &node, domain.ObservedFailed, problem)
			return nil
		}
	}
	if node.DesiredState == domain.DesiredStopped && actual.State == domain.ObservedRunning {
		if node.ObservedState == domain.ObservedUnknown {
			if err = r.transition(ctx, &node, domain.ObservedRunning, nil); err != nil {
				return err
			}
		}
		if err = r.transition(ctx, &node, domain.ObservedStopping, nil); err != nil {
			return err
		}
		if err = runNodePhase(ctx, r.timeouts.Stop, func(phaseCtx context.Context) error { return runtime.Stop(phaseCtx, node) }); err != nil {
			code := "stop_failed"
			if errors.Is(err, context.DeadlineExceeded) {
				code = "stop_timeout"
			}
			problem := structuredProblem(err, nodeProblem(node, code, "stopping", "runtime and owned resources retained for retry", "inspect the runtime process and retry stopping", 3))
			_ = r.transition(ctx, &node, domain.ObservedFailed, problem)
			return nil
		}
		actual.State = domain.ObservedStopped
	}
	if node.DesiredState == domain.DesiredStopped && actual.State == domain.ObservedStopped && node.ObservedState != domain.ObservedStopped {
		if node.ObservedState != domain.ObservedUnknown && node.ObservedState != domain.ObservedStopping {
			if err = r.transition(ctx, &node, domain.ObservedStopping, nil); err != nil {
				return err
			}
		}
		return r.transition(ctx, &node, domain.ObservedStopped, nil)
	}
	return nil
}

func runNodePhase(ctx context.Context, timeout time.Duration, operation func(context.Context) error) error {
	phaseCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return operation(phaseCtx)
}

func startProblem(node domain.Node, err error) domain.Problem {
	code := "start_failed"
	cleanup := "runtime adapter attempted partial-start cleanup"
	hint := "inspect runtime logs and owned resources before retrying"
	message := strings.ToLower(err.Error())
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		code = "start_timeout"
		cleanup = "start cancellation requested; owned partial runtime is retained for discovery and cleanup validation"
	case strings.Contains(message, "qmp readiness timed out"):
		code = "qmp_readiness_timeout"
		cleanup = "QEMU process and created TAPs were removed; durable overlay and logs were retained"
		hint = "inspect qemu.log, machine arguments, and QMP socket permissions before retrying"
	case strings.Contains(message, "exited before qmp"):
		code = "runtime_early_exit"
		cleanup = "exited QEMU process and created TAPs were removed; durable overlay and logs were retained"
		hint = "inspect qemu.log for image, firmware, CPU, or device errors before retrying"
	}
	return nodeProblem(node, code, "starting", cleanup, hint, 3)
}

func nodeProblem(node domain.Node, code, phase, cleanup, hint string, retryAfter int) domain.Problem {
	return domain.Problem{Code: code, Retryable: true, ResourceType: "node", ResourceID: node.ID, Phase: phase, Cleanup: cleanup, OperatorHint: hint, RetryAfterSeconds: retryAfter}
}

func (r *NodeReconciler) transition(ctx context.Context, node *domain.Node, state domain.ObservedState, problem *domain.Problem) error {
	if node.ObservedState == state && problem == nil {
		return nil
	}
	if err := domain.ValidateNodeTransition(node.ObservedState, state); err != nil {
		return err
	}
	if err := r.store.SetNodeObservedState(ctx, node.ID, state, problem); err != nil {
		return err
	}
	node.ObservedState = state
	node.LastError = problem
	return nil
}
