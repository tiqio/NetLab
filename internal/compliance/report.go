package compliance

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
)

type Summary struct {
	CandidateID string         `json:"candidate_id,omitempty"`
	Counts      map[string]int `json:"counts"`
	Conclusion  string         `json:"conclusion"`
}

func BuildSummary(ledger Ledger) Summary {
	counts := map[string]int{}
	for _, finding := range ledger.Findings {
		counts[finding.Status]++
	}
	conclusion := "not_ready"
	if len(ledger.Findings) > 0 && counts["open"]+counts["partial"]+counts["blocked"]+counts["stale"]+counts["expired"] == 0 {
		conclusion = "ready"
	}
	candidate := ""
	if ledger.CandidateID != nil {
		candidate = *ledger.CandidateID
	}
	return Summary{CandidateID: candidate, Counts: counts, Conclusion: conclusion}
}

func ValidateReportConsistency(ledger Ledger, run AcceptanceRun) error {
	ledgerConclusion := BuildSummary(ledger).Conclusion
	runReady := run.Conclusion == "passed"
	if (ledgerConclusion == "ready") != runReady {
		return fmt.Errorf("contradictory conclusions: ledger=%s acceptance=%s", ledgerConclusion, run.Conclusion)
	}
	if ledger.CandidateID != nil && run.CandidateID != "" && *ledger.CandidateID != run.CandidateID {
		return fmt.Errorf("contradictory candidate identity: ledger=%s acceptance=%s", *ledger.CandidateID, run.CandidateID)
	}
	return nil
}

func WriteReport(writer io.Writer, ledger Ledger, jsonOutput bool) error {
	summary := BuildSummary(ledger)
	if jsonOutput {
		return json.NewEncoder(writer).Encode(summary)
	}
	_, _ = fmt.Fprintf(writer, "Candidate: %s\nConclusion: %s\n", summary.CandidateID, summary.Conclusion)
	keys := make([]string, 0, len(summary.Counts))
	for key := range summary.Counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		_, _ = fmt.Fprintf(writer, "%s: %d\n", key, summary.Counts[key])
	}
	return nil
}
