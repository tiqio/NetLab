package contract

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	httpapi "github.com/netlab/netlab/internal/api/http"
	"github.com/netlab/netlab/internal/app/command"
	"github.com/netlab/netlab/internal/domain"
)

type placementCommandStub struct {
	laboratoryID domain.ID
	revision     domain.Revision
	updates      []domain.PlacementUpdate
}

func TestGeneratedClientExposesUpdateTopologyPlacements(t *testing.T) {
	body, err := os.ReadFile("../../web/src/api/index.ts")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "updateTopologyPlacements:") || !strings.Contains(string(body), "/placements`") {
		t.Fatal("generated SPA client does not expose the placement mutation")
	}
}

func (s *placementCommandStub) Update(_ context.Context, laboratoryID domain.ID, revision domain.Revision, updates []domain.PlacementUpdate) (command.TopologyPlacementResult, error) {
	s.laboratoryID, s.revision, s.updates = laboratoryID, revision, updates
	return command.TopologyPlacementResult{LaboratoryRevision: revision.Next(), Placements: []domain.TopologyPlacement{{LaboratoryID: laboratoryID, ResourceID: updates[0].ResourceID, ResourceType: updates[0].ResourceType, X: updates[0].X, Y: updates[0].Y, Revision: 1}}}, nil
}

func TestUpdateTopologyPlacementsHTTPContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &placementCommandStub{}
	engine := gin.New()
	httpapi.NewTopologyPlacementHandlers(service).Register(engine)
	body, _ := json.Marshal(map[string]any{"placements": []map[string]any{{"resource_id": "node-1", "resource_type": "node", "x": 12.5, "y": -4}}})
	request := httptest.NewRequest(http.MethodPut, "/api/v1/labs/lab-1/placements", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("If-Match", "3")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || recorder.Header().Get("ETag") != "4" {
		t.Fatalf("status=%d etag=%q body=%s", recorder.Code, recorder.Header().Get("ETag"), recorder.Body.String())
	}
	if service.laboratoryID != "lab-1" || service.revision != 3 || len(service.updates) != 1 {
		t.Fatalf("service call mismatch: %+v", service)
	}
}
