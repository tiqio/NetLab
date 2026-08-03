package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/netlab/netlab/internal/domain"
)

func TestHandleErrorMapsStructuredProblems(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name       string
		problem    domain.Problem
		status     int
		retryAfter string
	}{
		{name: "revision", problem: domain.Problem{Code: "revision_conflict"}, status: http.StatusPreconditionFailed},
		{name: "capacity", problem: domain.Problem{Code: "resource_exhausted"}, status: http.StatusTooManyRequests},
		{name: "temporary capacity", problem: domain.Problem{Code: "resource_exhausted", Retryable: true}, status: http.StatusServiceUnavailable, retryAfter: "1"},
		{name: "runtime", problem: domain.Problem{Code: "temporary_unavailable", Retryable: true, RetryAfterSeconds: 7}, status: http.StatusServiceUnavailable, retryAfter: "7"},
		{name: "reconciler", problem: domain.Problem{Code: "operation_failed", Retryable: true, RetryAfterSeconds: 3}, status: http.StatusServiceUnavailable, retryAfter: "3"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(recorder)
			handleError(context, test.problem)
			if recorder.Code != test.status || recorder.Header().Get("Retry-After") != test.retryAfter {
				t.Fatalf("status=%d retry-after=%q", recorder.Code, recorder.Header().Get("Retry-After"))
			}
			var body domain.Problem
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil || body.Code != test.problem.Code {
				t.Fatalf("body=%s err=%v", recorder.Body.String(), err)
			}
		})
	}
}

func TestHandleErrorPreservesWrappedPointerProblem(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	handleError(context, fmt.Errorf("runtime: %w", &domain.Problem{Code: "temporary_unavailable", Message: "QMP unavailable", Retryable: true, ResourceType: "node", ResourceID: "node", Phase: "starting", Cleanup: "no runtime created", OperatorHint: "retry after QMP recovers", RetryAfterSeconds: 6}))
	if recorder.Code != http.StatusServiceUnavailable || recorder.Header().Get("Retry-After") != "6" {
		t.Fatalf("status=%d retry-after=%q body=%s", recorder.Code, recorder.Header().Get("Retry-After"), recorder.Body.String())
	}
	var problem domain.Problem
	if err := json.Unmarshal(recorder.Body.Bytes(), &problem); err != nil || problem.Code != "temporary_unavailable" || problem.ResourceID != "node" || problem.Phase != "starting" {
		t.Fatalf("problem=%+v err=%v", problem, err)
	}
}
