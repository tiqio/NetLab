---

description: "Dependency-ordered implementation tasks for constitution gap closure"
---

# Tasks: Constitution Gap Closure

**Input**: Design documents from `specs/002-constitution-gap-closure/`

**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/`, `quickstart.md`

**Tests**: Tests are required because this feature changes runtime state, networking, lifecycle,
REST/MCP/event contracts, deployment security, recovery, and release acceptance.

**Organization**: Tasks are grouped by user story so each story can be implemented and verified as an
independent increment after the shared foundation is complete.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel because it uses different files and has no dependency on another
  incomplete task in the same group.
- **[Story]**: Maps implementation work to the user story identifiers in `spec.md`.
- Every task names the exact file or directory it changes.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Create the repository structure and command surfaces needed by all five stories.

- [X] T001 Create the compliance package skeleton and package documentation in `internal/compliance/doc.go`
- [X] T002 [P] Create the compliance command entry point and subcommand dispatch skeleton in `cmd/netlab-compliance/main.go`
- [X] T003 [P] Document tracked evidence-directory rules and prohibited artifact classes in `compliance/README.md`
- [X] T004 [P] Add constitution acceptance command placeholders and environment variables in `Makefile`
- [X] T005 [P] Add non-secret production, validation, and evidence path examples in `deploy/config/netlab.example.yaml`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish candidate identity, shared schemas, deterministic errors, and validation helpers
used across all user stories.

**⚠️ CRITICAL**: No user-story implementation begins until this phase is complete.

- [X] T006 Define release candidate identity, contract digest, scope digest, and host capability value objects in `internal/domain/release.go`
- [X] T007 [P] Define stable structured problem codes and cleanup/operator-hint fields in `internal/domain/problem.go`
- [X] T008 [P] Add release version, candidate ID, binary digest, contract digest, and build timestamp configuration in `internal/support/config/config.go`
- [X] T009 Inject deterministic release metadata during builds and expose it to server construction in `cmd/netlabd/main.go`
- [X] T010 [P] Add reusable JSON Schema, digest, candidate, and prohibited-content test helpers in `tests/testsupport/compliance.go`
- [X] T011 [P] Add reusable target-host capability and owned-resource baseline helpers in `tests/testsupport/host_baseline.go`
- [X] T012 [P] Add reusable multi-client ordered-event and revision-conflict assertions in `tests/testsupport/client_parity.go`
- [X] T013 Add candidate identity and structured problem serialization coverage in `internal/domain/release_test.go`

**Checkpoint**: Candidate identity and shared verification primitives are ready for story work.

---

## Phase 3: User Story 1 - Obtain a Truthful Compliance Baseline (Priority: P1) 🎯 MVP

**Goal**: Maintain one version-controlled ledger in which every mandatory constitution statement has a
truthful status backed by current evidence or an approved, expiring exception.

**Independent Test**: Validate the ledger, reproduce at least one passing, one open, and one exception
entry, then change a covered scope digest and verify that the formerly accepted evidence becomes stale.

### Tests for User Story 1

- [X] T014 [P] [US1] Add contract tests for all four compliance JSON schemas and cross-document references in `tests/contract/compliance_schema_test.go`
- [X] T015 [P] [US1] Add CLI behavior and exit-code tests for validate, report, and capture-candidate in `internal/compliance/cli_test.go`
- [X] T016 [P] [US1] Add freshness tests for candidate, contract, scope, host-capability, and expiration changes in `internal/compliance/freshness_test.go`
- [X] T017 [P] [US1] Add contradiction tests preventing task completion or partial evidence from verifying a finding in `internal/compliance/ledger_test.go`
- [X] T018 [P] [US1] Add evidence redaction tests for credentials, bootstrap secrets, image bytes, packet payloads, and unsafe artifact paths in `tests/security/compliance_evidence_test.go`

### Implementation for User Story 1

- [X] T019 [P] [US1] Implement compliance finding, evidence, exception, deployment, template, and acceptance-run models in `internal/compliance/model.go`
- [X] T020 [P] [US1] Implement deterministic loading of the four contract schemas from `specs/002-constitution-gap-closure/contracts/` in `internal/compliance/schema.go`
- [X] T021 [US1] Implement schema validation and cross-reference invariants in `internal/compliance/validate.go`
- [X] T022 [US1] Implement evidence freshness, supersession, exception expiry, and automatic finding reopening in `internal/compliance/freshness.go`
- [X] T023 [US1] Implement prohibited-content scanning and evidence path allowlisting in `internal/compliance/redaction.go`
- [X] T024 [US1] Implement non-contradictory human and JSON compliance reporting in `internal/compliance/report.go`
- [X] T025 [US1] Complete validate, report, and capture-candidate CLI commands in `cmd/netlab-compliance/main.go`
- [X] T026 [US1] Create the initial constitution-to-requirement ledger with all open audit findings in `compliance/constitution-ledger.json`
- [X] T027 [US1] Add the repeatable compliance validation wrapper used by local and CI gates in `scripts/validate-compliance.sh`

**Checkpoint**: The project has a reproducible, truthful baseline independent of runtime remediation.

---

## Phase 4: User Story 2 - Operate One Secure Authoritative Service (Priority: P1)

**Goal**: Ensure exactly one externally reachable production instance serves the approved candidate and
contract, with preview instances isolated and unapproved source networks denied at the host boundary.

**Independent Test**: Mutate one lab from browser, HTTP, and MCP clients through the authoritative URL,
verify ordered shared state, then prove an unapproved source cannot reach any management endpoint.

### Tests for User Story 2

- [X] T028 [P] [US2] Add deployment-authority schema and exactly-one-authoritative-instance contract tests in `tests/contract/deployment_authority_test.go`
- [X] T029 [P] [US2] Add release identity and current-route parity tests for `/api/v1/capabilities` in `tests/contract/release_identity_http_test.go`
- [X] T030 [P] [US2] Add browser, HTTP, MCP, and event-stream shared-state acceptance coverage in `tests/e2e/journeys/authoritativeControlPlane.spec.ts`
- [X] T031 [P] [US2] Add target-host listener, source-network denial, and preview isolation tests in `tests/security/management_boundary_test.go`
- [X] T032 [P] [US2] Add repository, config, logs, fixtures, and evidence credential-pattern tests in `tests/security/credential_rotation_test.go`

### Implementation for User Story 2

- [X] T033 [P] [US2] Implement the release identity query service in `internal/app/query/release.go`
- [X] T034 [US2] Extend capabilities responses with candidate and contract identity in `internal/api/http/server.go`
- [X] T035 [US2] Register and document the current capabilities and runtime-ownership routes in `internal/api/http/routes.go`
- [X] T036 [P] [US2] Create the production, validation, draining, isolated, and retired instance inventory in `compliance/deployment-authority.json`
- [X] T037 [US2] Implement deployment inventory validation and live listener comparison in `internal/compliance/deployment.go`
- [X] T038 [P] [US2] Converge the authoritative systemd service to one production bind and candidate configuration in `deploy/systemd/netlab.service`
- [X] T039 [P] [US2] Add an isolated loopback-only validation service override in `deploy/systemd/netlab-validation.service`
- [X] T040 [P] [US2] Add an nftables trusted-management-network policy covering HTTP, MCP, console, capture, and preview ports in `deploy/nftables/netlab-management.nft`
- [X] T041 [US2] Implement production authority, route digest, listener, and network-boundary verification in `scripts/verify-production-authority.sh`
- [X] T042 [US2] Add non-secret credential rotation attestation fields and operator procedure in `docs/production-readiness.md`

**Checkpoint**: One secure endpoint is authoritative and all supported clients share its state.

---

## Phase 5: User Story 3 - Trust Runtime State and Failure Reporting (Priority: P1)

**Goal**: Make desired state, observed state, task results, ordered events, UI semantics, MCP responses,
and host reality agree for every supported resource, including deletion and injected failures.

**Independent Test**: Exercise lifecycle, deletion, concurrent revision conflict, restart reconciliation,
and failure injection for each runtime object; confirm UI, REST, MCP, stream, database, and host state
converge within ten seconds without refresh.

### Tests for User Story 3

- [X] T043 [P] [US3] Add migration and repository tests for runtime capability revisions and transactional outbox writes in `internal/store/sqlite/capability_repository_test.go`
- [X] T044 [P] [US3] Add network-object observed-state and ordered-event transaction tests in `internal/store/sqlite/network_state_repository_test.go`
- [X] T045 [P] [US3] Add REST contract tests for node capabilities and structured problems in `tests/contract/node_capabilities_http_test.go`
- [X] T046 [P] [US3] Add MCP parity tests for `get_node_capabilities` and mutation conflict semantics in `tests/contract/node_capabilities_mcp_test.go`
- [X] T047 [P] [US3] Add event contract and monotonic ordering tests for network state and capability changes in `tests/contract/runtime_observation_events_test.go`
- [X] T048 [P] [US3] Add failure-injection tests covering resource, phase, retryability, cleanup, and operator hints in `tests/integration/structured_failure_test.go`
- [X] T049 [P] [US3] Add deletion, adoption, quarantine, and orphan-cleanup recovery tests in `tests/recovery/runtime_truth_recovery_test.go`
- [X] T050 [P] [US3] Add frontend store tests preventing stale snapshots from resurrecting deleted resources in `web/src/stores/laboratory.runtimeTruth.test.ts`
- [X] T051 [P] [US3] Add topology health-semantic tests proving `active` renders healthy in `web/src/features/topology/topologyRuntimeState.test.ts`
- [X] T052 [P] [US3] Add capability diagnostics and unavailable-QGA component tests in `web/src/features/nodes/NodeCapabilityPanel.test.ts`

### Implementation for User Story 3

- [X] T053 [P] [US3] Define runtime capability observation types, transitions, and validation in `internal/domain/runtime_capability.go`
- [X] T054 [P] [US3] Add runtime capability observation and network-state indexes in `internal/store/sqlite/migrations/0008_runtime_observations.sql`
- [X] T055 [US3] Implement capability observation persistence, latest-by-capability reads, and transactional outbox appends in `internal/store/sqlite/capability_repository.go`
- [X] T056 [US3] Implement transactional network-object observed-state updates and deletion tombstones in `internal/store/sqlite/network_repository.go`
- [X] T057 [P] [US3] Implement capability and runtime-state query handlers in `internal/app/query/runtime_capabilities.go`
- [X] T058 [US3] Implement startup and periodic capability probing, owned-resource adoption, and ambiguity quarantine in `internal/app/reconcile/runtime_observations.go`
- [X] T059 [US3] Normalize durable operation errors and idempotent cleanup results at command boundaries in `internal/app/command/problems.go`
- [X] T060 [US3] Publish ordered `network_object.observed_state_changed` and `node.capability_changed` events in `internal/api/stream/events.go`
- [X] T061 [US3] Implement `/api/v1/nodes/{nodeId}/capabilities` with revisioned structured responses in `internal/api/http/node_capability_handlers.go`
- [X] T062 [US3] Implement the equivalent `get_node_capabilities` MCP query tool in `internal/api/mcp/node_capability_tools.go`
- [X] T063 [US3] Reconcile snapshots and ordered events without stale resurrection or silent conflict replay in `web/src/stores/laboratory.ts`
- [X] T064 [US3] Render healthy network-object state and capability diagnostics in `web/src/features/nodes/NodeCapabilityPanel.vue`

**Checkpoint**: Runtime truth and terminal failures are consistent across every control surface.

---

## Phase 6: User Story 4 - Prove Supported Templates with Real Workloads (Priority: P2)

**Goal**: Prove each declared template with its genuine, legally supplied workload and provide automatic
networking, bootstrap, console, and capability readiness without claiming placeholder images as support.

**Independent Test**: For every available QEMU and Docker template version, create a node with the exact
immutable image, apply its declared bootstrap, validate console/health/capabilities/networking, restart
it, and delete it with no owned-resource leak.

### Tests for User Story 4

- [X] T065 [P] [US4] Add NAT configuration and migration tests for DHCPv4, DHCPv6, DNS, and router advertisements in `internal/store/sqlite/nat_service_repository_test.go`
- [X] T066 [P] [US4] Add privileged dnsmasq lifecycle, lease, RA, adoption, cancellation, and cleanup tests in `tests/integration/nat_addressing_test.go`
- [X] T067 [P] [US4] Add QMP, QGA, serial, VNC, bootstrap, hotplug, guest-command, and port-mapping probe tests in `internal/runtime/qemu/capability_probe_test.go`
- [X] T068 [P] [US4] Add genuine Ubuntu, VyOS, and FancyWAN bootstrap and restart acceptance tests in `tests/integration/qemu_template_readiness_test.go`
- [X] T069 [P] [US4] Add BusyBox and Ubuntu container immutable-digest acceptance tests in `tests/integration/docker_template_readiness_test.go`
- [X] T070 [P] [US4] Add FortiGate legal-media blocking and expiring-exception tests in `tests/contract/fortigate_readiness_test.go`
- [X] T071 [P] [US4] Add DHCPv4, DHCPv6, SLAAC, bridge, routed, and NAT scenario matrix tests in `tests/integration/automatic_networking_test.go`
- [X] T072 [P] [US4] Add template readiness, blocked state, and capability-prerequisite UI tests in `web/src/features/templates/TemplateReadiness.test.ts`

### Implementation for User Story 4

- [X] T073 [P] [US4] Define NAT service desired configuration and runtime observation types in `internal/domain/nat_service.go`
- [X] T074 [P] [US4] Add NAT service configuration, helper ownership, and lease metadata tables in `internal/store/sqlite/migrations/0009_nat_services.sql`
- [X] T075 [US4] Implement NAT service configuration persistence and ownership queries in `internal/store/sqlite/nat_service_repository.go`
- [X] T076 [US4] Implement validated dnsmasq configuration generation without shell interpolation in `internal/runtime/linuxnet/dnsmasq_config.go`
- [X] T077 [US4] Implement owned foreground dnsmasq start, supervision, adoption, diagnostics, and cleanup in `internal/runtime/linuxnet/dnsmasq.go`
- [X] T078 [US4] Integrate NAT helper checkpoints, cancellation compensation, and state events in `internal/app/reconcile/network_object_tasks.go`
- [X] T079 [P] [US4] Implement deterministic QEMU runtime capability probes and readiness details in `internal/runtime/qemu/capability_probe.go`
- [X] T080 [US4] Gate guest commands and advertised template capabilities on current probe results in `internal/runtime/qemu/guest.go`
- [X] T081 [US4] Persist immutable image digest and genuine-workload readiness during template launch in `internal/app/command/topology_tasks.go`
- [X] T082 [P] [US4] Create the six-family readiness matrix with genuine, blocked, and exception states in `compliance/template-readiness.json`
- [X] T083 [US4] Expose template readiness and capability prerequisites through the template query in `internal/app/query/templates.go`
- [X] T084 [US4] Render genuine validation, blocked media, accepted exception, and unavailable capability states in `web/src/features/templates/TemplateCatalog.vue`
- [X] T085 [US4] Implement operator-supplied genuine image acceptance without retaining image contents in `acceptance/operator-image-acceptance.sh`
- [X] T086 [US4] Add repeatable QEMU and automatic-networking readiness steps to `acceptance/qemu-acceptance.sh`

**Checkpoint**: Supported template claims are backed by genuine workload evidence or explicit exceptions.

---

## Phase 7: User Story 5 - Complete Current-Build Acceptance and Governance (Priority: P2)

**Goal**: Run every mandatory gate against the exact production candidate, restore the ownership
baseline, redact retained evidence, and publish one reviewable, non-contradictory release conclusion.

**Independent Test**: Execute the complete quickstart sequence against the candidate, including service
and host restart plus 100 lifecycle cycles, then validate the final evidence package and cleanup baseline.

### Tests for User Story 5

- [X] T087 [P] [US5] Add acceptance-run schema and aggregate conclusion tests in `tests/contract/acceptance_run_test.go`
- [X] T088 [P] [US5] Add service restart adoption and client convergence tests in `tests/recovery/candidate_service_restart_test.go`
- [X] T089 [P] [US5] Add host restart recovery-policy, durable-task, helper-adoption, and quarantine tests in `tests/recovery/candidate_host_restart_test.go`
- [X] T090 [P] [US5] Extend the 100-cycle test to cover QEMU, Docker, netns, bridges, NAT helpers, links, captures, and artifacts in `tests/integration/leak_cycle_test.go`
- [X] T091 [P] [US5] Add multi-browser, HTTP, MCP, console, capture, filter, and live-rewire candidate acceptance in `tests/e2e/journeys/currentCandidateAcceptance.spec.ts`
- [X] T092 [P] [US5] Add candidate evidence package redaction, digest, and cleanup-baseline security tests in `tests/security/acceptance_package_test.go`

### Implementation for User Story 5

- [X] T093 [P] [US5] Implement acceptance-run aggregation, gate states, skipped-test reasons, and final conclusion rules in `internal/compliance/acceptance.go`
- [X] T094 [US5] Extend compliance reporting to reject conflicting detailed and summary conclusions in `internal/compliance/report.go`
- [X] T095 [P] [US5] Add service restart execution, observation, and baseline comparison in `acceptance/t225-service-restart.sh`
- [X] T096 [P] [US5] Add operator-controlled host restart execution and post-boot evidence collection in `acceptance/host-restart.sh`
- [X] T097 [P] [US5] Add structured lifecycle failure injection and cleanup evidence collection in `acceptance/lifecycle-failures.sh`
- [X] T098 [US5] Implement the candidate-wide local, target, recovery, security, leak, and browser gate orchestrator in `scripts/run-constitution-acceptance.sh`
- [X] T099 [US5] Add acceptance, compliance, security, and 100-cycle targets with candidate propagation in `Makefile`
- [X] T100 [US5] Generate metadata-only evidence indexes and one final report under `compliance/evidence/README.md`
- [X] T101 [US5] Document reviewer identity, approval state, exception review, and unavailable-git fallback in `docs/release-governance.md`
- [X] T102 [US5] Execute the full current-candidate validation procedure and record non-secret results in `compliance/evidence/current-candidate.json`
- [X] T103 [US5] Reconcile every ledger finding against current evidence and publish the release conclusion in `compliance/constitution-ledger.json`

**Checkpoint**: The exact candidate has one coherent acceptance decision and a restored host baseline.

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Complete contract publication, documentation, cleanup, and final repository-wide validation.

- [X] T104 [P] Synchronize implemented REST schemas and operation IDs with `specs/002-constitution-gap-closure/contracts/openapi-delta.yaml`
- [X] T105 [P] Synchronize MCP capability semantics and errors with `specs/002-constitution-gap-closure/contracts/mcp-tools.md`
- [X] T106 [P] Synchronize ordered runtime event payloads and client rules with `specs/002-constitution-gap-closure/contracts/events.md`
- [X] T107 [P] Update deployment, template media, trusted-network, and acceptance operator guidance in `README.md`
- [X] T108 Run formatting, static analysis, unit, contract, web, security, and build gates from `Makefile`
- [X] T109 Run privileged integration, recovery, failure-injection, and 100-cycle leak gates from `specs/002-constitution-gap-closure/quickstart.md`
- [X] T110 Run local and target-host browser acceptance and verify no stale UI resurrection using `specs/002-constitution-gap-closure/quickstart.md`
- [X] T111 Validate all evidence schemas, redaction results, exceptions, cleanup baselines, and final conclusion using `scripts/validate-compliance.sh`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 — Setup**: No dependencies; starts immediately.
- **Phase 2 — Foundational**: Depends on Phase 1 and blocks all user stories.
- **Phase 3 — US1**: Depends on Phase 2; produces the baseline used to measure all later remediation.
- **Phase 4 — US2**: Depends on Phase 2 and can run alongside US1, but final evidence is recorded through US1 tooling.
- **Phase 5 — US3**: Depends on Phase 2 and can run alongside US1/US2.
- **Phase 6 — US4**: Depends on Phase 2; NAT lifecycle integration also consumes US3 structured state and events.
- **Phase 7 — US5**: Depends on US1 through US4 because it aggregates their current-candidate evidence.
- **Phase 8 — Polish**: Depends on every story selected for the release.

### User Story Dependency Graph

```text
Setup -> Foundation -> US1 -----------------------> US5 -> Polish
                    -> US2 -----------------------> US5
                    -> US3 -> US4 ----------------> US5
                    -> US4 (non-state work) ------> US5
```

### Within Each User Story

- Write the listed tests first and verify they fail for the expected missing behavior.
- Implement domain models and migrations before repositories and application services.
- Implement services before REST, MCP, event-stream, and frontend integrations.
- Complete failure, cancellation, recovery, and cleanup behavior before acceptance evidence is recorded.
- Do not mark a story complete until its independent test passes against the same candidate identity.

## Parallel Opportunities

### User Story 1

```text
T014 schema contracts | T015 CLI tests | T016 freshness tests | T017 contradiction tests | T018 redaction tests
T019 compliance models | T020 schema loader
```

### User Story 2

```text
T028 authority contract | T029 release route contract | T030 client journey | T031 boundary security | T032 credential scan
T033 release query | T036 deployment inventory | T038 production unit | T039 validation unit | T040 nftables policy
```

### User Story 3

```text
T043 capability repository | T044 network transaction | T045 REST | T046 MCP | T047 events
T048 failures | T049 recovery | T050 store UI | T051 topology UI | T052 capability UI
T053 capability domain | T054 migration
```

### User Story 4

```text
T065 NAT persistence | T066 dnsmasq lifecycle | T067 QEMU probes | T068 QEMU templates
T069 Docker templates | T070 FortiGate policy | T071 addressing matrix | T072 template UI
T073 NAT domain | T074 migration | T079 QEMU probe implementation | T082 readiness matrix
```

### User Story 5

```text
T087 acceptance contract | T088 service recovery | T089 host recovery
T090 leak cycles | T091 multi-client acceptance | T092 package security
T093 acceptance aggregation | T095 service script | T096 host script | T097 failure script
```

## Implementation Strategy

### MVP First — Truthful Baseline

1. Complete Setup and Foundational phases.
2. Complete User Story 1 only.
3. Run `scripts/validate-compliance.sh` and reproduce representative ledger entries.
4. Stop and review the truthful baseline before changing production or runtime behavior.

### Incremental Delivery

1. **US1** establishes what is actually complete, open, blocked, stale, or excepted.
2. **US2** converges deployment authority and trusted-network exposure.
3. **US3** makes runtime state, deletion, conflicts, and errors trustworthy.
4. **US4** closes genuine template, automatic networking, and capability-readiness gaps.
5. **US5** repeats every gate against one candidate and publishes the release decision.

### Parallel Team Strategy

1. Complete Setup and Foundation together.
2. Run US1, US2, and US3 with separate owners after the foundation checkpoint.
3. Start US4 runtime-state integration after US3 state/event contracts stabilize; other US4 work may
   proceed earlier.
4. Start US5 only after all required story checkpoints are green for the same candidate.

## Notes

- `[P]` tasks use disjoint files and can be executed concurrently within the stated dependency boundary.
- Commercial or proprietary images remain operator supplied and are never committed or copied into evidence.
- Evidence contains metadata, digests, outcomes, and cleanup facts only—never credentials or packet payloads.
- Host commands must use validated argument vectors or bounded generated configuration, never interpolated
  untrusted shell strings.
- A checked task is not proof; only accepted, current evidence can verify a compliance finding.

## Phase 9: Convergence

- [X] T112 CRITICAL deploy the exact candidate as the sole authoritative NetLab service on a verified non-conflicting endpoint, isolate validation instances, and regenerate the truthful deployment inventory per FR-005, FR-006, and SC-003 (contradicts)
- [X] T113 CRITICAL derive, apply, and externally test trusted-management nftables rules for every UI, API, MCP, console, capture, artifact, production, and validation endpoint per FR-007 and SC-004 (partial)
- [ ] T114 CRITICAL rotate every previously exposed deployment credential out of band, record a non-secret rotation attestation, and repeat repository, log, fixture, capture, and evidence scans per FR-008 and Constitution VII (missing)
- [X] T115 CRITICAL implement terminal reconciliation for disappeared or orphaned ownership observations, clean the two residual namespaces and stale capture records, and add privileged recovery and leak regression tests per FR-012, FR-014, SC-012, Constitution V, and Constitution VI (partial)
- [X] T116 aggregate authoritative browser artifacts, scenario IDs, complete resource baselines, target capabilities, image provenance, and measured redaction results in `scripts/run-constitution-acceptance.sh` per FR-026 and SC-014 (partial)
- [ ] T117 execute the operator-controlled full host restart against the authoritative candidate and retain metadata proving durable task recovery, ordered events, ownership adoption, client convergence, and terminal cleanup per FR-024 and SC-011 (missing)
- [ ] T118 add genuine pointer and keyboard journeys for every required interaction at 1920x1080 and 1024x768, then pass the target-host interaction coverage gate per FR-025, SC-013, and SC-015 (partial)
- [ ] T119 import operator-approved genuine VyOS and FancyWAN media and prove bootstrap, console, health, stop/start recovery, immutable digest, and cleanup behavior per FR-016, SC-007, and SC-008 (missing)
- [ ] T120 complete FortiGate license review and genuine operator-supplied media acceptance, or record an approved expiring exception without automatically fetching commercial media per FR-017 and SC-007 (missing)
- [ ] T121 add repeatable target-host acceptance for IPv4, IPv6, DHCPv4, DHCPv6, SLAAC, bridge, routed, and NAT automatic networking without undocumented guest intervention per FR-020 and SC-009 (partial)
- [X] T122 reject placeholder candidate IDs and all-zero or empty release digests when starting an authoritative deployment, inject immutable build and contract identity, and add configuration tests per FR-006 and FR-026 (contradicts)
- [ ] T123 regenerate `compliance/template-readiness.json` for the exact candidate from immutable image records and accepted genuine-workload, bootstrap, console, capability, lifecycle, and cleanup evidence per FR-015, FR-018, and SC-007 (partial)
- [ ] T124 rerun every mandatory local, privileged, restart, leak, browser, image, networking, authority, boundary, redaction, and compliance gate against the authoritative candidate; restore the cleanup baseline and reconcile the ledger and reviewer decision per FR-025 through FR-028 and US5 (partial)
