package reconcile

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/app/ports"
	"github.com/netlab/netlab/internal/domain"
)

type workloadReconcileRepository struct {
	mu       sync.Mutex
	workload domain.TrafficWorkload
}

func (r *workloadReconcileRepository) ListAllTrafficWorkloads(context.Context) ([]domain.TrafficWorkload, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return []domain.TrafficWorkload{r.workload}, nil
}
func (r *workloadReconcileRepository) GetTrafficWorkload(context.Context, domain.ID) (domain.TrafficWorkload, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.workload, nil
}
func (r *workloadReconcileRepository) UpdateTrafficWorkloadState(_ context.Context, _ domain.ID, expected domain.Revision, desired, observed string, problem *domain.Problem, _ domain.ID) (domain.TrafficWorkload, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.workload.Revision != expected {
		return r.workload, errors.New("revision conflict")
	}
	r.workload.Revision++
	r.workload.DesiredState, r.workload.ObservedState, r.workload.LastError = desired, observed, problem
	return r.workload, nil
}
func (r *workloadReconcileRepository) RecordTrafficWorkloadOutcome(_ context.Context, _ domain.ID, success bool, bytes int64, problem *domain.Problem) (domain.TrafficWorkload, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.workload.Attempts++
	if success {
		r.workload.Successes++
		r.workload.MatchedBytes += bytes
	} else {
		r.workload.Failures++
	}
	r.workload.LastError = problem
	return r.workload, nil
}

type workloadResolverFake struct{}

func (workloadResolverFake) ResolveTrafficWorkloadTarget(context.Context, domain.TrafficWorkload) (ports.TrafficWorkloadTarget, error) {
	return ports.TrafficWorkloadTarget{Kind: "namespace", Namespace: "test"}, nil
}

type workloadExecutorFake struct {
	err    error
	called chan struct{}
}

func (e workloadExecutorFake) ExecuteTrafficWorkload(context.Context, domain.TrafficWorkload, ports.TrafficWorkloadTarget) (ports.TrafficWorkloadExecution, error) {
	select {
	case e.called <- struct{}{}:
	default:
	}
	return ports.TrafficWorkloadExecution{MatchedBytes: 64}, e.err
}

func reconcilerWorkload() domain.TrafficWorkload {
	return domain.TrafficWorkload{ID: "workload", LaboratoryID: "lab", Name: "ping", Revision: 2, Source: domain.TrafficWorkloadEndpoint{Kind: "network_object", ResourceID: "pc"}, Protocol: "icmp", AddressFamily: "ipv4", Destination: domain.TrafficWorkloadDestination{Address: "192.0.2.1"}, IntervalSeconds: 1, TimeoutSeconds: 1, DesiredState: "running", ObservedState: "queued"}
}

func TestTrafficWorkloadReconcilerRecordsSuccessfulAndFailedAttempts(t *testing.T) {
	for _, test := range []struct {
		name                string
		err                 error
		successes, failures int64
	}{{"success", nil, 1, 0}, {"failure", errors.New("unreachable"), 0, 1}} {
		t.Run(test.name, func(t *testing.T) {
			repository := &workloadReconcileRepository{workload: reconcilerWorkload()}
			called := make(chan struct{}, 1)
			reconciler := NewTrafficWorkloadReconciler(repository, workloadResolverFake{}, workloadExecutorFake{err: test.err, called: called}, nil)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if err := reconciler.Reconcile(ctx); err != nil {
				t.Fatal(err)
			}
			select {
			case <-called:
			case <-time.After(time.Second):
				t.Fatal("workload did not execute")
			}
			deadline := time.Now().Add(time.Second)
			for time.Now().Before(deadline) {
				repository.mu.Lock()
				value := repository.workload
				repository.mu.Unlock()
				if value.Attempts == 1 {
					if value.Successes != test.successes || value.Failures != test.failures {
						t.Fatalf("workload=%+v", value)
					}
					return
				}
				time.Sleep(5 * time.Millisecond)
			}
			t.Fatal("outcome not recorded")
		})
	}
}
