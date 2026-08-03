<!--
Sync Impact Report
- Version change: 1.0.0 -> 1.1.0
- Modified principles: none
- Added principles: Local-First Development and Traceable Deployment
- Modified sections: Product and Technology Boundaries; Delivery Workflow and Quality Gates
- Added sections: none
- Removed sections: none
- Templates requiring updates:
  - ✅ .specify/templates/plan-template.md
  - ✅ .specify/templates/spec-template.md
  - ✅ .specify/templates/tasks-template.md
  - ✅ .specify/templates/commands/*.md (directory absent; no files to update)
  - ✅ AGENTS.md
  - ✅ specs/002-constitution-gap-closure/quickstart.md
- Follow-up TODOs:
  - ⚠ Restore or re-clone the repository Git metadata before the next implementation milestone;
    the current workspace has no `.git` directory and therefore cannot create traceable commits.
-->
# NetLab Constitution

## Core Principles

### I. Shared State and Control-Plane Parity

Topology definitions, node lifecycle state, interface state, captures, and task results MUST have
one server-authoritative representation shared by every browser session, API client, and MCP client.
Browser and automation sessions MUST NOT invalidate one another or observe account-scoped runtime
state. Every state-changing UI operation MUST use the same documented application service and
authorization boundary as the public API. Concurrent mutations MUST use explicit revision,
idempotency, or conflict semantics rather than last-writer ambiguity. This principle prevents the
session and visibility failures that motivated replacing EVE-NG.

### II. Deliberate Runtime Scope

The product MUST remain a single-host network simulator centered on QEMU virtual machines, Docker
containers, Linux bridges, NAT bridges, and Linux network namespaces. Supported QEMU template
families are FancyWAN, Ubuntu, FortiGate, and VyOS; supported Docker template families are BusyBox
and Ubuntu. A template MUST describe selectable image versions, interfaces, console methods,
default resources, network drivers, bootstrap capabilities, and health behavior without embedding
vendor-specific orchestration in the core scheduler. Cisco support, exhaustive EVE-NG device
compatibility, clustering, and password-based user authentication are out of scope unless this
constitution is amended.

### III. Live Topology and Observability

Users and automation MUST be able to connect and disconnect links while nodes run, and QEMU NIC
changes MUST use QMP with an observable, reversible result. The platform MUST provide Telnet and VNC
console access, live packet capture compatible with Wireshark workflows, and traffic filters that
show the path of matching packets. Captures MUST be streamable or downloadable through stable API
references with metadata, bounded retention, cancellation, and explicit truncation/error status;
an MCP response MUST return structured capture metadata and a retrievable artifact or bounded packet
summary rather than unbounded binary data. Runtime topology and observation data MUST remain
consistent across UI, API, and MCP clients.

### IV. Automation-First Interfaces

All durable resources and long-running actions MUST be accessible through versioned HTTP APIs, and
LLM-oriented operations MUST additionally be exposed through MCP tools with typed inputs and
structured outputs. Operations that may be retried MUST accept idempotency keys or be inherently
idempotent. Long-running operations MUST return task identifiers and expose status, progress,
result, error, and cancellation state. QEMU nodes MUST support cloud-init seed ISO workflows for
VyOS, FancyWAN, and Ubuntu where the guest supports them; arbitrary guest command execution through
the QEMU guest agent; host port forwarding; and declared NIC driver selection. UI-only control paths
are prohibited.

### V. Resource Safety and Isolation

Every node MUST have explicit CPU, memory, storage, interface, and process ownership metadata.
QEMU CPU controls MUST support a virtual CPU count independent from a host CPU-time quota, including
the case where two guest vCPUs are limited to the time budget of one host core. Docker, QEMU,
network namespaces, bridges, port mappings, captures, and helper processes MUST be created through
validated adapters and MUST be cleaned up after normal deletion, partial failure, restart, or crash
reconciliation. Arbitrary guest commands MUST be treated as privileged operations: inputs, timeouts,
output limits, exit status, and audit records are mandatory. Host shell interpolation of untrusted
values is prohibited.

### VI. Lifecycle Correctness and Recoverability

Node and link lifecycles MUST be modeled as explicit state machines with legal transitions,
timeouts, and actionable failure states. SQLite stores desired state and durable operation history;
actual QEMU, Docker, process, and Linux networking state MUST be reconciled at service startup and
periodically thereafter. Each runtime adapter MUST have unit tests for command/QMP construction and
failure mapping, contract tests for API/MCP schemas, and integration tests for create, start, stop,
hot-plug, capture, and cleanup behavior. Tests for a changed lifecycle MUST be written before or
with the implementation and MUST demonstrate the failure before the fix when correcting a defect.
No feature is complete if it only passes a mocked happy path.

### VII. Image, Secret, and Execution Hygiene

The repository and distributable artifacts MUST NOT contain proprietary appliance images,
third-party credentials, host passwords, cloud-init secrets, or captured user traffic. Image
records MUST include source, version, checksum, format, and license/entitlement notes; operators are
responsible for supplying images they are legally entitled to run. FortiGate and other commercial
images MAY be used only after license review and MUST NOT be fetched automatically from unofficial
collections by production code. Deployment credentials MUST be supplied out of band, rotated after
exposure, and excluded from logs, fixtures, examples, and generated documentation.

### VIII. Local-First Development and Traceable Deployment

All source, specification, migration, build, and test changes MUST be implemented in the local Git
worktree before any production-target modification. Each independently testable milestone MUST pass
its applicable local quality gates and MUST be recorded as a focused Git commit before deployment.
Deployments to `10.72.1.7` MUST use artifacts built from an identified clean commit; the deployed
commit SHA, artifact digest, migration state, deployment time, and target-host test result MUST be
recorded. Source files MUST NOT be edited directly on `10.72.1.7`. If target validation fails, the
fix MUST return to the local worktree, receive tests and a new milestone commit, and then be
redeployed. Before changing an established subsystem, implementers SHOULD inspect relevant
`git log` and `git blame` history to preserve intentional compatibility and understand prior fixes.
This workflow provides reproducible rollback, reviewable history, and a reliable distinction between
development state and the deployed system.

## Product and Technology Boundaries

- The backend MUST use Go with Gin for HTTP serving, `github.com/digitalocean/go-qemu` for QEMU/QMP
  integration where suitable, and SQLite for durable single-host state.
- The frontend MUST be a Vue single-page application and MUST consume the same API contracts offered
  to external clients; direct access to the database or runtime sockets is prohibited.
- The production deployment and privileged validation target is `10.72.1.7`, a single Linux server
  with KVM/QEMU, Docker, Linux bridges, NAT, namespaces, traffic control, packet capture tooling, and
  sufficient privileges. Multi-node scheduling is prohibited.
- The PC/network-namespace capability MUST cover IPv4, IPv6, DHCPv4, DHCPv6, and IPv6 SLAAC.
- Device image versions MUST be data-driven template variants. Replacing an image version MUST NOT
  require changing scheduler or topology-domain code.
- The absence of account/password authentication defines a trusted deployment boundary, not an
  absence of safety controls. Management endpoints MUST support configurable bind addresses and
  deployment documentation MUST require network-layer access restriction outside isolated labs.
- Reference inspection of an existing EVE-NG installation is permitted for behavioral comparison,
  but copied code, proprietary assets, credentials, and undocumented compatibility dependencies are
  prohibited.

## Delivery Workflow and Quality Gates

- Each feature specification MUST state affected resource lifecycles, UI/API/MCP parity, concurrent
  access behavior, failure recovery, observability, cleanup, and measurable acceptance criteria.
- Plans MUST document the desired-state model, runtime adapter boundaries, state transitions,
  SQLite transactions, asynchronous task semantics, rollback/reconciliation, and privilege impact.
- Tasks MUST include contract tests, adapter tests, integration tests on a Linux virtualization host,
  cleanup/leak checks, and documentation for externally visible behavior. Tests are not optional for
  runtime, networking, API, MCP, capture, or lifecycle changes.
- Pull requests MUST identify constitution impacts and MUST pass formatting, static analysis, unit,
  contract, and applicable integration tests. Any skipped hardware/privileged test MUST have a
  recorded reason and a repeatable target-host validation procedure.
- Implementation MUST proceed through explicit local milestones. A milestone is complete only when
  its focused tests pass and its changes are committed. Uncommitted or dirty-worktree artifacts MUST
  NOT be deployed to `10.72.1.7`.
- Deployment MUST record the source commit SHA and artifact digest before service replacement, then
  run the documented smoke, contract, privileged integration, recovery, and leak checks applicable
  to the milestone. Rollback MUST select a previously recorded commit/artifact rather than manually
  patching the target host.
- Changes MUST be delivered in independently demonstrable slices. Complexity beyond the declared
  single-host scope requires written justification in the implementation plan and a constitution
  amendment when it changes product boundaries.

## Governance

This constitution supersedes conflicting project practices and generated templates. Amendments MUST
be proposed as a reviewed documentation change that explains motivation, migration impact, affected
templates, and compatibility consequences. Approval requires explicit project-owner acceptance.

Constitution versions use semantic versioning: MAJOR for removal or incompatible redefinition of a
principle or product boundary, MINOR for a new principle or materially expanded mandatory guidance,
and PATCH for non-semantic clarification. Every amendment MUST update the Sync Impact Report,
version, and amendment date and MUST propagate changes to dependent Spec Kit templates.

Every feature plan MUST perform a Constitution Check before research and after design. Reviewers MUST
reject unexplained violations. Temporary exceptions MUST identify an owner, scope, risk, expiration
condition, and removal task; an exception cannot silently redefine the constitution.

**Version**: 1.1.0 | **Ratified**: 2026-07-23 | **Last Amended**: 2026-08-03
