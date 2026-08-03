# Data Model: Frontend Interaction Acceptance

The feature adds test-domain records and evidence contracts. It does not add product database tables. Persisted
test records are sanitized JSON artifacts under ignored acceptance-output directories.

## AcceptanceRun

Represents one complete local or target-host execution.

| Field | Type | Rules |
|---|---|---|
| `id` | string | UUID or collision-resistant identifier; immutable |
| `started_at` / `finished_at` | timestamp | UTC RFC 3339; finish required for terminal runs |
| `environment` | EnvironmentSnapshot | Captured before mutations |
| `status` | enum | `preflighting`, `running`, `cleaning`, `passed`, `failed` |
| `viewports` | Viewport[] | Must include 1920×1080 and 1024×768 for complete runs |
| `interaction_results` | InteractionResult[] | One terminal result per applicable inventory entry and context |
| `version_coverage` | VersionCoverage[] | Covers every available supported version |
| `owned_resources` | OwnedResource[] | Append-only discovery ledger until cleanup completes |
| `cleanup` | CleanupRecord | Required for every terminal run |
| `evidence_policy_version` | string | Selects redaction/retention rules |

### State Transitions

`preflighting → running → cleaning → passed`

`preflighting → cleaning → failed`

`running → cleaning → failed`

`cleaning → failed` when any owned resource remains or cleanup cannot be verified.

No terminal run may return to `running` or be reused as a later run.

## EnvironmentSnapshot

Records the immutable preflight facts used to decide coverage.

| Field | Type | Rules |
|---|---|---|
| `base_url` | string | HTTP(S) origin only; no embedded credentials |
| `target_kind` | enum | `local-disposable` or `remote-privileged` |
| `service_version` | string | Required when exposed by service |
| `capabilities` | map | Product capability name to declared availability |
| `templates` | TemplateObservation[] | Supported template/device family and available versions |
| `browser` | object | Browser name/version and viewport project |
| `desktop_wireshark` | CapabilityDecision | Optional environmental capability only |
| `baseline` | ResourceBaseline | Pre-existing counts/identities needed to prove non-interference |

## InteractionDefinition

A version-controlled inventory entry for a user-visible control or gesture.

| Field | Type | Rules |
|---|---|---|
| `id` | string | Stable dotted identifier; unique across inventory |
| `area` | string | Workspace feature area |
| `label` | string | Human-recognizable control name |
| `locator` | object | Prefer role and accessible name; test IDs only when semantics are insufficient |
| `applicable_states` | string[] | At least one declared state |
| `activation` | enum[] | One or both of `pointer`, `keyboard`; gestures may add `drag` |
| `outcome_class` | enum | `presentation`, `navigation`, `mutation`, `task`, `stream`, `download`, `error` |
| `operation` | string | Required for authoritative mutations; maps to operation registry/API |
| `required_capabilities` | string[] | Missing supported capability fails preflight |
| `optional_environment_capabilities` | string[] | Only these may cause a reasoned skip |
| `cleanup_effect` | string | Expected owned resources released or retained as metadata |
| `sensitive_evidence` | string[] | Content categories that must never be retained |

## InteractionResult

Terminal result for one definition in a concrete context.

| Field | Type | Rules |
|---|---|---|
| `interaction_id` | string | References `InteractionDefinition.id` |
| `status` | enum | `passed`, `failed`, `skipped` |
| `viewport` | Viewport | Required |
| `activation` | enum | Must be allowed by definition |
| `precondition` | string | Sanitized, reproducible state summary |
| `action` | string | Sanitized user action summary |
| `expected` / `actual` | string | Must be non-empty; no sensitive payloads |
| `visible_feedback_ms` | integer | Non-negative; checked against 500 ms goal |
| `pending_identity_ms` | integer | Required for long operations; checked against 2 s goal |
| `task_observation` | TaskObservation | Required for durable operations |
| `authoritative_state` | object | Sanitized resource IDs, revisions, and states only |
| `skip_decision` | CapabilityDecision | Required only for `skipped` |
| `evidence_refs` | string[] | Relative paths within run output |

### Validation Rules

- `skipped` is valid only for an explicitly optional environmental capability with a verified missing
  prerequisite.
- A product-declared supported capability that is missing, unavailable, or nonfunctional produces `failed`.
- A mutation cannot pass without refreshed authoritative state; durable mutations also require a terminal task.
- A presentation-only control must assert its documented local effect; a silent no-op always fails.

## TaskObservation

| Field | Type | Rules |
|---|---|---|
| `task_id` | string | Required and stable |
| `kind` | string | Must match the interaction's operation |
| `states` | string[] | Ordered observed progression |
| `terminal_state` | enum | `succeeded`, `failed`, or `cancelled` |
| `duration_ms` | integer | Non-negative and bounded by declared timeout |
| `problem` | object | Structured, sanitized problem when not succeeded |
| `cleanup_state` | string | Required for timeout, cancellation, or failure |

## CapabilityDecision

| Field | Type | Rules |
|---|---|---|
| `name` | string | Stable capability key |
| `class` | enum | `product-supported`, `product-unsupported`, `environment-optional` |
| `available` | boolean | Based on preflight evidence |
| `decision` | enum | `run`, `fail`, `skip` |
| `reason` | string | Required for `fail` or `skip` |
| `evidence` | string | Sanitized source of decision |

Only `environment-optional` may have `decision: skip`.

## OwnedResource

Tracks every resource created or adopted by the acceptance run.

| Field | Type | Rules |
|---|---|---|
| `resource_type` | enum | Laboratory, node, interface, link, network object, mapping, capture, filter, task, artifact, process, container, VM, namespace, bridge, rule, socket, or temporary file |
| `resource_id` | string | Product or host identity |
| `owner_run_id` | string | Must equal the current run |
| `laboratory_id` | string | Present when laboratory-scoped |
| `created_at` | timestamp | UTC RFC 3339 |
| `cleanup_method` | string | UI deletion first for journey resources; bounded API/host fallback for interrupted runs |
| `cleanup_state` | enum | `owned`, `deleting`, `deleted`, `missing`, `leaked` |
| `remediation` | string | Required for `leaked` |

### State Transitions

`owned → deleting → deleted`

`owned → missing` when authoritative discovery proves it is already absent.

`deleting → leaked` after bounded cleanup fails. Any `leaked` resource fails the run.

## CleanupRecord

| Field | Type | Rules |
|---|---|---|
| `started_at` / `finished_at` | timestamp | Required |
| `trigger` | enum | `success`, `failure`, `timeout`, `interrupt` |
| `attempted` | boolean | Must be true for every terminal run |
| `resources` | OwnedResource[] | Final state for every ledger entry |
| `baseline_restored` | boolean | Must be true to pass |
| `remaining_count` | integer | Must be zero to pass |
| `remediation` | string[] | Required when remaining count is non-zero |

## VersionCoverage

| Field | Type | Rules |
|---|---|---|
| `runtime` | enum | `qemu`, `docker`, or `lightweight` |
| `device_family` | string | Supported template or lightweight kind |
| `version_id` | string | Available template version identity |
| `image_id` | string | Sanitized operator image identity/checksum reference |
| `coverage_level` | enum | `full-journey` or `lifecycle-connectivity` |
| `result` | enum | `passed` or `failed`; supported versions cannot be skipped |
| `interactions` | string[] | Inventory IDs used as evidence |

For each supported runtime/device family, exactly one or more versions may be `full-journey`, and every other
available version must have `lifecycle-connectivity` coverage.

## ClientObservation

| Field | Type | Rules |
|---|---|---|
| `client_id` | string | Distinguishes browser A, browser B, and API automation |
| `mutation_id` | string | Shared correlation identity |
| `event_sequence` | integer | Last ordered event observed |
| `resource_revision` | integer | Final authoritative revision |
| `observed_at` | timestamp | UTC RFC 3339 |
| `convergence_ms` | integer | Must be within 5 seconds for required shared-state scenarios |
| `local_preferences_hash` | string | Used only to prove browser-local independence; contains no preference values |
