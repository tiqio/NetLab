package command_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/app/command"
	storesqlite "github.com/netlab/netlab/internal/store/sqlite"
)

func TestIdempotencyReplayConflictExpiryAndConcurrency(t *testing.T) {
	database, err := storesqlite.Open(context.Background(), "file:idempotency?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	service := command.NewIdempotencyService(storesqlite.NewRepositories(database), 20*time.Millisecond)
	var calls atomic.Int32
	operation := func(context.Context) (int, []byte, error) {
		calls.Add(1)
		return 201, []byte(`{"id":"one"}`), nil
	}
	first, err := service.Execute(context.Background(), "create", "abcdefgh", []byte(`{"name":"one"}`), operation)
	if err != nil || first.Replay {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	replay, err := service.Execute(context.Background(), "create", "abcdefgh", []byte(`{"name":"one"}`), operation)
	if err != nil || !replay.Replay || calls.Load() != 1 {
		t.Fatalf("replay=%+v calls=%d err=%v", replay, calls.Load(), err)
	}
	if _, err = service.Execute(context.Background(), "create", "abcdefgh", []byte(`{"name":"other"}`), operation); err != command.ErrIdempotencyConflict {
		t.Fatalf("conflict=%v", err)
	}
	time.Sleep(25 * time.Millisecond)
	if _, err = service.Execute(context.Background(), "create", "abcdefgh", []byte(`{"name":"other"}`), operation); err != nil || calls.Load() != 2 {
		t.Fatalf("expiry calls=%d err=%v", calls.Load(), err)
	}

	var concurrentCalls atomic.Int32
	concurrentOperation := func(context.Context) (int, []byte, error) {
		concurrentCalls.Add(1)
		time.Sleep(5 * time.Millisecond)
		return 202, []byte(`{"task":"one"}`), nil
	}
	var wait sync.WaitGroup
	for range 8 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			if _, executeErr := service.Execute(context.Background(), "start", "concurrent-key", []byte(`{"node":"one"}`), concurrentOperation); executeErr != nil {
				t.Errorf("concurrent: %v", executeErr)
			}
		}()
	}
	wait.Wait()
	if concurrentCalls.Load() != 1 {
		t.Fatalf("concurrent operation calls=%d", concurrentCalls.Load())
	}
}
