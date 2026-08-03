# Tasks: Frontend Interaction Acceptance

**Input**: Design documents from `specs/003-frontend-interaction-acceptance/`

**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/`, `quickstart.md`

**Tests**: Tests are mandatory because this feature exists to prove frontend behavior against authoritative and
privileged runtime state. Story tests must be written before the corresponding harness or product fixes and must
demonstrate the current shallow, silent, stale, or leaking behavior where a defect exists.

**Organization**: Tasks are grouped by user story so each story remains independently demonstrable. Shared
acceptance infrastructure is established first; target-host execution occurs only after local and contract gates
pass.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel because it edits different files and does not depend on incomplete tasks
- **[Story]**: Maps the task to a user story from `spec.md`
- Every task names its implementation or evidence path

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Establish acceptance-suite directories, dependencies, commands, and isolated outputs.

- [X] T001 Create the acceptance fixture, page-object, journey, and matrix directories with tracked placeholders in `tests/e2e/fixtures/.gitkeep`, `tests/e2e/pages/.gitkeep`, `tests/e2e/journeys/.gitkeep`, and `tests/e2e/matrices/.gitkeep`
- [X] T002 Add direct development dependencies for JSON Schema validation and runner utilities plus local/target acceptance scripts in `web/package.json` and `web/package-lock.json`
- [X] T003 Configure distinct disposable-local and remote-target Playwright profiles with 1920×1080 and 1024×768 projects in `web/playwright.config.ts` and `web/playwright.target.config.ts`
- [X] T004 Add ignored acceptance evidence, traces, screenshots, and temporary ledgers without ignoring versioned manifests in `.gitignore`
- [X] T005 Add local, target-host, schema-contract, and acceptance-artifact commands to `Makefile`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Implement the shared inventory, capability, task, ownership, cleanup, and evidence foundations used
by every story.

**⚠️ CRITICAL**: No user-story implementation begins until this phase passes its contract tests.

- [X] T006 Define AcceptanceRun, EnvironmentSnapshot, InteractionDefinition, InteractionResult, TaskObservation, CapabilityDecision, OwnedResource, CleanupRecord, VersionCoverage, and ClientObservation types in `tests/e2e/fixtures/acceptanceTypes.ts`
- [X] T007 [P] Add failing JSON Schema self-validation and representative valid/invalid document tests in `tests/contract/frontend_acceptance_schema_test.go`
- [X] T008 Implement inventory and evidence schema loading with duplicate interaction-ID detection in `tests/e2e/fixtures/schemaValidation.ts`
- [X] T009 [P] Implement evidence redaction rules that reject credentials, authorization headers, console output, guest-command output, packet payloads, capture files, bootstrap data, and proprietary image content in `tests/e2e/fixtures/redaction.ts`
- [X] T010 [P] Implement the append-only run-owned resource ledger and sanitized persistence in `tests/e2e/fixtures/resourceLedger.ts`
- [X] T011 Implement unconditional bounded cleanup coordination for success, failure, timeout, and interruption using the resource ledger in `tests/e2e/fixtures/cleanupCoordinator.ts`
- [X] T012 [P] Implement health, capability, template/image, browser, Wireshark-helper, and clean-baseline preflight decisions in `tests/e2e/fixtures/preflight.ts`
- [X] T013 [P] Implement bounded visible-feedback, task-identity, task-terminal, event-sequence, stream-readiness, authoritative-state, and cleanup waiters in `tests/e2e/fixtures/waiters.ts`
- [X] T014 [P] Implement role-first locator, pointer activation, keyboard activation, dialog, and refreshed-state primitives in `tests/e2e/pages/BasePage.ts`
- [X] T015 Compose run identity, API request context, two browser contexts, preflight, resource ledger, cleanup, and evidence fixtures in `tests/e2e/fixtures/acceptanceFixture.ts`
- [X] T016 Add fixture-level tests for ineligible skips, supported-capability failures, timeout handling, signal-safe cleanup, and redaction in `tests/e2e/fixtures/acceptanceFixture.test.ts`
- [X] T017 Extend source-control artifact scanning to reject acceptance outputs and sensitive evidence in `scripts/check-frontend-artifacts.sh`

**Checkpoint**: Shared acceptance records validate against contracts, supported capabilities cannot be skipped,
and every fixture termination path attempts cleanup.

---

## Phase 3: User Story 1 - Complete Real User Journeys (Priority: P1) 🎯 MVP

**Goal**: Drive the complete create–connect–operate–delete workflow through visible frontend controls and prove
task completion, refreshed authoritative state, complete image-version coverage, and zero leaks.

**Independent Test**: Starting from an empty disposable or target environment, use only visible controls to
create and rename a laboratory, create representative QEMU/Docker/lightweight resources, connect and rewire
them, operate nodes, follow durable tasks, and delete the laboratory; every mutation survives refresh and the
post-run resource baseline is clean.

### Tests for User Story 1

- [X] T018 [P] [US1] Add a failing frontend-only laboratory create, select, rename, duplicate, export/import entry, refresh, cancel-delete, and confirm-delete journey in `tests/e2e/journeys/laboratoryLifecycle.spec.ts`
- [X] T019 [P] [US1] Add a failing template browse, version/image selection, stale-catalog revalidation, invalid submission, duplicate-submit, and node-creation journey in `tests/e2e/journeys/templateCreation.spec.ts`
- [X] T020 [P] [US1] Add a failing topology select, multi-select, port-select, connect, disconnect, reconnect, move, pan, zoom, route, inspector, and keyboard journey in `tests/e2e/journeys/topologyEditing.spec.ts`
- [X] T021 [P] [US1] Add a failing node lifecycle, resource edit, interface add/remove, guest command, port mapping, network-object diagnostics, task navigation, and destructive cleanup journey in `tests/e2e/journeys/nodeOperations.spec.ts`
- [X] T022 [P] [US1] Add a failing representative-full-journey and remaining-version lifecycle/connectivity matrix test in `tests/e2e/matrices/templateVersionCoverage.spec.ts`
- [X] T023 [P] [US1] Add a failing interrupted-journey cleanup and clean-baseline restoration test in `tests/e2e/journeys/primaryJourneyCleanup.spec.ts`

### Implementation for User Story 1

- [X] T024 [P] [US1] Implement laboratory toolbar, selector, transfer, revision-conflict, refresh, and deletion drivers in `tests/e2e/pages/LaboratoryPage.ts`
- [X] T025 [P] [US1] Implement template catalog, image/version picker, availability, stale-catalog, and node-creation drivers in `tests/e2e/pages/TemplatePage.ts`
- [X] T026 [P] [US1] Implement ECharts topology coordinate conversion, selection, drag, connect, disconnect, reconnect, zoom, pan, routing, and inspector drivers in `tests/e2e/pages/TopologyPage.ts`
- [X] T027 [P] [US1] Implement node lifecycle, resources, interfaces, guest command, port mapping, network object, and task-center drivers in `tests/e2e/pages/NodeOperationsPage.ts` and `tests/e2e/pages/TaskCenterPage.ts`
- [X] T028 [US1] Correct laboratory controls that fail the new real-service journey in `web/src/features/laboratories/LaboratoryToolbar.vue`, `web/src/features/laboratories/LaboratoryTransferDialog.vue`, and `web/src/stores/laboratory.ts`
- [X] T029 [US1] Correct template/version selection, availability, stale catalog, submission, and duplicate-prevention behavior in `web/src/features/templates/TemplateCatalog.vue`, `web/src/features/templates/TemplatePicker.vue`, `web/src/features/topology/DevicePalette.vue`, and `web/src/features/topology/CreateTopologyResourceDialog.vue`
- [X] T030 [US1] Correct topology gestures, live wiring, routing persistence, selection, and inspector actions exposed by the journey in `web/src/features/topology/TopologyCanvas.vue`, `web/src/features/topology/useLinkEditing.ts`, `web/src/features/topology/topologyLayout.ts`, and `web/src/features/topology/TopologyInspector.vue`
- [X] T031 [US1] Correct node, interface, resource, guest-command, mapping, network-object, and task actions exposed by the journey in `web/src/features/nodes/NodeOperationsPanel.vue`, `web/src/features/nodes/InterfaceOperations.vue`, `web/src/features/nodes/NodeResourcesEditor.vue`, `web/src/features/nodes/GuestCommandPanel.vue`, `web/src/features/nodes/PortMappingsPanel.vue`, `web/src/features/nodes/LightweightNodeEditor.vue`, and `web/src/features/tasks/TaskCenter.vue`
- [X] T032 [US1] Implement the complete run-owned create–connect–operate–delete orchestration and version coverage recording in `tests/e2e/journeys/completeRealJourney.ts`

**Checkpoint**: User Story 1 independently proves the primary frontend workflow against authoritative state and
leaves no run-owned resources.

---

## Phase 4: User Story 2 - No Silent or Misleading Controls (Priority: P1)

**Goal**: Inventory every visible interaction and guarantee an intentional pointer/keyboard outcome, explicit
unavailable reason, or eligible environmental skip; no enabled control may silently do nothing.

**Independent Test**: For every applicable workspace state and both required viewports, compare discovered
interactive elements with the versioned inventory, activate each permitted control by pointer and keyboard,
and fail on missing inventory entries, silent no-ops, false success, clipped actions, or unexplained disabled
states.

### Tests for User Story 2

- [X] T033 [P] [US2] Add failing uniqueness, required-metadata, operation-registry parity, and capability-classification tests for the interaction inventory in `tests/e2e/matrices/interactionInventory.test.ts`
- [X] T034 [P] [US2] Add a failing DOM-to-inventory coverage and enabled-control no-op detection suite in `tests/e2e/matrices/controlCoverage.spec.ts`
- [X] T035 [P] [US2] Add failing pointer/keyboard parity, visible-focus, clipping, and reachable-action tests at 1920×1080 and 1024×768 in `tests/e2e/matrices/viewportInputMatrix.spec.ts`
- [X] T036 [P] [US2] Add failing form validation, boundary, units, dependency, server-error, unsaved-change, duplicate-submit, conflict, merge, retry, cancel, and confirm tests in `tests/e2e/matrices/formAndDialogMatrix.spec.ts`
- [X] T037 [P] [US2] Add failing empty, loading, stale, reconnecting, unsupported, permission, quota, conflict, partial-failure, cleanup, cancellation, retryable, terminal-failure, and success component matrices in `web/src/components/common/InteractionStateMatrix.test.ts`

### Implementation for User Story 2

- [X] T038 [US2] Populate the versioned role-first interaction inventory for every applicable workspace control and ECharts gesture in `tests/e2e/matrices/interaction-inventory.json`
- [X] T039 [P] [US2] Implement DOM discovery, applicable-state filtering, inventory matching, and newly-added-control failure reporting in `tests/e2e/matrices/controlInventoryAudit.ts`
- [X] T040 [P] [US2] Implement observable outcome assertions for presentation, navigation, mutation, task, stream, download, and structured-error classes in `tests/e2e/fixtures/outcomeAssertions.ts`
- [X] T041 [P] [US2] Implement pointer/keyboard equivalence, focus visibility, viewport clipping, and dialog reachability helpers in `tests/e2e/fixtures/inputAccessibility.ts`
- [X] T042 [US2] Correct global shell, command palette, operations drawer, and common control disabled-reason/focus behavior in `web/src/components/shell/LaboratoryShell.vue`, `web/src/components/shell/CommandPalette.vue`, `web/src/components/shell/OperationsDrawer.vue`, and `web/src/components/ui/button/Button.vue`
- [X] T043 [US2] Correct confirmation, form validation, recoverable input, structured error, loading, empty, and status presentation behavior in `web/src/components/common/ConfirmationDialog.vue`, `web/src/components/common/StructuredProblem.vue`, `web/src/components/common/LoadingState.vue`, `web/src/components/common/EmptyState.vue`, and `web/src/components/common/StatePresentation.vue`
- [X] T044 [US2] Add documented presentation-only outcomes and accessible names for any intentional local-only controls in `web/src/components/shell/LaboratoryShell.vue`, `web/src/features/topology/TopologyWorkspace.vue`, and `tests/e2e/matrices/interaction-inventory.json`

**Checkpoint**: User Story 2 independently detects unimplemented clicks, inventory drift, misleading enablement,
keyboard gaps, and viewport-specific unreachable controls.

---

## Phase 5: User Story 3 - Exercise Diagnostics and Interactive Workspaces (Priority: P2)

**Goal**: Prove real Telnet/VNC, capture, Wireshark handoff, Traffic Filter, stream lifecycle, refresh,
reconnect, and unsupported-mode behavior without silently changing node state.

**Independent Test**: With suitable running target-host resources, open and switch Telnet/VNC sessions, force
reconnect and resize, start interface and link captures, observe a non-empty live stream and retained metadata,
validate Wireshark handoff, run Traffic Filter path observation, refresh during each stream, stop everything,
and verify node lifecycle and cleanup state remain correct.

### Tests for User Story 3

- [X] T045 [P] [US3] Add a failing real Telnet/VNC create, switch, reconnect, resize, close, refresh interruption, and unsupported-mode journey in `tests/e2e/journeys/consoleWorkspace.spec.ts`
- [X] T046 [P] [US3] Add a failing interface/link capture start, discovery, refresh, non-empty stream, artifact metadata, quota, truncation, stop, failure, and cleanup journey in `tests/e2e/journeys/captureWorkspace.spec.ts`
- [X] T047 [P] [US3] Add a failing Traffic Filter input, scope, start, discovery, path, timing, confidence, duplicate, loop, ambiguity, stop, failure, and restart-recovery journey in `tests/e2e/journeys/trafficFilterWorkspace.spec.ts`
- [X] T048 [P] [US3] Add a failing page-refresh and service-stream reconnect matrix for console, capture, Traffic Filter, task, and event sessions in `tests/e2e/matrices/streamReconnect.spec.ts`
- [X] T049 [P] [US3] Add a failing Wireshark handoff command, non-empty stream, optional native-launch, and ineligible-skip contract test in `tests/e2e/journeys/wiresharkHandoff.spec.ts`

### Implementation for User Story 3

- [X] T050 [P] [US3] Implement Telnet/VNC session, tab switching, renderer state, resize, refresh, reconnect, and close drivers in `tests/e2e/pages/ConsolePage.ts`
- [X] T051 [P] [US3] Implement capture scope, lifecycle, live-byte observation, artifact metadata, quota, truncation, Wireshark handoff, and cleanup drivers in `tests/e2e/pages/CapturePage.ts`
- [X] T052 [P] [US3] Implement Traffic Filter scope, packet-match input, path observation, timing/confidence, restart, and cleanup drivers in `tests/e2e/pages/TrafficFilterPage.ts`
- [X] T053 [US3] Correct Telnet/VNC session state, switching, reconnect, resize, close, and unsupported feedback in `web/src/features/diagnostics/ConsoleWorkspace.vue`
- [X] T054 [US3] Correct real capture discovery, refresh, stream readiness, retained metadata, Wireshark handoff, quota/truncation, stop, and cleanup behavior in `web/src/features/diagnostics/CapturePanel.vue` and `web/src/api/diagnostics.ts`
- [X] T055 [US3] Correct Traffic Filter input mapping, path observations, timing/confidence, duplicate/loop/ambiguity rendering, stop, failure, and restart behavior in `web/src/features/diagnostics/TrafficFilterPanel.vue`, `web/src/features/diagnostics/TrafficFilterChart.vue`, and `web/src/features/diagnostics/trafficFilterMatch.ts`
- [X] T056 [US3] Implement the approved optional desktop Wireshark launcher adapter and sanitized launch receipt in `tests/e2e/fixtures/wiresharkLauncher.ts`

**Checkpoint**: User Story 3 independently proves that diagnostic controls operate on real streams and report
reconnect, unsupported, failure, and cleanup outcomes accurately.

---

## Phase 6: User Story 4 - Repeatable Regression Evidence (Priority: P2)

**Goal**: Produce deterministic, schema-valid, sanitized evidence for two consecutive runs, concurrent clients,
failure cleanup, capability gates, and reproducible failed controls.

**Independent Test**: Execute the complete suite twice from the same clean baseline with two browsers and one
automation client; compare inventories and version coverage, verify shared convergence within five seconds,
inject one controlled failure, and confirm every run produces sanitized evidence and zero residual resources.

### Tests for User Story 4

- [X] T057 [P] [US4] Add failing schema validation, redaction, terminal-result completeness, timing, and cleanup-invariant tests for evidence output in `tests/e2e/fixtures/evidenceReporter.test.ts`
- [X] T058 [P] [US4] Add a failing two-browser plus HTTP-client shared-state, revision-conflict, ordered-event, reconnect, and deleted-resource convergence journey in `tests/e2e/journeys/concurrentClients.spec.ts`
- [X] T059 [P] [US4] Add a failing frontend/API/MCP equivalent-operation final-state parity test in `tests/integration/frontend_control_parity_test.go`
- [X] T060 [P] [US4] Add a failing two-run inventory, version-coverage, timing, and result-difference comparison test in `tests/e2e/journeys/repeatability.spec.ts`
- [X] T061 [P] [US4] Add a failing controlled-timeout/interruption cleanup, leak report, and contaminated-baseline rejection test in `tests/e2e/journeys/failureCleanup.spec.ts`
- [X] T062 [P] [US4] Add a failing supported-capability failure and environment-optional skip eligibility test in `tests/e2e/journeys/capabilityGate.spec.ts`

### Implementation for User Story 4

- [X] T063 [P] [US4] Implement schema-valid sanitized JSON evidence, failure screenshot/trace policy, timing aggregation, and exact-control failure summaries in `tests/e2e/fixtures/evidenceReporter.ts`
- [X] T064 [P] [US4] Implement two-browser and automation-client observation correlation, event ordering, revision comparison, and convergence timing in `tests/e2e/fixtures/clientObserver.ts`
- [X] T065 [P] [US4] Implement representative-version selection, remaining-version coverage aggregation, and complete-family failure rules in `tests/e2e/fixtures/versionCoverage.ts`
- [X] T066 [P] [US4] Implement deterministic result normalization and consecutive-run comparison in `tests/e2e/fixtures/runComparator.ts`
- [X] T067 [US4] Implement target-host orchestration, signal traps, preflight, Playwright invocation, evidence finalization, and unconditional cleanup in `acceptance/frontend-acceptance.sh`
- [X] T068 [US4] Implement named controlled failure injection after owned-resource creation in `tests/e2e/fixtures/failureInjection.ts`
- [X] T069 [US4] Emit human-readable pass/fail/skip, timing, version coverage, concurrent convergence, and cleanup summaries beside JSON evidence in `tests/e2e/fixtures/runSummary.ts`

**Checkpoint**: User Story 4 independently produces reproducible evidence, rejects invalid skips, proves shared
state across clients, and cleans resources after both normal and injected-failure runs.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Remove misleading legacy coverage, integrate all gates, document operation, and execute the required
real-host validation.

- [X] T070 Replace route-mocked lightweight Playwright success with a real-service journey or move its state-only assertions to Vitest in `tests/e2e/us6_lightweight_workflow.spec.ts` and `web/src/features/nodes/LightweightNodeEditor.test.ts`
- [X] T071 [P] Replace heading/tab-presence-only Playwright checks with inventory-backed outcome assertions in `tests/e2e/accessibility.spec.ts`, `tests/e2e/frontend_diagnostics.spec.ts`, `tests/e2e/frontend_responsive_keyboard.spec.ts`, `tests/e2e/frontend_templates_automation.spec.ts`, `tests/e2e/frontend_topology_workspace.spec.ts`, `tests/e2e/us1_shared_topology.spec.ts`, and `tests/e2e/us3_automation.spec.ts`
- [X] T072 [P] Add schema, local E2E, target-host E2E, failure-cleanup, and evidence-redaction commands to `acceptance/README.md`
- [X] T073 Update `specs/003-frontend-interaction-acceptance/quickstart.md` to match the implemented command names, prerequisites, outputs, and safe remediation flow
- [X] T074 Run formatting, lint, unit, contract, frontend artifact, build, and disposable Playwright gates and record sanitized results in `specs/003-frontend-interaction-acceptance/validation.md`
- [X] T075 Deploy the built frontend/service to `10.72.1.7`, run one complete real QEMU/Docker/lightweight/Console/capture/Traffic-Filter acceptance pass, and append sanitized evidence references to `specs/003-frontend-interaction-acceptance/validation.md`
- [X] T076 Run the complete target-host suite a second time from the same clean baseline, compare normalized results, and append repeatability findings to `specs/003-frontend-interaction-acceptance/validation.md`
- [X] T077 Run the controlled failure-injection target-host suite, verify automatic cleanup and zero run-owned host resources, and append the leak audit to `specs/003-frontend-interaction-acceptance/validation.md`
- [X] T078 Verify the final repository and distributable contain no credentials, proprietary images, packet captures, console payloads, guest-command output, bootstrap secrets, or acceptance runtime artifacts using `scripts/check-frontend-artifacts.sh` and document the result in `specs/003-frontend-interaction-acceptance/validation.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: Starts immediately.
- **Phase 2 (Foundational)**: Depends on Phase 1 and blocks all user stories.
- **User Story 1 (P1)**: Depends on Phase 2 and delivers the MVP primary journey.
- **User Story 2 (P1)**: Depends on Phase 2; can proceed in parallel with User Story 1 after the shared fixtures
  exist, but final inventory population must include controls stabilized by User Story 1.
- **User Story 3 (P2)**: Depends on Phase 2 and suitable real runtime resources; can develop page drivers and
  component fixes in parallel, while privileged proof waits for User Story 1 resource creation.
- **User Story 4 (P2)**: Depends on Phase 2; evidence components can proceed early, but repeatability and cleanup
  scenarios depend on completed journeys from User Stories 1–3.
- **Phase 7 (Polish)**: Depends on all user stories; target-host deployment starts only after local gates pass.

### User Story Dependency Graph

```text
Setup → Foundation → US1 (primary real journey) ─┬→ US3 (real diagnostics) ─┐
                         └→ US2 (control audit) ─┼─────────────────────────┤
                                                └→ US4 (evidence) ←───────┘
                                                                      ↓
                                                        Polish + target validation
```

### Within Each User Story

- Write the listed failing tests first and confirm they expose the missing or shallow behavior.
- Implement page drivers and test-domain helpers before changing product components.
- Apply the smallest product fix that makes authoritative behavior observable and correct.
- Re-run the story independently, then run dependent story gates.
- Never allow a failed story test to bypass cleanup or preserve its laboratory.

## Parallel Execution Examples

### User Story 1

```text
Parallel: T018 laboratory tests, T019 template tests, T020 topology tests,
          T021 node-operation tests, T022 version matrix, T023 cleanup failure
Then:     T024–T027 page drivers in parallel
Then:     T028–T031 focused product fixes
Finally:  T032 complete journey orchestration
```

### User Story 2

```text
Parallel: T033 inventory contract, T034 control audit, T035 viewport/input,
          T036 forms/dialogs, T037 state matrix
Then:     T039–T041 audit/assertion helpers in parallel
Then:     T038 inventory population and T042–T044 product accessibility/state fixes
```

### User Story 3

```text
Parallel: T045 console, T046 capture, T047 Traffic Filter, T048 reconnect,
          T049 Wireshark contract tests
Then:     T050–T052 page drivers in parallel
Then:     T053–T056 product and launcher fixes
```

### User Story 4

```text
Parallel: T057 evidence, T058 concurrent clients, T059 API/MCP parity,
          T060 repeatability, T061 failure cleanup, T062 capability gate
Then:     T063–T066 evidence/observer/version/comparison helpers in parallel
Then:     T067–T069 runner, injection, and summary integration
```

## Implementation Strategy

### MVP First

1. Complete Setup and Foundational phases.
2. Complete User Story 1 and demonstrate the primary frontend journey against the disposable service.
3. Run User Story 1 on `10.72.1.7` with representative QEMU, Docker, and lightweight resources.
4. Do not claim MVP success unless cleanup restores the clean baseline.

### Incremental Delivery

1. **US1** establishes real mutation and lifecycle trust.
2. **US2** closes silent no-ops, disabled-state ambiguity, keyboard gaps, and inventory drift.
3. **US3** proves real interactive streams and diagnostic behavior.
4. **US4** makes coverage repeatable, comparable, sanitized, and safe after failure.
5. **Polish** removes legacy shallow checks and executes the complete two-run target-host gate.

### Completion Gate

- All 78 tasks are complete.
- Every applicable inventory interaction has an eligible terminal result at both required viewports.
- Every supported runtime/device family and available version satisfies its required coverage level.
- Two browsers, HTTP automation, and applicable MCP operations converge to the same final state.
- Both successful and injected-failure target-host runs leave zero run-owned resources.
- Evidence validates against the contracts and contains no prohibited sensitive content.

## Phase 8: Convergence

- [X] T079 CRITICAL enforce automated filename, page-region, and content safety checks before retaining or publishing Playwright screenshots, traces, and HTML reports; delete unsafe artifacts per Constitution VII and FR-024 (contradicts)
- [X] T080 CRITICAL implement a run-level append-only resource ledger, signal forwarding, and an independent bounded cleanup phase that restores the initial baseline after success, assertion failure, timeout, or process interruption per FR-022, FR-023, and FR-029 (partial)
- [X] T081 integrate `writeEvidence()` and `renderRunSummary()` with the Playwright fixture and `acceptance/frontend-acceptance.sh` so every complete or failed run emits aggregated schema-valid sanitized JSON and human-readable terminal evidence per FR-024, SC-011, and plan: evidence finalization (partial)
- [X] T082 extend `tests/e2e/matrices/interaction-inventory.json` and its DOM parity suites to cover Console open/reconnect/session switch/session close, capture start/refresh/stop/stream/retained-file/Wireshark, and Traffic Filter start/refresh/stop controls per FR-001 and FR-012–FR-014 (partial)
- [X] T083 migrate `tests/e2e/frontend_topology_workspace.spec.ts` and `tests/e2e/us3_automation.spec.ts` to the acceptance fixture, register every created laboratory immediately, and prove unconditional deletion and baseline restoration per FR-022, FR-029, and Constitution VI (contradicts)
- [X] T084 complete the runner contract by validating MCP and required browser capabilities, Wireshark helper eligibility, evidence-directory permissions, clean-baseline policy, and documented exit codes 2/75/77 before execution per FR-025 and contracts/runner-contract.md (partial)
- [X] T085 apply `NETLAB_ACCEPTANCE_TIMEOUT_SCALE` consistently to visible-feedback, durable-task, stream, reconnect, runtime, and cleanup waiters while retaining explicit bounded timeout failures per FR-021 and contracts/runner-contract.md (partial)

## Phase 9: Convergence

- [ ] T086 CRITICAL make the DOM parity audit enumerate every visible control in every applicable workspace state and require one stable inventory-backed terminal result per interaction, activation method, and viewport per FR-001, FR-002, FR-003, and SC-001 (contradicts)
- [ ] T087 CRITICAL make cleanup dispatch explicit for every ledger resource type, inject failure only after real runtime resources exist, and verify zero owned processes, VMs, containers, interfaces, namespaces, links, mappings, captures, artifacts, sockets, and temporary files before marking resources deleted per FR-022, FR-023, SC-010, Constitution V, and Constitution VI (contradicts)
- [ ] T088 CRITICAL scan filenames, DOM snapshots, error contexts, screenshots, traces, videos, reports, and all other browser artifacts for prohibited content before retention, retaining only sanitized metadata or deleting the artifact per FR-024, SC-011, and Constitution VII (contradicts)
- [ ] T089 emit one aggregated schema-valid run evidence document with stable inventory interaction IDs, exact page/control/precondition/action/expected/actual/resource/task/cleanup fields, plus a matching human-readable summary, instead of substituting hashed `test.*` records per FR-024 and SC-011 (partial)
- [ ] T090 measure real visible-feedback and pending-identity latency and require primary mutations to observe durable task identity, legal progress, terminal state, refreshed resource state, timeout, cancellation, retry, and cleanup outcomes per FR-006, FR-021, SC-002, SC-006, and SC-007 (partial)
- [ ] T091 complete frontend laboratory, template, and form journeys for real export/import outcomes, stale catalog revalidation, unavailable images, invalid boundaries and dependencies, duplicate submission, server field errors, revision conflict, merge, retry, and recoverable input preservation per FR-007, FR-008, FR-015, and FR-016 (partial)
- [ ] T092 iterate every available version rather than one version per family and perform the required representative full journey or remaining-version create/start/connect/delete coverage, enforcing `assertCompleteVersionCoverage()` in the final run gate per FR-008 and SC-013 (partial)
- [ ] T093 complete topology acceptance for selection, multi-selection, port selection, connect, disconnect, reconnect, move, pan, zoom, grouping, local routing, inspector actions, clearing selection, refresh persistence, and keyboard traversal per FR-009 and US3/AC4 (partial)
- [ ] T094 complete node acceptance across every supported resource kind for start, stop, edit, delete, CPU/memory limits, interface add/remove, QGA commands, port mapping add/remove, network-object creation/attachment/diagnostics, terminal tasks, and destructive cleanup per FR-010 (partial)
- [ ] T095 complete Task Center acceptance for laboratory, resource, operation-kind, state, time, and text filters, resource navigation, eligible cancellation, terminal replay, timestamps, results, and structured errors per FR-011 and US3/AC5 (partial)
- [ ] T096 complete deterministic Telnet and VNC session creation, explicit session switching, reconnect, terminal resize, close, refresh interruption, unsupported-mode behavior, and unchanged node lifecycle assertions per FR-012 and US3/AC1 (partial)
- [ ] T097 exercise real interface and link captures with non-empty stream data, discovery, refresh, quota, truncation, retained artifact metadata, completion, failure, stop, cleanup, and frontend Wireshark handoff using the actual capture stream and an eligible desktop-launch skip reason per FR-013 and SC-014 (partial)
- [ ] T098 generate observable scoped traffic and validate Traffic Filter input mapping, discovery, path observations, timing, confidence, duplicate, loop, ambiguity, stop, failure, refresh rejoin, and restart recovery per FR-014 and US3/AC3 (partial)
- [ ] T099 add pointer and equivalent keyboard execution for every primary journey at 1920×1080 and 1024×768, asserting focus visibility, reachability, and unclipped required or confirmation controls per FR-017 and SC-003 (partial)
- [ ] T100 collect real ordered event and convergence observations from two browsers, HTTP automation, and applicable MCP operations—including browser-originated mutation, revision conflict, reconnect, and deleted-resource handling—instead of synthesized client records or direct application-service parity per FR-019, FR-020, and SC-008 (contradicts)
- [ ] T101 build applicable empty, loading, stale, reconnecting, unsupported, permission, quota, conflict, partial-failure, cleanup, cancellation, retryable-failure, terminal-failure, and success scenarios, and refresh active Console, capture, Traffic Filter, task, and event sessions to verify rejoin or explicit reconnect-required state per FR-027 and FR-028 (partial)
- [ ] T102 compare two consecutive complete real-run evidence bundles automatically for inventory, capability decisions, version coverage, and terminal outcomes, failing on unexplained differences instead of comparing cloned synthetic evidence per SC-009 (partial)
