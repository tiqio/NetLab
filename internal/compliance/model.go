package compliance

import "time"

type Ledger struct {
	SchemaVersion       string              `json:"schema_version"`
	ConstitutionVersion string              `json:"constitution_version"`
	CandidateID         *string             `json:"candidate_id,omitempty"`
	AcceptanceRunID     *string             `json:"acceptance_run_id,omitempty"`
	ReleaseConclusion   string              `json:"release_conclusion"`
	GeneratedAt         time.Time           `json:"generated_at"`
	Findings            []Finding           `json:"findings"`
	Exceptions          []AcceptedException `json:"exceptions"`
}

type Finding struct {
	ID             string    `json:"id"`
	Principle      string    `json:"principle"`
	Statement      string    `json:"statement"`
	Severity       string    `json:"severity"`
	Status         string    `json:"status"`
	Owner          string    `json:"owner"`
	RequirementIDs []string  `json:"requirement_ids"`
	EvidenceIDs    []string  `json:"evidence_ids"`
	ExceptionID    *string   `json:"exception_id,omitempty"`
	NextAction     *string   `json:"next_action,omitempty"`
	CandidateID    *string   `json:"candidate_id,omitempty"`
	LastReviewedAt time.Time `json:"last_reviewed_at"`
}

type AcceptedException struct {
	ID                  string     `json:"id"`
	FindingID           string     `json:"finding_id"`
	Owner               string     `json:"owner"`
	Scope               string     `json:"scope"`
	Risk                string     `json:"risk"`
	Motivation          string     `json:"motivation"`
	ApprovedBy          *string    `json:"approved_by,omitempty"`
	ApprovedAt          *time.Time `json:"approved_at,omitempty"`
	ExpirationCondition string     `json:"expiration_condition"`
	RemovalTask         string     `json:"removal_task"`
	Status              string     `json:"status"`
}

type EvidenceRecord struct {
	SchemaVersion  string           `json:"schema_version"`
	ID             string           `json:"id"`
	Kind           string           `json:"kind"`
	Status         string           `json:"status"`
	CandidateID    string           `json:"candidate_id"`
	ReleaseVersion string           `json:"release_version"`
	BinaryDigest   *string          `json:"binary_digest,omitempty"`
	ContractDigest string           `json:"contract_digest"`
	ScopeDigest    string           `json:"scope_digest"`
	FindingIDs     []string         `json:"finding_ids"`
	Procedure      string           `json:"procedure"`
	Target         map[string]any   `json:"target"`
	StartedAt      time.Time        `json:"started_at"`
	FinishedAt     time.Time        `json:"finished_at"`
	Outcome        string           `json:"outcome"`
	Cleanup        CleanupResult    `json:"cleanup"`
	Redaction      RedactionResult  `json:"redaction"`
	Artifacts      []ArtifactRecord `json:"artifacts"`
	Supersedes     []string         `json:"supersedes,omitempty"`
}

type CleanupResult struct {
	BaselineRestored bool     `json:"baseline_restored"`
	RemainingCount   int      `json:"remaining_count"`
	Remediation      []string `json:"remediation,omitempty"`
}

type RedactionResult struct {
	Passed                 bool `json:"passed"`
	ProhibitedContentCount int  `json:"prohibited_content_count"`
}

type ArtifactRecord struct {
	Kind                  string `json:"kind"`
	Path                  string `json:"path"`
	Digest                string `json:"digest"`
	ContainsBinaryPayload bool   `json:"contains_binary_payload"`
}

type DeploymentAuthority struct {
	SchemaVersion                 string                         `json:"schema_version"`
	HostID                        string                         `json:"host_id"`
	ApprovedManagementScopes      []string                       `json:"approved_management_scopes"`
	CredentialRotationAttestation *CredentialRotationAttestation `json:"credential_rotation_attestation,omitempty"`
	Instances                     []DeploymentInstance           `json:"instances"`
	VerifiedAt                    time.Time                      `json:"verified_at"`
}

type CredentialRotationAttestation struct {
	RotatedAt           time.Time `json:"rotated_at"`
	AttestedBy          string    `json:"attested_by"`
	SecretValueRecorded bool      `json:"secret_value_recorded"`
}

type DeploymentInstance struct {
	ID                  string     `json:"id"`
	Role                string     `json:"role"`
	ListenAddress       string     `json:"listen_address"`
	StateDirectory      string     `json:"state_directory"`
	DatabasePath        string     `json:"database_path"`
	ServiceName         string     `json:"service_name"`
	CandidateID         string     `json:"candidate_id"`
	ContractDigest      string     `json:"contract_digest"`
	ExternallyReachable bool       `json:"externally_reachable"`
	ManagementScope     []string   `json:"management_scope,omitempty"`
	VerifiedAt          *time.Time `json:"verified_at,omitempty"`
}

type TemplateReadinessMatrix struct {
	SchemaVersion string           `json:"schema_version"`
	CandidateID   string           `json:"candidate_id"`
	GeneratedAt   time.Time        `json:"generated_at"`
	Templates     []map[string]any `json:"templates"`
}

type AcceptanceRun struct {
	SchemaVersion       string                `json:"schema_version"`
	ID                  string                `json:"id"`
	CandidateID         string                `json:"candidate_id"`
	Status              string                `json:"status"`
	GateResults         map[string]GateResult `json:"gate_results"`
	ScenarioEvidenceIDs []string              `json:"scenario_evidence_ids"`
	Exceptions          []string              `json:"exceptions"`
	CleanupBaseline     ResourceBaseline      `json:"cleanup_baseline"`
	CleanupFinal        ResourceBaseline      `json:"cleanup_final"`
	RedactionResult     RedactionResult       `json:"redaction_result"`
	Conclusion          string                `json:"conclusion"`
	StartedAt           time.Time             `json:"started_at"`
	FinishedAt          *time.Time            `json:"finished_at,omitempty"`
}

type GateResult struct {
	Status  string `json:"status"`
	Reason  string `json:"reason,omitempty"`
	Details string `json:"details,omitempty"`
}

type ResourceBaseline struct {
	Digest    string         `json:"digest"`
	Resources map[string]int `json:"resources"`
}
