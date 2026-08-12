package mcp

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/netlab/netlab/internal/domain"
)

type trafficWorkloadMCPFake struct{ value domain.TrafficWorkload }

func (f *trafficWorkloadMCPFake) List(context.Context, domain.ID) ([]domain.TrafficWorkload, error) {
	return []domain.TrafficWorkload{f.value}, nil
}
func (f *trafficWorkloadMCPFake) Get(context.Context, domain.ID) (domain.TrafficWorkload, error) {
	return f.value, nil
}
func (f *trafficWorkloadMCPFake) Create(_ context.Context, w domain.TrafficWorkload, _ string) (domain.OperationTask, error) {
	f.value = w
	return domain.OperationTask{ID: "task", ResourceID: "workload"}, nil
}
func (f *trafficWorkloadMCPFake) Start(context.Context, domain.ID, domain.Revision, string) (domain.OperationTask, error) {
	return domain.OperationTask{ID: "start"}, nil
}
func (f *trafficWorkloadMCPFake) Stop(context.Context, domain.ID, domain.Revision, string) (domain.OperationTask, error) {
	return domain.OperationTask{ID: "stop"}, nil
}
func (f *trafficWorkloadMCPFake) Delete(context.Context, domain.ID, domain.Revision, string) (domain.OperationTask, error) {
	return domain.OperationTask{ID: "delete"}, nil
}

func TestTrafficWorkloadMCPParity(t *testing.T) {
	fake := &trafficWorkloadMCPFake{value: domain.TrafficWorkload{ID: "workload", Attempts: 3, Successes: 2, Failures: 1, MatchedBytes: 128}}
	tools := TrafficWorkloadTools(fake)
	if len(tools) != 6 {
		t.Fatalf("tools=%d", len(tools))
	}
	ctx := &gin.Context{Request: httptest.NewRequest("POST", "/mcp", nil)}
	result, err := tools[1].Handler(ctx, map[string]any{"workload_id": "workload"})
	if err != nil || result.(domain.TrafficWorkload).MatchedBytes != 128 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	created, err := tools[2].Handler(ctx, map[string]any{"laboratory_id": "lab", "name": "ping", "source": map[string]any{"kind": "node", "resource_id": "node"}, "protocol": "icmp", "address_family": "ipv4", "destination": map[string]any{"address": "192.0.2.1"}, "interval_seconds": 5, "timeout_seconds": 2, "idempotency_key": "key"})
	if err != nil || created.(map[string]any)["workload_id"] != domain.ID("workload") {
		t.Fatalf("created=%+v err=%v", created, err)
	}
}
