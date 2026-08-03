package compliance

import (
	"fmt"
	"sort"
	"strings"
)

var mandatoryAcceptanceGates = []string{"format", "static_analysis", "unit", "contract", "frontend", "privileged_integration", "recovery", "security", "leak", "browser"}

func ConcludeAcceptance(run AcceptanceRun, approvedExceptions map[string]bool) (string, error) {
	if run.SchemaVersion != "1.0" || strings.TrimSpace(run.ID) == "" || strings.TrimSpace(run.CandidateID) == "" {
		return "", fmt.Errorf("acceptance run identity is incomplete")
	}
	if run.FinishedAt == nil || run.FinishedAt.Before(run.StartedAt) {
		return "", fmt.Errorf("acceptance run terminal timestamps are invalid")
	}
	for _, gate := range mandatoryAcceptanceGates {
		result, ok := run.GateResults[gate]
		if !ok {
			return "", fmt.Errorf("mandatory gate %s is missing", gate)
		}
		switch result.Status {
		case "passed":
		case "skipped":
			if strings.TrimSpace(result.Reason) == "" {
				return "", fmt.Errorf("skipped gate %s requires a reason", gate)
			}
			return "blocked", nil
		case "blocked":
			return "blocked", nil
		case "failed", "cancelled":
			return "failed", nil
		default:
			return "", fmt.Errorf("gate %s has invalid status %q", gate, result.Status)
		}
	}
	if run.CleanupBaseline.Digest == "" || run.CleanupFinal.Digest == "" || run.CleanupBaseline.Digest != run.CleanupFinal.Digest {
		return "failed", nil
	}
	if !run.RedactionResult.Passed || run.RedactionResult.ProhibitedContentCount > 0 {
		return "failed", nil
	}
	for _, exceptionID := range run.Exceptions {
		if !approvedExceptions[exceptionID] {
			return "blocked", nil
		}
	}
	return "passed", nil
}

func GateNames() []string {
	result := append([]string(nil), mandatoryAcceptanceGates...)
	sort.Strings(result)
	return result
}
