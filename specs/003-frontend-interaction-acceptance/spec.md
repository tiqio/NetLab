# Feature Specification: Frontend Interaction Acceptance

**Feature Branch**: `[003-frontend-interaction-acceptance]`

**Created**: 2026-07-27

**Status**: Draft

**Input**: User description: "能不能去前端去进行一系列点击测试，不要弄成半吊子点击也没有效果"

## Clarifications

### Session 2026-07-27

- Q: 真实运行环境应覆盖哪些前端功能？ → A: 所有受支持的前端功能都必须在 `10.72.1.7` 上使用真实 QEMU、Docker、轻量节点、Console、抓包、Traffic Filter、接线和清理流程进行验收。
- Q: 验收运行失败后如何处理其创建的运行资源？ → A: 无论成功或失败都必须自动尝试清理当前运行拥有的全部资源，仅保留脱敏的元数据和验收证据。
- Q: 设备模板与镜像版本应采用什么验收覆盖策略？ → A: 每类运行时和设备家族至少选择一个代表版本完成全流程；其余所有可用版本必须完成创建、启动、基本连通和删除测试。
- Q: Wireshark 前端验收应达到什么程度？ → A: 必须验证真实抓包、实时数据流和 Wireshark 交接信息；环境允许自动化时还必须启动 Wireshark，否则明确记录跳过原因。
- Q: 缺失或不可用的能力应如何影响验收结果？ → A: 产品声明支持的能力缺失或不可用时验收必须失败；只有明确可选且验收环境确实不具备的能力才能记录具体原因后跳过。

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Complete Real User Journeys (Priority: P1)

As a network-lab operator, I want every primary frontend workflow to be exercised through visible controls
against the real service so that I can trust that clicking buttons, selecting menu items, submitting forms,
and manipulating the topology produces the intended result rather than a silent no-op.

**Why this priority**: A polished interface has no value if its controls are disconnected from authoritative
state. Verifying complete journeys is the minimum confidence required before users build real laboratories.

**Independent Test**: Start from an empty system and complete laboratory creation, rename, device creation,
topology connection, lifecycle operation, task observation, diagnostics, and deletion exclusively through the
frontend, verifying authoritative state after every action and leaving the system clean.

**Acceptance Scenarios**:

1. **Given** an empty system, **When** the operator creates and renames a laboratory through the frontend,
   **Then** the new name appears consistently in the laboratory selector, open-laboratory area, refreshed
   browser state, and authoritative shared state.
2. **Given** a laboratory and available templates, **When** the operator selects a device template, chooses a
   compatible version and image, enters valid values, and submits, **Then** the dialog closes only after the
   request is accepted, the durable operation is visible, and the created resource appears in the topology.
3. **Given** at least two compatible endpoints, **When** the operator connects, disconnects, and reconnects
   them through canvas and inspector controls, **Then** each operation reaches a terminal result and the
   topology reflects the same final connection state after refresh.
4. **Given** an operable node, **When** the operator starts, stops, edits, or deletes it through visible
   controls, **Then** the control shows acceptance or failure feedback, the related task can be followed to a
   terminal state, and the node's observed state converges accordingly.
5. **Given** a completed test laboratory, **When** the operator confirms deletion, **Then** the laboratory and
   all owned resources disappear, active streams are closed or reported interrupted, and no owned runtime
   resources remain.

---

### User Story 2 - No Silent or Misleading Controls (Priority: P1)

As an operator, I want every visible interactive control to have an intentional, testable outcome so that I
never click something that appears usable but does nothing, changes only temporary presentation without
explanation, or reports success before the service accepts the action.

**Why this priority**: Silent no-ops and false-positive success messages create operational risk and destroy
trust faster than an explicit unsupported or disabled state.

**Independent Test**: Build an inventory of all visible buttons, menu items, tabs, links, selectors, form
submissions, topology gestures, keyboard commands, and chart interactions for each applicable state; activate
each one and verify one of the allowed outcomes.

**Acceptance Scenarios**:

1. **Given** an enabled visible control, **When** it is activated by pointer, **Then** it produces a visible
   state change, navigation, dialog, authoritative operation, download/stream, or explicit error within the
   defined response window.
2. **Given** the same control is keyboard reachable, **When** it is activated by keyboard, **Then** it produces
   the equivalent outcome and retains a visible focus indication.
3. **Given** a control cannot operate in the current state, **When** the operator encounters it, **Then** it is
   disabled or marked unavailable and an adjacent or discoverable explanation identifies the missing
   prerequisite; it MUST NOT appear enabled and silently ignore activation.
4. **Given** a long-running action, **When** the service accepts it, **Then** the frontend shows the durable task
   identity and progress or pending state instead of claiming immediate completion.
5. **Given** an action fails, conflicts, times out, or partially cleans up, **When** the failure is returned,
   **Then** the frontend presents the authoritative problem, preserves recoverable input, and offers the next
   valid action without fabricating success.

---

### User Story 3 - Exercise Diagnostics and Interactive Workspaces (Priority: P2)

As an operator, I want consoles, packet capture, Traffic Filter, topology manipulation, task navigation, and
local workspace controls tested through realistic interaction sequences so that advanced functions are usable
beyond simply opening their panels.

**Why this priority**: These workflows contain multiple asynchronous and stateful transitions where shallow
tests frequently miss broken controls, stale identifiers, reconnect failures, and incomplete cleanup.

**Independent Test**: Use a laboratory with suitable running resources to open and switch console sessions,
start and stop interface and link captures, run a Traffic Filter, manipulate topology layout and connections,
navigate from tasks to resources, refresh the page, and verify session recovery or explicit reconnect state.

**Acceptance Scenarios**:

1. **Given** a node with supported console modes, **When** the operator opens multiple console sessions,
   switches between them, reconnects, resizes, and closes them, **Then** every control updates the correct
   session and closing a session does not change node lifecycle state.
2. **Given** a valid interface or link, **When** the operator starts a capture, opens the live stream or
   retained artifact, refreshes status, and stops capture, **Then** quota, packets, bytes, retention,
   truncation, completion, task, and stop state remain consistent with authoritative metadata.
3. **Given** observable traffic, **When** the operator starts a Traffic Filter and refreshes or rejoins it,
   **Then** matching path observations, timing, confidence, duplicate/loop/ambiguity indicators, and terminal
   state are visible and correspond to the selected scope.
4. **Given** a topology with multiple resources, **When** the operator pans, zooms, moves, groups, selects,
   multi-selects, routes links, connects ports, and uses keyboard traversal, **Then** every interaction produces
   the intended visible result and browser-local preferences survive refresh without altering shared topology.
5. **Given** a task related to a resource, **When** the operator filters tasks, opens its resource, cancels an
   eligible task, or re-reads terminal state, **Then** the task center shows the correct resource identity,
   cancellation availability, timestamps, result, or structured error.

---

### User Story 4 - Repeatable Regression Evidence (Priority: P2)

As a maintainer, I want interaction acceptance runs to produce repeatable, sanitized evidence and actionable
failure reports so that regressions identify the exact control, precondition, expected result, actual result,
and cleanup status.

**Why this priority**: Broad clicking without deterministic assertions only creates false confidence. Failures
must be reproducible and must not leave resources that contaminate later runs.

**Independent Test**: Run the complete interaction suite twice from a clean baseline at both supported
viewports and compare the resulting interaction inventory, terminal outcomes, error report, and cleanup audit.

**Acceptance Scenarios**:

1. **Given** an interaction acceptance run, **When** any action fails, **Then** the result identifies the page,
   control label, resource, precondition, action, expected outcome, actual outcome, and cleanup performed.
2. **Given** a completed run, **When** evidence is reviewed, **Then** it contains no credentials, bootstrap
   secrets, console payloads, packet payloads, proprietary image content, or other sensitive artifacts.
3. **Given** a failed or interrupted run, **When** cleanup executes, **Then** created laboratories and owned
   runtime resources are removed or explicitly listed with a safe remediation action.
4. **Given** the suite is rerun without product changes, **When** the same controls and capabilities are
   available, **Then** the interaction inventory and pass/fail outcomes are deterministic.

### Edge Cases

- A control is visible but covered, off-screen, clipped, or unreachable at the minimum supported viewport.
- A control changes label or moves after an asynchronous event while the operator is about to activate it.
- Double-clicking or rapidly submitting the same action would otherwise create duplicate resources.
- A selected template version or image becomes unavailable between opening a form and submitting it.
- Another browser or automation client changes the same laboratory revision during form editing.
- A durable task remains queued, is cancelled, fails before starting, fails during cleanup, or completes after
  the operator navigates away.
- A resource is deleted before task-to-resource navigation occurs.
- The event stream disconnects, skips events, or reconnects while an action is pending.
- A console, capture stream, or Traffic Filter is interrupted by refresh or service restart.
- A browser-local preference contains stale IDs for resources that no longer exist.
- The backend returns an empty collection as null, empty, or temporarily unavailable.
- A download, stream, or external tool action is blocked by browser policy or unavailable on the operator's
  workstation.
- A destructive confirmation is cancelled, confirmed once, or activated repeatedly.
- A control is intentionally unsupported for the selected runtime or device version.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The product MUST maintain an interaction inventory covering every user-visible button, menu
  item, tab, link, selector, form submission, dialog action, topology gesture, keyboard command, chart action,
  console action, stream action, and download action in each applicable workspace state.
- **FR-002**: Every enabled interaction in the inventory MUST have at least one complete acceptance scenario
  that begins from a declared precondition, activates the control through the frontend, and verifies the
  resulting user-visible and authoritative state.
- **FR-003**: A test MUST fail if an enabled control produces no observable result, only changes an unrelated
  element, reports success without authoritative acceptance, or requires an undocumented second action.
- **FR-004**: Controls that cannot operate in the current state MUST be disabled or explicitly marked
  unavailable and MUST provide a discoverable explanation of the required prerequisite or unsupported
  capability.
- **FR-005**: The acceptance suite MUST distinguish local presentation outcomes from shared authoritative
  mutations and verify each against its correct source of truth.
- **FR-006**: Primary mutation journeys MUST verify durable task acceptance, task identity, progress or pending
  state, legal terminal state, resulting resource state, and any structured error or cleanup outcome.
- **FR-007**: The suite MUST exercise laboratory create, select, rename, duplicate, import/export entry,
  refresh, and delete controls, including cancellation and revision-conflict behavior where applicable.
- **FR-008**: The suite MUST exercise template browsing, image/version selection, availability explanations,
  stale-catalog revalidation, valid node creation, invalid submission, and duplicate-submit prevention. For
  each supported runtime and device family, at least one representative available version MUST complete the
  full applicable frontend journey; every other available version MUST complete creation, startup, basic
  connectivity, and deletion through the frontend.
- **FR-009**: The suite MUST exercise topology selection, multi-selection, port selection, connect,
  disconnect, reconnect, move, pan, zoom, grouping, local link routing, inspector opening, clearing selection,
  and keyboard traversal.
- **FR-010**: The suite MUST exercise node lifecycle, resource editing, interface add/remove, guest command,
  port mapping, network-object creation/attachment/diagnostics, and destructive cleanup controls for every
  supported resource kind available in the test environment.
- **FR-011**: The suite MUST exercise task filtering by laboratory scope, resource, operation kind, state,
  time, and text, plus resource navigation, eligible cancellation, terminal replay, timestamps, results, and
  structured errors.
- **FR-012**: The suite MUST exercise Telnet and VNC session creation, switching, reconnect, resize, close,
  refresh interruption, and unsupported-mode behavior without changing node lifecycle state.
- **FR-013**: The suite MUST exercise a real capture start, discovery, refresh, live packet stream, retained
  artifact, Wireshark handoff information, quota, truncation, stop, completion, failure, and cleanup from both
  interface and link scopes when supported. When the acceptance environment permits safe desktop automation,
  the suite MUST launch Wireshark through the frontend handoff and verify that it receives the capture; when
  it does not, the run MUST record the concrete environmental reason as an explicit skip.
- **FR-014**: The suite MUST exercise Traffic Filter input mapping, scope selection, start, discovery, refresh,
  path observations, timing, confidence, duplicate, loop, ambiguity, stop, failure, and restart recovery.
- **FR-015**: Each destructive or high-impact action MUST be tested for both cancel and confirm paths and MUST
  identify the affected resource, expected cleanup, running-resource impact, and stream interruption before
  submission.
- **FR-016**: Forms MUST be tested for required fields, valid boundaries, explicit units, dependency rules,
  server field errors, unsaved-change protection, duplicate submissions, revision conflicts, refresh, merge,
  retry, and preservation of recoverable user input.
- **FR-017**: Every primary journey MUST be tested by pointer and keyboard at 1920×1080 and 1024×768, with
  visible focus, reachable controls, and no clipped confirmation or required action.
- **FR-018**: Tests MUST use the deployed service on `10.72.1.7` for acceptance journeys and MUST exercise
  every supported frontend capability with real QEMU, Docker, lightweight-node, console, capture, Traffic
  Filter, topology connection, and cleanup resources. Controlled substitutes MAY be used only for isolated
  component state matrices and MUST NOT be the sole evidence that a visible control works.
- **FR-019**: Frontend actions, HTTP automation, and applicable MCP actions MUST converge to the same shared
  final state when they perform equivalent operations; browser-local presentation preferences MUST remain
  independent.
- **FR-020**: Concurrent-client scenarios MUST cover an action made in one browser or automation client being
  observed by another browser, including revision conflict, ordered convergence, reconnect, and deleted
  resource handling.
- **FR-021**: Long-running acceptance actions MUST have bounded waits, explicit timeout failures, safe retry or
  cancellation behavior, and MUST NOT be treated as passed merely because a request was accepted.
- **FR-022**: Each run MUST use identifiable synthetic resources and MUST automatically attempt to remove all
  resources it creates after either success or failure, including runtime processes, interfaces, namespaces,
  links, mappings, captures, retained artifacts, and temporary files. Failed resources MUST NOT be preserved
  for manual debugging as the default or an acceptance-run option.
- **FR-023**: If automatic cleanup cannot complete, the run MUST fail and report every remaining owned
  resource with a remediation action; later runs MUST NOT silently reuse contaminated state, and the retained
  failure record MUST NOT itself require preserving runtime resources.
- **FR-024**: Evidence MUST record interaction identity, precondition, action, expected outcome, actual
  outcome, terminal resource/task state, duration, viewport, and cleanup status without retaining sensitive
  content. After cleanup, only sanitized metadata and acceptance evidence MAY remain from the run.
- **FR-025**: The suite MUST fail when a product-declared supported capability is missing, unavailable, or
  nonfunctional. A control MAY be skipped only when its capability is explicitly optional and the acceptance
  environment demonstrably lacks the prerequisite; every permitted skip MUST record the concrete capability
  and environmental reason, and every unreported or ineligible skip MUST count as a failure.
- **FR-026**: A visible control MAY be accepted as intentionally presentation-only only when its expected local
  effect is documented and asserted; silent no-op behavior is never an allowed outcome.
- **FR-027**: Empty, loading, stale, reconnecting, unsupported, permission, quota, conflict, partial-failure,
  cleanup, cancellation, retryable failure, terminal failure, and success states MUST each have at least one
  interaction acceptance scenario where the state is applicable.
- **FR-028**: The suite MUST verify that refreshing the page after each major mutation retains authoritative
  results and either rejoins transient sessions or presents an explicit reconnect-required state.
- **FR-029**: The suite MUST run from a clean baseline and MUST verify the clean baseline again after completion.

### Key Entities

- **Interaction Inventory Item**: A user-visible control or gesture, its location, applicable states,
  prerequisites, allowed outcomes, keyboard equivalent, and owning user journey.
- **Acceptance Journey**: An ordered sequence of frontend actions and assertions that delivers a complete
  operator outcome rather than testing an isolated click.
- **Expected Outcome**: The permitted visible, navigation, local-preference, artifact, stream, durable-task, or
  authoritative-state result of an interaction.
- **Acceptance Run**: One execution identified by start time, environment, viewport, capability set, created
  resources, interaction results, and final cleanup status.
- **Interaction Result**: Pass, fail, or explicitly skipped status with duration, evidence, expected result,
  actual result, and related resource/task identities.
- **Cleanup Record**: The set of resources created by a run, their terminal cleanup state, remaining leaks,
  and remediation guidance.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of enabled controls in the interaction inventory produce their documented observable
  outcome; zero enabled controls are accepted as silent no-ops.
- **SC-002**: 100% of primary mutation journeys verify both durable terminal task state and refreshed final
  resource state, not only request acceptance.
- **SC-003**: The complete acceptance suite passes at both 1920×1080 and 1024×768 with all required controls
  visible, reachable, and keyboard operable.
- **SC-004**: At least one complete run on `10.72.1.7` covers laboratory lifecycle, real QEMU, Docker and
  lightweight topology creation, connection and rewiring, node operation, task navigation, Telnet/VNC
  console, interface/link capture, Traffic Filter, automation convergence, and destructive cleanup through
  the frontend.
- **SC-005**: 100% of disabled or unavailable controls tested provide a discoverable reason; no unavailable
  control appears enabled without producing feedback.
- **SC-006**: 95% of non-long-running interactions present visible feedback within 500 milliseconds, and all
  accepted long-running interactions expose a task or pending identity within 2 seconds.
- **SC-007**: Every accepted long-running operation reaches a verified terminal state within its declared
  timeout or produces a failure report containing timeout, cancellation, and cleanup status.
- **SC-008**: Two browsers and one automation client observe the same authoritative final state within 5
  seconds after each tested shared mutation, while browser-local layout preferences remain independent.
- **SC-009**: Two consecutive runs from the same clean baseline produce the same interaction inventory and no
  unexplained pass/fail differences.
- **SC-010**: After every successful or failed run, zero run-owned laboratories, processes, containers,
  virtual machines, interfaces, namespaces, links, mappings, captures, artifacts, or temporary files remain;
  only sanitized metadata and acceptance evidence remain.
- **SC-011**: Failure evidence identifies the exact control and reproducible action for 100% of failed
  interactions while containing zero credentials, secrets, console payloads, packet payloads, or proprietary
  image content.
- **SC-012**: A representative operator can complete the primary create-connect-operate-diagnose-delete
  journey without encountering an unexplained control or requiring undocumented recovery steps.
- **SC-013**: Every supported runtime and device family has at least one available version that passes its
  complete applicable frontend journey, and 100% of the remaining available versions pass frontend-driven
  creation, startup, basic connectivity, and deletion.
- **SC-014**: Every complete real-environment run verifies non-empty live capture data and usable Wireshark
  handoff information; desktop Wireshark is launched and receives the capture whenever safe automation is
  available, or the result contains an explicit environmental skip reason.
- **SC-015**: 100% of product-declared supported capabilities execute their required acceptance coverage with
  no skips; 100% of permitted skips identify an explicitly optional capability and a verified missing
  environmental prerequisite.

## Assumptions

- The target acceptance environment is the deployed single-host NetLab service on `10.72.1.7` and begins
  without user-owned laboratories; synthetic test laboratories may be created and deleted by the suite.
- Available templates and license-safe operator images determine which device-specific journeys run. The
  representative full-journey version for each supported runtime and device family is recorded in the run
  evidence; all other available versions receive the required lifecycle and connectivity coverage. A missing
  product-declared supported capability fails acceptance; only explicitly optional capabilities with verified
  missing environmental prerequisites are reported as skips rather than simulated as passing.
- Destructive tests operate only on resources created by the current acceptance run.
- Failed acceptance runs are diagnosed from sanitized evidence and ownership reports rather than by retaining
  their laboratories or runtime resources.
- External Wireshark launch is required when it can be automated safely; otherwise the handoff information,
  non-empty live stream, and explicit environmental skip reason provide the required evidence. Desktop-launch
  automation is an explicitly optional environment capability; capture, live-stream, and handoff behavior are
  product-declared supported capabilities and cannot be skipped.
- The trusted deployment has no account/password workflow, so authentication controls are outside this
  feature's interaction inventory.
- Visual hover decoration without an action is not considered an interactive control; focusable, clickable,
  selectable, draggable, dismissible, or submit-capable elements are included.

## Scope Boundaries

- This feature validates and completes existing frontend interactions; it does not add new device families,
  multi-host clustering, account authentication, or Cisco support.
- It does not treat direct backend requests as substitutes for frontend journeys, except to create controlled
  concurrent changes or verify authoritative final state.
- It does not require retaining packet payloads, console transcripts, proprietary images, credentials, or
  secrets as test evidence.
- It does not permit skipping a broken interaction by hiding the control unless the product requirement itself
  declares the capability unsupported and communicates that state to the operator.
