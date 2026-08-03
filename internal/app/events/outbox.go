package events

import (
	"context"
	"errors"
	"time"

	"github.com/netlab/netlab/internal/domain"
)

var ErrReplayExpired = errors.New("event replay window expired")

const (
	EventNetworkObjectLinkCreated      = "network_object_link.created"
	EventNetworkObjectLinkStateChanged = "network_object_link.state_changed"
	EventNetworkObjectLinkRecovered    = "network_object_link.recovered"
	EventNetworkObjectLinkDeleted      = "network_object_link.deleted"
	EventCaptureStateChanged           = "capture.state_changed"
	EventCaptureCompleted              = "capture.completed"
	EventTrafficFilterObservation      = "traffic_filter.observation"
)

func CaptureEvent(value domain.Capture, taskID domain.ID) domain.OutboxEvent {
	eventType := EventCaptureStateChanged
	if value.FinishedAt != nil || value.State == "completed" || value.State == "cancelled" || value.State == "failed" || value.State == "truncated" {
		eventType = EventCaptureCompleted
	}
	return domain.OutboxEvent{
		Type: eventType, LaboratoryID: value.LaboratoryID, ResourceType: "capture", ResourceID: value.ID, TaskID: taskID,
		Data:       map[string]any{"capture": value, "source_type": value.SourceType, "source_id": value.SourceID, "completion_reason": value.CompletionReason},
		OccurredAt: time.Now().UTC(),
	}
}

func TrafficObservationEvent(filter domain.TrafficFilter, observation domain.TrafficObservation, taskID domain.ID) domain.OutboxEvent {
	return domain.OutboxEvent{
		Type: EventTrafficFilterObservation, LaboratoryID: filter.LaboratoryID, ResourceType: observation.ResourceType, ResourceID: observation.ResourceID, TaskID: taskID,
		Data: map[string]any{"traffic_filter_id": filter.ID, "observation": observation, "color": filter.Color}, OccurredAt: time.Now().UTC(),
	}
}

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
