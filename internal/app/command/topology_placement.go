package command

import (
	"context"

	"github.com/netlab/netlab/internal/domain"
)

type TopologyPlacementRepository interface {
	UpdatePlacements(context.Context, domain.ID, domain.Revision, []domain.PlacementUpdate) (domain.Revision, []domain.TopologyPlacement, error)
}

type TopologyPlacementResult struct {
	LaboratoryRevision domain.Revision            `json:"laboratory_revision"`
	Placements         []domain.TopologyPlacement `json:"placements"`
}

type TopologyPlacementService struct{ repository TopologyPlacementRepository }

func NewTopologyPlacementService(repository TopologyPlacementRepository) *TopologyPlacementService {
	return &TopologyPlacementService{repository: repository}
}

func (s *TopologyPlacementService) Update(ctx context.Context, laboratoryID domain.ID, expectedRevision domain.Revision, updates []domain.PlacementUpdate) (TopologyPlacementResult, error) {
	if laboratoryID == "" {
		return TopologyPlacementResult{}, domain.Problem{Code: "invalid_request", Message: "laboratory id required", ResourceType: "laboratory"}
	}
	if expectedRevision < 1 {
		return TopologyPlacementResult{}, domain.Problem{Code: "precondition_required", Message: "valid laboratory revision required", ResourceType: "laboratory", ResourceID: laboratoryID}
	}
	if err := domain.ValidatePlacementBatch(updates); err != nil {
		return TopologyPlacementResult{}, domain.Problem{Code: "invalid_placement", Message: err.Error(), ResourceType: "laboratory", ResourceID: laboratoryID}
	}
	revision, placements, err := s.repository.UpdatePlacements(ctx, laboratoryID, expectedRevision, updates)
	if err != nil {
		return TopologyPlacementResult{}, err
	}
	return TopologyPlacementResult{LaboratoryRevision: revision, Placements: placements}, nil
}
