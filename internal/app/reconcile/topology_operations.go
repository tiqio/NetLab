package reconcile

import (
	"context"

	"github.com/netlab/netlab/internal/domain"
)

type LinkReplacementDataPlane interface {
	EnsureLink(context.Context, domain.Link, domain.Interface, domain.Interface) error
	DeleteLink(context.Context, domain.ID) error
}

type TopologyOperations struct{ dataPlane LinkReplacementDataPlane }

func NewTopologyOperations(dataPlane LinkReplacementDataPlane) *TopologyOperations {
	return &TopologyOperations{dataPlane: dataPlane}
}

func (o *TopologyOperations) EnsureLink(ctx context.Context, link domain.Link, endpointA, endpointB domain.Interface) error {
	if err := o.dataPlane.DeleteLink(ctx, link.ID); err != nil {
		return err
	}
	return o.dataPlane.EnsureLink(ctx, link, endpointA, endpointB)
}
