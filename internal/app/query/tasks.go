package query

import (
	"context"
	"time"

	"github.com/netlab/netlab/internal/domain"
)

type TaskRepository interface {
	GetTask(context.Context, domain.ID) (domain.OperationTask, error)
	ListTasks(context.Context, int) ([]domain.OperationTask, error)
	RequestTaskCancellation(context.Context, domain.ID, time.Time) (domain.OperationTask, error)
}

type TaskCanceller interface {
	Cancel(context.Context, domain.ID) error
}

type TaskService struct {
	repository TaskRepository
	canceller  TaskCanceller
}

func NewTaskService(repository TaskRepository, cancellers ...TaskCanceller) *TaskService {
	service := &TaskService{repository: repository}
	if len(cancellers) > 0 {
		service.canceller = cancellers[0]
	}
	return service
}

func (s *TaskService) Get(ctx context.Context, id domain.ID) (domain.OperationTask, error) {
	return s.repository.GetTask(ctx, id)
}

func (s *TaskService) List(ctx context.Context, limit int) ([]domain.OperationTask, error) {
	return s.repository.ListTasks(ctx, limit)
}

func (s *TaskService) Cancel(ctx context.Context, id domain.ID) (domain.OperationTask, error) {
	if _, err := s.repository.RequestTaskCancellation(ctx, id, time.Now().UTC()); err != nil {
		return domain.OperationTask{}, err
	}
	if s.canceller != nil {
		if err := s.canceller.Cancel(ctx, id); err != nil {
			return domain.OperationTask{}, err
		}
	}
	return s.repository.GetTask(ctx, id)
}
