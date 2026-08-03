# Tasks: Topology Interaction UX

**Input**: Design documents from `specs/004-topology-interaction-ux/`

**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/`, `quickstart.md`

**Tests**: Tests are required because this feature changes shared topology state, HTTP/MCP contracts, ordered
events, live link lifecycle, recovery, cleanup, and primary browser interactions.

**Organization**: Tasks are grouped by user story so each story has an independently demonstrable result.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel because it touches different files and does not depend on an incomplete task.
- **[Story]**: Maps the task to a user story in `spec.md`.
- Test tasks precede their corresponding implementation tasks.

## Phase 1: Setup and Baseline

**Purpose**: Establish feature-specific fixtures, contracts, and safe visual-asset rules before behavior changes.

- [X] T001 Record the current topology gesture, selection, connection, and EVE-NG behavioral baseline in `specs/004-topology-interaction-ux/baseline.md`
- [X] T002 [P] Add reusable placement, dense-topology, and multi-interface factories in `web/src/test/factories.ts`
- [X] T003 [P] Add topology placement and reconnect fixtures for Go tests in `tests/testsupport/topology_fixtures.go`
- [X] T004 [P] Add the topology symbol provenance and license manifest in `web/src/assets/topology/NOTICE.md`
- [X] T005 Reconcile feature contracts with the canonical API, event, and MCP documents in `specs/001-network-simulator-platform/contracts/openapi.yaml`, `specs/001-network-simulator-platform/contracts/events.md`, and `specs/001-network-simulator-platform/contracts/mcp-tools.md`

---

## Phase 2: Foundational Shared Topology Infrastructure

**Purpose**: Add shared placement persistence and common frontend interaction primitives required by every story.

**⚠️ CRITICAL**: User-story implementation starts only after this phase passes its tests.

- [X] T006 [P] Add failing domain validation tests for `TopologyPlacement` and placement batches in `internal/domain/topology_placement_test.go`
- [X] T007 [P] Add failing SQLite migration and repository tests for placement uniqueness, revision checks, cascade cleanup, and atomic batches in `internal/store/sqlite/topology_placement_repository_test.go`
- [X] T008 [P] Add failing snapshot and ordered-event contract tests for placements in `tests/contract/topology_placement_contract_test.go`
- [X] T009 Define `TopologyPlacement`, batch mutation inputs, validation bounds, and conflict problems in `internal/domain/topology_placement.go`
- [X] T010 Add the `topology_placements` schema, indexes, foreign-key cleanup, and migration registration in `internal/store/sqlite/migrations.go`
- [X] T011 Implement placement query and transactional batch update repositories in `internal/store/sqlite/topology_placement_repository.go`
- [X] T012 Extend topology repository interfaces and laboratory snapshots with placements in `internal/app/ports/topology.go` and `internal/domain/models.go`
- [X] T013 Publish `topology.placements_changed` in the same transaction as placement updates through `internal/store/sqlite/outbox.go`
- [X] T014 [P] Define framework-independent gesture, pointer, viewport, selection, and connection state types in `web/src/features/topology/interactionTypes.ts`
- [X] T015 [P] Implement coordinate transforms, zoom anchoring, drag thresholds, port hit testing, and route geometry in `web/src/features/topology/topologyGeometry.ts`
- [X] T016 [P] Add unit tests for coordinate transforms, finite bounds, hit testing, route geometry, and dense-topology edge cases in `web/src/features/topology/topologyGeometry.test.ts`
- [X] T017 Add shared placement and interaction result types to `web/src/api/generated.ts` and `web/src/types/workspace.ts`

**Checkpoint**: Placement persistence, snapshot/event representation, geometry, and interaction types are ready.

---

## Phase 3: User Story 1 - Natural Topology Navigation and Layout (Priority: P1) 🎯 MVP

**Goal**: Make wheel zoom, canvas pan, click/drag disambiguation, node/group movement, and refresh persistence predictable.

**Independent Test**: In a multi-node lab, use mouse, trackpad-equivalent wheel input, and keyboard to zoom around
the pointer, pan blank canvas, move one or several nodes, cancel a drag, fit the topology, refresh, and verify the
same shared coordinates without changing node lifecycle or links.

### Tests for User Story 1

- [X] T018 [P] [US1] Add failing interaction-controller tests for click-versus-drag threshold, pointer capture, pan, node drag, group drag, Escape, focus loss, and wheel precedence in `web/src/features/topology/topologyInteractionController.test.ts`
- [X] T019 [P] [US1] Add failing component tests for wheel anchoring, blank-canvas pan, node drag preview, fit view, and unclipped controls in `web/src/features/topology/TopologyCanvas.test.ts`
- [X] T020 [P] [US1] Add failing HTTP and generated-client contract tests for `updateTopologyPlacements` in `tests/contract/topology_placement_http_test.go`
- [X] T021 [P] [US1] Add failing MCP parity tests for `netlab.topology.set_positions` in `tests/contract/topology_placement_mcp_test.go`
- [X] T022 [P] [US1] Add failing integration tests for batch idempotency, revision conflicts, one outbox event per drag, and API/MCP final-state parity in `tests/integration/topology_placement_integration_test.go`
- [ ] T023 [P] [US1] Add failing Playwright journeys for both viewports covering wheel, pan, node/group drag, cancellation, refresh persistence, and another-browser coordinate convergence in `tests/e2e/journeys/topologyNavigation.spec.ts`

### Implementation for User Story 1

- [X] T024 [US1] Implement revisioned placement command handling, validation, idempotency, audit details, and event publication in `internal/app/command/topology_placement.go`
- [X] T025 [US1] Implement placement query loading and deterministic fallback resolution in `internal/app/query/topology_placement.go`
- [X] T026 [US1] Register placement HTTP handlers and response problems in `internal/api/http/topology_placement_handlers.go` and `cmd/netlabd/main.go`
- [X] T027 [US1] Register `netlab.topology.set_positions` through the same application command in `internal/api/mcp/topology_placement_tools.go` and `cmd/netlabd/main.go`
- [X] T028 [US1] Add `updateTopologyPlacements` to the SPA API client in `web/src/api/index.ts`
- [X] T029 [US1] Implement the idle, pressing, panning, and dragging-resources states in `web/src/features/topology/topologyInteractionController.ts`
- [X] T030 [US1] Refactor `TopologyCanvas.vue` to normalize chart, pointer, wheel, and keyboard signals through the interaction controller in `web/src/features/topology/TopologyCanvas.vue`
- [X] T031 [US1] Commit one bounded placement batch on drag release and recover conflicts from the authoritative snapshot in `web/src/features/topology/TopologyWorkspace.vue`
- [X] T032 [US1] Remove node placements from browser-local storage while preserving viewport and route migration in `web/src/composables/useWorkspacePreferences.ts`
- [X] T033 [US1] Add fit-all, fit-selection, reset-view, and reduced-motion-safe navigation controls in `web/src/features/topology/TopologyWorkspace.vue`
- [X] T034 [US1] Update the stable interaction inventory for wheel, pan, drag, fit, reset, cancel, and shared placement outcomes in `tests/e2e/matrices/interaction-inventory.json`

**Checkpoint**: User Story 1 provides an independently usable and testable topology navigation MVP.

---

## Phase 4: User Story 2 - Accurate EVE-NG-Familiar Connection Workflow (Priority: P1)

**Goal**: Provide hover connector discovery, drag-to-node connection, compact interface selection, direct port
connection, link context actions, disconnect, and atomic reconnect with rollback.

**Independent Test**: Connect two multi-interface nodes using the connector handle, auto-select a sole free port,
choose among several ports, cancel without mutation, disconnect, and reconnect one endpoint while running; verify
the old link survives every failed, cancelled, timed-out, or conflicted reconnect.

### Tests for User Story 2

- [X] T035 [P] [US2] Add failing state-machine tests for connector discovery, source retention, target highlighting, node-body drop, port chooser, direct-port drop, and cancellation in `web/src/features/topology/topologyConnectionController.test.ts`
- [X] T036 [P] [US2] Add failing component tests for connector handle visibility, valid/invalid targets, endpoint labels, chooser focus, and link context actions in `web/src/features/topology/TopologyCanvas.test.ts` and `web/src/features/topology/PortChooser.test.ts`
- [X] T037 [P] [US2] Add failing HTTP/OpenAPI tests for atomic `reconnectLink` and structured conflict responses in `tests/contract/link_reconnect_http_test.go`
- [X] T038 [P] [US2] Add failing MCP parity tests for `netlab.links.reconnect` in `tests/contract/link_reconnect_mcp_test.go`
- [X] T039 [P] [US2] Add failing application tests proving original endpoints remain authoritative until success and are restored on failure/cancellation in `internal/app/command/link_reconnect_test.go`
- [ ] T040 [P] [US2] Add failing privileged integration tests for live connect, disconnect, reconnect, rollback, timeout, cancellation, and zero runtime leaks in `tests/integration/link_reconnect_privileged_test.go`
- [X] T041 [P] [US2] Add failing restart recovery tests for interrupted reconnect tasks and orphan-link prevention in `tests/recovery/link_reconnect_recovery_test.go`
- [ ] T042 [P] [US2] Add failing Playwright journeys for EVE-NG-familiar hover-drag-connect, sole-port bypass, multi-port chooser, keyboard path, disconnect, reconnect, and recoverable errors in `tests/e2e/journeys/topologyConnection.spec.ts`

### Implementation for User Story 2

- [X] T043 [US2] Extend link command ports with atomic reconnect and compensation interfaces in `internal/app/ports/topology.go`
- [X] T044 [US2] Implement durable reconnect task validation, runtime application, compensation, cancellation, audit, and terminal events in `internal/app/command/link_reconnect.go`
- [X] T045 [US2] Reuse approved QEMU, Docker, bridge, and namespace adapters for reconnect without shell interpolation in `internal/app/reconcile/topology_operations.go`
- [X] T046 [US2] Add reconnect HTTP routing through the application command in `internal/api/http/topology_handlers.go`
- [X] T047 [US2] Add `netlab.links.reconnect` through the same command handler in `internal/api/mcp/topology_tools.go`
- [X] T048 [US2] Add reconnect and task-envelope methods to `web/src/api/index.ts` and `web/src/api/generated.ts`
- [X] T049 [US2] Implement connecting, choosing-target-port, and cancelling transitions in `web/src/features/topology/topologyInteractionController.ts`
- [X] T050 [US2] Render the EVE-NG-familiar connector handle, temporary line, target affordances, ports, and endpoint labels in `web/src/features/topology/TopologyCanvas.vue`
- [X] T051 [US2] Implement the compact accessible interface chooser with one-port auto-selection in `web/src/features/topology/PortChooser.vue`
- [X] T052 [US2] Implement link Inspect, Reconnect endpoint, Disconnect, and Edit local route actions in `web/src/features/topology/LinkContextMenu.vue`
- [X] T053 [US2] Replace frontend disconnect-then-connect reconnect logic with the atomic task workflow in `web/src/features/topology/TopologyWorkspace.vue`
- [X] T054 [US2] Add durable task feedback, retry, cancellation, conflict reload, and original-link preservation UI in `web/src/features/topology/TopologyWorkspace.vue`
- [X] T055 [US2] Update connection, chooser, disconnect, reconnect, cancellation, and context-action entries in `tests/e2e/matrices/interaction-inventory.json`

**Checkpoint**: User Story 2 delivers the complete connection workflow without half-connected topology states.

---

## Phase 5: User Story 3 - Recognizable Nodes, Ports, Links, and States (Priority: P2)

**Goal**: Make every supported resource kind and lifecycle state identifiable without opening the inspector.

**Independent Test**: Display QEMU, Docker, PC, bridge, NAT, L2, and L3 resources across running, stopped,
transitioning, failed, and selected states; users can identify kind/state and discover ports without relying on color.

### Tests for User Story 3

- [X] T056 [P] [US3] Add failing visual-semantic tests for every resource kind, lifecycle state, fallback symbol, and non-color label in `web/src/features/topology/topologyVisualSemantics.test.ts`
- [X] T057 [P] [US3] Add failing component tests for symbol rendering, port detail levels, traffic overlays, link visibility, and missing-asset fallback in `web/src/features/topology/TopologyCanvas.test.ts`
- [X] T058 [P] [US3] Add failing accessibility tests for names, roles, status text, contrast classes, and reduced motion in `web/src/features/topology/TopologyCanvas.a11y.test.ts`
- [X] T059 [P] [US3] Add failing browser recognition and dense-label journeys at both viewports in `tests/e2e/journeys/topologyVisualRecognition.spec.ts`

### Implementation for User Story 3

- [X] T060 [P] [US3] Create local neutral SVG symbols and fallbacks for QEMU, Docker, PC, bridge, NAT, L2, and L3 in `web/src/assets/topology/`
- [X] T061 [US3] Map resource kind, desired/observed state, selection, failure, and traffic semantics in `web/src/features/topology/topologyVisualSemantics.ts`
- [X] T062 [US3] Refactor ECharts graph data and overlays to use licensed symbols, stable labels, endpoint details, and progressive port disclosure in `web/src/features/topology/TopologyCanvas.vue`
- [X] T063 [US3] Add comfortable, compact, and minimal label-density preferences in `web/src/composables/useWorkspacePreferences.ts` and `web/src/types/workspace.ts`
- [X] T064 [US3] Document every topology visual asset source, version, license, and fallback in `web/src/assets/topology/NOTICE.md`

**Checkpoint**: User Story 3 is visually complete, accessible, offline-capable, and license-auditable.

---

## Phase 6: User Story 4 - Precise Selection and Keyboard Parity (Priority: P2)

**Goal**: Support single selection, additive selection, box selection, group movement, clear/cancel precedence, and
keyboard-equivalent navigation and connection.

**Independent Test**: Use pointer and keyboard independently to select overlapping resources, extend and reduce
selection, box-select, move a group, connect ports, cancel transient state, clear selection, and open the inspector.

### Tests for User Story 4

- [X] T065 [P] [US4] Add failing selection-model tests for single, toggle, range/box, group drag, background clear, and deleted-resource cleanup in `web/src/features/topology/topologySelection.test.ts`
- [X] T066 [P] [US4] Add failing keyboard-controller tests for traversal, extension, port navigation, connect, cancel priority, movement, and inspector activation in `web/src/features/topology/topologyKeyboardController.test.ts`
- [X] T067 [P] [US4] Add failing component tests for selection rectangle, visible focus, keyboard announcements, and minimum-viewport reachability in `web/src/features/topology/TopologyCanvas.test.ts`
- [X] T068 [P] [US4] Add failing pointer/keyboard parity journeys for both viewports in `tests/e2e/journeys/topologySelectionKeyboard.spec.ts`

### Implementation for User Story 4

- [X] T069 [P] [US4] Implement immutable selection and box-intersection helpers in `web/src/features/topology/topologySelection.ts`
- [X] T070 [P] [US4] Implement ordered resource/port keyboard navigation and announcements in `web/src/features/topology/topologyKeyboardController.ts`
- [X] T071 [US4] Add box-selecting and cancellation-priority transitions to `web/src/features/topology/topologyInteractionController.ts`
- [X] T072 [US4] Render selection rectangle, group-drag preview, focus affordances, and live announcements in `web/src/features/topology/TopologyCanvas.vue`
- [X] T073 [US4] Integrate keyboard navigation, group movement, clear selection, and inspector activation in `web/src/features/topology/TopologyWorkspace.vue`
- [X] T074 [US4] Add selection, group movement, keyboard connection, Escape, and focus inventory entries in `tests/e2e/matrices/interaction-inventory.json`

**Checkpoint**: User Story 4 provides equivalent pointer and keyboard outcomes at both supported viewports.

---

## Phase 7: User Story 5 - Shared Convergence, Recovery, and Performance (Priority: P3)

**Goal**: Keep all browser/API/MCP clients converged without session invalidation and recover safely from event gaps,
conflicts, refreshes, timeouts, partial failures, and service restarts.

**Independent Test**: Two browsers and an automation client concurrently move nodes, start nodes, and reconnect
links; inject stale revisions, event gaps, disconnects, failures, cancellation, and restart; verify convergence in
five seconds, original-link rollback, session continuity, and zero leaks.

### Tests for User Story 5

- [ ] T075 [P] [US5] Add failing event-order and reset-required contract tests for placement and reconnect events in `tests/contract/topology_interaction_events_test.go`
- [ ] T076 [P] [US5] Add failing two-browser/API/MCP convergence and session-continuity integration tests in `tests/integration/topology_interaction_concurrency_test.go`
- [ ] T077 [P] [US5] Add failing recovery tests for stale placements, deleted-resource cleanup, event gaps, interrupted drag refresh, and service restart in `tests/recovery/topology_interaction_recovery_test.go`
- [ ] T078 [P] [US5] Add failing dense-topology performance tests for 100 nodes, 200 links, bounded render updates, and one placement write per drag in `web/src/features/topology/topologyPerformance.test.ts`
- [ ] T079 [P] [US5] Add failing Playwright concurrency, reconnect, conflict, refresh, failure, and session-preservation journeys in `tests/e2e/journeys/topologySharedConvergence.spec.ts`
- [ ] T080 [P] [US5] Add failing target-host cleanup evidence for processes, interfaces, links, tasks, events, and temporary artifacts after reconnect failure in `tests/e2e/journeys/topologyRuntimeCleanup.spec.ts`

### Implementation for User Story 5

- [X] T081 [US5] Apply placement/reconnect events by revision and trigger snapshot reload on event gaps in `web/src/stores/topology.ts`
- [X] T082 [US5] Preserve browser sessions and pending operation identities across API/MCP-originated mutations in `web/src/stores/topology.ts` and `web/src/features/topology/TopologyWorkspace.vue`
- [ ] T083 [US5] Reconcile missing/deleted placement records and interrupted reconnect tasks during startup in `internal/app/reconcile/topology_recovery.go`
- [ ] T084 [US5] Add bounded audit and observability fields for placement latency, reconnect phase, rollback, conflicts, and event lag in `internal/app/audit/service.go` and `internal/support/observability/metrics.go`
- [X] T085 [US5] Add render batching, detail degradation, label culling, and interaction-safe throttling for dense topologies in `web/src/features/topology/TopologyCanvas.vue`
- [ ] T086 [US5] Record real client observations, task timings, event sequences, conflict recovery, and leak cleanup in the acceptance evidence fixtures under `tests/e2e/fixtures/`

**Checkpoint**: User Story 5 proves shared-state correctness, recovery, performance, and resource safety.

---

## Phase 8: Polish and Cross-Cutting Validation

**Purpose**: Finish migration, documentation, artifact hygiene, and complete validation gates.

- [ ] T087 [P] Update frontend interaction and EVE-NG comparison documentation in `specs/004-topology-interaction-ux/quickstart.md` and `specs/004-topology-interaction-ux/baseline.md`
- [ ] T088 [P] Update canonical API examples and generated client parity assertions in `tests/contract/us3_openapi_parity_test.go`
- [X] T089 [P] Add artifact-policy checks that reject remote icons, unlicensed symbols, EVE-NG assets, credentials, console output, and packet content in `scripts/check-frontend-artifacts.sh`
- [X] T090 Remove obsolete local placement storage and legacy topology gesture paths after migration in `web/src/composables/useWorkspacePreferences.ts`, `web/src/features/topology/TopologyCanvas.vue`, and `web/src/features/topology/TopologyWorkspace.vue`
- [X] T091 Run and fix formatting, lint, type checking, Go unit/contract/integration/recovery tests, frontend tests, and production build using the commands in `specs/004-topology-interaction-ux/quickstart.md`
- [ ] T092 Run the complete local Playwright acceptance suite twice and compare normalized outcomes using `acceptance/frontend-acceptance-repeat.sh`
- [ ] T093 Deploy the reviewed binary out of band and run the complete target-host acceptance suite twice against `http://10.72.1.7:8088` without recording credentials in `specs/004-topology-interaction-ux/validation.md`
- [ ] T094 Run controlled reconnect failure, cancellation, timeout, event-gap, and restart scenarios on the target host and record zero-leak evidence in `specs/004-topology-interaction-ux/validation.md`
- [ ] T095 Re-run the EVE-NG familiarity comparison without retaining credentials or proprietary artifacts and document behavioral outcomes in `specs/004-topology-interaction-ux/validation.md`

---

## Dependencies and Execution Order

### Phase Dependencies

- **Phase 1** has no dependencies.
- **Phase 2** depends on Phase 1 and blocks every user story.
- **US1 (Phase 3)** depends on Phase 2 and is the MVP.
- **US2 (Phase 4)** depends on Phase 2; its workspace integration should follow US1 controller integration.
- **US3 (Phase 5)** depends on the Phase 2 geometry/types and can proceed in parallel with most US2 backend work.
- **US4 (Phase 6)** depends on the Phase 2 interaction types and should integrate after US1 controller behavior stabilizes.
- **US5 (Phase 7)** depends on US1 placement events and US2 reconnect tasks; performance work also consumes US3/US4 UI.
- **Phase 8** depends on all selected user stories.

### User Story Dependency Graph

```text
Setup → Foundation → US1 (MVP)
                   ├──→ US2 ──┐
                   ├──→ US3 ──┼──→ US5 → Polish
                   └──→ US4 ──┘
```

### Within Each User Story

1. Add tests and prove they fail for the missing behavior.
2. Add domain/application/storage contracts before transport adapters.
3. Route HTTP and MCP through the same application command.
4. Implement pure interaction/geometry modules before Vue/ECharts integration.
5. Complete browser and privileged validation before marking the story done.

## Parallel Execution Examples

### User Story 1

```text
T018 interaction-controller tests
T020 HTTP contract tests
T021 MCP contract tests
T023 Playwright navigation journey
```

After T024–T028, T029 controller work and T032 preference migration can proceed in parallel.

### User Story 2

```text
T035 connection state tests
T037 HTTP reconnect contract
T038 MCP reconnect contract
T040 privileged runtime test
T042 Playwright connection journey
```

After T044, HTTP/MCP adapters T046–T048 can proceed alongside frontend controller and component work T049–T052.

### User Story 3

```text
T056 visual semantic tests
T058 accessibility tests
T059 browser recognition journey
T060 SVG symbol creation
```

### User Story 4

```text
T065 selection tests
T066 keyboard tests
T067 component focus tests
T068 browser parity journey
```

### User Story 5

```text
T075 event contracts
T076 convergence integration
T077 recovery tests
T078 performance tests
T079 browser convergence journey
T080 target cleanup journey
```

## Implementation Strategy

### MVP First

1. Complete Setup and Foundation.
2. Complete US1 only.
3. Demonstrate predictable wheel, pan, node/group drag, shared placement, refresh persistence, and session-safe
   convergence before expanding connection behavior.

### Incremental Delivery

1. **US1**: Natural navigation and shared layout.
2. **US2**: EVE-NG-familiar connection workflow with atomic reconnect.
3. **US3**: Recognizable licensed visual language.
4. **US4**: Full selection and keyboard parity.
5. **US5**: Multi-client convergence, recovery, dense performance, and leak proof.

### Completion Rule

A task is complete only when its referenced implementation exists, its required tests pass, cleanup restores the
baseline, and no credentials, EVE-NG assets, proprietary images, packet data, console output, or unsafe browser
artifacts are retained.

## Phase 9: Convergence

- [X] T096 CRITICAL correct topology creation copy to state that confirmed placements are shared while viewport and manual routes are browser-local, and add a regression assertion in `web/src/features/topology/CreateTopologyResourceDialog.vue` and its component tests per Constitution I and FR-004 (contradicts)
- [X] T097 Add focus-anchored keyboard zoom actions, visible announcements, inventory coverage, and dual-viewport browser tests in `web/src/features/topology/topologyKeyboardController.ts`, `web/src/features/topology/TopologyWorkspace.vue`, and `tests/e2e/journeys/topologyNavigation.spec.ts` per FR-001 and FR-017 (missing)
- [X] T098 Implement node-hover tracking and progressive port-name, connection-state, and availability disclosure without defeating dense-topology degradation in `web/src/features/topology/TopologyCanvas.vue` and its component/browser tests per FR-008 and US3/AC3 (partial)
- [X] T099 Replace the fixed local-route toggle with draggable browser-local link control points, reset/cancel behavior, and persistence tests in `web/src/features/topology/TopologyCanvas.vue`, `web/src/features/topology/TopologyWorkspace.vue`, and `web/src/composables/useWorkspacePreferences.ts` per FR-014 (partial)
- [X] T100 Add pointer and keyboard select-all for applicable topology resources with visible focus/selection feedback and dual-viewport tests in `web/src/features/topology/topologySelection.ts`, `web/src/features/topology/topologyKeyboardController.ts`, and `tests/e2e/journeys/topologySelectionKeyboard.spec.ts` per FR-015 (missing)
- [X] T101 Centralize Escape, pointer-cancel, window-blur, visibility-loss, overlay, drag, box-selection, connection, and selection cancellation into one priority arbiter with no double dispatch in `web/src/features/topology/TopologyCanvas.vue`, `web/src/features/topology/TopologyWorkspace.vue`, and interaction tests per FR-018, FR-023, and the interrupted-drag edge case (partial)
- [X] T102 Encode desired and observed lifecycle states together in non-color topology labels and accessibility summaries, then extend real-browser recognition coverage to QEMU, Docker, lifecycle combinations, and dense labels in `web/src/features/topology/topologyVisualSemantics.ts`, `web/src/features/topology/TopologyCanvas.vue`, and `tests/e2e/journeys/topologyVisualRecognition.spec.ts` per FR-007, FR-027, and SC-005 (partial)
- [X] T103 Complete revision-ordered placement and reconnect event application, laboratory-scoped sequence handling, reset-required snapshot reload, and pending-operation preservation in the actual `web/src/stores/laboratory.ts` store with contract and concurrency tests per FR-021 and T081/T082 (partial)
- [X] T104 Remove browser-local link control points when authoritative links or laboratories are deleted and add refresh/recovery regression tests in `web/src/composables/useWorkspacePreferences.ts` and `web/src/stores/laboratory.ts` per FR-024 (partial)
