# Research: Constitution Gap Closure

## Decision 1: Use a Versioned File-Backed Compliance Ledger

**Decision**: Store the constitution ledger, deployment authority inventory, template readiness matrix,
and evidence metadata as JSON files validated by JSON Schema and a Go command.

**Rationale**: Compliance status is release evidence rather than shared topology state. A deterministic,
reviewable file is easy to diff, validate in CI, and bind to an exact candidate without adding a second
runtime database or management service.

**Alternatives considered**:

- Store compliance findings in SQLite and expose them through the product API: rejected because it adds
  runtime authority and migration complexity for release-process data.
- Maintain Markdown only: rejected because contradictory status, missing ownership, and stale evidence
  cannot be reliably validated.
- Use an external governance product: rejected because it makes local release gates non-reproducible and
  introduces an unnecessary dependency.

## Decision 2: Bind Evidence to Candidate and Scope Digests

**Decision**: Each evidence record includes release version, binary/build digest, contract digest,
covered requirement IDs, target facts, and cleanup/redaction results. Evidence becomes stale when the
candidate or covered scope digest changes.

**Rationale**: Dates alone do not reveal whether newer frontend, runtime, contract, or deployment changes
invalidate a historical pass.

**Alternatives considered**:

- Expire all evidence after a fixed number of days: rejected because unchanged behavior may remain valid
  while same-day code changes may invalidate evidence immediately.
- Trust task completion markers: rejected because the current audit found completed tasks with partial
  evidence.

## Decision 3: Keep One External Authority and Isolate Validation

**Decision**: The target host has one externally reachable `authoritative` instance. Any validation
instance must bind to loopback or an isolated management namespace and use a separate, clearly labeled
state directory. Deployment verification fails when two authoritative/reachable instances exist.

**Rationale**: Separate visible instances with separate databases recreate the state-divergence problem
that motivated the product.

**Alternatives considered**:

- Merge multiple instances through shared SQLite: rejected because concurrent service ownership and
  runtime reconciliation would be unsafe.
- Allow multiple public instances with operator naming: rejected because browsers and automation can
  still select different authorities.

## Decision 4: Enforce the Trusted Boundary with Host Networking

**Decision**: Provide an nftables deployment policy that permits configured management CIDRs and rejects
other access to application, MCP, console, and artifact listeners. The application retains configurable
bind addresses and startup warnings.

**Rationale**: The constitution explicitly defines a trusted deployment boundary without application
authentication. Host networking is the correct enforcement layer and can protect every protocol.

**Alternatives considered**:

- Add application authentication: rejected because it changes an explicit product boundary.
- Rely only on startup warnings: rejected because a warning does not deny access.
- Bind production to loopback behind a reverse proxy: viable but not required; retained as an operator
  deployment alternative when the proxy enforces the same boundary.

## Decision 5: Fix Network-Object Truth at Store, Event, and UI Layers

**Decision**: Persist network-object observed-state changes and their outbox events in one transaction,
then teach frontend semantics that `active` is healthy/running. Add snapshot/event/MCP parity tests.

**Rationale**: The dynamic NAT test showed both failure modes: the host and database can be active while
the client lacks a state event, and the visual semantic currently treats `active` as stopped/unknown.

**Alternatives considered**:

- Refresh the full laboratory after every network task: rejected because it is race-prone and bypasses
  ordered incremental state.
- Map `active` to `running` only in the API: rejected because it hides the domain distinction and does not
  solve missing events.

## Decision 6: Store Runtime Capability Observations Separately

**Decision**: Add durable per-node observations for QMP, QGA, serial, VNC, image, and bootstrap readiness,
with independent revisions and ordered events.

**Rationale**: Node lifecycle can be healthy while an optional capability such as QGA is unavailable.
Conflating capability readiness with node running state produces generic timeouts or false claims.

**Alternatives considered**:

- Put probe results into opaque node configuration: rejected because configuration is desired input and
  lacks independent revision/history.
- Fail every QEMU start when QGA is absent: rejected because QGA is optional for some templates.
- Probe only when a guest command is requested: rejected because the UI and automation cannot know
  readiness before attempting the operation.

## Decision 7: Use an Owned `dnsmasq` Helper for NAT DHCP and RA

**Decision**: Extend NAT configuration with optional DHCPv4, DHCPv6, DNS, and router-advertisement
settings. Run `dnsmasq` in the foreground under a deterministic owned systemd unit and record its unit,
PID, configuration, lease files, and state in runtime ownership.

**Rationale**: `dnsmasq` is present on the acceptance host, supports DHCPv4/DHCPv6/RA in one bounded
process, and fits the existing supervised-helper and startup-adoption model. It removes undocumented
manual guest networking from NAT validation.

**Alternatives considered**:

- Require every guest to use static cloud-init addressing: rejected because attachment order and address
  allocation become operator-specific and do not cover DHCP requirements.
- Embed a DHCP server in Go: rejected because it adds protocol and security complexity outside project
  goals.
- Use separate DHCP and RA daemons: rejected because it increases ownership and recovery surfaces.

## Decision 8: Distinguish Genuine Workload Acceptance from Mechanics Tests

**Decision**: Template readiness records separate `mechanics_validated` from `genuine_workload_validated`.
Only the latter closes device-family acceptance. Commercial media may be `blocked` or covered by an
approved expiring exception.

**Rationale**: An Ubuntu-derived image can validate QEMU, cloud-init, QMP, and console mechanics but does
not prove VyOS, FancyWAN, or FortiGate behavior.

**Alternatives considered**:

- Treat any bootable qcow2 as template validation: rejected because it misrepresents product support.
- Automatically fetch vendor images: rejected by image hygiene and licensing rules.

## Decision 9: Normalize Errors at Operation Boundaries

**Decision**: Every durable handler supplies an operation-specific terminal problem. Repository
not-found errors are translated according to operation semantics, and cleanup validation occurs before
idempotent success.

**Rationale**: A runner-level fallback cannot know whether absence is expected, retryable, already
cleaned, or a data-integrity problem.

**Alternatives considered**:

- Improve only the generic task-runner fallback: rejected because it still lacks phase-specific cleanup
  and operator guidance.
- Preserve raw adapter/database strings: rejected because REST/MCP/task parity requires stable codes.

## Decision 10: Produce One Candidate-Wide Acceptance Report

**Decision**: A single orchestration command records local gates, target-host scenarios, candidate
identity, retained evidence, cleanup baseline, redaction, exceptions, and final conclusion. Detailed
scenario evidence remains linked but cannot override the top-level status silently.

**Rationale**: The current validation document contains an older partial conclusion and later broader
passes, which makes release status ambiguous.

**Alternatives considered**:

- Continue appending free-form evidence to one long report: rejected because obsolete conclusions remain
  authoritative-looking.
- Keep separate uncoordinated local and target reports: rejected because candidate and cleanup identity
  can diverge.
