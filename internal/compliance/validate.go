package compliance

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/netlab/netlab/internal/domain"
)

var findingIDPattern = regexp.MustCompile(`^CONST-[A-Z0-9-]+$`)

type Documents struct {
	Ledger     Ledger
	Deployment DeploymentAuthority
	Templates  TemplateReadinessMatrix
	Evidence   []EvidenceRecord
	Acceptance *AcceptanceRun
}

func LoadDocuments(ledgerPath, deploymentPath, templatesPath, evidenceDirectory string) (Documents, error) {
	var documents Documents
	if err := decodeFile(ledgerPath, &documents.Ledger); err != nil {
		return documents, err
	}
	if err := decodeFile(deploymentPath, &documents.Deployment); err != nil {
		return documents, err
	}
	if err := decodeFile(templatesPath, &documents.Templates); err != nil {
		return documents, err
	}
	entries, err := os.ReadDir(evidenceDirectory)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return documents, err
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(evidenceDirectory, entry.Name())
		var envelope struct {
			GateResults json.RawMessage `json:"gate_results"`
		}
		if err := decodeFileLoose(path, &envelope); err != nil {
			return documents, err
		}
		if len(envelope.GateResults) > 0 {
			var run AcceptanceRun
			if err := decodeFile(path, &run); err != nil {
				return documents, err
			}
			documents.Acceptance = &run
			continue
		}
		var evidence EvidenceRecord
		if err := decodeFile(path, &evidence); err != nil {
			return documents, err
		}
		documents.Evidence = append(documents.Evidence, evidence)
	}
	return documents, nil
}

func decodeFileLoose(path string, target any) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

func decodeFile(path string, target any) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

func ValidateDocuments(documents Documents) error {
	if documents.Ledger.SchemaVersion != "1.0" || documents.Deployment.SchemaVersion != "1.0" || documents.Templates.SchemaVersion != "1.0" {
		return errors.New("all compliance documents must use schema version 1.0")
	}
	evidenceByID := make(map[string]EvidenceRecord, len(documents.Evidence))
	for _, evidence := range documents.Evidence {
		if err := validateEvidence(evidence); err != nil {
			return fmt.Errorf("evidence %s: %w", evidence.ID, err)
		}
		if _, duplicate := evidenceByID[evidence.ID]; duplicate {
			return fmt.Errorf("duplicate evidence id %s", evidence.ID)
		}
		evidenceByID[evidence.ID] = evidence
	}
	exceptions := make(map[string]AcceptedException, len(documents.Ledger.Exceptions))
	for _, exception := range documents.Ledger.Exceptions {
		exceptions[exception.ID] = exception
	}
	seen := map[string]struct{}{}
	for _, finding := range documents.Ledger.Findings {
		if !findingIDPattern.MatchString(finding.ID) {
			return fmt.Errorf("invalid finding id %s", finding.ID)
		}
		if _, duplicate := seen[finding.ID]; duplicate {
			return fmt.Errorf("duplicate finding id %s", finding.ID)
		}
		seen[finding.ID] = struct{}{}
		if strings.TrimSpace(finding.Owner) == "" {
			return fmt.Errorf("finding %s has no owner", finding.ID)
		}
		if finding.Status != "verified" && (finding.NextAction == nil || strings.TrimSpace(*finding.NextAction) == "") {
			return fmt.Errorf("finding %s requires next action", finding.ID)
		}
		if finding.Status == "verified" && len(finding.EvidenceIDs) == 0 {
			return fmt.Errorf("verified finding %s has no evidence", finding.ID)
		}
		for _, evidenceID := range finding.EvidenceIDs {
			evidence, ok := evidenceByID[evidenceID]
			if !ok {
				return fmt.Errorf("finding %s references missing evidence %s", finding.ID, evidenceID)
			}
			if finding.Status == "verified" && (evidence.Status != "accepted" || evidence.Outcome != "passed") {
				return fmt.Errorf("finding %s uses non-accepted evidence %s", finding.ID, evidenceID)
			}
		}
		if finding.Status == "accepted_exception" || finding.Status == "expired" {
			if finding.ExceptionID == nil {
				return fmt.Errorf("finding %s requires exception", finding.ID)
			}
			exception, ok := exceptions[*finding.ExceptionID]
			if !ok || exception.FindingID != finding.ID {
				return fmt.Errorf("finding %s has invalid exception", finding.ID)
			}
		}
	}
	authoritative := 0
	for _, instance := range documents.Deployment.Instances {
		if !domain.ValidSHA256Digest(instance.ContractDigest) {
			return fmt.Errorf("instance %s has invalid contract digest", instance.ID)
		}
		if instance.Role == "authoritative" && instance.ExternallyReachable {
			authoritative++
		}
	}
	if authoritative != 1 {
		return fmt.Errorf("expected exactly one externally reachable authoritative instance, got %d", authoritative)
	}
	if len(documents.Templates.Templates) < 6 {
		return errors.New("template readiness requires at least six families")
	}
	expectedConclusion := BuildSummary(documents.Ledger).Conclusion
	if documents.Ledger.ReleaseConclusion != expectedConclusion {
		return fmt.Errorf("contradictory ledger conclusion: detailed=%s summary=%s", expectedConclusion, documents.Ledger.ReleaseConclusion)
	}
	if documents.Acceptance != nil {
		conclusion, err := ConcludeAcceptance(*documents.Acceptance, approvedExceptions(documents.Ledger))
		if err != nil {
			return err
		}
		if conclusion != documents.Acceptance.Conclusion {
			return fmt.Errorf("contradictory acceptance conclusion: detailed=%s summary=%s", conclusion, documents.Acceptance.Conclusion)
		}
		if err = ValidateReportConsistency(documents.Ledger, *documents.Acceptance); err != nil {
			return err
		}
		if documents.Ledger.AcceptanceRunID == nil || *documents.Ledger.AcceptanceRunID != documents.Acceptance.ID {
			return fmt.Errorf("ledger acceptance run reference does not match current aggregate")
		}
	}
	return nil
}

func approvedExceptions(ledger Ledger) map[string]bool {
	result := map[string]bool{}
	for _, exception := range ledger.Exceptions {
		result[exception.ID] = (exception.Status == "approved" || exception.Status == "active") && exception.ApprovedBy != nil && exception.ApprovedAt != nil
	}
	return result
}

func validateEvidence(evidence EvidenceRecord) error {
	if evidence.SchemaVersion != "1.0" || evidence.ID == "" || evidence.CandidateID == "" || evidence.ReleaseVersion == "" {
		return errors.New("missing required identity")
	}
	if !domain.ValidSHA256Digest(evidence.ContractDigest) || !domain.ValidSHA256Digest(evidence.ScopeDigest) {
		return errors.New("invalid digest")
	}
	if evidence.BinaryDigest != nil && !domain.ValidSHA256Digest(*evidence.BinaryDigest) {
		return errors.New("invalid binary digest")
	}
	if len(evidence.FindingIDs) == 0 || evidence.Procedure == "" {
		return errors.New("finding ids and procedure required")
	}
	if evidence.FinishedAt.Before(evidence.StartedAt) {
		return errors.New("finish precedes start")
	}
	if evidence.Status == "accepted" && (evidence.Outcome != "passed" || !evidence.Cleanup.BaselineRestored || !evidence.Redaction.Passed) {
		return errors.New("accepted evidence must pass with cleanup and redaction")
	}
	for _, artifact := range evidence.Artifacts {
		if artifact.ContainsBinaryPayload || !domain.ValidSHA256Digest(artifact.Digest) {
			return errors.New("invalid artifact metadata")
		}
	}
	return nil
}
