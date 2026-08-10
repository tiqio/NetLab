package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/netlab/netlab/internal/domain"
)

type topologyConnectionCommandsStub struct {
	createdSource domain.ConnectionEndpoint
	createdTarget domain.ConnectionEndpoint
}

func (s *topologyConnectionCommandsStub) Create(_ context.Context, laboratoryID domain.ID, source, target domain.ConnectionEndpoint, _ domain.TopologyConnectionConfig, _ string) (domain.TopologyConnection, domain.OperationTask, error) {
	s.createdSource, s.createdTarget = source, target
	connection := domain.TopologyConnection{ID: "connection", LaboratoryID: laboratoryID, Source: source, Target: target, BackingKind: domain.ConnectionBackingLink, BackingID: "connection", Revision: 1, DesiredState: "connected", ObservedState: "pending"}
	taskValue := domain.OperationTask{ID: "task", Kind: "link.connect", ResourceType: "link", ResourceID: connection.ID, State: domain.TaskQueued}
	return connection, taskValue, nil
}
func (*topologyConnectionCommandsStub) List(context.Context, domain.ID) ([]domain.TopologyConnection, error) {
	return nil, nil
}
func (*topologyConnectionCommandsStub) Get(context.Context, domain.ID) (domain.TopologyConnection, error) {
	return domain.TopologyConnection{}, nil
}
func (*topologyConnectionCommandsStub) Delete(context.Context, domain.ID, domain.Revision, string) (domain.TopologyConnection, domain.OperationTask, error) {
	return domain.TopologyConnection{}, domain.OperationTask{}, nil
}

type topologyConnectionReadStub struct{}

func (topologyConnectionReadStub) ListConnectionEndpoints(context.Context, domain.ID) ([]domain.ConnectionEndpoint, error) {
	return nil, nil
}
func (topologyConnectionReadStub) GetLaboratory(context.Context, domain.ID) (domain.Laboratory, error) {
	return domain.Laboratory{ID: "lab", Revision: 7}, nil
}

func TestUnifiedTopologyConnectionCreateContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	commands := &topologyConnectionCommandsStub{}
	engine := gin.New()
	NewTopologyConnectionHandlers(commands, topologyConnectionReadStub{}).Register(engine)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/labs/lab/connections", strings.NewReader(`{
      "source":{"kind":"node_interface","resource_id":"node-a","port_id":"if-a"},
      "target":{"kind":"node_interface","resource_id":"node-b","port_id":"if-b"}
    }`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("If-Match", "7")
	request.Header.Set("Idempotency-Key", "connection-key")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"connection"`) || !strings.Contains(response.Body.String(), `"task"`) || !strings.Contains(response.Body.String(), `"laboratory_revision":7`) {
		t.Fatalf("body=%s", response.Body.String())
	}
	if commands.createdSource.PortID != "if-a" || commands.createdTarget.PortID != "if-b" {
		t.Fatalf("source=%+v target=%+v", commands.createdSource, commands.createdTarget)
	}
}

func TestUnifiedTopologyConnectionCreateRequiresCurrentRevision(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	NewTopologyConnectionHandlers(&topologyConnectionCommandsStub{}, topologyConnectionReadStub{}).Register(engine)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/labs/lab/connections", strings.NewReader(`{"source":{"kind":"node_interface","resource_id":"a","port_id":"if-a"},"target":{"kind":"node_interface","resource_id":"b","port_id":"if-b"}}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("If-Match", "6")
	request.Header.Set("Idempotency-Key", "stale")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusPreconditionFailed || !strings.Contains(response.Body.String(), "revision_conflict") {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}
