export type AcceptanceStatus =
  "preflighting" | "running" | "cleaning" | "passed" | "failed";

export type InteractionStatus = "passed" | "failed" | "skipped";
export type Activation = "pointer" | "keyboard" | "drag";
export type OutcomeClass =
  | "presentation"
  | "navigation"
  | "mutation"
  | "task"
  | "stream"
  | "download"
  | "error";

export interface Viewport {
  width: number;
  height: number;
}

export interface CapabilityDecision {
  name: string;
  class: "product-supported" | "product-unsupported" | "environment-optional";
  available: boolean;
  decision: "run" | "fail" | "skip";
  reason?: string;
  evidence?: string;
}

export interface EnvironmentSnapshot {
  base_url: string;
  target_kind: "local-disposable" | "remote-privileged";
  service_version?: string;
  release?: {
    version: string;
    candidate_id: string;
    binary_digest?: string;
    contract_digest: string;
    built_at?: string;
  };
  capabilities: Record<string, boolean>;
  capability_decisions: CapabilityDecision[];
  templates: TemplateObservation[];
  browser?: { name: string; version?: string };
  baseline_clean: boolean;
  baseline_laboratory_ids: string[];
  baseline_runtime_ownership: RuntimeOwnershipObservation[];
}

export interface RuntimeOwnershipObservation {
  resource_type: string;
  resource_id: string;
  object_kind: string;
  object_name: string;
  cleanup_state: string;
  ownership_class?: "managed" | "acceptance_owned" | "foreign_observed";
}

export interface TemplateObservation {
  template_id: string;
  device_family: string;
  runtime: "qemu" | "docker" | "lightweight";
  versions: Array<{
    version_id: string;
    image_id?: string;
    available: boolean;
  }>;
}

export interface InteractionDefinition {
  id: string;
  area: string;
  label: string;
  locator: { role: string; name?: string; test_id?: string };
  applicable_states: string[];
  activation: Activation[];
  outcome_class: OutcomeClass;
  operation?: string;
  required_capabilities?: string[];
  optional_environment_capabilities?: string[];
  cleanup_effect: string;
  sensitive_evidence: string[];
}

export interface TaskObservation {
  task_id: string;
  kind: string;
  states: string[];
  terminal_state: "succeeded" | "failed" | "cancelled";
  duration_ms: number;
  cleanup_state: string;
  problem_code?: string;
}

export interface InteractionResult {
  interaction_id: string;
  status: InteractionStatus;
  viewport: Viewport;
  activation: Activation;
  precondition: string;
  action: string;
  expected: string;
  actual: string;
  duration_ms: number;
  visible_feedback_ms?: number;
  pending_identity_ms?: number;
  task?: TaskObservation;
  resource_ids?: string[];
  cleanup_status: string;
  skip?: {
    capability: string;
    class: "environment-optional";
    reason: string;
    evidence: string;
  };
  evidence_refs?: string[];
}

export type ResourceCleanupState =
  "owned" | "deleting" | "deleted" | "missing" | "leaked";

export interface OwnedResource {
  resource_type: string;
  resource_id: string;
  owner_run_id: string;
  laboratory_id?: string;
  revision?: number;
  created_at: string;
  cleanup_method: string;
  cleanup_state: ResourceCleanupState;
  remediation?: string;
}

export interface CleanupRecord {
  started_at: string;
  finished_at: string;
  trigger: "success" | "failure" | "timeout" | "interrupt";
  attempted: true;
  resources: OwnedResource[];
  baseline_restored: boolean;
  remaining_count: number;
  remediation: string[];
}

export interface VersionCoverage {
  runtime: "qemu" | "docker" | "lightweight";
  device_family: string;
  version_id: string;
  image_id?: string;
  coverage_level: "full-journey" | "lifecycle-connectivity";
  result: "passed" | "failed";
  interactions: string[];
}

export interface ClientObservation {
  client_id: string;
  mutation_id: string;
  event_sequence: number;
  resource_revision: number;
  observed_at: string;
  convergence_ms: number;
  local_preferences_hash?: string;
}

export interface AcceptanceEvidence {
  schema_version: "1.0.0";
  run_id: string;
  status: "passed" | "failed";
  started_at: string;
  finished_at: string;
  environment: EnvironmentSnapshot;
  viewports: Viewport[];
  interaction_results: InteractionResult[];
  version_coverage: VersionCoverage[];
  cleanup: CleanupRecord;
  client_observations?: ClientObservation[];
}
