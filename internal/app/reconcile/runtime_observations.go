package reconcile

import (
	"context"
	"fmt"
	"time"

	"github.com/netlab/netlab/internal/domain"
)

type RuntimeObservationStore interface {
	ListAllNodes(context.Context) ([]domain.Node, error)
	ObserveRuntimeCapability(context.Context, domain.RuntimeCapabilityObservation) (domain.RuntimeCapabilityObservation, error)
}

type CapabilityProber interface {
	Probe(context.Context, domain.Node) ([]domain.RuntimeCapabilityObservation, error)
}

type CapabilityProbeFunc func(context.Context, domain.Node) ([]domain.RuntimeCapabilityObservation, error)

func (function CapabilityProbeFunc) Probe(ctx context.Context, node domain.Node) ([]domain.RuntimeCapabilityObservation, error) {
	return function(ctx, node)
}

type RuntimeObservationReconciler struct {
	store  RuntimeObservationStore
	prober CapabilityProber
}

func NewRuntimeObservationReconciler(store RuntimeObservationStore, prober CapabilityProber) *RuntimeObservationReconciler {
	return &RuntimeObservationReconciler{store: store, prober: prober}
}
func (r *RuntimeObservationReconciler) Name() string { return "runtime-capability-observations" }

func (r *RuntimeObservationReconciler) Reconcile(ctx context.Context) error {
	if r == nil || r.store == nil || r.prober == nil {
		return nil
	}
	nodes, err := r.store.ListAllNodes(ctx)
	if err != nil {
		return err
	}
	for _, node := range nodes {
		observations, probeErr := r.prober.Probe(ctx, node)
		if probeErr != nil {
			observations = []domain.RuntimeCapabilityObservation{{NodeID: node.ID, Capability: domain.CapabilityImage, State: domain.CapabilityFailed, Required: true, Problem: &domain.Problem{Code: "capability_probe_failed", Message: probeErr.Error(), Retryable: true, ResourceType: "node", ResourceID: node.ID, Phase: "capability_probe", Cleanup: "runtime unchanged", OperatorHint: "inspect runtime diagnostics and retry"}}}
		}
		for _, observation := range observations {
			observation.NodeID = node.ID
			observation.ObservedAt = time.Now().UTC()
			if _, err := r.store.ObserveRuntimeCapability(ctx, observation); err != nil {
				return fmt.Errorf("observe %s %s: %w", node.ID, observation.Capability, err)
			}
		}
	}
	return nil
}
