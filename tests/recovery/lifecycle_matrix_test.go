package recovery

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/netlab/netlab/internal/app/ports"
	"github.com/netlab/netlab/internal/app/reconcile"
	"github.com/netlab/netlab/internal/domain"
)

func TestServiceRestartLifecycleMatrixAndOrphanQuarantine(t *testing.T) {
	for _, desired := range []domain.DesiredState{domain.DesiredRunning, domain.DesiredStopped} {
		store := &matrixStore{node: domain.Node{ID: "node-1", Kind: "pc", DesiredState: desired, ObservedState: domain.ObservedUnknown}}
		runtime := &matrixRuntime{state: domain.ObservedStopped}
		if desired == domain.DesiredStopped {
			runtime.state = domain.ObservedRunning
		}
		if err := reconcile.NewTopologyReconciler(store, runtime).Reconcile(context.Background()); err != nil {
			t.Fatal(err)
		}
		if desired == domain.DesiredRunning && runtime.starts != 1 {
			t.Fatalf("starts=%d", runtime.starts)
		}
		if desired == domain.DesiredStopped && runtime.stops != 1 {
			t.Fatalf("stops=%d", runtime.stops)
		}
	}
	root := t.TempDir()
	_ = os.Mkdir(filepath.Join(root, "known"), 0700)
	_ = os.Mkdir(filepath.Join(root, "orphan"), 0700)
	moved, err := reconcile.QuarantineOrphans(root, map[string]bool{"known": true})
	if err != nil {
		t.Fatal(err)
	}
	if len(moved) != 1 || filepath.Base(moved[0]) != "orphan" {
		t.Fatalf("moved=%v", moved)
	}
}

type matrixStore struct{ node domain.Node }

func (s *matrixStore) ListAllNodes(context.Context) ([]domain.Node, error) {
	return []domain.Node{s.node}, nil
}
func (s *matrixStore) SetNodeObservedState(_ context.Context, _ domain.ID, state domain.ObservedState, _ *domain.Problem) error {
	s.node.ObservedState = state
	return nil
}

type matrixRuntime struct {
	state         domain.ObservedState
	starts, stops int
}

func (r *matrixRuntime) Inspect(context.Context, domain.Node) (ports.ActualNode, error) {
	return ports.ActualNode{State: r.state}, nil
}
func (r *matrixRuntime) Start(context.Context, domain.Node) error {
	r.starts++
	r.state = domain.ObservedRunning
	return nil
}
func (r *matrixRuntime) Stop(context.Context, domain.Node) error {
	r.stops++
	r.state = domain.ObservedStopped
	return nil
}
func (r *matrixRuntime) Delete(context.Context, domain.Node) error { return nil }
