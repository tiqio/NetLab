package integration

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	httpapi "github.com/netlab/netlab/internal/api/http"
	"github.com/netlab/netlab/internal/app/reconcile"
	"github.com/netlab/netlab/internal/app/task"
	"github.com/netlab/netlab/internal/domain"
	"github.com/netlab/netlab/internal/support/observability"
)

func TestRequestCaptureAndTaskQueueLimits(t *testing.T) {
	server := httpapi.NewServer("127.0.0.1:0", slog.New(slog.NewTextHandler(io.Discard, nil)), &observability.Metrics{})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/missing", bytes.NewReader(make([]byte, 4<<20+1)))
	request.ContentLength = 4<<20 + 1
	response := httptest.NewRecorder()
	server.Engine().ServeHTTP(response, request)
	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}

	captures := reconcile.NewCaptureManager(t.TempDir(), 1, 1024, time.Hour)
	if _, err := captures.Start(context.Background(), reconcile.CaptureRequest{Interface: "tap0", MaxBytes: 2048}); err == nil {
		t.Fatal("global capture ceiling not enforced")
	}

	store := newLimitTaskStore()
	runner := task.NewRunner(store, 1, 1)
	defer runner.Close()
	blocked := make(chan struct{})
	started := make(chan struct{})
	var startedOnce sync.Once
	runner.Register("block", func(context.Context, *domain.OperationTask) (map[string]any, error) {
		startedOnce.Do(func() { close(started) })
		<-blocked
		return nil, nil
	})
	if err := runner.Enqueue(context.Background(), domain.OperationTask{Kind: "block", ResourceType: "test", ResourceID: "1"}); err != nil {
		t.Fatal(err)
	}
	<-started
	if err := runner.Enqueue(context.Background(), domain.OperationTask{Kind: "block", ResourceType: "test", ResourceID: "2"}); err != nil {
		t.Fatal(err)
	}
	if err := runner.Enqueue(context.Background(), domain.OperationTask{Kind: "block", ResourceType: "test", ResourceID: "3"}); err == nil {
		t.Fatal("task queue backpressure missing")
	}
	if store.count() != 2 {
		t.Fatalf("queued records=%d", store.count())
	}
	close(blocked)
}

type limitTaskStore struct {
	mu     sync.Mutex
	values map[domain.ID]domain.OperationTask
}

func newLimitTaskStore() *limitTaskStore {
	return &limitTaskStore{values: map[domain.ID]domain.OperationTask{}}
}
func (s *limitTaskStore) CreateTask(_ context.Context, value domain.OperationTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[value.ID] = value
	return nil
}
func (s *limitTaskStore) UpdateTask(_ context.Context, value domain.OperationTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[value.ID] = value
	return nil
}
func (s *limitTaskStore) GetTask(_ context.Context, id domain.ID) (domain.OperationTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.values[id], nil
}
func (s *limitTaskStore) count() int { s.mu.Lock(); defer s.mu.Unlock(); return len(s.values) }
