package contract_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	httpapi "github.com/netlab/netlab/internal/api/http"
	"github.com/netlab/netlab/internal/app/query"
	"github.com/netlab/netlab/internal/domain"
)

type capabilityStoreStub struct {
	values []domain.RuntimeCapabilityObservation
	err    error
}

func (s capabilityStoreStub) ListRuntimeCapabilities(context.Context, domain.ID) ([]domain.RuntimeCapabilityObservation, error) {
	return s.values, s.err
}

func TestNodeCapabilitiesHTTPContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	service := query.NewRuntimeCapabilityService(capabilityStoreStub{values: []domain.RuntimeCapabilityObservation{{NodeID: "node-1", Capability: domain.CapabilityQGA, Revision: 1, State: domain.CapabilityUnavailable, ObservedAt: time.Now()}}})
	httpapi.NewNodeCapabilityHandlers(service).Register(engine)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/nodes/node-1/capabilities", nil)
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"qga"`) {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}
