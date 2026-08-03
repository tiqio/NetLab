package query

import (
	"context"
	"errors"

	"github.com/netlab/netlab/internal/domain"
)

type RuntimeCapabilityRepository interface {
	ListRuntimeCapabilities(context.Context, domain.ID) ([]domain.RuntimeCapabilityObservation, error)
}

type RuntimeCapabilityService struct{ repository RuntimeCapabilityRepository }

func NewRuntimeCapabilityService(repository RuntimeCapabilityRepository) *RuntimeCapabilityService {
	return &RuntimeCapabilityService{repository: repository}
}

func (s *RuntimeCapabilityService) Get(ctx context.Context, nodeID domain.ID) ([]domain.RuntimeCapabilityObservation, error) {
	values, err := s.repository.ListRuntimeCapabilities(ctx, nodeID)
	if errors.Is(err, domain.ErrNotFound) {
		return nil, domain.Problem{Code: domain.ProblemCodeNotFound, Message: "node not found", ResourceType: "node", ResourceID: nodeID, Phase: "query_capabilities", Cleanup: "no cleanup required", OperatorHint: "refresh the topology and choose an existing node"}
	}
	return values, err
}
