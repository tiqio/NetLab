package reconcile

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/netlab/netlab/internal/domain"
)

type RecoveryStore interface {
	ListAllNodes(context.Context) ([]domain.Node, error)
	PrepareHostRecovery(context.Context) error
}
type Recovery struct {
	store      RecoveryStore
	reconciler Reconciler
}

func NewRecovery(store RecoveryStore, reconciler Reconciler) *Recovery {
	return &Recovery{store: store, reconciler: reconciler}
}
func (r *Recovery) Adopt(ctx context.Context) error { return r.reconciler.Reconcile(ctx) }
func (r *Recovery) Restore(ctx context.Context) error {
	if err := r.store.PrepareHostRecovery(ctx); err != nil {
		return err
	}
	return r.reconciler.Reconcile(ctx)
}

func RunBounded(ctx context.Context, values []domain.Node, qemuLimit, otherLimit int, operation func(context.Context, domain.Node) error) error {
	if qemuLimit < 1 {
		qemuLimit = 1
	}
	if otherLimit < 1 {
		otherLimit = 1
	}
	qemuSlots := make(chan struct{}, qemuLimit)
	otherSlots := make(chan struct{}, otherLimit)
	var wait sync.WaitGroup
	errorsChannel := make(chan error, len(values))
	for _, value := range values {
		value := value
		slots := otherSlots
		if value.Kind == "qemu" {
			slots = qemuSlots
		}
		wait.Add(1)
		go func() {
			defer wait.Done()
			select {
			case slots <- struct{}{}:
			case <-ctx.Done():
				errorsChannel <- ctx.Err()
				return
			}
			defer func() { <-slots }()
			if err := operation(ctx, value); err != nil {
				errorsChannel <- fmt.Errorf("recover %s: %w", value.ID, err)
			}
		}()
	}
	wait.Wait()
	close(errorsChannel)
	for err := range errorsChannel {
		if err != nil {
			return err
		}
	}
	return nil
}

func QuarantineOrphans(runtimeRoot string, known map[string]bool) ([]string, error) {
	entries, err := os.ReadDir(runtimeRoot)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	quarantine := filepath.Join(runtimeRoot, "quarantine")
	var moved []string
	for _, entry := range entries {
		if entry.Name() == "quarantine" || known[entry.Name()] {
			continue
		}
		if err = os.MkdirAll(quarantine, 0o700); err != nil {
			return moved, err
		}
		destination := filepath.Join(quarantine, entry.Name())
		if err = os.Rename(filepath.Join(runtimeRoot, entry.Name()), destination); err != nil {
			return moved, err
		}
		moved = append(moved, destination)
	}
	return moved, nil
}
