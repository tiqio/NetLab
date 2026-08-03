package reconcile

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/netlab/netlab/internal/domain"
)

type RecoveryTaskStore interface {
	CreateTask(context.Context, domain.OperationTask) error
	UpdateTask(context.Context, domain.OperationTask) error
}

type RecoveryCoordinator struct {
	store        RecoveryTaskStore
	participants []Reconciler
}

type RecoveryResourceOutcome struct {
	ResourceType string            `json:"resource_type"`
	ResourceID   domain.ID         `json:"resource_id"`
	State        string            `json:"state"`
	Error        string            `json:"error,omitempty"`
	RuntimeID    string            `json:"runtime_id,omitempty"`
	Details      map[string]string `json:"details,omitempty"`
}

type RecoveryCheckpointParticipant interface {
	ReconcileWithCheckpoints(context.Context, func(RecoveryResourceOutcome) error) error
}

func NewRecoveryCoordinator(store RecoveryTaskStore, participants ...Reconciler) *RecoveryCoordinator {
	return &RecoveryCoordinator{store: store, participants: participants}
}

func (c *RecoveryCoordinator) Execute(ctx context.Context, mode string, prepare func(context.Context) error) (domain.OperationTask, error) {
	now := time.Now().UTC()
	task := domain.OperationTask{ID: domain.NewID(), Kind: "system.recovery", ResourceType: "host", ResourceID: domain.ID(mode), State: domain.TaskRunning, ProgressTotal: len(c.participants) + 1, CreatedAt: now, StartedAt: &now}
	task.Result = map[string]any{"mode": mode, "completed_participants": []string{}}
	if err := c.store.CreateTask(ctx, task); err != nil {
		return task, err
	}
	fail := func(err error) (domain.OperationTask, error) {
		finished := time.Now().UTC()
		task.State = domain.TaskFailed
		task.FinishedAt = &finished
		task.Error = structuredProblem(err, domain.Problem{Code: "recovery_failed", Retryable: true, TaskID: task.ID, ResourceType: "host", ResourceID: domain.ID(mode), Phase: "recovery", Cleanup: "completed participant outcomes are retained", OperatorHint: "inspect failed participant outcomes and retry recovery", RetryAfterSeconds: 5})
		_ = c.store.UpdateTask(context.Background(), task)
		return task, err
	}
	if prepare != nil {
		if err := prepare(ctx); err != nil {
			return fail(err)
		}
	}
	task.ProgressCurrent = 1
	if err := c.store.UpdateTask(ctx, task); err != nil {
		return fail(err)
	}
	completed := make([]string, 0, len(c.participants))
	outcomes := make([]map[string]any, 0, len(c.participants))
	failures := make([]string, 0)
	for _, participant := range c.participants {
		var participantErr error
		if checkpointed, ok := participant.(RecoveryCheckpointParticipant); ok {
			participantErr = checkpointed.ReconcileWithCheckpoints(ctx, func(outcome RecoveryResourceOutcome) error {
				outcomes = append(outcomes, recoveryOutcomeMap(outcome))
				task.ProgressTotal++
				task.ProgressCurrent++
				task.Result = map[string]any{"mode": mode, "completed_participants": append([]string(nil), completed...), "resource_outcomes": append([]map[string]any(nil), outcomes...)}
				return c.store.UpdateTask(ctx, task)
			})
		} else {
			participantErr = participant.Reconcile(ctx)
		}
		if participantErr != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", participant.Name(), participantErr))
			outcomes = append(outcomes, map[string]any{"resource_type": participant.Name(), "state": "failed", "error": participantErr.Error()})
		} else {
			completed = append(completed, participant.Name())
			outcomes = append(outcomes, map[string]any{"resource_type": participant.Name(), "state": "recovered"})
		}
		task.ProgressCurrent++
		task.Result = map[string]any{"mode": mode, "completed_participants": append([]string(nil), completed...), "resource_outcomes": append([]map[string]any(nil), outcomes...)}
		if err := c.store.UpdateTask(ctx, task); err != nil {
			return fail(err)
		}
	}
	if len(failures) > 0 {
		return fail(fmt.Errorf("recovery completed with failures: %s", strings.Join(failures, "; ")))
	}
	finished := time.Now().UTC()
	task.State = domain.TaskSucceeded
	task.FinishedAt = &finished
	task.Result = map[string]any{"mode": mode, "participants": len(c.participants), "completed_participants": completed, "resource_outcomes": outcomes}
	return task, c.store.UpdateTask(ctx, task)
}

func recoveryOutcomeMap(outcome RecoveryResourceOutcome) map[string]any {
	value := map[string]any{"resource_type": outcome.ResourceType, "resource_id": outcome.ResourceID, "state": outcome.State}
	if outcome.Error != "" {
		value["error"] = outcome.Error
	}
	if outcome.RuntimeID != "" {
		value["runtime_id"] = outcome.RuntimeID
	}
	if len(outcome.Details) > 0 {
		value["details"] = outcome.Details
	}
	return value
}

type NetworkRecoveryReconciler struct {
	labs interface {
		ListLaboratories(context.Context) ([]domain.Laboratory, error)
	}
	networks *NetworkObjectService
}

func NewNetworkRecoveryReconciler(labs interface {
	ListLaboratories(context.Context) ([]domain.Laboratory, error)
}, networks *NetworkObjectService) *NetworkRecoveryReconciler {
	return &NetworkRecoveryReconciler{labs: labs, networks: networks}
}
func (r *NetworkRecoveryReconciler) Name() string { return "network-recovery" }
func (r *NetworkRecoveryReconciler) Reconcile(ctx context.Context) error {
	return r.ReconcileWithCheckpoints(ctx, func(RecoveryResourceOutcome) error { return nil })
}
func (r *NetworkRecoveryReconciler) ReconcileWithCheckpoints(ctx context.Context, checkpoint func(RecoveryResourceOutcome) error) error {
	values, err := r.labs.ListLaboratories(ctx)
	if err != nil {
		return err
	}
	for _, value := range values {
		if value.LifecycleState == "active" {
			if err = r.networks.RestoreLaboratoryWithCheckpoints(ctx, value.ID, checkpoint); err != nil {
				return err
			}
		}
	}
	return nil
}

type PortMappingRecoveryStore interface {
	ListAllPortMappings(context.Context) ([]domain.PortMapping, error)
	SetPortMappingState(context.Context, domain.ID, string, *domain.Problem) error
}
type PortMappingRecoveryRuntime interface {
	Apply(context.Context, domain.PortMapping) error
}
type PortMappingRecoveryReconciler struct {
	store   PortMappingRecoveryStore
	runtime PortMappingRecoveryRuntime
}

type CaptureRecoveryReconciler struct{ captures *CaptureManager }

func NewCaptureRecoveryReconciler(captures *CaptureManager) *CaptureRecoveryReconciler {
	return &CaptureRecoveryReconciler{captures: captures}
}
func (r *CaptureRecoveryReconciler) Name() string                    { return "capture-recovery" }
func (r *CaptureRecoveryReconciler) Reconcile(context.Context) error { return nil }
func (r *CaptureRecoveryReconciler) ReconcileWithCheckpoints(_ context.Context, checkpoint func(RecoveryResourceOutcome) error) error {
	if r.captures == nil {
		return nil
	}
	for _, capture := range r.captures.List() {
		state := "recovered"
		message := ""
		if capture.State == "failed" && capture.CompletionReason == "service_restart" {
			state = "reconnect_required"
			if capture.LastError != nil {
				message = capture.LastError.Message
			}
		} else if capture.State == "failed" {
			state = "failed"
			if capture.LastError != nil {
				message = capture.LastError.Message
			}
		}
		if err := checkpoint(RecoveryResourceOutcome{ResourceType: "capture", ResourceID: capture.ID, State: state, Error: message, RuntimeID: string(capture.ID), Details: map[string]string{"completion_reason": capture.CompletionReason}}); err != nil {
			return err
		}
	}
	return nil
}

func NewPortMappingRecoveryReconciler(store PortMappingRecoveryStore, runtime PortMappingRecoveryRuntime) *PortMappingRecoveryReconciler {
	return &PortMappingRecoveryReconciler{store: store, runtime: runtime}
}
func (r *PortMappingRecoveryReconciler) Name() string { return "port-mapping-recovery" }
func (r *PortMappingRecoveryReconciler) Reconcile(ctx context.Context) error {
	return r.ReconcileWithCheckpoints(ctx, func(RecoveryResourceOutcome) error { return nil })
}
func (r *PortMappingRecoveryReconciler) ReconcileWithCheckpoints(ctx context.Context, checkpoint func(RecoveryResourceOutcome) error) error {
	if r.runtime == nil {
		return nil
	}
	values, err := r.store.ListAllPortMappings(ctx)
	if err != nil {
		return err
	}
	for _, value := range values {
		if err = r.runtime.Apply(ctx, value); err != nil {
			problem := structuredProblem(err, domain.Problem{Code: "port_mapping_recovery_failed", Retryable: true, ResourceType: "port_mapping", ResourceID: value.ID, Phase: "recovery", Cleanup: "mapping record retained for retry", OperatorHint: "inspect host port conflicts and retry recovery", RetryAfterSeconds: 3})
			_ = r.store.SetPortMappingState(ctx, value.ID, "failed", problem)
			if checkpointErr := checkpoint(RecoveryResourceOutcome{ResourceType: "port_mapping", ResourceID: value.ID, State: "failed", Error: err.Error()}); checkpointErr != nil {
				return checkpointErr
			}
			continue
		}
		_ = r.store.SetPortMappingState(ctx, value.ID, "active", nil)
		if err = checkpoint(RecoveryResourceOutcome{ResourceType: "port_mapping", ResourceID: value.ID, State: "recovered"}); err != nil {
			return err
		}
	}
	return nil
}
