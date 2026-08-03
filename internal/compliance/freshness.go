package compliance

import "time"

type FreshnessInput struct {
	CandidateID    string
	ContractDigest string
	ScopeDigest    string
	Now            time.Time
}

func EvidenceIsFresh(evidence EvidenceRecord, input FreshnessInput) bool {
	return evidence.Status == "accepted" && evidence.Outcome == "passed" &&
		evidence.CandidateID == input.CandidateID && evidence.ContractDigest == input.ContractDigest &&
		evidence.ScopeDigest == input.ScopeDigest && evidence.Cleanup.BaselineRestored && evidence.Redaction.Passed
}

func ReconcileFreshness(ledger *Ledger, evidence []EvidenceRecord, input FreshnessInput) {
	byID := make(map[string]EvidenceRecord, len(evidence))
	for _, record := range evidence {
		byID[record.ID] = record
	}
	for index := range ledger.Findings {
		finding := &ledger.Findings[index]
		if finding.Status != "verified" {
			continue
		}
		fresh := false
		for _, id := range finding.EvidenceIDs {
			if EvidenceIsFresh(byID[id], input) {
				fresh = true
				break
			}
		}
		if !fresh {
			finding.Status = "stale"
			action := "repeat validation against the current candidate and scope"
			finding.NextAction = &action
			finding.LastReviewedAt = input.Now.UTC()
		}
	}
}
