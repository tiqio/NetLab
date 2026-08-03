package task

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/domain"
)

type recoveryStore struct {
	mu     sync.Mutex
	values map[domain.ID]domain.OperationTask
}

func (s *recoveryStore) CreateTask(_ context.Context, value domain.OperationTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[value.ID] = value
	return nil
}
func (s *recoveryStore) UpdateTask(_ context.Context, value domain.OperationTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[value.ID] = value
	return nil
}
func (s *recoveryStore) GetTask(_ context.Context, id domain.ID) (domain.OperationTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.values[id], nil
}
func (s *recoveryStore) ListRecoverableTasks(context.Context, int) ([]domain.OperationTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return []domain.OperationTask{s.values["task"]}, nil
}

func TestRecoverRequeuesInterruptedTask(t *testing.T) {
	store := &recoveryStore{values: map[domain.ID]domain.OperationTask{"task": {ID: "task", Kind: "recover", State: domain.TaskRunning, CreatedAt: time.Now()}}}
	runner := NewRunner(store, 1, 4)
	defer runner.Close()
	done := make(chan struct{})
	runner.Register("recover", func(context.Context, *domain.OperationTask) (map[string]any, error) { close(done); return nil, nil })
	if err := runner.Recover(context.Background()); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("recovered task did not run")
	}
}
