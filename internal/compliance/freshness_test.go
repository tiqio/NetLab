package compliance

import (
	"testing"
	"time"
)

func TestEvidenceFreshnessRequiresExactCandidateContractAndScope(t *testing.T) {
	digest := "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	record := EvidenceRecord{Status: "accepted", Outcome: "passed", CandidateID: "candidate-1", ContractDigest: digest, ScopeDigest: digest, Cleanup: CleanupResult{BaselineRestored: true}, Redaction: RedactionResult{Passed: true}}
	input := FreshnessInput{CandidateID: "candidate-1", ContractDigest: digest, ScopeDigest: digest, Now: time.Now()}
	if !EvidenceIsFresh(record, input) {
		t.Fatal("expected current evidence")
	}
	input.CandidateID = "candidate-2"
	if EvidenceIsFresh(record, input) {
		t.Fatal("candidate change must stale evidence")
	}
}

func TestReconcileFreshnessReopensVerifiedFindingAsStale(t *testing.T) {
	action := ""
	ledger := Ledger{Findings: []Finding{{ID: "CONST-I-01", Status: "verified", EvidenceIDs: []string{"e1"}, NextAction: &action}}}
	ReconcileFreshness(&ledger, nil, FreshnessInput{Now: time.Now()})
	if ledger.Findings[0].Status != "stale" || ledger.Findings[0].NextAction == nil {
		t.Fatalf("unexpected finding: %#v", ledger.Findings[0])
	}
}
