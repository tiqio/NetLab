package console

import (
	"context"
	"io"
	"net"
	"testing"
	"time"
)

func TestBridgeCopiesBinaryFramesAndClosesWithoutLifecycleSideEffects(t *testing.T) {
	clientSide, proxyClient := net.Pipe()
	proxyBackend, backendSide := net.Pipe()
	done := make(chan error, 1)
	go func() {
		done <- Bridge(context.Background(), proxyClient, proxyBackend, Limits{IdleTimeout: time.Second, MaximumSession: time.Second})
	}()
	payload := []byte{0, 1, 2, 255, 'v', 'n', 'c'}
	go func() { _, _ = clientSide.Write(payload) }()
	received := make([]byte, len(payload))
	if _, err := io.ReadFull(backendSide, received); err != nil {
		t.Fatal(err)
	}
	if string(received) != string(payload) {
		t.Fatalf("received=%v", received)
	}
	_ = clientSide.Close()
	_ = backendSide.Close()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("proxy did not close")
	}
}

func TestBridgeAppliesBandwidthLimit(t *testing.T) {
	clientSide, proxyClient := net.Pipe()
	proxyBackend, backendSide := net.Pipe()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		_ = Bridge(ctx, proxyClient, proxyBackend, Limits{BytesPerSecond: 1000, MaximumSession: time.Second})
	}()
	payload := make([]byte, 100)
	started := time.Now()
	go func() { _, _ = clientSide.Write(payload) }()
	if _, err := io.ReadFull(backendSide, make([]byte, len(payload))); err != nil {
		t.Fatal(err)
	}
	if time.Since(started) < 80*time.Millisecond {
		t.Fatal("bandwidth limit was not applied")
	}
	_ = clientSide.Close()
	_ = backendSide.Close()
}
