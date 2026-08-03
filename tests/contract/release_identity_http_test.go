package contract_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	httpapi "github.com/netlab/netlab/internal/api/http"
	"github.com/netlab/netlab/internal/app/query"
	"github.com/netlab/netlab/internal/domain"
)

func TestCapabilitiesExposeExactReleaseIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	handlers := httpapi.NewAutomationHandlers(nil, nil, nil, nil, nil)
	handlers.SetReleaseService(query.NewReleaseService(domain.ReleaseIdentity{Version: "1.0.0", CandidateID: "candidate-1", ContractDigest: domain.DigestBytes([]byte("contract"))}))
	handlers.Register(engine)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/capabilities", nil)
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var body struct {
		Release domain.ReleaseIdentity `json:"release"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Release.CandidateID != "candidate-1" || body.Release.ContractDigest == "" {
		t.Fatalf("unexpected release: %#v", body.Release)
	}
}
