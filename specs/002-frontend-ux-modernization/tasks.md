# Tasks: Frontend UX Modernization

**Input**: Design documents from `specs/002-frontend-ux-modernization/`

**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/`, `quickstart.md`

**Tests**: Component tests use controlled mocks; contract/integration tests use the real test service;
critical Playwright flows use the actual Go backend; privileged runtime flows are validated on `10.72.1.7`.

**Organization**: Tasks are grouped by user story so each story can be implemented and validated as an
independent increment after the shared foundation is complete.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel because it writes different files and has no dependency on an incomplete task.
- **[Story]**: Maps the task to a user story in `spec.md`.
- Every task includes the exact file or directory it owns.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Establish the design system, directory layout, and frontend test configuration.

- [X] T001 Add Shadcn Vue, Tailwind CSS v4, ECharts, Reka UI, Lucide, and required utility dependencies and scripts in `web/package.json`
- [X] T002 Configure Shadcn Vue aliases and Tailwind/Vite integration in `web/components.json`, `web/vite.config.ts`, and `web/src/styles/index.css`
- [X] T003 [P] Create NetLab color, spacing, density, typography, lifecycle, traffic, and focus tokens in `web/src/styles/theme.css`
- [X] T004 [P] Create feature and shared-component directory exports in `web/src/components/index.ts`, `web/src/components/ui/index.ts`, `web/src/components/shell/index.ts`, and `web/src/composables/index.ts`
- [X] T005 Configure shared Vitest setup, cleanup, accessibility matchers, and typed mock reset behavior in `web/vitest.config.ts` and `web/src/test/setup.ts`
- [X] T006 Configure Playwright desktop, minimum-viewport, and real-backend projects with sanitized artifact settings in `web/playwright.config.ts`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Build shared primitives, state boundaries, renderer adapters, and test factories required by all stories.

**⚠️ CRITICAL**: No user story implementation begins until this phase is complete.

- [X] T007 Implement the baseline Shadcn Vue primitives used across the application in `web/src/components/ui/button/`, `web/src/components/ui/dialog/`, `web/src/components/ui/dropdown-menu/`, `web/src/components/ui/form/`, `web/src/components/ui/input/`, `web/src/components/ui/select/`, `web/src/components/ui/tabs/`, `web/src/components/ui/tooltip/`, `web/src/components/ui/sheet/`, and `web/src/components/ui/resizable/`
- [X] T008 [P] Add shared status badge, empty state, loading state, confirmation dialog, and resource identity components in `web/src/components/common/`
- [X] T009 [P] Define authoritative projection, local workspace preference, pending submission, task, problem, and diagnostic presentation types in `web/src/types/workspace.ts`
- [X] T010 [P] Add deterministic API resource, task, event, problem, topology, and chart fixture factories in `web/src/test/factories.ts`
- [X] T011 [P] Add typed controlled API, event-stream, ECharts, ResizeObserver, xterm, and noVNC mocks in `web/src/test/mocks/`
- [X] T012 Implement and unit test versioned laboratory-scoped preference validation, migration, clamping, debounced persistence, and safe fallback in `web/src/composables/useWorkspacePreferences.ts` and `web/src/composables/useWorkspacePreferences.test.ts`
- [X] T013 Implement and unit test the modular ECharts lifecycle wrapper with stable IDs, resize observation, event cleanup, partial updates, and disposal in `web/src/components/charts/EChart.vue` and `web/src/components/charts/EChart.test.ts`
- [X] T014 [P] Implement deterministic collision-aware placement and topology view-model derivation in `web/src/features/topology/topologyLayout.ts` and `web/src/features/topology/topologyLayout.test.ts`
- [X] T015 [P] Implement the shared structured-problem presenter with retry, cleanup, operator-action, field-error, and revision-conflict views in `web/src/components/common/StructuredProblem.vue` and `web/src/components/common/StructuredProblem.test.ts`
- [X] T016 Refactor ordered-event convergence, replay-gap refresh, reconnect state, and task/resource normalization without changing API authority in `web/src/stores/laboratory.ts` and `web/src/stores/laboratory.test.ts`
- [X] T017 Add a typed UI operation registry mapping visible mutations to generated API operations, revisions, idempotency, tasks, and navigation targets in `web/src/api/operationRegistry.ts` and `web/src/api/operationRegistry.test.ts`
- [X] T018 Add contract tests that verify every operation in `contracts/backend-integration-matrix.md` is represented by generated types and the real test service in `tests/contract/frontend_operation_matrix_test.go`

**Checkpoint**: Shared components, local-state boundaries, ECharts lifecycle, generated API mapping, and
authoritative convergence behavior are ready for user-story work.

---

## Phase 3: User Story 1 - Build and Operate a Topology Visually (Priority: P1) 🎯 MVP

**Goal**: Deliver the recognizable five-region workspace and complete graphical laboratory/topology workflow.

**Independent Test**: From an empty laboratory, add supported QEMU, Docker, and lightweight nodes, create
network objects and links, rewire a running connection, start nodes, inspect state, and delete the laboratory
using only the graphical workspace; confirm a second client sees the same resources but not local placement.

### Tests for User Story 1

- [X] T019 [P] [US1] Add component tests for toolbar, resizable shell, panel persistence, responsive drawer state, and destructive confirmations in `web/src/components/shell/LaboratoryShell.test.ts`
- [X] T020 [P] [US1] Add component tests for ECharts topology nodes, links, interfaces, selection, multi-selection, pan/zoom/fit, drag persistence, pending state, and deterministic remote placement in `web/src/features/topology/TopologyCanvas.test.ts`
- [X] T021 [P] [US1] Add component tests for searchable categorized device palette and compatible template/image selection in `web/src/features/topology/DevicePalette.test.ts`
- [X] T022 [P] [US1] Add component tests for node/link/network-object inspectors, valid actions, revision changes, and deleted-while-open behavior in `web/src/features/topology/TopologyInspector.test.ts`
- [X] T023 [P] [US1] Add real-service integration tests for laboratory CRUD, node creation, network objects, connect/disconnect, live rewiring, tasks, and event convergence in `tests/integration/frontend_topology_workflow_test.go`
- [X] T024 [P] [US1] Add Playwright coverage for the independent topology workflow and two-browser local-layout isolation in `tests/e2e/frontend_topology_workspace.spec.ts`

### Implementation for User Story 1

- [X] T025 [P] [US1] Implement laboratory create, rename, duplicate, export, import, and delete controls with confirmations in `web/src/features/laboratories/LaboratoryToolbar.vue`
- [X] T026 [P] [US1] Implement searchable categorized QEMU, Docker, and lightweight device palette with capability-aware template/image selection in `web/src/features/topology/DevicePalette.vue`
- [X] T027 [US1] Implement the resizable top/left/center/right/bottom workspace shell and panel persistence integration in `web/src/components/shell/LaboratoryShell.vue`
- [X] T028 [US1] Implement the ECharts topology canvas with stable resource IDs, local coordinates, legends, tooltips, accessible descriptions, selection, multi-selection, pan/zoom/fit, and drag handling in `web/src/features/topology/TopologyCanvas.vue`
- [X] T029 [US1] Implement direct node/network-object placement and create dialogs using existing typed API operations in `web/src/features/topology/CreateTopologyResourceDialog.vue`
- [X] T030 [US1] Implement interface compatibility feedback, link preview, connect, disconnect, and live rewire interactions in `web/src/features/topology/useLinkEditing.ts`
- [X] T031 [US1] Implement context-sensitive node, link, and network-object inspectors with desired/observed state, revision, runtime identity, pending task, and structured problem details in `web/src/features/topology/TopologyInspector.vue`
- [X] T032 [US1] Integrate authoritative store state, local preferences, shell regions, topology canvas, palette, inspector, and task drawer while decomposing the existing monolith in `web/src/features/topology/TopologyWorkspace.vue`
- [X] T033 [US1] Update routing and route-level loading/error/empty behavior for laboratory selection and workspace entry in `web/src/router/index.ts` and `web/src/App.vue`

**Checkpoint**: User Story 1 is a deployable graphical topology MVP independent of later operational and
diagnostic enhancements.

---

## Phase 4: User Story 2 - Operate Nodes with Clear Feedback (Priority: P1)

**Goal**: Provide complete node operations, durable task visibility, resource trends, and actionable failures.

**Independent Test**: Start/stop a node, hot-add/remove an interface, change resources/NIC driver, execute a
guest command, manage a port mapping, cancel a task, observe CPU/memory trends, and recover from injected
revision, runtime, quota, and cleanup failures without duplicate submission.

### Tests for User Story 2

- [X] T034 [P] [US2] Add component tests for lifecycle, interface, resource, guest-command, and port-mapping forms including capability gating and duplicate-submit protection in `web/src/features/nodes/NodeOperationsPanel.test.ts`
- [X] T035 [P] [US2] Add component tests for task filters, progress, cancellation, task-to-resource navigation, reconnect recovery, and deleted/missing resources in `web/src/features/tasks/TaskCenter.test.ts`
- [X] T036 [P] [US2] Add component tests for revision conflict preservation and structured runtime/quota/cleanup errors in `web/src/features/nodes/NodeOperationProblemFlows.test.ts`
- [X] T037 [P] [US2] Add component tests for ECharts CPU, memory, task-progress, and operation-statistics views including empty and degraded states in `web/src/features/analytics/ResourceCharts.test.ts`
- [X] T038 [P] [US2] Add real-service integration tests for node operations, task cancellation, idempotent replay, conflict handling, and structured problem fields in `tests/integration/frontend_node_operations_test.go`
- [X] T039 [P] [US2] Add Playwright coverage for a successful operation, a cancellable task, and an injected actionable failure in `tests/e2e/frontend_node_operations.spec.ts`

### Implementation for User Story 2

- [X] T040 [P] [US2] Implement filtered durable task center with progress, timestamps, cancellation, resource navigation, and structured problems in `web/src/features/tasks/TaskCenter.vue`
- [X] T041 [P] [US2] Implement reusable pending-submission and safe-replay handling around generated API operations in `web/src/composables/useDurableOperation.ts`
- [X] T042 [US2] Refactor node lifecycle and capability-aware operation controls to use the operation registry and durable task presenter in `web/src/features/nodes/NodeOperationsPanel.vue`
- [X] T043 [P] [US2] Implement interface hot-add/remove and NIC-driver forms with staged operation feedback in `web/src/features/nodes/InterfaceOperations.vue`
- [X] T044 [P] [US2] Implement vCPU, CPU-time quota, memory, and other supported resource editing with revision preservation in `web/src/features/nodes/NodeResourcesEditor.vue`
- [X] T045 [P] [US2] Implement bounded guest-command execution and result/problem presentation without browser persistence in `web/src/features/nodes/GuestCommandPanel.vue`
- [X] T046 [P] [US2] Implement host-port mapping list, collision feedback, create, and delete workflows in `web/src/features/nodes/PortMappingsPanel.vue`
- [X] T047 [US2] Implement modular ECharts resource trends, task progress, and operation statistics with shared legends/tooltips in `web/src/features/analytics/ResourceCharts.vue`
- [X] T048 [US2] Integrate task center and node-operation panels into the inspector and bottom drawer without losing active operations during navigation in `web/src/components/shell/OperationsDrawer.vue`

**Checkpoint**: User Story 2 independently proves that the SPA never implies completion before durable
server state and gives actionable recovery information for failures.

---

## Phase 5: User Story 3 - Use Consoles and Network Diagnostics (Priority: P1)

**Goal**: Provide reconnectable Telnet/VNC consoles, capture operations, and ECharts Traffic Filter paths.

**Independent Test**: Open and reconnect supported consoles, start/monitor/stream/download/stop a capture,
run a known Traffic Filter, inspect direction/count/timing/confidence/loop ambiguity, and close/reopen panels
without changing node lifecycle or leaking renderer resources.

### Tests for User Story 3

- [X] T049 [P] [US3] Add component tests for console discovery, tabs, reconnect, resize, unsupported modes, close semantics, and xterm/noVNC cleanup in `web/src/features/diagnostics/ConsoleWorkspace.test.ts`
- [X] T050 [P] [US3] Add component tests for capture start/status/stop/stream/download, quota, truncation, retention, and sanitized metadata in `web/src/features/diagnostics/CapturePanel.test.ts`
- [X] T051 [P] [US3] Add component tests for ECharts Traffic Filter path observations, no-match, loops, duplicates, multiple paths, confidence, and ambiguity in `web/src/features/diagnostics/TrafficFilterChart.test.ts`
- [X] T052 [P] [US3] Add real-service integration tests for console metadata, capture lifecycle/artifacts, Traffic Filter lifecycle/observations, event convergence, and redaction in `tests/integration/frontend_diagnostics_test.go`
- [X] T053 [P] [US3] Add Playwright coverage for console hosting, capture lifecycle, and Traffic Filter visualization against the real backend in `tests/e2e/frontend_diagnostics.spec.ts`

### Implementation for User Story 3

- [X] T054 [P] [US3] Implement reconnectable Telnet xterm and VNC noVNC renderer hosts with explicit transport/observer disposal in `web/src/features/diagnostics/ConsoleWorkspace.vue`
- [X] T055 [P] [US3] Implement capture target selection, task/resource status, limits, truncation, retention, stream, download, and stop controls in `web/src/features/diagnostics/CapturePanel.vue`
- [X] T056 [P] [US3] Implement Traffic Filter characteristic forms and durable operation lifecycle in `web/src/features/diagnostics/TrafficFilterPanel.vue`
- [X] T057 [US3] Implement the ECharts Traffic Filter path chart joined to topology coordinates with direction, packet count, timing, confidence, loop, and ambiguity indicators in `web/src/features/diagnostics/TrafficFilterChart.vue`
- [X] T058 [US3] Integrate console, capture, Traffic Filter, and network-object diagnostic tabs into the resizable bottom drawer in `web/src/features/diagnostics/DiagnosticsPanel.vue`
- [X] T059 [US3] Add sanitized artifact URL and streaming helpers that never persist packet or console payloads in `web/src/api/diagnostics.ts`

**Checkpoint**: User Story 3 independently validates the complete live diagnostic workflow and renderer cleanup.

---

## Phase 6: User Story 4 - Manage Templates, Images, and Automation (Priority: P2)

**Goal**: Complete template/image discovery and make automation-created resources and tasks fully visible.

**Independent Test**: Browse template capabilities and image provenance/digest/license/validation data, import
an allowed image reference, create a node with a selected version, inspect automation guidance, and verify an
API/MCP-created topology mutation appears with identical resource, task, result, and failure state.

### Tests for User Story 4

- [X] T060 [P] [US4] Add component tests for template capability comparison, image versions, provenance, digest, license notes, availability, validation, and stale-dialog refresh in `web/src/features/templates/TemplateCatalog.test.ts`
- [X] T061 [P] [US4] Add component tests for image import task feedback and secret/proprietary-data redaction in `web/src/features/templates/ImageImportDialog.test.ts`
- [X] T062 [P] [US4] Add component tests for automation operation discovery, API/MCP parity guidance, and automation-created task/resource navigation in `web/src/views/AutomationView.test.ts`
- [X] T063 [P] [US4] Add real-service integration tests for templates, images, image import metadata, lab export/import metadata, automation mutations, and audit visibility in `tests/integration/frontend_templates_automation_test.go`
- [X] T064 [P] [US4] Add Playwright coverage for template/image browsing and an automation-created resource appearing in the SPA in `tests/e2e/frontend_templates_automation.spec.ts`

### Implementation for User Story 4

- [X] T065 [P] [US4] Implement searchable template catalog and capability/version comparison using existing generated template/image types in `web/src/features/templates/TemplateCatalog.vue`
- [X] T066 [P] [US4] Implement image provenance, digest, license, validation, availability, and import-task presentation in `web/src/features/templates/ImageImportDialog.vue`
- [X] T067 [US4] Refactor template selection to handle live catalog changes and prevent stale incompatible submissions in `web/src/features/templates/TemplatePicker.vue`
- [X] T068 [US4] Complete the route-level template/image management view with Shadcn Vue layout and consistent loading/error/empty states in `web/src/views/TemplatesView.vue`
- [X] T069 [US4] Complete the automation view with API/MCP operation discovery, shared-state explanations, audit/task navigation, and no UI-only controls in `web/src/views/AutomationView.vue`
- [X] T070 [US4] Add laboratory export/import artifact status, missing-image reporting, and redacted metadata presentation in `web/src/features/laboratories/LaboratoryTransferDialog.vue`

**Checkpoint**: User Story 4 independently proves template/image transparency and UI/API/MCP state parity.

---

## Phase 7: User Story 5 - Work Efficiently Across Screen Sizes and Input Methods (Priority: P2)

**Goal**: Make primary workflows keyboard accessible, non-color-dependent, responsive at 1024×768, and performant.

**Independent Test**: Complete core topology, node-operation, task, and diagnostic workflows with keyboard-only
input at desktop and 1024×768 viewports; verify visible focus, descriptive labels, responsive drawers, non-color
state cues, and the 100-node/300-link interaction target.

### Tests for User Story 5

- [X] T071 [P] [US5] Expand automated accessibility coverage for landmarks, names, focus order, dialogs, tabs, menus, forms, and non-color state semantics in `tests/e2e/accessibility.spec.ts`
- [X] T072 [P] [US5] Add Playwright minimum-viewport and keyboard-only workflows for topology, operations, tasks, consoles, and diagnostics in `tests/e2e/frontend_responsive_keyboard.spec.ts`
- [X] T073 [P] [US5] Add deterministic 100-node/300-link ECharts interaction benchmarks and renderer leak assertions in `web/src/features/topology/TopologyCanvas.performance.test.ts`
- [X] T074 [P] [US5] Add component tests for global shortcuts, topology keyboard selection, command palette, focus restoration, and conflict-safe dialogs in `web/src/composables/useWorkspaceKeyboard.test.ts`

### Implementation for User Story 5

- [X] T075 [P] [US5] Implement workspace keyboard navigation, topology selection commands, focus restoration, and guarded global shortcuts in `web/src/composables/useWorkspaceKeyboard.ts`
- [X] T076 [P] [US5] Implement a searchable keyboard command palette using only existing operation-registry actions in `web/src/components/shell/CommandPalette.vue`
- [X] T077 [US5] Implement 1024×768 responsive shell fallbacks that convert secondary regions to accessible sheets/tabs while preserving canvas and active operation visibility in `web/src/components/shell/LaboratoryShell.vue`
- [X] T078 [US5] Add text, icon, shape, pattern, ARIA, tooltip, legend, and inspector equivalents for topology lifecycle and traffic states in `web/src/features/topology/topologyVisualSemantics.ts`
- [X] T079 [US5] Profile and optimize ECharts option updates, placement persistence, long task/palette rendering, and mount/unmount cleanup to meet the interaction target in `web/src/features/topology/TopologyCanvas.vue`

**Checkpoint**: User Story 5 independently proves baseline accessibility, responsive usability, and measured
workspace performance.

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Verify integration quality, hygiene, documentation, and privileged acceptance across all stories.

- [X] T080 [P] Add user-facing workspace terminology, interaction, console, capture, and Traffic Filter guidance without EVE-NG branding/assets in `docs/frontend-workspace.md`
- [X] T081 [P] Add a tracked-file and browser-artifact hygiene check for credentials, cloud-init data, packet captures, proprietary images, and sensitive Playwright outputs in `scripts/check-frontend-artifacts.sh` and `Makefile`
- [X] T082 Reconcile generated API types and document any verified backend contract defects before adding backend changes in `web/src/api/generated.ts` and `specs/002-frontend-ux-modernization/contracts/backend-integration-matrix.md`
- [X] T083 Run and fix focused formatting, lint, component, contract, and real-backend E2E validation through `Makefile`, without changing unrelated failures
- [X] T084 Execute the complete scenarios in `specs/002-frontend-ux-modernization/quickstart.md` and record sanitized results in `specs/002-frontend-ux-modernization/validation.md`
- [X] T085 Run privileged QEMU, Docker, live rewiring, console, capture, Traffic Filter, guest-command, port-mapping, and cleanup acceptance on `10.72.1.7` and record metadata-only results in `specs/002-frontend-ux-modernization/validation-host.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies; T003 and T004 can run after T001/T002 ownership is coordinated.
- **Foundational (Phase 2)**: Depends on Setup and blocks every user story.
- **US1, US2, US3 (P1)**: Start after Foundational. They may run in parallel, but US2 and US3 integrate
  into shell extension points established by US1; their feature components and tests can proceed independently.
- **US4 and US5 (P2)**: Start after Foundational. US5's final shell integration follows US1's shell work.
- **Polish (Phase 8)**: Runs after all stories selected for release are complete.

### User Story Dependencies

- **User Story 1**: No story dependency; establishes the MVP shell and topology workspace.
- **User Story 2**: No functional dependency on US1; final inspector/drawer integration uses US1 shell slots.
- **User Story 3**: No functional dependency on US1; final diagnostics drawer integration uses US1 shell slots.
- **User Story 4**: Independent after Foundational; shares generated API and task/problem presenters.
- **User Story 5**: Keyboard utilities and tests are independent; responsive shell completion follows US1.

### Within Each User Story

- Write the listed tests first and confirm they fail for the intended missing behavior.
- Build isolated feature components and composables before route/shell integration.
- Use generated API models and the operation registry before submitting mutations.
- Treat accepted HTTP responses as task acceptance, not operation completion.
- Complete the independent test before moving the story checkpoint to done.

## Parallel Opportunities

- Setup styling tokens, directory exports, and test configuration have disjoint ownership.
- Foundational types, test factories/mocks, placement logic, and problem presentation can run in parallel.
- Component, integration, and E2E test files within each story can be authored in parallel.
- After Foundational, US1 through US4 feature components can be implemented by separate owners.
- US5 keyboard/semantics work can proceed while US1 shell and topology behavior is stabilized.

## Parallel Example: User Story 1

```text
Task T019: shell component tests in web/src/components/shell/LaboratoryShell.test.ts
Task T020: topology canvas tests in web/src/features/topology/TopologyCanvas.test.ts
Task T021: device palette tests in web/src/features/topology/DevicePalette.test.ts
Task T022: inspector tests in web/src/features/topology/TopologyInspector.test.ts
Task T023: real-service integration tests in tests/integration/frontend_topology_workflow_test.go
Task T024: Playwright workflow in tests/e2e/frontend_topology_workspace.spec.ts
```

## Parallel Example: User Story 3

```text
Task T054: console renderer host in web/src/features/diagnostics/ConsoleWorkspace.vue
Task T055: capture panel in web/src/features/diagnostics/CapturePanel.vue
Task T056: Traffic Filter form in web/src/features/diagnostics/TrafficFilterPanel.vue
Task T059: diagnostic API helpers in web/src/api/diagnostics.ts
```

## Implementation Strategy

### MVP First

1. Complete Setup and Foundational.
2. Complete User Story 1 only.
3. Run T019–T024 and the Workspace MVP section of `quickstart.md`.
4. Demonstrate the topology workspace before adding operational and diagnostic depth.

### Incremental Delivery

1. **US1**: Recognizable topology construction and live link workflow.
2. **US2**: Complete node operations, durable tasks, failures, and resource charts.
3. **US3**: Consoles, packet capture, and Traffic Filter visualization.
4. **US4**: Template/image transparency and automation parity.
5. **US5**: Accessibility, responsive behavior, and measured performance.
6. **Polish**: Hygiene, full validation, and privileged host acceptance.

### Parallel Team Strategy

After Phase 2, assign disjoint owners to topology shell/canvas, node operations/tasks, diagnostics, and
templates/automation. Keep shared edits to `LaboratoryShell.vue`, `TopologyWorkspace.vue`, and generated API
types serialized through a single integrator to avoid merge conflicts.

## Notes

- `[P]` tasks use disjoint files or directories and have no incomplete dependency.
- Browser-local visual layout never becomes authoritative laboratory state.
- Specialized ECharts, xterm, noVNC, and packet streaming components are not replaced by generic UI widgets.
- Do not add credentials, proprietary images, bootstrap secrets, packet payloads, or sensitive traces to source.
- Do not implement a UI workaround for a missing backend contract; record and fix the contract first.

## Phase 9: Convergence

- [X] T086 Add a revision-safe laboratory rename dialog wired to the existing authoritative `updateLab` operation, preserving entered values on conflicts and refreshing shared state after success per FR-003 (missing)
- [X] T087 Integrate live template and compatible image-version selection into the actual node-creation workflow, revalidate the server catalog before submission, explain unavailable versions, and remove the disconnected picker path per FR-004, FR-020, US1/AC1, and US4/AC1 (partial)
- [X] T088 Implement real ECharts viewport and zoom persistence plus user-facing topology grouping and editable local link-route controls without writing visual preferences to laboratory state per FR-006 (partial)
- [X] T089 Add direct topology port/edge interactions for connect, reconnect, and disconnect operations with pre-submission compatibility and invalid-target explanations per FR-007 (partial)
- [X] T090 Add authoritative capture and Traffic Filter discovery, event convergence, refresh rejoin, and control-service restart recovery; if list/discovery contracts are absent, record and fix the backend contract before hydrating the UI per FR-015 and SC-012 (partial)
- [X] T091 Enable capture workflows from selected links as well as node interfaces and present elapsed time, durable task state, quota, truncation, retention, stream, artifact, completion, and stop state through terminal convergence per FR-017, US3/AC2, and SC-007 (partial)
- [X] T092 Correct Traffic Filter request mapping to the declared source/destination address and port fields and complete the ECharts path view with topology-coordinate joining, timing, confidence, duplicate, loop, and ambiguity semantics per FR-018 and US3/AC3 (contradicts)
- [X] T093 Expand the task center with laboratory, resource, operation-kind, state, and time filters plus reliable navigation when a resource is pending, deleted, or outside the active laboratory per FR-022 (partial)
- [X] T094 Add field-level validation, explicit units and dependency rules, unsaved-change protection, server-problem-to-field mapping, duplicate-submit guards, and revision-conflict merge/refresh/retry flows that preserve user input per FR-011 and FR-023 (partial)
- [X] T095 Replace direct general-purpose native buttons, textareas, tables, selects, progress elements, menus, alerts, and notifications in feature components with shared Shadcn/ui primitives while retaining dedicated topology, terminal, VNC, and packet renderers per FR-023A (contradicts)
- [X] T096 Add confirmation dialogs for node, link, network-object, and other destructive or high-impact actions that identify affected resources, cleanup expectations, running-node impact, and active-stream interruption before submission per FR-024 (missing)
- [X] T097 Replace CSS/native task-progress and capture-volume graphics with modular ECharts visualizations using shared legends, tooltips, selection, zoom behavior, and empty/degraded states per FR-019 (contradicts)
- [X] T098 Implement keyboard topology traversal, selection, multi-selection, inspector/action invocation, and visible non-color focus/state equivalents, then validate primary workflows with keyboard-only tests per FR-025 and US5/AC1 (partial)
- [X] T099 Expand controlled component tests to cover success, loading, empty, stale, unsupported, permission, quota, conflict, retryable failure, terminal failure, reconnect, cancellation, cleanup, and renderer-disposal variants for every reusable workflow per FR-029 and SC-008 (partial)
- [X] T100 Add actual-backend Playwright workflows that create and delete laboratories through the UI, build mixed QEMU/Docker/lightweight topologies, select template/images, connect and rewire links, operate nodes and tasks, launch consoles, run captures and Traffic Filters, observe automation-created state, and verify destructive cleanup at 1024×768 and 1920×1080 per FR-031, SC-010, and SC-013 (partial)
- [X] T101 Replace the mocked single-mount scale check with deterministic 100-node/300-link ECharts benchmarks measuring 95th-percentile pan, zoom, selection, status-update, and inspector latency plus repeated mount/unmount observer, chart, socket, and renderer leak assertions per SC-006 (partial)
- [X] T102 Add closable, reconnectable, resizable multi-session Telnet/VNC tabs or panels with session switching and explicit reconnect-required state after refresh or service interruption per FR-016 (partial)
- [X] T103 Complete durable task presentation with started/finished timestamps, terminal result payloads, explicit cancellation availability, safe replay controls, related resource identity, and structured terminal errors without creating UI-only tasks per FR-010 (partial)
- [X] T104 Add distinct reusable presentations and clear next actions for empty, loading, stale, reconnecting, unsupported, permission, quota, conflict, partial-failure, cleanup, and terminal-error states across the workspace per FR-028 (partial)
