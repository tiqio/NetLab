package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/netlab/netlab/internal/app/query"
	"github.com/netlab/netlab/internal/domain"
	"github.com/netlab/netlab/internal/runtime/ownership"
)

type ownershipQueryStore struct {
	values []ownership.Record
}

func (s ownershipQueryStore) ListRuntimeOwnership(context.Context) ([]ownership.Record, error) {
	return s.values, nil
}

func TestRuntimeOwnershipList(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	service := query.NewRuntimeOwnershipService(ownershipQueryStore{values: []ownership.Record{{
		ResourceType: "node",
		ResourceID:   domain.ID("node-1"),
		ObjectKind:   "process",
		ObjectName:   "qemu-1",
		CleanupState: "active",
	}}})
	NewRuntimeOwnershipHandlers(service).Register(engine)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/runtime-ownership", nil)
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if response.Body.String() == "null" || response.Body.String() == "[]" {
		t.Fatalf("ownership response missing: %s", response.Body.String())
	}
}
