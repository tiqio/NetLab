# Feature Specification: Constitution Gap Closure

**Feature Branch**: `[002-constitution-gap-closure]`

**Created**: 2026-07-29

**Status**: Draft

**Input**: User description: "审查宪法，看有哪些还没有完成"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Obtain a Truthful Compliance Baseline (Priority: P1)

As the project owner, I need every mandatory constitution statement mapped to current, reproducible
evidence so that an item cannot be described as complete merely because a task checkbox was marked.

**Why this priority**: All later remediation depends on knowing which capabilities are genuinely
complete, partially proven, blocked by operator assets, or not implemented.

**Independent Test**: Review the generated compliance ledger and independently reproduce a sample
from every constitution principle, including at least one passing item, one open item, and one
accepted exception.

**Acceptance Scenarios**:

1. **Given** the current constitution and project artifacts, **When** the audit is generated, **Then**
   every mandatory statement has exactly one status, an owner, evidence date, evidence location, and
   next action.
2. **Given** a task marked complete whose evidence says "partial", **When** the audit reconciles the
   conflict, **Then** the compliance status remains open or partial rather than complete.
3. **Given** evidence from an older build, **When** behavior-affecting changes have occurred since that
   evidence, **Then** the evidence is marked stale until repeated on the current release candidate.

---

### User Story 2 - Operate One Secure Authoritative Service (Priority: P1)

As an operator, I need one clearly designated production control plane whose state, contracts, and
network exposure match the approved release so that browsers and automation cannot accidentally use
different databases or unsafe preview instances.

**Why this priority**: Multiple independently visible control planes recreate the state divergence
the product was created to eliminate, while unrestricted unauthenticated access violates the trusted
deployment boundary.

**Independent Test**: From two browsers, an HTTP client, and an MCP client, create and mutate one
laboratory through the designated endpoint and verify that all clients observe one ordered state;
also verify that unapproved source networks cannot reach management endpoints.

**Acceptance Scenarios**:

1. **Given** production and preview instances on the same host, **When** deployment convergence is
   completed, **Then** exactly one externally reachable instance is authoritative and every other
   instance is retired or restricted to an isolated validation boundary.
2. **Given** the approved release contracts, **When** the production endpoint is inspected, **Then**
   every documented route and capability is present and no obsolete contract is served.
3. **Given** an unauthenticated management endpoint, **When** a client outside the approved management
   network attempts access, **Then** network-layer controls deny the connection.
4. **Given** credentials previously exposed during development, **When** production readiness is
   assessed, **Then** the credentials have been rotated and no credential value appears in repository,
   logs, fixtures, captures, or generated evidence.

---

### User Story 3 - Trust Runtime State and Failure Reporting (Priority: P1)

As a topology user or automation client, I need desired state, actual state, task outcomes, and cleanup
results to agree across the canvas, API, MCP, and host reality so that I can safely continue operating
without manual refreshes or host inspection.

**Why this priority**: A resource that works on the host while appearing stopped, or a failed task that
only says "not found", is not operationally trustworthy.

**Independent Test**: Exercise every supported resource lifecycle and compare the final UI, HTTP,
MCP, event-stream, audit, and host-observed state, including injected partial failures and deletion.

**Acceptance Scenarios**:

1. **Given** an active bridge, NAT bridge, namespace node, QEMU node, Docker node, link, capture, or
   traffic filter, **When** convergence completes, **Then** all clients report the same actual state as
   the host within the defined convergence window.
2. **Given** a resource is deleted, **When** its durable deletion task succeeds, **Then** it disappears
   from every client without refresh and all owned resources are removed.
3. **Given** a lifecycle failure, **When** the task reaches a terminal state, **Then** the result names
   the affected resource, phase, retryability, cleanup result, and operator action rather than a generic
   error.
4. **Given** two clients mutate the same revision concurrently, **When** the operations are processed,
   **Then** one ordered result is accepted and the other receives an explicit replay or conflict result.

---

### User Story 4 - Prove Supported Templates with Real Workloads (Priority: P2)

As an operator, I need each declared template family validated with a representative, legally supplied
image and its genuine bootstrap and health behavior so that template support does not mean merely
booting an Ubuntu substitute under another label.

**Why this priority**: The constitution explicitly declares supported QEMU and Docker families, and
users rely on that declaration when building laboratories.

**Independent Test**: Select each available template version, create a node with its intended image,
apply supported bootstrap data, start it, reach its declared console or health check, and cleanly
delete it without leaked resources.

**Acceptance Scenarios**:

1. **Given** Ubuntu, VyOS, and FancyWAN QEMU templates, **When** operator-approved representative images
   are available, **Then** each boots its real workload, consumes its declared bootstrap format, exposes
   accurate health, and survives stop/start recovery.
2. **Given** a FortiGate image has not passed license review, **When** template readiness is reported,
   **Then** FortiGate is explicitly marked blocked by operator asset or accepted exception and is not
   represented as dynamically validated.
3. **Given** BusyBox and Ubuntu container templates, **When** version selection and startup are tested,
   **Then** the selected immutable image digest is the one actually running.
4. **Given** a QEMU template declares guest-command capability, **When** its chosen image lacks a ready
   guest agent, **Then** creation or health reporting clearly identifies the missing prerequisite rather
   than advertising a working capability.

---

### User Story 5 - Complete Current-Build Acceptance and Governance (Priority: P2)

As a release reviewer, I need all constitution quality gates repeated against the exact production
candidate and summarized without contradictory conclusions so that release approval is based on
current evidence.

**Why this priority**: Historical acceptance is valuable but cannot prove recent frontend, lifecycle,
networking, or deployment changes.

**Independent Test**: Execute the documented validation sequence against the candidate, inspect the
resulting evidence package, and confirm that all cleanup baselines return to their original state.

**Acceptance Scenarios**:

1. **Given** the current candidate, **When** unit, contract, frontend, privileged integration, recovery,
   security, and resource-leak validation completes, **Then** one report records every result and any
   skipped test has a reason and repeatable procedure.
2. **Given** service restart and full host restart scenarios, **When** recovery completes, **Then** owned
   resources are adopted or restored according to laboratory policy and all clients converge on the
   same final state.
3. **Given** a failure-injection run, **When** cleanup finishes, **Then** the ownership baseline is fully
   restored and retained evidence contains no credentials, bootstrap secrets, image bytes, or packet
   payloads.
4. **Given** the final report, **When** a reviewer compares its summary with detailed evidence, **Then**
   no item is simultaneously described as complete and partial.

### Edge Cases

- A required vendor image is unavailable or legally unusable when the release is otherwise ready.
- A previously passing acceptance result becomes stale after a runtime, networking, frontend, or
  contract change.
- The production service is current but an old preview service remains reachable with separate state.
- Network controls protect one management port but leave another preview or MCP endpoint exposed.
- A resource is operational on the host but its stored actual state remains provisioning, stopped, or
  unknown.
- An asynchronous task succeeds while a delayed list or snapshot response attempts to restore deleted
  data in a client.
- Cleanup succeeds for the primary resource but leaves an interface, process, namespace, socket,
  bridge attachment, rule, capture artifact, or ownership record.
- A test is marked passed using a placeholder image that does not exercise the declared device family.
- Acceptance evidence is retained after a failed run and includes data that should have been redacted.
- The current workspace lacks the review metadata required to prove that governance gates were applied.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The project MUST maintain a constitution compliance ledger mapping every mandatory
  constitution statement to `verified`, `partial`, `open`, `blocked`, or `accepted_exception`.
- **FR-002**: Every compliance entry MUST record an accountable owner, evidence date, candidate
  identity, reproducible validation procedure, evidence location, and next required action.
- **FR-003**: A completed task checkbox MUST NOT be accepted as proof when its linked evidence is
  partial, stale, contradictory, missing, or produced by a materially different build.
- **FR-004**: Accepted exceptions MUST record owner, scope, risk, expiration condition, approval, and a
  removal task; an unavailable legal vendor image MUST use this mechanism rather than a false pass.
- **FR-005**: The deployment MUST designate exactly one externally reachable production control plane
  for a host and MUST retire or isolate every additional instance with independent state.
- **FR-006**: The production control plane MUST expose the approved current contract consistently to
  UI, HTTP API, MCP, event-stream, and operational clients.
- **FR-007**: The management boundary MUST deny access from networks outside the approved trusted
  management scope, including every production, preview, API, MCP, console, and artifact endpoint.
- **FR-008**: Previously exposed deployment credentials MUST be rotated before release approval, and
  evidence generation MUST verify that prohibited secrets and artifacts are absent.
- **FR-009**: Desired and actual state MUST be authoritative, legal-transitioned, and observable for
  laboratories, nodes, interfaces, links, network objects, attachments, captures, traffic filters,
  consoles, port mappings, tasks, and owned runtime resources.
- **FR-010**: For every state-changing operation, UI, HTTP API, and applicable MCP behavior MUST use the
  same command semantics and produce equivalent task, event, conflict, cancellation, and audit results.
- **FR-011**: Concurrent mutations MUST provide explicit revision, idempotency replay, or conflict
  behavior and MUST NOT silently overwrite a newer accepted state.
- **FR-012**: Deletion MUST remove the resource from every connected client after durable success and
  MUST remove or explicitly account for every owned process, interface, namespace, bridge attachment,
  rule, socket, seed, helper, console, capture, and artifact.
- **FR-013**: Terminal failures MUST identify resource and task identity, lifecycle phase, retryability,
  retry timing when applicable, cleanup state, and an actionable operator response.
- **FR-014**: Startup and periodic reconciliation MUST adopt valid owned resources, quarantine ambiguous
  ownership, preserve unowned host resources, and publish convergence results to all control planes.
- **FR-015**: Every declared supported template family MUST have a readiness record distinguishing
  implemented template support, available legal image versions, genuine workload validation, bootstrap
  validation, console validation, health validation, and cleanup validation.
- **FR-016**: Ubuntu, VyOS, and FancyWAN automatic configuration MUST be proven with representative
  genuine workloads; a substitute operating system MAY test generic mechanics but MUST NOT satisfy
  device-family acceptance.
- **FR-017**: FortiGate acceptance MUST require operator-supplied, license-reviewed media and MUST never
  automatically retrieve commercial images from unofficial collections.
- **FR-018**: Image records MUST retain source, exact version, immutable checksum, format, availability,
  validation result, and license or entitlement notes without storing image bytes in project evidence.
- **FR-019**: A QEMU image or template that declares guest-command support MUST expose a deterministic
  readiness result for the guest agent and an actionable unsupported state when the prerequisite is
  missing.
- **FR-020**: Automatic node networking MUST be validated for the declared IPv4, IPv6, DHCPv4, DHCPv6,
  SLAAC, bridge, routed, and NAT modes without relying on undocumented manual guest intervention.
- **FR-021**: Live link changes and QEMU NIC hot-add or removal MUST publish reversible stage results,
  preserve running-node state, and compensate partial failures without leaked host or guest devices.
- **FR-022**: Telnet, VNC, guest commands, port mappings, capture streaming, Wireshark handoff, and
  traffic-filter path results MUST be validated through stable external interfaces with bounded
  timeout, cancellation, output, retention, and truncation behavior.
- **FR-023**: CPU-time, memory, process, interface, capture, and artifact limits MUST be measured under
  real load and MUST produce explicit admission or runtime failures when exceeded.
- **FR-024**: Service restart and host restart acceptance MUST verify recovery policy, durable task
  continuation, ordered event visibility, stable or explicitly recreated ownership, and terminal
  cleanup.
- **FR-025**: The current release candidate MUST pass formatting, static analysis, unit, contract,
  frontend, privileged integration, recovery, security, and leak validation; every skipped gate MUST
  have a recorded reason and repeatable target-host procedure.
- **FR-026**: Acceptance evidence MUST identify the exact candidate, target host capabilities, image
  provenance, scenario outcome, cleanup baseline, and redaction result.
- **FR-027**: Compliance and validation reports MUST present one consistent conclusion and MUST not
  retain obsolete top-level statements that contradict newer detailed evidence.
- **FR-028**: Compliance findings MUST reopen automatically or through review when their evidence
  expires, the covered contract changes, or a regression demonstrates different behavior.

### Key Entities

- **Compliance Finding**: One constitution obligation, its status, severity, owner, rationale, evidence,
  next action, and relationship to any exception.
- **Evidence Record**: A reproducible result tied to an exact candidate, host, scenario, timestamp,
  outcome, cleanup baseline, redaction result, and retention state.
- **Accepted Exception**: A time-bounded approved deviation with owner, scope, risk, expiration condition,
  and removal work.
- **Deployment Instance**: A control-plane instance with authority status, endpoint scope, candidate
  identity, state location, reachability boundary, and retirement status.
- **Template Readiness Record**: A template family and version mapped to legal media, real workload,
  bootstrap, console, guest command, health, lifecycle, and cleanup evidence.
- **Acceptance Run**: A candidate-wide validation execution with gate results, failure injections,
  retained evidence, cleanup outcome, and final release conclusion.

### Audit Baseline Findings

The July 29, 2026 review identifies the following unfinished or unproven obligations:

1. **Completion evidence is inconsistent**: all 231 implementation tasks are checked, while the primary
   validation report still opens with a partial-acceptance conclusion and several checked tasks retain
   a "partial" annotation. Later report sections claim broader completion, leaving no single truthful
   release conclusion.
2. **Current-build evidence is stale**: substantial frontend, lifecycle, console, image, and deployment
   changes occurred after the principal July 24 acceptance report. Historical evidence does not prove
   the current candidate without rerun and candidate identification.
3. **Production deployment is drifting**: the designated production endpoint does not expose at least
   one route present in the current approved contract, while a second externally reachable preview
   control plane uses separate state on the same host.
4. **Trusted-network enforcement is incomplete**: management services listen on all interfaces and the
   reviewed host reports no active host firewall boundary for those endpoints.
5. **Credential rotation is outstanding**: credentials exposed during development require out-of-band
   rotation before production approval, even though no matching credential was found in tracked source.
6. **Actual-state presentation is not fully truthful**: a dynamically verified active NAT bridge was
   presented in the topology as stopped or unknown, demonstrating a host/store/UI convergence gap.
7. **Failure diagnostics are not uniformly actionable**: retained production tasks include generic
   "not found" and "handler-specific cleanup status unavailable" results for link operations.
8. **Real template-family validation is incomplete**: current registered QEMU media covers Ubuntu;
   prior acceptance exercised FancyWAN and VyOS generic paths with an Ubuntu-derived workload rather
   than their genuine operating systems, and FortiGate has no legal dynamic acceptance evidence.
9. **Image-dependent capability readiness needs tightening**: the official Ubuntu image boots and
   consumes seed data, but guest-command readiness required installing a guest agent, and automatic NAT
   networking required operator correction before it persisted across restart.
10. **Production rollout is incomplete**: recent deletion synchronization, console, Docker networking,
    image import, and topology interaction fixes have not been proven on the single designated
    production instance as one current candidate.
11. **Governance proof is unavailable in this workspace**: no repository review metadata is present to
    demonstrate pull-request constitution impact review for the current accumulated changes.

Capabilities already supported by code or historical evidence—including durable tasks, shared events,
QEMU, Docker, namespaces, QMP, QGA interfaces, consoles, capture, traffic filters, port mappings,
resource ownership, recovery, and extensive automated test suites—remain regression scope rather than
being treated as absent. They become constitution-complete only when the current-candidate evidence and
deployment findings above are closed.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of mandatory constitution statements have one non-contradictory ledger status and a
  reproducible evidence or exception record.
- **SC-002**: Zero findings are marked verified when their newest evidence is partial, stale, produced by
  a substitute workload, or tied to a different candidate.
- **SC-003**: Exactly one externally reachable authoritative control plane remains on the target host;
  all other instances are retired or isolated from users and automation.
- **SC-004**: 100% of unapproved network sources are denied access to management endpoints, while
  approved management clients retain required access.
- **SC-005**: Across 100 consecutive valid lifecycle and live-topology operations, UI, HTTP API, MCP,
  event stream, and host observation agree on final state within 10 seconds in at least 95% of attempts,
  with every remaining attempt reaching a specific terminal failure.
- **SC-006**: 100% of injected terminal failures report resource identity, phase, retryability, cleanup
  result, and operator action; no result uses an unexplained generic "not found" outcome.
- **SC-007**: Every supported template family has either genuine workload acceptance evidence or an
  approved time-bounded blocked/exception record; no substitute workload is counted as device-family
  validation.
- **SC-008**: Ubuntu, VyOS, and FancyWAN automatic configuration succeeds without undocumented manual
  intervention in at least 19 of 20 clean-node attempts per validated version.
- **SC-009**: IPv4, IPv6, DHCPv4, DHCPv6, SLAAC, bridge, routed, and NAT scenarios each pass 100
  consecutive flows with no unexplained loss and preserve their intended configuration after restart.
- **SC-010**: Live rewiring and supported QEMU NIC changes complete within 10 seconds in at least 95% of
  valid attempts and leave zero owned-resource leaks after failure or deletion.
- **SC-011**: Current-candidate service restart converges within 60 seconds and full host recovery
  reaches a successful or specific terminal outcome for every previously running resource within five
  minutes.
- **SC-012**: The 100-cycle cleanup test returns process, interface, namespace, bridge, rule, socket,
  cgroup, capture, artifact, and ownership counts to their recorded baseline with zero unowned resource
  deletion.
- **SC-013**: All required quality gates pass against the exact deployed candidate, and every skipped
  privileged scenario has an approved reason and repeatable target-host command.
- **SC-014**: Evidence scanning reports zero proprietary images, credentials, bootstrap secrets, private
  keys, packet payloads, or unauthorized captures in source and retained release evidence.
- **SC-015**: At least 90% of evaluation users complete template selection, topology creation, live
  rewiring, console access, and capture handoff without assistance on the production candidate.

## Assumptions

- The constitution version remains 1.0.0 during this feature; changing product boundaries requires a
  separate reviewed amendment.
- The project owner can designate an approved management network and rotate deployment credentials.
- Commercial or proprietary images are supplied by the operator only after license review and are not
  required to be stored in project source or evidence.
- A legally unavailable FortiGate image may remain a documented, expiring blocked item rather than
  delaying unrelated open-source runtime completion.
- Historical acceptance evidence may be reused only when its covered behavior and candidate identity
  remain unchanged and the evidence is still reproducible.
- Preview instances may continue temporarily only when isolated from normal users, automation clients,
  and production state.

## Dependencies

- Access to the designated target virtualization host and its host-level networking controls.
- Operator-approved representative images for genuine workload validation.
- A single identified production candidate and deployable release artifact.
- Ability to execute privileged integration, restart, recovery, capture, and leak tests.
- Project-owner approval for exceptions, credential rotation confirmation, and final release status.

## Out of Scope

- Adding Cisco support, clustering, account/password authentication, or exhaustive EVE-NG compatibility.
- Automatically downloading proprietary commercial images.
- Treating a documentation-only status change as remediation without reproducible runtime evidence.
- Redesigning the entire product when an existing capability only requires current-candidate validation
  or deployment convergence.
