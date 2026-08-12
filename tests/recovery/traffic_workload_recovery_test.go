package recovery_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/app/ports"
	"github.com/netlab/netlab/internal/app/reconcile"
	"github.com/netlab/netlab/internal/domain"
)

type recoveryWorkloadStore struct {
	mu    sync.Mutex
	value domain.TrafficWorkload
}

func (s *recoveryWorkloadStore) ListAllTrafficWorkloads(context.Context) ([]domain.TrafficWorkload, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return []domain.TrafficWorkload{s.value}, nil
}
func (s *recoveryWorkloadStore) GetTrafficWorkload(context.Context, domain.ID) (domain.TrafficWorkload, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.value, nil
}
func (s *recoveryWorkloadStore) UpdateTrafficWorkloadState(_ context.Context, _ domain.ID, _ domain.Revision, desired, observed string, p *domain.Problem, _ domain.ID) (domain.TrafficWorkload, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.value.Revision++
	s.value.DesiredState = desired
	s.value.ObservedState = observed
	s.value.LastError = p
	return s.value, nil
}
func (s *recoveryWorkloadStore) RecordTrafficWorkloadOutcome(_ context.Context, _ domain.ID, success bool, bytes int64, p *domain.Problem) (domain.TrafficWorkload, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.value.Attempts++
	if success {
		s.value.Successes++
		s.value.MatchedBytes += bytes
	} else {
		s.value.Failures++
	}
	s.value.LastError = p
	return s.value, nil
}

type recoveryResolver struct{}

func (recoveryResolver) ResolveTrafficWorkloadTarget(context.Context, domain.TrafficWorkload) (ports.TrafficWorkloadTarget, error) {
	return ports.TrafficWorkloadTarget{Kind: "namespace", Namespace: "pc"}, nil
}

type recoveryExecutor struct{ called chan struct{} }

func (e recoveryExecutor) ExecuteTrafficWorkload(context.Context, domain.TrafficWorkload, ports.TrafficWorkloadTarget) (ports.TrafficWorkloadExecution, error) {
	e.called <- struct{}{}
	return ports.TrafficWorkloadExecution{MatchedBytes: 64}, nil
}
func TestRunningTrafficWorkloadResumesAfterServiceRestart(t *testing.T) {
	store := &recoveryWorkloadStore{value: domain.TrafficWorkload{ID: "w", LaboratoryID: "lab", Name: "ping", Revision: 4, Source: domain.TrafficWorkloadEndpoint{Kind: "network_object", ResourceID: "pc"}, Protocol: "icmp", AddressFamily: "ipv4", Destination: domain.TrafficWorkloadDestination{Address: "192.0.2.1"}, IntervalSeconds: 1, TimeoutSeconds: 1, DesiredState: "running", ObservedState: "running"}}
	called := make(chan struct{}, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	reconcile.NewTrafficWorkloadReconciler(store, recoveryResolver{}, recoveryExecutor{called}, nil).Start(ctx)
	select {
	case <-called:
	case <-time.After(time.Second):
		t.Fatal("running workload was not resumed")
	}
}
