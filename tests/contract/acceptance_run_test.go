package contract_test

import (
	"testing"
	"time"

	"github.com/netlab/netlab/internal/compliance"
)

func completeAcceptanceRun() compliance.AcceptanceRun {
	now := time.Now().UTC()
	gates := map[string]compliance.GateResult{}
	for _, name := range compliance.GateNames() {
		gates[name] = compliance.GateResult{Status: "passed"}
	}
	return compliance.AcceptanceRun{SchemaVersion: "1.0", ID: "run-1", CandidateID: "candidate-1", Status: "passed", GateResults: gates, CleanupBaseline: compliance.ResourceBaseline{Digest: "same", Resources: map[string]int{}}, CleanupFinal: compliance.ResourceBaseline{Digest: "same", Resources: map[string]int{}}, RedactionResult: compliance.RedactionResult{Passed: true}, Conclusion: "passed", StartedAt: now, FinishedAt: &now}
}

func TestAcceptanceRunRequiresEveryMandatoryGate(t *testing.T) {
	run := completeAcceptanceRun()
	delete(run.GateResults, "recovery")
	if _, err := compliance.ConcludeAcceptance(run, nil); err == nil {
		t.Fatal("missing mandatory gate accepted")
	}
}

func TestAcceptanceRunBlocksDocumentedSkipAndFailsDirtyBaseline(t *testing.T) {
	run := completeAcceptanceRun()
	run.GateResults["browser"] = compliance.GateResult{Status: "skipped", Reason: "target host unavailable; rerun make test-e2e-target"}
	if conclusion, err := compliance.ConcludeAcceptance(run, nil); err != nil || conclusion != "blocked" {
		t.Fatalf("conclusion=%s err=%v", conclusion, err)
	}
	run = completeAcceptanceRun()
	run.CleanupFinal.Digest = "different"
	if conclusion, err := compliance.ConcludeAcceptance(run, nil); err != nil || conclusion != "failed" {
		t.Fatalf("conclusion=%s err=%v", conclusion, err)
	}
}
