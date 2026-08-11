package reconcile

import (
	"context"

	"github.com/netlab/netlab/internal/app/command"
	"github.com/netlab/netlab/internal/domain"
)

type TopologyConnectionRecoveryStore interface {
	RecoverTopologyConnectionReservations(context.Context) ([]domain.TopologyConnectionRecoveryOutcome, error)
}

type TopologyConnectionRecoveryReconciler struct {
	store TopologyConnectionRecoveryStore
}

func NewTopologyConnectionRecoveryReconciler(store TopologyConnectionRecoveryStore) *TopologyConnectionRecoveryReconciler {
	return &TopologyConnectionRecoveryReconciler{store: store}
}

func (r *TopologyConnectionRecoveryReconciler) Name() string { return "topology-connection-recovery" }

func (r *TopologyConnectionRecoveryReconciler) Reconcile(ctx context.Context) error {
	return r.ReconcileWithCheckpoints(ctx, func(RecoveryResourceOutcome) error { return nil })
}

func (r *TopologyConnectionRecoveryReconciler) ReconcileWithCheckpoints(ctx context.Context, checkpoint func(RecoveryResourceOutcome) error) error {
	outcomes, err := r.store.RecoverTopologyConnectionReservations(ctx)
	if err != nil {
		return err
	}
	for _, outcome := range outcomes {
		state := outcome.State
		if state == "" {
			state = "recovered"
		}
		if err = checkpoint(RecoveryResourceOutcome{ResourceType: outcome.ResourceType, ResourceID: outcome.ResourceID, State: state, Error: outcome.Error, Details: map[string]string{"action": outcome.Action}}); err != nil {
			return err
		}
	}
	return nil
}

func NewUnifiedTopologyConnectionService(repository command.TopologyConnectionRepository, links command.TopologyConnectionLinkOperations, networks command.TopologyConnectionNetworkOperations) *command.TopologyConnectionService {
	return command.NewTopologyConnectionService(repository, links, networks)
}
