package contract

import (
	"context"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/netlab/netlab/internal/api/mcp"
	"github.com/netlab/netlab/internal/domain"
)

type reconnectMCPStub struct{ calls int }

func (s *reconnectMCPStub) Reconnect(_ context.Context, _ domain.ID, _ domain.Revision, _ domain.ID, _ domain.ID, _ string) (domain.OperationTask, error) {
	s.calls++
	return domain.OperationTask{ID: "task-1", Kind: "link.reconnect", State: domain.TaskQueued}, nil
}

func TestReconnectLinkMCPParity(t *testing.T) {
	service := &reconnectMCPStub{}
	tools := mcp.LinkReconnectTools(service)
	if len(tools) != 1 || tools[0].Name != "netlab.links.reconnect" {
		t.Fatalf("tools=%+v", tools)
	}
	value, err := tools[0].Handler(&gin.Context{}, map[string]any{"link_id": "link-1", "expected_revision": float64(1), "retained_endpoint_id": "if-a", "replacement_endpoint_id": "if-c", "idempotency_key": "reconnect-key"})
	if err != nil || service.calls != 1 || value == nil {
		t.Fatalf("value=%+v calls=%d err=%v", value, service.calls, err)
	}
}
