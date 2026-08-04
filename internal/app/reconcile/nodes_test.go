package reconcile

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/app/ports"
	"github.com/netlab/netlab/internal/domain"
)

type lifecycleStore struct {
	node        domain.Node
	transitions []domain.ObservedState
	problems    []*domain.Problem
}

func (s *lifecycleStore) ListAllNodes(context.Context) ([]domain.Node, error) {
	return []domain.Node{s.node}, nil
}

func (s *lifecycleStore) SetNodeObservedState(_ context.Context, _ domain.ID, state domain.ObservedState, problem *domain.Problem) error {
	if err := domain.ValidateNodeTransition(s.node.ObservedState, state); err != nil {
		return err
	}
	s.node.ObservedState = state
	s.node.LastError = problem
	s.transitions = append(s.transitions, state)
	s.problems = append(s.problems, problem)
	return nil
}

type lifecycleRuntime struct {
	state        domain.ObservedState
	owner        map[string]string
	provisionErr error
	startErr     error
	stopErr      error
	startWait    bool
}

type countingDockerRuntime struct {
	state      domain.ObservedState
	startCalls int
	startErr   error
}

type restoringNamespaceRuntime struct {
	running bool
}

func (r *restoringNamespaceRuntime) Inspect(context.Context, domain.Node) (ports.ActualNode, error) {
	if !r.running {
		return ports.ActualNode{State: domain.ObservedStopped}, nil
	}
	return ports.ActualNode{State: domain.ObservedRunning, Owner: map[string]string{"netns": "nl-restored"}}, nil
}

func (r *restoringNamespaceRuntime) Start(context.Context, domain.Node) error {
	r.running = true
	return nil
}

func (r *restoringNamespaceRuntime) Stop(context.Context, domain.Node) error { return nil }
func (r *restoringNamespaceRuntime) Delete(context.Context, domain.Node) error {
	return nil
}

func (r *countingDockerRuntime) Inspect(context.Context, domain.Node) (ports.ActualNode, error) {
	return ports.ActualNode{State: r.state, Owner: map[string]string{"container_id": "container-1", "pid": "4242"}}, nil
}

func (r *countingDockerRuntime) Start(context.Context, domain.Node) error {
	r.startCalls++
	return r.startErr
}

func (r *countingDockerRuntime) Stop(context.Context, domain.Node) error { return nil }
func (r *countingDockerRuntime) Delete(context.Context, domain.Node) error {
	return nil
}

func (r lifecycleRuntime) Inspect(context.Context, domain.Node) (ports.ActualNode, error) {
	return ports.ActualNode{State: r.state, Owner: r.owner}, nil
}

func TestNodeRecoveryCheckpointIncludesStableRuntimeID(t *testing.T) {
	store := &lifecycleStore{node: domain.Node{ID: "node", Kind: "qemu", DesiredState: domain.DesiredRunning, ObservedState: domain.ObservedRunning}}
	reconciler := NewNodeReconciler(store, RuntimeDispatch{QEMU: lifecycleRuntime{state: domain.ObservedRunning, owner: map[string]string{"pid": "4242", "qmp": "/run/qmp.sock"}}})
	var outcomes []RecoveryResourceOutcome
	if err := reconciler.ReconcileWithCheckpoints(context.Background(), func(outcome RecoveryResourceOutcome) error { outcomes = append(outcomes, outcome); return nil }); err != nil {
		t.Fatal(err)
	}
	if len(outcomes) != 1 || outcomes[0].RuntimeID != "4242" || outcomes[0].Details["qmp"] != "/run/qmp.sock" {
		t.Fatalf("outcomes=%+v", outcomes)
	}
}

func TestNodeRecoveryRestoresMissingRuntimeBeforeCheckpoint(t *testing.T) {
	store := &lifecycleStore{node: domain.Node{ID: "node", Kind: "pc", DesiredState: domain.DesiredRunning, ObservedState: domain.ObservedRunning}}
	runtime := &restoringNamespaceRuntime{}
	reconciler := NewNodeReconciler(store, RuntimeDispatch{Lightweight: runtime})
	var outcomes []RecoveryResourceOutcome
	if err := reconciler.ReconcileWithCheckpoints(context.Background(), func(outcome RecoveryResourceOutcome) error {
		outcomes = append(outcomes, outcome)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(outcomes) != 1 || outcomes[0].State != "recovered" || outcomes[0].RuntimeID != "nl-restored" {
		t.Fatalf("outcomes=%+v", outcomes)
	}
	expected := []domain.ObservedState{domain.ObservedFailed, domain.ObservedProvisioning, domain.ObservedStarting, domain.ObservedRunning}
	if !reflect.DeepEqual(store.transitions, expected) {
		t.Fatalf("transitions=%v want=%v", store.transitions, expected)
	}
}
func (r lifecycleRuntime) Provision(context.Context, domain.Node) error { return r.provisionErr }
func (r lifecycleRuntime) Start(ctx context.Context, _ domain.Node) error {
	if r.startWait {
		<-ctx.Done()
		return ctx.Err()
	}
	return r.startErr
}
func (r lifecycleRuntime) Stop(context.Context, domain.Node) error   { return r.stopErr }
func (r lifecycleRuntime) Delete(context.Context, domain.Node) error { return nil }

func TestNodeReconcilerRecordsActionableStartFailure(t *testing.T) {
	store := &lifecycleStore{node: domain.Node{ID: "node", Kind: "qemu", DesiredState: domain.DesiredRunning, ObservedState: domain.ObservedStopped}}
	runtime := lifecycleRuntime{state: domain.ObservedStopped, startErr: errors.New("invalid qcow2")}
	if err := NewNodeReconciler(store, RuntimeDispatch{QEMU: runtime}).Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(store.transitions) != 3 || store.transitions[0] != domain.ObservedProvisioning || store.transitions[1] != domain.ObservedStarting || store.transitions[2] != domain.ObservedFailed {
		t.Fatalf("transitions=%v", store.transitions)
	}
	problem := store.problems[2]
	if problem == nil || problem.Code != "start_failed" || problem.Phase != "starting" || problem.Message != "invalid qcow2" || problem.ResourceID != "node" || problem.Cleanup == "" || problem.OperatorHint == "" || problem.RetryAfterSeconds == 0 {
		t.Fatalf("problem=%+v", problem)
	}
}

func TestNodeReconcilerPersistsProvisioningFailure(t *testing.T) {
	store := &lifecycleStore{node: domain.Node{ID: "node", Kind: "qemu", DesiredState: domain.DesiredRunning, ObservedState: domain.ObservedStopped}}
	runtime := lifecycleRuntime{state: domain.ObservedStopped, provisionErr: errors.New("invalid qcow2 backing image")}
	if err := NewNodeReconciler(store, RuntimeDispatch{QEMU: runtime}).Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(store.transitions) != 2 || store.transitions[0] != domain.ObservedProvisioning || store.transitions[1] != domain.ObservedFailed {
		t.Fatalf("transitions=%v", store.transitions)
	}
	problem := store.problems[1]
	if problem == nil || problem.Code != "provision_failed" || problem.Phase != "provisioning" || problem.Cleanup == "" || problem.OperatorHint == "" {
		t.Fatalf("problem=%+v", problem)
	}
}

func TestNodeReconcilerBoundsStartPhase(t *testing.T) {
	store := &lifecycleStore{node: domain.Node{ID: "node", Kind: "qemu", DesiredState: domain.DesiredRunning, ObservedState: domain.ObservedStopped}}
	reconciler := NewNodeReconciler(store, RuntimeDispatch{QEMU: lifecycleRuntime{state: domain.ObservedStopped, startWait: true}})
	reconciler.SetPhaseTimeouts(NodePhaseTimeouts{Inspect: time.Second, Provision: time.Second, Start: 10 * time.Millisecond, Apply: time.Second, Stop: time.Second})
	if err := reconciler.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	problem := store.problems[len(store.problems)-1]
	if problem == nil || problem.Code != "start_timeout" || problem.Phase != "starting" || !problem.Retryable || problem.Cleanup == "" || problem.OperatorHint == "" {
		t.Fatalf("problem=%+v", problem)
	}
}

func TestNodeReconcilerPersistsSuccessfulStartCheckpoints(t *testing.T) {
	store := &lifecycleStore{node: domain.Node{ID: "node", Kind: "qemu", DesiredState: domain.DesiredRunning, ObservedState: domain.ObservedStopped}}
	if err := NewNodeReconciler(store, RuntimeDispatch{QEMU: lifecycleRuntime{state: domain.ObservedStopped}}).Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	expected := []domain.ObservedState{domain.ObservedProvisioning, domain.ObservedStarting, domain.ObservedRunning}
	if len(store.transitions) != len(expected) {
		t.Fatalf("transitions=%v", store.transitions)
	}
	for index := range expected {
		if store.transitions[index] != expected[index] {
			t.Fatalf("transitions=%v", store.transitions)
		}
	}
}

func TestNodeReconcilerReappliesRunningDockerConfigurationDuringRecovery(t *testing.T) {
	store := &lifecycleStore{node: domain.Node{ID: "node", Kind: "docker", DesiredState: domain.DesiredRunning, ObservedState: domain.ObservedRunning}}
	runtime := &countingDockerRuntime{state: domain.ObservedRunning}
	if err := NewNodeReconciler(store, RuntimeDispatch{Docker: runtime}).ReconcileWithCheckpoints(context.Background(), func(RecoveryResourceOutcome) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if runtime.startCalls != 1 || len(store.transitions) != 0 {
		t.Fatalf("startCalls=%d transitions=%v", runtime.startCalls, store.transitions)
	}
}

func TestNodeReconcilerDoesNotReportRunningWhenDockerConfigurationFails(t *testing.T) {
	store := &lifecycleStore{node: domain.Node{ID: "node", Kind: "docker", DesiredState: domain.DesiredRunning, ObservedState: domain.ObservedRunning}}
	runtime := &countingDockerRuntime{state: domain.ObservedRunning, startErr: errors.New("route gateway unavailable")}
	if err := NewNodeReconciler(store, RuntimeDispatch{Docker: runtime}).Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if runtime.startCalls != 1 || len(store.transitions) != 1 || store.transitions[0] != domain.ObservedFailed {
		t.Fatalf("startCalls=%d transitions=%v", runtime.startCalls, store.transitions)
	}
	problem := store.problems[0]
	if problem == nil || problem.Code != "runtime_configuration_failed" || problem.Phase != "runtime_configuration" || problem.OperatorHint == "" {
		t.Fatalf("problem=%+v", problem)
	}
}

func TestNodeReconcilerClassifiesQEMUStartFailures(t *testing.T) {
	for _, test := range []struct{ name, message, code string }{
		{name: "early exit", message: "QEMU exited before QMP became ready: invalid accelerator", code: "runtime_early_exit"},
		{name: "readiness timeout", message: "QMP readiness timed out: connection refused", code: "qmp_readiness_timeout"},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := &lifecycleStore{node: domain.Node{ID: "node", Kind: "qemu", DesiredState: domain.DesiredRunning, ObservedState: domain.ObservedStopped}}
			runtime := lifecycleRuntime{state: domain.ObservedStopped, startErr: errors.New(test.message)}
			if err := NewNodeReconciler(store, RuntimeDispatch{QEMU: runtime}).Reconcile(context.Background()); err != nil {
				t.Fatal(err)
			}
			problem := store.problems[len(store.problems)-1]
			if problem == nil || problem.Code != test.code || problem.Cleanup == "" || problem.OperatorHint == "" {
				t.Fatalf("problem=%+v", problem)
			}
		})
	}
}

func TestNodeReconcilerPersistsSuccessfulStopCheckpoints(t *testing.T) {
	store := &lifecycleStore{node: domain.Node{ID: "node", Kind: "qemu", DesiredState: domain.DesiredStopped, ObservedState: domain.ObservedRunning}}
	if err := NewNodeReconciler(store, RuntimeDispatch{QEMU: lifecycleRuntime{state: domain.ObservedRunning}}).Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	expected := []domain.ObservedState{domain.ObservedStopping, domain.ObservedStopped}
	if len(store.transitions) != len(expected) || store.transitions[0] != expected[0] || store.transitions[1] != expected[1] {
		t.Fatalf("transitions=%v", store.transitions)
	}
}

func TestNodeReconcilerDetectsUnexpectedRuntimeExit(t *testing.T) {
	store := &lifecycleStore{node: domain.Node{ID: "node", Kind: "qemu", DesiredState: domain.DesiredRunning, ObservedState: domain.ObservedRunning}}
	if err := NewNodeReconciler(store, RuntimeDispatch{QEMU: lifecycleRuntime{state: domain.ObservedStopped}}).Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(store.transitions) != 1 || store.transitions[0] != domain.ObservedFailed || store.problems[0].Code != "runtime_exited" {
		t.Fatalf("transitions=%v problems=%+v", store.transitions, store.problems)
	}
}

func TestNodeReconcilerRecordsActionableStopFailure(t *testing.T) {
	store := &lifecycleStore{node: domain.Node{ID: "node", Kind: "qemu", DesiredState: domain.DesiredStopped, ObservedState: domain.ObservedRunning}}
	runtime := lifecycleRuntime{state: domain.ObservedRunning, stopErr: errors.New("QMP unavailable")}
	if err := NewNodeReconciler(store, RuntimeDispatch{QEMU: runtime}).Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(store.transitions) != 2 || store.transitions[0] != domain.ObservedStopping || store.transitions[1] != domain.ObservedFailed || store.problems[1].Code != "stop_failed" || store.problems[1].Phase != "stopping" || store.problems[1].Cleanup == "" || store.problems[1].OperatorHint == "" || store.problems[1].RetryAfterSeconds == 0 {
		t.Fatalf("transitions=%v problems=%+v", store.transitions, store.problems)
	}
}
