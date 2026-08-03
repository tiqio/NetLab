package query

import (
	"context"
	"github.com/netlab/netlab/internal/domain"
)

type LaboratoryReader interface {
	ListLaboratories(context.Context) ([]domain.Laboratory, error)
	Snapshot(context.Context, domain.ID) (domain.TopologySnapshot, error)
}
type LaboratoryService struct{ reader LaboratoryReader }

func NewLaboratoryService(reader LaboratoryReader) *LaboratoryService {
	return &LaboratoryService{reader: reader}
}
func (s *LaboratoryService) List(ctx context.Context) ([]domain.Laboratory, error) {
	return s.reader.ListLaboratories(ctx)
}
func (s *LaboratoryService) Snapshot(ctx context.Context, id domain.ID) (domain.TopologySnapshot, error) {
	snapshot, err := s.reader.Snapshot(ctx, id)
	if err != nil {
		return snapshot, err
	}
	snapshot.Placements = ResolveTopologyPlacements(snapshot)
	return snapshot, nil
}
