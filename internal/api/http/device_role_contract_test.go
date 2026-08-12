package httpapi

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/netlab/netlab/internal/domain"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type readinessFake struct{}

func (readinessFake) Get(context.Context, domain.ID) (domain.DeviceReadiness, error) {
	return domain.DeviceReadiness{NodeID: "node", Cable: domain.DeviceReadinessLevel{State: "ready"}, Guest: domain.DeviceReadinessLevel{State: "ready"}, Management: domain.DeviceReadinessLevel{State: "prerequisite"}, DataPath: domain.DeviceReadinessLevel{State: "unverified"}}, nil
}
func TestDeviceReadinessHTTPSeparatesStates(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	handler := NewNodeOperationsHandlers(nil, nil, nil, nil, nil)
	handler.SetDeviceReadiness(readinessFake{})
	handler.Register(engine)
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/nodes/node/device-readiness", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"management":{"state":"prerequisite"}`) || !strings.Contains(response.Body.String(), `"data_path":{"state":"unverified"}`) {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}
