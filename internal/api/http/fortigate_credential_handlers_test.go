package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/netlab/netlab/internal/domain"
)

type fortiGateCredentialServiceFake struct {
	username string
	current  string
	staged   string
}

func (f *fortiGateCredentialServiceFake) Metadata(context.Context, domain.ID) (domain.NodeCredentialMetadata, error) {
	return domain.NodeCredentialMetadata{NodeID: "node-a", Kind: domain.CredentialKindConsoleAdmin, Configured: true, State: "authenticated"}, nil
}
func (f *fortiGateCredentialServiceFake) Put(_ context.Context, _ domain.ID, username string, current, staged []byte) (domain.NodeCredentialMetadata, error) {
	f.username, f.current, f.staged = username, string(current), string(staged)
	return domain.NodeCredentialMetadata{NodeID: "node-a", Kind: domain.CredentialKindConsoleAdmin, Configured: true, Staged: len(staged) > 0, State: "pending_verification"}, nil
}
func (f *fortiGateCredentialServiceFake) Delete(context.Context, domain.ID) error { return nil }
func (f *fortiGateCredentialServiceFake) Verify(context.Context, domain.ID, string) (domain.OperationTask, error) {
	return domain.OperationTask{ID: "task-a", Kind: "fortigate.credential.verify", Input: map[string]any{"credential_ref": "node:node-a:console_admin"}}, nil
}
func (f *fortiGateCredentialServiceFake) Bootstrap(context.Context, domain.ID, string) (domain.OperationTask, error) {
	return domain.OperationTask{ID: "task-b", Kind: "fortigate.bootstrap", Input: map[string]any{"credential_ref": "node:node-a:console_admin"}}, nil
}

func TestFortiGateCredentialResponseNeverEchoesSecrets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &fortiGateCredentialServiceFake{}
	engine := gin.New()
	NewFortiGateCredentialHandlers(service, []string{"10.72.0.0/16"}).Register(engine)
	request := httptest.NewRequest(http.MethodPut, "/api/v1/nodes/node-a/credentials/console-admin", strings.NewReader(`{"username":"admin","current_password":"old-secret","new_password":"new-secret"}`))
	request.Header.Set("Content-Type", "application/json")
	request.RemoteAddr = "10.72.1.20:12345"
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if service.current != "old-secret" || service.staged != "new-secret" {
		t.Fatal("service did not receive submitted credentials")
	}
	if strings.Contains(recorder.Body.String(), "old-secret") || strings.Contains(recorder.Body.String(), "new-secret") || strings.Contains(recorder.Body.String(), `"username"`) {
		t.Fatalf("response echoed credential material: %s", recorder.Body.String())
	}
}

func TestFortiGateCredentialRoutesRejectNonManagementSource(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	NewFortiGateCredentialHandlers(&fortiGateCredentialServiceFake{}, []string{"10.72.0.0/16"}).Register(engine)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/nodes/node-a/credentials", nil)
	request.RemoteAddr = "192.0.2.10:12345"
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var problem domain.Problem
	_ = json.Unmarshal(recorder.Body.Bytes(), &problem)
	if problem.Code != "management_scope_required" {
		t.Fatalf("problem=%+v", problem)
	}
}
