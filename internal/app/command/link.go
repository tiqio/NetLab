package command

import (
	"context"
	"fmt"
	"github.com/netlab/netlab/internal/domain"
)

type LinkRepository interface {
	CreateLink(context.Context, domain.Link) error
	DeleteLink(context.Context, domain.ID) error
}
type LinkService struct{ repository LinkRepository }

func NewLinkService(repository LinkRepository) *LinkService {
	return &LinkService{repository: repository}
}
func (s *LinkService) Connect(ctx context.Context, labID, a, b domain.ID) (domain.Link, error) {
	if a == b {
		return domain.Link{}, fmt.Errorf("link endpoints must differ")
	}
	link := domain.Link{ID: domain.NewID(), LaboratoryID: labID, EndpointAID: a, EndpointBID: b, Revision: 1, DesiredState: "connected", ObservedState: "pending"}
	return link, s.repository.CreateLink(ctx, link)
}
func (s *LinkService) Disconnect(ctx context.Context, id domain.ID) error {
	if repository, ok := s.repository.(interface {
		MarkLinkDisconnected(context.Context, domain.ID) error
	}); ok {
		return repository.MarkLinkDisconnected(ctx, id)
	}
	return s.repository.DeleteLink(ctx, id)
}
