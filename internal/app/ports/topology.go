package ports

import (
	"context"

	"github.com/netlab/netlab/internal/domain"
)

type LinkReconnectRepository interface {
	GetLink(context.Context, domain.ID) (domain.Link, error)
	GetInterface(context.Context, domain.ID) (domain.Interface, error)
	CommitLinkReconnect(context.Context, domain.ID, domain.Revision, domain.ID, domain.ID) (domain.Link, error)
}

type LinkReconnectRuntime interface {
	EnsureLink(context.Context, domain.Link, domain.Interface, domain.Interface) error
}
