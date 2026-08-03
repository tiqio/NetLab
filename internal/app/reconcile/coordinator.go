package reconcile

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

type Reconciler interface {
	Name() string
	Reconcile(context.Context) error
}
type Coordinator struct {
	interval    time.Duration
	logger      *slog.Logger
	reconcilers []Reconciler
	stop        chan struct{}
	wg          sync.WaitGroup
}

func NewCoordinator(interval time.Duration, logger *slog.Logger, reconcilers ...Reconciler) *Coordinator {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	return &Coordinator{interval: interval, logger: logger, reconcilers: reconcilers, stop: make(chan struct{})}
}
func (c *Coordinator) Start(ctx context.Context) {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		ticker := time.NewTicker(c.interval)
		defer ticker.Stop()
		c.run(ctx)
		for {
			select {
			case <-ctx.Done():
				return
			case <-c.stop:
				return
			case <-ticker.C:
				c.run(ctx)
			}
		}
	}()
}
func (c *Coordinator) run(ctx context.Context) {
	for _, r := range c.reconcilers {
		if err := r.Reconcile(ctx); err != nil && c.logger != nil {
			c.logger.Error("reconciliation failed", "reconciler", r.Name(), "error", err)
		}
	}
}
func (c *Coordinator) Close() { close(c.stop); c.wg.Wait() }
