package command

import (
	"errors"

	"github.com/netlab/netlab/internal/domain"
)

func NormalizeOperationProblem(err error, fallback domain.Problem, absenceIsSuccess bool) *domain.Problem {
	if err == nil {
		return nil
	}
	if errors.Is(err, domain.ErrNotFound) && absenceIsSuccess {
		return nil
	}
	if errors.Is(err, domain.ErrNotFound) {
		problem := fallback
		problem.Code = domain.ProblemCodeNotFound
		problem.Message = "owned resource was not found"
		problem.Retryable = false
		if problem.Cleanup == "" {
			problem.Cleanup = "resource absence confirmed; related ownership still requires verification"
		}
		if problem.OperatorHint == "" {
			problem.OperatorHint = "refresh state and inspect runtime ownership before retrying"
		}
		return &problem
	}
	problem := domain.NormalizeProblem(err, fallback)
	return &problem
}
