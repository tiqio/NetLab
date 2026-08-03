package stream

import (
	"context"
	"github.com/netlab/netlab/internal/app/events"
	"github.com/netlab/netlab/internal/domain"
	"testing"
)

type eventStore struct{ events []domain.OutboxEvent }

func (s eventStore) OutboxAfter(_ context.Context, after int64, _ int) ([]domain.OutboxEvent, error) {
	var values []domain.OutboxEvent
	for _, event := range s.events {
		if event.Sequence > after {
			values = append(values, event)
		}
	}
	return values, nil
}

func TestReplayCursorContinuesAcrossPublisherReplacement(t *testing.T) {
	store := &eventStore{events: []domain.OutboxEvent{{Sequence: 10}, {Sequence: 11}}}
	first := events.NewPublisher(store)
	batch, err := first.Replay(context.Background(), 9, 100)
	if err != nil || len(batch) != 2 {
		t.Fatalf("batch=%+v err=%v", batch, err)
	}
	cursor := batch[len(batch)-1].Sequence
	store.events = append(store.events, domain.OutboxEvent{Sequence: 12})
	second := events.NewPublisher(store)
	batch, err = second.Replay(context.Background(), cursor, 100)
	if err != nil || len(batch) != 1 || batch[0].Sequence != 12 {
		t.Fatalf("batch=%+v err=%v", batch, err)
	}
}
func TestReplayBatchAndExpired(t *testing.T) {
	publisher := events.NewPublisher(eventStore{events: []domain.OutboxEvent{{Sequence: 1}, {Sequence: 2}}})
	count, err := ReplayBatch(context.Background(), publisher, 0)
	if err != nil || count != 2 {
		t.Fatalf("count=%d err=%v", count, err)
	}
	publisher.SetRetainedFrom(10)
	if _, err = ReplayBatch(context.Background(), publisher, 1); err == nil {
		t.Fatal("expected reset")
	}
}

func TestReplayPreservesOrderedNetworkObjectLinkLifecycle(t *testing.T) {
	publisher := events.NewPublisher(eventStore{events: []domain.OutboxEvent{
		{Sequence: 21, Type: EventNetworkObjectLinkCreated, ResourceType: "network_object_link", ResourceID: "link-1", Revision: 1},
		{Sequence: 22, Type: EventNetworkObjectLinkStateChanged, ResourceType: "network_object_link", ResourceID: "link-1", Revision: 1},
		{Sequence: 23, Type: EventNetworkObjectLinkRecovered, ResourceType: "network_object_link", ResourceID: "link-1", Revision: 1, TaskID: "recovery-task"},
	}})
	batch, err := publisher.Replay(context.Background(), 20, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(batch) != 3 || batch[0].Type != EventNetworkObjectLinkCreated || batch[1].Type != EventNetworkObjectLinkStateChanged || batch[2].Type != EventNetworkObjectLinkRecovered || batch[2].TaskID != "recovery-task" {
		t.Fatalf("batch=%+v", batch)
	}
}
