# Data Model: Constitution Gap Closure

## Overview

This feature has two data planes:

1. **Release evidence plane**: version-controlled JSON documents for compliance, deployment authority,
   template readiness, exceptions, and acceptance evidence. These documents are not runtime topology
   state and are never served as another control plane.
2. **Runtime observation plane**: server-authoritative SQLite records for per-node capability readiness
   and existing network-object desired/actual state, coupled to ordered outbox events.

## Compliance Finding

Represents one mandatory constitution statement or product boundary.

| Field | Type | Rules |
|---|---|---|
| `id` | string | Stable identifier such as `CONST-I-01`; unique in ledger |
| `principle` | string | Constitution section or boundary name |
| `statement` | string | One testable mandatory obligation |
| `severity` | enum | `critical`, `high`, `medium`, `low` |
| `status` | enum | `open`, `partial`, `blocked`, `verified`, `accepted_exception`, `stale`, `expired` |
| `owner` | string | Accountable role or person; cannot be empty for non-verified states |
| `requirement_ids` | string[] | Related feature requirements |
| `evidence_ids` | string[] | Accepted evidence records supporting status |
| `exception_id` | string? | Required when status is `accepted_exception` or `expired` |
| `next_action` | string | Required unless status is `verified` |
| `candidate_id` | string? | Candidate currently covered by verification |
| `last_reviewed_at` | timestamp | Required UTC timestamp |

### State Transitions

```text
open -> partial|blocked|verified|accepted_exception
partial -> open|blocked|verified|accepted_exception
blocked -> open|partial|verified|accepted_exception
verified -> stale
accepted_exception -> expired|verified|open
stale -> open|partial|blocked|verified|accepted_exception
expired -> open|blocked|verified|accepted_exception
```

`verified` requires at least one accepted, non-stale evidence record tied to the current candidate or
an unchanged scope digest. `accepted_exception` requires an approved exception. Task completion alone
cannot transition a finding to `verified`.

## Evidence Record

Represents reproducible proof for one or more compliance findings.

| Field | Type | Rules |
|---|---|---|
| `id` | string | Unique and immutable |
| `kind` | enum | `unit`, `contract`, `frontend`, `integration`, `recovery`, `leak`, `security`, `manual`, `deployment`, `review` |
| `status` | enum | `collected`, `validated`, `accepted`, `rejected`, `stale`, `superseded` |
| `candidate_id` | string | Exact candidate identifier |
| `release_version` | string | Human-readable version |
| `binary_digest` | SHA-256 | Required for deployable-candidate evidence |
| `contract_digest` | SHA-256 | Digest of approved external contracts |
| `scope_digest` | SHA-256 | Digest of files/requirements covered by the evidence |
| `finding_ids` | string[] | At least one finding |
| `procedure` | string | Repeatable command or documented scenario |
| `target` | object | Host capability facts without credentials |
| `started_at`, `finished_at` | timestamp | `finished_at >= started_at` |
| `outcome` | enum | `passed`, `failed`, `skipped`, `blocked` |
| `cleanup` | object | Baseline, final counts, remaining resources, remediation |
| `redaction` | object | Scan result and prohibited-content count |
| `artifacts` | array | Metadata paths/digests only; prohibited payloads disallowed |
| `supersedes` | string[] | Older evidence replaced by this record |

### Lifecycle

```text
collected -> validated -> accepted|rejected
accepted -> stale|superseded
rejected -> superseded
stale -> superseded
```

An accepted record becomes stale when the candidate, contract digest, scope digest, or relevant host
capability changes.

## Accepted Exception

Represents a time-bounded approved deviation.

| Field | Type | Rules |
|---|---|---|
| `id` | string | Unique |
| `finding_id` | string | Exactly one finding |
| `owner` | string | Required |
| `scope` | string | Explicitly bounded |
| `risk` | string | Required |
| `motivation` | string | Required |
| `approved_by` | string | Project owner or delegated reviewer |
| `approved_at` | timestamp | Required |
| `expiration_condition` | string | Date or objective condition |
| `removal_task` | string | Required actionable work reference |
| `status` | enum | `proposed`, `approved`, `expired`, `closed`, `rejected` |

### State Transitions

```text
proposed -> approved|rejected
approved -> expired|closed
expired -> approved|closed
```

No exception may silently alter a constitution statement.

## Deployment Instance

Represents a NetLab control-plane deployment on the target host.

| Field | Type | Rules |
|---|---|---|
| `id` | string | Unique instance name |
| `role` | enum | `candidate`, `authoritative`, `isolated`, `draining`, `retired` |
| `host_id` | string | Stable non-secret host identifier |
| `listen_address` | string | Required |
| `management_scope` | string[] | Approved CIDRs or isolation boundary |
| `state_directory` | string | Must not be shared by concurrent instances |
| `database_path` | string | Must not be shared by concurrent instances |
| `service_name` | string | Required |
| `candidate_id` | string | Exact deployed candidate |
| `contract_digest` | SHA-256 | Must match approved contract |
| `externally_reachable` | boolean | Only one authoritative instance may be true per host |
| `verified_at` | timestamp | Required for authoritative/isolated states |

### State Transitions

```text
candidate -> authoritative|isolated|retired
authoritative -> draining
draining -> retired
isolated -> candidate|retired
```

Invariant: one host has at most one `authoritative` instance with `externally_reachable=true`.

## Template Readiness Record

Represents the evidence status of a declared template family and image version.

| Field | Type | Rules |
|---|---|---|
| `template_key` | string | Declared template identifier |
| `template_version` | string | Exact version |
| `runtime_kind` | enum | `qemu`, `docker` |
| `status` | enum | `unavailable`, `mechanics_validated`, `genuine_validated`, `blocked`, `accepted_exception` |
| `image` | object? | Name, version, digest, format, source type, license status; no image path requiring secret access |
| `genuine_workload` | boolean | Must be true for `genuine_validated` |
| `bootstrap` | object | Declared, tested, outcome, evidence ID |
| `console` | object | Declared modes and tested modes |
| `capabilities` | object | QMP, QGA, hotplug, guest command, port mapping readiness |
| `lifecycle` | object | Create/start/stop/restart/delete outcomes |
| `cleanup` | object | Leak result and evidence ID |
| `exception_id` | string? | Required for `accepted_exception` |

An Ubuntu substitute may set `mechanics_validated=true` for a non-Ubuntu template but cannot set
`genuine_workload=true` or status `genuine_validated`.

## Runtime Capability Observation

Server-authoritative actual capability state for one node.

| Field | Type | Rules |
|---|---|---|
| `node_id` | ID | Foreign key to node; cascade delete |
| `capability` | enum | `image`, `bootstrap`, `qmp`, `qga`, `serial`, `vnc`, `hotplug`, `guest_exec`, `port_mapping` |
| `revision` | integer | Monotonic per `(node_id, capability)` |
| `state` | enum | `unknown`, `probing`, `ready`, `unavailable`, `failed` |
| `required` | boolean | Derived from template version |
| `details` | JSON | Bounded non-secret diagnostics |
| `problem` | structured problem? | Required for `unavailable` or `failed` when actionable |
| `observed_at` | timestamp | Required |

### State Transitions

```text
unknown -> probing
probing -> ready|unavailable|failed
ready -> probing|unavailable|failed
unavailable|failed -> probing
```

State and ordered outbox event commit in one transaction. A missing optional QGA does not change node
`observed_state=running`; it changes only the QGA/guest-exec capability state.

## NAT Service Configuration and Observation

Extends the existing NAT network-object configuration and runtime ownership.

### Desired Configuration

| Field | Type | Rules |
|---|---|---|
| `ipv4_prefix` | CIDR | Existing; non-overlapping owned prefix |
| `ipv6_prefix` | CIDR? | Required for DHCPv6 or RA |
| `uplink` | interface name | Validated host interface |
| `dhcpv4.enabled` | boolean | Default false for backward compatibility |
| `dhcpv4.range_start`, `range_end` | IPv4 | Inside prefix, exclude gateway |
| `dhcpv4.lease_seconds` | integer | Bounded |
| `dhcpv6.enabled` | boolean | Requires IPv6 prefix |
| `dhcpv6.range_start`, `range_end` | IPv6 | Inside prefix |
| `router_advertisement.enabled` | boolean | Requires IPv6 prefix |
| `dns_servers` | IP[] | Bounded non-empty when DHCP enabled |

### Runtime Observation

Uses existing ownership records for helper unit, PID, generated config, lease files, bridge, and rules.
Diagnostics expose helper state, lease counts, RA state, addresses, translation state, and cleanup state.

```text
absent -> provisioning -> active
provisioning -> failed
active -> degraded|failed|deleting
degraded|failed -> provisioning|deleting
deleting -> absent
```

## Acceptance Run

Aggregates one release-candidate validation.

| Field | Type | Rules |
|---|---|---|
| `id` | string | Unique |
| `candidate_id` | string | Required |
| `status` | enum | `planned`, `running`, `passed`, `failed`, `blocked`, `cancelled` |
| `gate_results` | map | Every required gate present |
| `scenario_evidence_ids` | string[] | Linked evidence records |
| `exceptions` | string[] | Approved exceptions only |
| `cleanup_baseline` | object | Captured before run |
| `cleanup_final` | object | Captured after run |
| `redaction_result` | object | Required terminal result |
| `conclusion` | string | One non-contradictory release statement |
| `started_at`, `finished_at` | timestamp | Required according to state |

```text
planned -> running
running -> passed|failed|blocked|cancelled
```

`passed` requires all mandatory gates passed, baseline restored, redaction passed, and every exception
approved and unexpired.

## Relationships

- A Compliance Finding references many Evidence Records; one Evidence Record may support many findings.
- A Compliance Finding has zero or one active Accepted Exception.
- An Acceptance Run aggregates Evidence Records and approved exceptions for one candidate.
- A Deployment Instance references the candidate and contract digest used by its Acceptance Run.
- A Template Readiness Record references image metadata, capability evidence, and optional exception.
- A Node has many Runtime Capability Observations and one latest observation per capability.
- A NAT NetworkObject owns one supervised helper and its runtime ownership records.
