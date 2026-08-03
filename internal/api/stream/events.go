package stream

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/coder/websocket"
	"github.com/netlab/netlab/internal/app/events"
	"net/http"
	"strconv"
	"time"
)

const (
	EventNetworkObjectObservedStateChanged = "network_object.observed_state_changed"
	EventNodeCapabilityChanged             = "node.capability_changed"
	EventNetworkObjectLinkCreated          = events.EventNetworkObjectLinkCreated
	EventNetworkObjectLinkStateChanged     = events.EventNetworkObjectLinkStateChanged
	EventNetworkObjectLinkRecovered        = events.EventNetworkObjectLinkRecovered
	EventNetworkObjectLinkDeleted          = events.EventNetworkObjectLinkDeleted
)

type EventHandler struct {
	publisher *events.Publisher
	poll      time.Duration
}

func NewEventHandler(publisher *events.Publisher) *EventHandler {
	return &EventHandler{publisher: publisher, poll: 250 * time.Millisecond}
}
func (h *EventHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	after, _ := strconv.ParseInt(r.URL.Query().Get("after"), 10, 64)
	connection, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: []string{"*"}})
	if err != nil {
		return
	}
	defer connection.CloseNow()
	ctx := r.Context()
	for {
		batch, err := h.publisher.Replay(ctx, after, 100)
		if errors.Is(err, events.ErrReplayExpired) {
			_ = connection.Write(ctx, websocket.MessageText, []byte(`{"type":"stream.reset_required"}`))
			_ = connection.Close(websocket.StatusNormalClosure, "snapshot required")
			return
		}
		if err != nil {
			_ = connection.Close(websocket.StatusInternalError, err.Error())
			return
		}
		for _, event := range batch {
			body, _ := json.Marshal(event)
			writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err = connection.Write(writeCtx, websocket.MessageText, body)
			cancel()
			if err != nil {
				return
			}
			after = event.Sequence
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(h.poll):
		}
	}
}
func ReplayBatch(ctx context.Context, publisher *events.Publisher, after int64) (int, error) {
	batch, err := publisher.Replay(ctx, after, 100)
	return len(batch), err
}
