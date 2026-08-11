package reconcile

import (
	"github.com/netlab/netlab/internal/domain"
)

func structuredProblem(err error, fallback domain.Problem) *domain.Problem {
	problem := domain.NormalizeProblem(err, fallback)
	if problem.Details == nil {
		problem.Details = map[string]any{}
	}
	problem.Details["phase"] = problem.Phase
	problem.Details["cleanup"] = problem.Cleanup
	return &problem
}

func normalizeTerminalError(target *error, fallback domain.Problem) {
	if target == nil || *target == nil {
		return
	}
	problem := domain.NormalizeProblem(*target, fallback)
	*target = problem
}

func terminalProblem(resourceType string, resourceID domain.ID, phase string) domain.Problem {
	return domain.Problem{
		Code:              "operation_failed",
		Retryable:         true,
		ResourceType:      resourceType,
		ResourceID:        resourceID,
		Phase:             phase,
		Cleanup:           "completed steps remain reconciled; owned partial state is retained for retry",
		OperatorHint:      "inspect the failed resource and retry the operation",
		RetryAfterSeconds: 3,
	}
}
