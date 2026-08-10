package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/netlab/netlab/internal/domain"
)

type contendedTopologyCommands struct {
	mu       sync.Mutex
	occupied bool
	tasks    map[string]domain.OperationTask
}

func (s *contendedTopologyCommands) Create(_ context.Context, lab domain.ID, source, target domain.ConnectionEndpoint, _ domain.TopologyConnectionConfig, key string) (domain.TopologyConnection, domain.OperationTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if taskValue, ok := s.tasks[key]; ok {
		return domain.TopologyConnection{ID: taskValue.ResourceID, LaboratoryID: lab, Source: source, Target: target, BackingKind: domain.ConnectionBackingLink, Revision: 1}, taskValue, nil
	}
	if s.occupied {
		return domain.TopologyConnection{}, domain.OperationTask{}, domain.Problem{Code: "port_in_use", Message: "endpoint already reserved", Retryable: true}
	}
	s.occupied = true
	taskValue := domain.OperationTask{ID: domain.NewID(), Kind: "link.connect", ResourceType: "link", ResourceID: domain.NewID(), State: domain.TaskQueued}
	s.tasks[key] = taskValue
	return domain.TopologyConnection{ID: taskValue.ResourceID, LaboratoryID: lab, Source: source, Target: target, BackingKind: domain.ConnectionBackingLink, Revision: 1}, taskValue, nil
}
func (*contendedTopologyCommands) List(context.Context, domain.ID) ([]domain.TopologyConnection, error) {
	return nil, nil
}
func (*contendedTopologyCommands) Get(context.Context, domain.ID) (domain.TopologyConnection, error) {
	return domain.TopologyConnection{}, domain.ErrNotFound
}
func (*contendedTopologyCommands) Delete(context.Context, domain.ID, domain.Revision, string) (domain.TopologyConnection, domain.OperationTask, error) {
	return domain.TopologyConnection{}, domain.OperationTask{}, nil
}

func topologyConnectionRequest(engine *gin.Engine, key string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/labs/lab/connections", strings.NewReader(`{"source":{"kind":"node_interface","resource_id":"a","port_id":"if-a"},"target":{"kind":"node_interface","resource_id":"b","port_id":"if-b"}}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("If-Match", "7")
	request.Header.Set("Idempotency-Key", key)
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	return response
}

func TestUnifiedTopologyConnectionHTTPSerializesClientsAndAllowsNewKeyAfterRelease(t *testing.T) {
	gin.SetMode(gin.TestMode)
	commands := &contendedTopologyCommands{tasks: map[string]domain.OperationTask{}}
	engine := gin.New()
	NewTopologyConnectionHandlers(commands, topologyConnectionReadStub{}).Register(engine)
	responses := make(chan *httptest.ResponseRecorder, 2)
	var waitGroup sync.WaitGroup
	for _, key := range []string{"client-a", "client-b"} {
		waitGroup.Add(1)
		go func(key string) { defer waitGroup.Done(); responses <- topologyConnectionRequest(engine, key) }(key)
	}
	waitGroup.Wait()
	close(responses)
	accepted, conflicts := 0, 0
	for response := range responses {
		if response.Code == http.StatusAccepted {
			accepted++
		}
		if response.Code == http.StatusConflict && strings.Contains(response.Body.String(), "port_in_use") {
			conflicts++
		}
	}
	if accepted != 1 || conflicts != 1 {
		t.Fatalf("accepted=%d conflicts=%d", accepted, conflicts)
	}
	commands.mu.Lock()
	commands.occupied = false
	commands.mu.Unlock()
	if response := topologyConnectionRequest(engine, "client-retry-new-key"); response.Code != http.StatusAccepted {
		t.Fatalf("retry status=%d body=%s", response.Code, response.Body.String())
	}
}
