package domain

import (
	"errors"
	"strings"
)

var ErrNotFound = errors.New("not found")

const (
	ProblemCodeConflict              = "revision_conflict"
	ProblemCodeNotFound              = "not_found"
	ProblemCodeCapabilityUnavailable = "capability_unavailable"
	ProblemCodeCleanupFailed         = "cleanup_failed"
	ProblemCodeInvalidRequest        = "invalid_request"
)

func (p Problem) ValidateTerminal() error {
	if strings.TrimSpace(p.Code) == "" || strings.TrimSpace(p.Message) == "" {
		return Problem{Code: ProblemCodeInvalidRequest, Message: "problem code and message required"}
	}
	if p.ResourceType != "" && p.ResourceID == "" {
		return Problem{Code: ProblemCodeInvalidRequest, Message: "resource id required when resource type is present"}
	}
	return nil
}
