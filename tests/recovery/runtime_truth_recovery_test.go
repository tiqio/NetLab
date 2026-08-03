package recovery_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/netlab/netlab/internal/app/reconcile"
	"github.com/netlab/netlab/internal/domain"
)

type observationStoreFake struct {
	nodes        []domain.Node
	observations []domain.RuntimeCapabilityObservation
}

func TestAmbiguousOwnedRuntimeIsQuarantined(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "unknown-node"), 0o700); err != nil {
		t.Fatal(err)
	}
	moved, err := reconcile.QuarantineOrphans(root, map[string]bool{})
	if err != nil {
		t.Fatal(err)
	}
	if len(moved) != 1 || filepath.Base(moved[0]) != "unknown-node" {
		t.Fatalf("moved=%v", moved)
	}
}

func (s *observationStoreFake) ListAllNodes(context.Context) ([]domain.Node, error) {
	return s.nodes, nil
}
func (s *observationStoreFake) ObserveRuntimeCapability(_ context.Context, observation domain.RuntimeCapabilityObservation) (domain.RuntimeCapabilityObservation, error) {
	observation.Revision = 1
	s.observations = append(s.observations, observation)
	return observation, nil
}

func TestRuntimeObservationRecoveryRecordsProbeFailure(t *testing.T) {
	store := &observationStoreFake{nodes: []domain.Node{{ID: "node-1", Kind: "qemu"}}}
	reconciler := reconcile.NewRuntimeObservationReconciler(store, reconcile.CapabilityProbeFunc(func(context.Context, domain.Node) ([]domain.RuntimeCapabilityObservation, error) {
		return nil, errors.New("QMP unavailable")
	}))
	if err := reconciler.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(store.observations) != 1 || store.observations[0].State != domain.CapabilityFailed || store.observations[0].Problem == nil {
		t.Fatalf("observations=%#v", store.observations)
	}
}
