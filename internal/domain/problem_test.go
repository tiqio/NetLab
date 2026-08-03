package domain

import (
	"fmt"
	"testing"
)

func TestNormalizeProblemPreservesWrappedPointerAndAddsContext(t *testing.T) {
	original := &Problem{Code: "temporary_unavailable", Message: "QMP unavailable", Retryable: true, RetryAfterSeconds: 3}
	problem := NormalizeProblem(fmt.Errorf("connect monitor: %w", original), Problem{ResourceType: "node", ResourceID: "node", TaskID: "task", Phase: "hot_add", Cleanup: "TAP removed", OperatorHint: "retry after QMP recovers"})
	if problem.Code != original.Code || problem.Message != original.Message || problem.ResourceID != "node" || problem.TaskID != "task" || problem.RetryAfterSeconds != 3 || problem.Phase != "hot_add" {
		t.Fatalf("problem=%+v", problem)
	}
}
