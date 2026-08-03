package integration_test

import (
	"errors"
	"testing"

	"github.com/netlab/netlab/internal/app/command"
	"github.com/netlab/netlab/internal/domain"
)

func TestTerminalFailureContainsOperationAndCleanupContext(t *testing.T) {
	fallback := domain.Problem{Code: "link_cleanup_failed", Message: "link cleanup failed", Retryable: true, ResourceType: "link", ResourceID: "link-1", TaskID: "task-1", Phase: "cleanup", Cleanup: "bridge removed; interface ownership pending", OperatorHint: "remove the stale ownership record and retry"}
	problem := command.NormalizeOperationProblem(errors.New("adapter failed"), fallback, false)
	if problem == nil || problem.ResourceID == "" || problem.TaskID == "" || problem.Phase == "" || problem.Cleanup == "" || problem.OperatorHint == "" {
		t.Fatalf("incomplete problem: %#v", problem)
	}
}

func TestIdempotentDeleteTreatsConfirmedAbsenceAsSuccess(t *testing.T) {
	if problem := command.NormalizeOperationProblem(domain.ErrNotFound, domain.Problem{ResourceType: "node", ResourceID: "node-1"}, true); problem != nil {
		t.Fatalf("unexpected problem: %#v", problem)
	}
}
