package compliance

import (
	"testing"
	"time"
)

func TestVerifiedFindingRejectsPartialEvidence(t *testing.T) {
	digest := "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	documents := Documents{
		Ledger:     Ledger{SchemaVersion: "1.0", Findings: []Finding{{ID: "CONST-I-01", Principle: "I", Statement: "truth", Severity: "high", Status: "verified", Owner: "owner", EvidenceIDs: []string{"e1"}, LastReviewedAt: time.Now()}}},
		Deployment: DeploymentAuthority{SchemaVersion: "1.0", Instances: []DeploymentInstance{{ID: "prod", Role: "authoritative", ExternallyReachable: true, ContractDigest: digest}}},
		Templates:  TemplateReadinessMatrix{SchemaVersion: "1.0", Templates: make([]map[string]any, 6)},
		Evidence:   []EvidenceRecord{{SchemaVersion: "1.0", ID: "e1", Status: "validated", Outcome: "passed", CandidateID: "c", ReleaseVersion: "v", ContractDigest: digest, ScopeDigest: digest, FindingIDs: []string{"CONST-I-01"}, Procedure: "test", StartedAt: time.Now(), FinishedAt: time.Now(), Cleanup: CleanupResult{BaselineRestored: true}, Redaction: RedactionResult{Passed: true}}},
	}
	if ValidateDocuments(documents) == nil {
		t.Fatal("partial evidence must not verify a finding")
	}
}
