# Implementation Plan: Constitution Gap Closure

**Branch**: `[002-constitution-gap-closure]` | **Date**: 2026-07-29 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/002-constitution-gap-closure/spec.md`

## Summary

Close the remaining constitution gaps by introducing a versioned, machine-validated compliance ledger;
converging the target host to one protected authoritative deployment; fixing actual-state and structured
failure parity; proving real template/image, guest-agent, bootstrap, and automatic networking readiness;
and rerunning the complete quality and recovery sequence against one identifiable production candidate.
The implementation reuses the existing domain, durable task, outbox event, ownership, reconciliation,
REST, MCP, and Vue paths rather than creating a parallel control plane.

## Technical Context

**Language/Version**: Go 1.26.x; TypeScript 5.8.x; Vue 3.5.x

**Primary Dependencies**: Gin 1.12.x, SQLite via `modernc.org/sqlite`,
`github.com/digitalocean/go-qemu`, Docker Engine client, `gopkg.in/yaml.v3`, Pinia, ECharts,
xterm.js, noVNC, host `dnsmasq`, nftables, iproute2, QEMU/KVM, Docker, and systemd

**Storage**: Existing SQLite WAL state and outbox; existing runtime ownership records; new SQLite
runtime-capability observations; version-controlled JSON compliance/evidence metadata; retained test
evidence outside distributable artifacts

**Testing**: Go unit, contract, security, privileged integration, recovery, and 100-cycle leak suites;
Vitest; Playwright local and target-host acceptance; deployment and evidence-schema validation

**Target Platform**: One x86_64 Ubuntu Linux host with KVM/QEMU, Docker, cgroup v2, bridges,
namespaces, nftables, packet-capture tools, systemd, and sufficient privileges

**Project Type**: Single-host web service with Vue SPA, REST/event/MCP interfaces, runtime adapters,
operator deployment assets, and compliance tooling

**Performance Goals**: Compliance validation finishes within 2 seconds for the project ledger;
runtime state converges across clients within 10 seconds for at least 95% of valid operations;
service restart adoption finishes within 60 seconds; host restart recovery finishes within 5 minutes;
valid live rewiring and NIC changes finish within 10 seconds in at least 95% of attempts

**Constraints**: No proprietary image, credential, bootstrap secret, packet payload, or private key in
source or evidence; no second externally reachable state authority; no shell interpolation of
untrusted values; no multi-host scheduling; commercial image validation is operator supplied and may
remain a time-bounded exception

**Scale/Scope**: Seven constitution principles and all mandatory boundaries/gates; one target host;
six declared template families; QEMU, Docker, and namespace runtimes; bridge/NAT/PC/L2/L3 objects;
UI, REST, event, MCP, console, capture, and artifact surfaces; 100 lifecycle/leak cycles

## Constitution Check

*GATE: Passed before Phase 0 research. Re-checked after Phase 1 design below.*

- **Shared state — PASS**: Runtime truth remains in the existing server-authoritative SQLite and
  observed-host model. Network-object and node-capability updates become transactional state plus
  ordered outbox events. The compliance ledger is release evidence, not a second topology database.
- **Control parity — PASS**: Runtime mutations continue through existing application command handlers
  and durable tasks. New node-capability reads are exposed equivalently to SPA, REST, and MCP. No UI
  mutation directly touches SQLite or runtime sockets.
- **Runtime scope — PASS**: The design remains single-host and uses only QEMU, Docker, Linux bridge,
  NAT, namespaces, cgroups, nftables, packet capture, and owned systemd helpers. No Cisco, cluster, or
  authentication scope is introduced.
- **Live operations — PASS**: Existing QMP/link/console/capture/filter behavior remains authoritative.
  This feature adds truthful active-state semantics and verifies reversible live operations rather
  than replacing them.
- **Safety and recovery — PASS**: NAT DHCP/RA helpers receive explicit ownership, bounded configuration,
  cancellation compensation, startup adoption, diagnostics, and deletion cleanup. Deployment exposure,
  credential rotation evidence, task errors, resource limits, and audit results are explicit gates.
- **Verification — PASS**: The plan requires unit, contract, frontend, target-host privileged,
  failure-injection, service/host recovery, security, and 100-cycle leak evidence tied to the exact
  candidate. Skips require an approved reason and repeatable procedure.
- **Image and secret hygiene — PASS**: Template readiness uses metadata, checksums, entitlement notes,
  and operator-provided paths only. Commercial images and secret values never enter source or retained
  evidence. Credential rotation is represented by non-secret attestation.

No constitution violation or temporary design exception is required. A missing legal FortiGate image
is a product-readiness exception record, not a change to the constitution or an implementation bypass.

## Project Structure

### Documentation (this feature)

```text
specs/002-constitution-gap-closure/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── compliance-ledger.schema.json
│   ├── evidence-record.schema.json
│   ├── deployment-authority.schema.json
│   ├── template-readiness.schema.json
│   ├── openapi-delta.yaml
│   ├── events.md
│   ├── mcp-tools.md
│   └── compliance-cli.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
├── netlabd/
└── netlab-compliance/              # Ledger/evidence validation and report command

internal/
├── compliance/                     # File-backed compliance domain, validation, freshness, reporting
├── domain/                         # Runtime capability and structured problem types
├── app/
│   ├── command/                    # Durable runtime mutations and failure mapping
│   ├── query/                      # Capability/readiness queries
│   ├── reconcile/                  # Network state, helper adoption, capability observation
│   └── task/                       # Terminal error and cancellation semantics
├── runtime/
│   ├── linuxnet/                   # NAT DHCPv4/DHCPv6/RA helper and diagnostics
│   ├── qemu/                       # QMP/QGA probes and image readiness
│   └── ownership/                  # Owned helper/process/resource records
├── store/sqlite/                   # Capability migration, transactional state/outbox writes
└── api/
    ├── http/                       # Release identity and node capability routes
    ├── mcp/                        # Equivalent capability query tool
    └── stream/                     # Ordered state/capability events

web/src/
├── api/                            # Generated contract types and operations
├── stores/                         # Event reconciliation without stale resurrection
├── features/templates/             # Genuine readiness and blocked/exception presentation
├── features/topology/              # Active network-object semantics and state parity
└── features/nodes/                 # QGA/QMP/console capability diagnostics

compliance/
├── constitution-ledger.json        # Current authoritative compliance status
├── deployment-authority.json       # Production/preview authority inventory
├── template-readiness.json         # Real image and capability readiness matrix
└── evidence/                       # Metadata/index files only; no prohibited payloads

deploy/
├── systemd/                        # One production service and isolated validation override
├── nftables/                       # Trusted-management-network policy template
└── config/                         # Explicit production/validation bind examples

scripts/
├── validate-compliance.sh
├── verify-production-authority.sh
└── run-constitution-acceptance.sh

tests/
├── contract/                       # Schemas, OpenAPI, MCP, event and parity checks
├── integration/                    # NAT helper, state parity, QGA readiness, failure paths
├── recovery/                       # Helper/service/host adoption
├── security/                       # Redaction, exposure and credential-pattern checks
└── e2e/                            # Multi-client and production-candidate workflows
```

**Structure Decision**: Compliance evidence is a repository-level release concern and is therefore
implemented as file-backed tooling under `internal/compliance`, `cmd/netlab-compliance`, and
`compliance/`. Runtime truth remains in existing domain/application/SQLite/runtime packages. The new
tool never mutates laboratory state and does not become another network service.

## Desired-State and Lifecycle Design

- `NetworkObject.desired_state=active` remains client intent. Runtime configuration changes observed
  state through `provisioning -> active`, `provisioning|active -> failed`, and
  `active|failed -> deleting -> deleted`.
- Network-object state writes use one SQLite transaction that updates state/error and appends the
  ordered outbox event. Clients never infer active state merely from task success.
- `RuntimeCapabilityObservation` records actual QMP, QGA, serial, VNC, image, and bootstrap readiness.
  Reconciliation may transition `unknown -> probing -> ready|unavailable|failed`; a new observation
  supersedes the prior record and emits an ordered event.
- NAT service intent is stored in network-object configuration. Owned DHCP/RA helper state is observed
  and adopted from runtime ownership; it does not create a second desired-state record.
- `ComplianceFinding` transitions `open|partial|blocked -> verified|accepted_exception`; verified
  findings become `stale` when candidate or evidence scope changes, and exceptions become `expired`
  when their condition is reached.
- `DeploymentInstance` transitions `candidate -> authoritative`; the former authority transitions
  `authoritative -> draining -> retired`. Validation instances are `isolated`, never authoritative.

## Runtime Adapter and Privilege Boundaries

- The Linux NAT adapter owns bridge addresses, nftables rules, lease state, generated helper
  configuration, systemd helper unit, PID, and diagnostics. All external commands use argv arrays and
  validated prefixes, ranges, interface names, and uplinks.
- `dnsmasq` is the target-host DHCPv4/DHCPv6/RA provider because it is already present on the acceptance
  host and can run in the foreground under an owned, restart-adoptable systemd unit. The adapter must
  report `capability_unsupported` when the binary is absent and must not silently provide only NAT.
- QEMU capability probing uses existing QMP/QGA Unix sockets through runtime adapters. A missing QGA
  is `unavailable` with image/template guidance, not a generic timeout failure.
- Firewall policy remains outside the application process. Deployment assets provide an explicit
  nftables management chain, approved CIDR variables, validation commands, and rollback procedure.
- The production convergence script verifies candidate digest, database/state directory, listener,
  service role, and current contracts before retiring the prior instance.

## Transactions, Tasks, Rollback, and Reconciliation

- Network-object observed state, structured error, revision timestamp, and outbox event commit together.
- Capability observations and their outbox events commit together and use monotonic per-capability
  revisions to reject stale probes.
- NAT creation remains a durable `network_object.create` task. Helper provisioning is checkpointed after
  bridge/rule setup and before active state. Cancellation or failure removes the owned helper, lease
  files, addresses, rules, and ownership entries, then records structured cleanup.
- Node startup remains the owning task for initial QMP/QGA/console probes. QGA absence does not roll back
  an otherwise healthy QEMU boot unless the selected template version declares QGA as required.
- Link, network, and deletion handlers wrap repository/runtime not-found errors with operation-specific
  terminal problems. Idempotent delete may treat already-absent state as success only after ownership
  and snapshot validation.
- Startup and periodic reconciliation adopt QEMU, Docker, namespace, bridge, NAT helper, capture,
  console, cgroup, socket, and rule ownership; ambiguous resources are quarantined and audited.
- Compliance generation is deterministic and read-only. Invalid or stale evidence fails the release
  gate but never changes runtime state.

## Implementation Phases

### Phase A — Evidence Truth and Candidate Identity

1. Add compliance ledger/evidence schemas, validator, freshness rules, report generator, and tests.
2. Add release identity (`version`, build identifier, contract digest) to capability responses and
   evidence capture.
3. Populate the initial ledger from the July 29 audit and reconcile contradictory legacy report text.

### Phase B — Single Secure Production Authority

1. Add deployment authority inventory and verification tooling.
2. Provide production and isolated-validation service profiles plus trusted-network nftables policy.
3. Deploy one current candidate, migrate/adopt state as planned, restrict or retire preview instances,
   verify contracts, and record non-secret credential-rotation attestation.

### Phase C — Truthful Runtime State and Failures

1. Make network-object state updates transactional and event-producing.
2. Add `active` visual semantics and event-driven client reconciliation tests.
3. Normalize link and other remaining generic task failures with cleanup/action data.
4. Add durable runtime capability observations and SPA/REST/MCP parity.

### Phase D — Genuine Template and Automatic Network Readiness

1. Add template readiness metadata and distinguish mechanics-only, genuine, blocked, and accepted
   exception evidence.
2. Add QGA readiness probing and actionable unsupported behavior.
3. Extend NAT objects with owned DHCPv4/DHCPv6/RA service configuration, adoption, diagnostics,
   cancellation, and cleanup.
4. Validate Ubuntu, VyOS, FancyWAN, BusyBox, and Ubuntu container workloads; validate FortiGate only
   with operator-approved media or record a time-bounded exception.

### Phase E — Current-Candidate Acceptance and Governance

1. Run all local quality gates and target-host privileged workflows against the exact candidate.
2. Exercise multi-client parity, live rewiring/QMP, console, capture/filter, port mapping, quotas,
   service restart, host restart, failure injection, and 100-cycle cleanup.
3. Scan evidence for prohibited content, record review metadata, close or explicitly retain findings,
   and publish one non-contradictory final report.

## Post-Design Constitution Re-Check

- **Shared state/control parity**: PASS. New runtime observations use the same store/outbox/query paths;
  compliance files are read-only evidence.
- **Approved runtime scope**: PASS. The only new runtime dependency is an owned single-host DHCP/RA
  helper for the already-approved NAT/networking scope.
- **Live topology/observability**: PASS. Active-state events, capability observations, and failure
  diagnostics close known observability gaps.
- **Automation-first interfaces**: PASS. Release identity and node capabilities have REST/MCP/UI parity;
  mutations remain durable tasks.
- **Resource safety/isolation**: PASS. Helper ownership, validation, firewall policy, cleanup, and
  quarantine are explicit.
- **Lifecycle correctness/recoverability**: PASS. State transitions, checkpoints, cancellation,
  startup adoption, host recovery, and leak tests are specified.
- **Image/secret hygiene**: PASS. Genuine image evidence is metadata-only and commercial media remains
  operator-controlled.

## Complexity Tracking

No constitution violation requires justification. The compliance CLI and file schemas are deliberately
kept outside the running control plane to avoid adding another database, service, or authorization
surface.
