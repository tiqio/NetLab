package ports

import (
	"context"

	"github.com/netlab/netlab/internal/domain"
)

type ObservationLocator struct {
	ResourceType  string
	ResourceID    domain.ID
	NamespaceName string
	InterfaceName string
	Orientation   string
}

type ObservationLocatorResolver interface {
	ResolveObservationLocator(context.Context, string, domain.ID) (ObservationLocator, error)
}

type ManagedRouteOwner struct {
	NodeID        domain.ID
	ProcessID     int
	InterfaceName string
}

type ExactManagedRouteRuntime interface {
	ReplaceManagedRoutes(context.Context, ManagedRouteOwner, []domain.RouteConfig) error
	RemoveManagedRoutes(context.Context, ManagedRouteOwner) error
}

type LinkReconnectRepository interface {
	GetLink(context.Context, domain.ID) (domain.Link, error)
	GetInterface(context.Context, domain.ID) (domain.Interface, error)
	CommitLinkReconnect(context.Context, domain.ID, domain.Revision, domain.ID, domain.ID) (domain.Link, error)
}

type LinkReconnectRuntime interface {
	EnsureLink(context.Context, domain.Link, domain.Interface, domain.Interface) error
}
