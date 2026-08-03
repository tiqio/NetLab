package query

import (
	"context"

	"github.com/netlab/netlab/internal/runtime/ownership"
)

type RuntimeOwnershipRepository interface {
	ListRuntimeOwnership(context.Context) ([]ownership.Record, error)
}

type RuntimeOwnershipService struct {
	repository RuntimeOwnershipRepository
}

func NewRuntimeOwnershipService(repository RuntimeOwnershipRepository) *RuntimeOwnershipService {
	return &RuntimeOwnershipService{repository: repository}
}

func (s *RuntimeOwnershipService) List(ctx context.Context) ([]ownership.Record, error) {
	return s.repository.ListRuntimeOwnership(ctx)
}
