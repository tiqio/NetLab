package contract

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	httpapi "github.com/netlab/netlab/internal/api/http"
	"github.com/netlab/netlab/internal/domain"
)

type reconnectCommandStub struct {
	linkID, retained, replacement domain.ID
	revision                      domain.Revision
}

func (s *reconnectCommandStub) Reconnect(_ context.Context, linkID domain.ID, revision domain.Revision, retained, replacement domain.ID, _ string) (domain.OperationTask, error) {
	s.linkID, s.revision, s.retained, s.replacement = linkID, revision, retained, replacement
	return domain.OperationTask{ID: "task-1", Kind: "link.reconnect", State: domain.TaskQueued}, nil
}

func TestReconnectLinkHTTPContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &reconnectCommandStub{}
	engine := gin.New()
	httpapi.NewLinkReconnectHandlers(service).Register(engine)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/links/link-1/reconnect", bytes.NewBufferString(`{"retained_endpoint_id":"if-a","replacement_endpoint_id":"if-c"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("If-Match", "4")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusAccepted || service.linkID != "link-1" || service.revision != 4 || service.retained != "if-a" || service.replacement != "if-c" {
		t.Fatalf("status=%d service=%+v body=%s", recorder.Code, service, recorder.Body.String())
	}
}
