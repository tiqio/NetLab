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

type trafficWorkloadHTTPFake struct{ value domain.TrafficWorkload }

func (f *trafficWorkloadHTTPFake) List(context.Context, domain.ID) ([]domain.TrafficWorkload, error) {
	return []domain.TrafficWorkload{f.value}, nil
}
func (f *trafficWorkloadHTTPFake) Get(context.Context, domain.ID) (domain.TrafficWorkload, error) {
	return f.value, nil
}
func (f *trafficWorkloadHTTPFake) Create(_ context.Context, w domain.TrafficWorkload, _ string) (domain.OperationTask, error) {
	f.value = w
	return domain.OperationTask{ID: "task", ResourceID: "workload", State: domain.TaskQueued}, nil
}
func (f *trafficWorkloadHTTPFake) Start(context.Context, domain.ID, domain.Revision, string) (domain.OperationTask, error) {
	return domain.OperationTask{ID: "start", State: domain.TaskQueued}, nil
}
func (f *trafficWorkloadHTTPFake) Stop(context.Context, domain.ID, domain.Revision, string) (domain.OperationTask, error) {
	return domain.OperationTask{ID: "stop", State: domain.TaskQueued}, nil
}
func (f *trafficWorkloadHTTPFake) Delete(context.Context, domain.ID, domain.Revision, string) (domain.OperationTask, error) {
	return domain.OperationTask{ID: "delete", State: domain.TaskQueued}, nil
}

func TestTrafficWorkloadHTTPContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fake := &trafficWorkloadHTTPFake{value: domain.TrafficWorkload{ID: "workload", LaboratoryID: "lab", Revision: 1}}
	engine := gin.New()
	NewTrafficWorkloadHandlers(fake).Register(engine)
	body := `{"laboratory_id":"lab","name":"ping","source":{"kind":"node","resource_id":"node"},"protocol":"icmp","address_family":"ipv4","destination":{"address":"192.0.2.1"},"interval_seconds":5,"timeout_seconds":2}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/traffic-workloads", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "create")
	res := httptest.NewRecorder()
	engine.ServeHTTP(res, req)
	if res.Code != http.StatusAccepted || !strings.Contains(res.Body.String(), `"workload_id":"workload"`) {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	for _, path := range []string{"start", "stop"} {
		req = httptest.NewRequest(http.MethodPost, "/api/v1/traffic-workloads/workload/"+path, nil)
		req.Header.Set("If-Match", "1")
		res = httptest.NewRecorder()
		engine.ServeHTTP(res, req)
		if res.Code != http.StatusAccepted {
			t.Fatalf("%s status=%d body=%s", path, res.Code, res.Body.String())
		}
	}
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/traffic-workloads/workload", nil)
	res = httptest.NewRecorder()
	engine.ServeHTTP(res, req)
	var problem domain.Problem
	_ = json.Unmarshal(res.Body.Bytes(), &problem)
	if res.Code != http.StatusPreconditionRequired || problem.Code != "precondition_required" {
		t.Fatalf("status=%d problem=%+v", res.Code, problem)
	}
}
