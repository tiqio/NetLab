package reconcile

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/netlab/netlab/internal/app/ports"
	"github.com/netlab/netlab/internal/domain"
)

type TrafficWorkloadReconcilerRepository interface {
	ListAllTrafficWorkloads(context.Context) ([]domain.TrafficWorkload, error)
	GetTrafficWorkload(context.Context, domain.ID) (domain.TrafficWorkload, error)
	UpdateTrafficWorkloadState(context.Context, domain.ID, domain.Revision, string, string, *domain.Problem, domain.ID) (domain.TrafficWorkload, error)
	RecordTrafficWorkloadOutcome(context.Context, domain.ID, bool, int64, *domain.Problem) (domain.TrafficWorkload, error)
}

type TrafficWorkloadReconciler struct {
	repository TrafficWorkloadReconcilerRepository
	resolver   ports.TrafficWorkloadTargetResolver
	executor   ports.TrafficWorkloadExecutor
	filters    ports.TrafficWorkloadFilterCorrelator
	interval   time.Duration

	mu      sync.Mutex
	running map[domain.ID]context.CancelFunc
}

func NewTrafficWorkloadReconciler(repository TrafficWorkloadReconcilerRepository, resolver ports.TrafficWorkloadTargetResolver, executor ports.TrafficWorkloadExecutor, filters ports.TrafficWorkloadFilterCorrelator) *TrafficWorkloadReconciler {
	return &TrafficWorkloadReconciler{repository: repository, resolver: resolver, executor: executor, filters: filters, interval: time.Second, running: map[domain.ID]context.CancelFunc{}}
}

func (r *TrafficWorkloadReconciler) Start(ctx context.Context) {
	_ = r.Reconcile(ctx)
	ticker := time.NewTicker(r.interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				r.stopAll()
				return
			case <-ticker.C:
				_ = r.Reconcile(ctx)
			}
		}
	}()
}

func (r *TrafficWorkloadReconciler) Reconcile(ctx context.Context) error {
	values, err := r.repository.ListAllTrafficWorkloads(ctx)
	if err != nil {
		return err
	}
	seen := make(map[domain.ID]bool, len(values))
	for _, value := range values {
		seen[value.ID] = true
		if value.DesiredState == "running" {
			r.ensureRunning(ctx, value)
		} else {
			r.ensureStopped(ctx, value)
		}
	}
	r.mu.Lock()
	for id, cancel := range r.running {
		if !seen[id] {
			cancel()
			delete(r.running, id)
		}
	}
	r.mu.Unlock()
	return nil
}

func (r *TrafficWorkloadReconciler) ensureRunning(parent context.Context, workload domain.TrafficWorkload) {
	r.mu.Lock()
	if r.running[workload.ID] != nil {
		r.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(parent)
	r.running[workload.ID] = cancel
	r.mu.Unlock()
	go r.run(ctx, workload.ID)
}

func (r *TrafficWorkloadReconciler) ensureStopped(ctx context.Context, workload domain.TrafficWorkload) {
	r.mu.Lock()
	cancel := r.running[workload.ID]
	if cancel != nil {
		cancel()
		delete(r.running, workload.ID)
	}
	r.mu.Unlock()
	if workload.ObservedState != "stopped" {
		_, _ = r.repository.UpdateTrafficWorkloadState(ctx, workload.ID, workload.Revision, "stopped", "stopped", nil, "")
	}
}

func (r *TrafficWorkloadReconciler) run(ctx context.Context, id domain.ID) {
	defer func() {
		r.mu.Lock()
		delete(r.running, id)
		r.mu.Unlock()
	}()
	for {
		workload, err := r.repository.GetTrafficWorkload(ctx, id)
		if err != nil || workload.DesiredState != "running" {
			return
		}
		if workload.ObservedState != "running" {
			workload, err = r.repository.UpdateTrafficWorkloadState(ctx, id, workload.Revision, "running", "running", nil, "")
			if err != nil {
				return
			}
		}
		started := time.Now().UTC()
		target, resolveErr := r.resolver.ResolveTrafficWorkloadTarget(ctx, workload)
		var execution ports.TrafficWorkloadExecution
		if resolveErr == nil {
			execution, err = r.executor.ExecuteTrafficWorkload(ctx, workload, target)
		} else {
			err = resolveErr
		}
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			problem := domain.NormalizeProblem(err, domain.Problem{Code: "workload_exchange_failed", Message: err.Error(), Retryable: true, ResourceType: "traffic_workload", ResourceID: id, Phase: "workload_execute", Cleanup: "no owned background process remains"})
			_, _ = r.repository.RecordTrafficWorkloadOutcome(ctx, id, false, 0, &problem)
		} else {
			_, _ = r.repository.RecordTrafficWorkloadOutcome(ctx, id, true, execution.MatchedBytes, nil)
			if r.filters != nil {
				r.filters.CorrelateSuccessfulWorkload(workload, started)
			}
		}
		delay := time.Duration(workload.IntervalSeconds) * time.Second
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func (r *TrafficWorkloadReconciler) stopAll() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, cancel := range r.running {
		cancel()
		delete(r.running, id)
	}
}

func trafficWorkloadNotFound(err error) bool { return errors.Is(err, domain.ErrNotFound) }
