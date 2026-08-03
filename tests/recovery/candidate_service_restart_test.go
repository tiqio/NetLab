package recovery_test

import (
	"testing"
	"time"

	"github.com/netlab/netlab/internal/compliance"
)

func TestCandidateServiceRestartGateRequiresClientConvergence(t *testing.T) {
	now := time.Now().UTC()
	gates := map[string]compliance.GateResult{}
	for _, name := range compliance.GateNames() {
		gates[name] = compliance.GateResult{Status: "passed"}
	}
	gates["recovery"] = compliance.GateResult{Status: "failed", Details: "runtime adopted but ordered client convergence failed"}
	run := compliance.AcceptanceRun{SchemaVersion: "1.0", ID: "restart", CandidateID: "candidate", GateResults: gates, CleanupBaseline: compliance.ResourceBaseline{Digest: "same"}, CleanupFinal: compliance.ResourceBaseline{Digest: "same"}, RedactionResult: compliance.RedactionResult{Passed: true}, StartedAt: now, FinishedAt: &now}
	conclusion, err := compliance.ConcludeAcceptance(run, nil)
	if err != nil || conclusion != "failed" {
		t.Fatalf("conclusion=%s err=%v", conclusion, err)
	}
}
