package events

import (
	"context"
	"errors"

	"github.com/netlab/netlab/internal/domain"
)

var ErrReplayExpired = errors.New("event replay window expired")

type Store interface {
	OutboxAfter(context.Context, int64, int) ([]domain.OutboxEvent, error)
}

type Publisher struct {
	store        Store
	retainedFrom int64
}

func NewPublisher(store Store) *Publisher           { return &Publisher{store: store} }
func (p *Publisher) SetRetainedFrom(sequence int64) { p.retainedFrom = sequence }
func (p *Publisher) Replay(ctx context.Context, after int64, limit int) ([]domain.OutboxEvent, error) {
	if after > 0 && after < p.retainedFrom {
		return nil, ErrReplayExpired
	}
	if limit < 1 || limit > 1000 {
		limit = 100
	}
	return p.store.OutboxAfter(ctx, after, limit)
}
