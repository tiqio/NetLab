package httpapi

import (
	"context"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/support/observability"
)

func TestStartReadySignalsOnlyAfterListenerBind(t *testing.T) {
	server := NewServer("127.0.0.1:0", nil, &observability.Metrics{})
	ready := make(chan struct{})
	finished := make(chan error, 1)
	go func() {
		finished <- server.StartReady(func() error {
			close(ready)
			return nil
		})
	}()
	select {
	case <-ready:
	case <-time.After(time.Second):
		t.Fatal("listener did not report readiness")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
	if err := <-finished; err != nil {
		t.Fatal(err)
	}
}
