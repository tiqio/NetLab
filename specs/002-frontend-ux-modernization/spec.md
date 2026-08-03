# Feature Specification: Frontend UX Modernization

**Feature Branch**: `[002-frontend-ux-modernization]`

**Created**: 2026-07-27

**Status**: Draft

**Input**: User description: "完善前端功能与界面，采用 Shadcn/ui 组件库，进行完善的前端测试以正确对接后端逻辑，整体风格和布局借鉴 EVE-NG 以降低认知负担，图相关逻辑采用 ECharts.js。"

## Clarifications

### Session 2026-07-27

- Q: ECharts 应覆盖哪些图形化视图？ → A: 拓扑画布、Traffic Filter 路径、资源趋势及统计图全部使用 ECharts。
- Q: 默认工作区采用什么布局？ → A: 顶部工具栏、左侧设备栏、中央拓扑、右侧属性栏和底部任务/Console 抽屉；侧栏与底栏可折叠并调整尺寸。
- Q: Shadcn/ui 应覆盖哪些界面？ → A: 所有通用界面组件统一迁移；拓扑画布、终端及其他专业渲染区域保留专用实现。
- Q: 前端自动化测试应连接什么环境？ → A: 组件测试使用受控 mock；契约与集成测试连接真实测试服务；关键端到端流程连接实际后端，特权场景在 10.72.1.7 验证。
- Q: 拓扑位置和视图状态如何保存？ → A: 全部保存在当前浏览器，不作为实验室共享状态，也不影响其他浏览器或自动化客户端。

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Build and Operate a Topology Visually (Priority: P1)

As a network laboratory user, I want a familiar topology workspace with a device palette, central
canvas, property inspector, and operation status area so that I can create, connect, start, stop, and
inspect nodes without learning a novel interaction model.

**Why this priority**: Topology construction and operation are the primary product workflows. If this
workspace is unclear or incomplete, every runtime capability becomes difficult to use.

**Independent Test**: Starting from an empty laboratory, a user can add QEMU, Docker, and lightweight
nodes, connect them, change a connection while nodes are running, start the nodes, inspect their state,
and delete the laboratory using only the graphical workspace.

**Acceptance Scenarios**:

1. **Given** an empty laboratory, **When** the user selects or drags a supported node type onto the
   canvas, chooses a valid template version, and confirms, **Then** the node appears at the selected
   position with its authoritative lifecycle state.
2. **Given** two nodes with available interfaces, **When** the user connects the interfaces, **Then**
   the connection appears immediately as pending and converges to the shared backend-observed state.
3. **Given** a running topology, **When** the user selects a node or link, **Then** the inspector shows
   only valid actions, current configuration, runtime status, revision-sensitive controls, and any
   actionable failure information.
4. **Given** a topology changed by an automation client or second browser, **When** the shared event is
   received, **Then** the canvas resources, inspector, and operation history update without requiring a
   reload while the current browser retains its own local placement and viewport preferences.
5. **Given** a user familiar with EVE-NG, **When** the user first opens the workspace, **Then** the
   placement of the topology, device palette, node actions, console access, and laboratory controls is
   recognizable without copying EVE-NG code, branding, or proprietary visual assets.

---

### User Story 2 - Operate Nodes with Clear Feedback (Priority: P1)

As an operator, I want node lifecycle, interface, resource, guest-command, and port-mapping operations
to provide consistent progress and errors so that I can distinguish accepted work, completed work,
retryable failures, and cleanup requirements.

**Why this priority**: Runtime operations are asynchronous and may partially fail. The interface must
not imply success before the backend reaches a terminal result.

**Independent Test**: A user can start and stop a node, hot-add and remove an interface, execute a guest
command, create and delete a port mapping, observe CPU and memory status, cancel a cancellable task, and
recover from an injected backend error with the prescribed action.

**Acceptance Scenarios**:

1. **Given** a valid lifecycle operation, **When** it is submitted, **Then** the interface displays the
   durable task identifier, progress, resource identity, and eventual terminal result.
2. **Given** a revision conflict, unavailable runtime, resource limit, or partial cleanup failure,
   **When** the backend rejects or fails the operation, **Then** the user sees the lifecycle phase,
   retryability, cleanup status, recommended action, and retry timing without losing current context.
3. **Given** an operation is still running, **When** the user navigates between laboratory panels,
   **Then** its status remains visible and no duplicate mutation is submitted accidentally.
4. **Given** a control-service restart, **When** the user reconnects, **Then** the interface restores
   shared node and task state and clearly identifies streams or sessions that require reconnection.

---

### User Story 3 - Use Consoles and Network Diagnostics (Priority: P1)

As a network engineer, I want consoles, packet capture, traffic-path analysis, and resource diagnostics
inside the laboratory workspace so that troubleshooting does not require unrelated tools or pages.

**Why this priority**: Console and packet visibility are core network-simulation workflows and must be
as accessible as topology editing.

**Independent Test**: From a running node or link, a user can open a Telnet or VNC console, start and
stop a capture, stream or download its artifact, define a traffic match, and see matching interfaces
and links highlighted on the topology.

**Acceptance Scenarios**:

1. **Given** a running node, **When** the user opens a supported console, **Then** it appears in a
   closable, reconnectable workspace panel without changing node lifecycle state.
2. **Given** a node interface or link, **When** the user starts a capture, **Then** elapsed time, bytes,
   truncation, retention, stream availability, and stop controls remain visible until terminal state.
3. **Given** a valid traffic characteristic, **When** matching packets are observed, **Then** the
   topology highlights every observed interface and link with counts, direction, timing, and ambiguity.
4. **Given** capture or console connectivity is interrupted, **When** the user retries, **Then** the
   session reconnects or reports a structured terminal reason without restarting the node.

---

### User Story 4 - Manage Templates, Images, and Automation (Priority: P2)

As an administrator or automation user, I want template versions, image availability, tasks, audit
events, and automation entry points presented consistently so that graphical and automated workflows
operate on the same shared resources.

**Why this priority**: These workflows are less frequent than topology operation but are required to
prepare devices and understand automated changes.

**Independent Test**: A user can review templates and legal image versions, import a supported image,
create a node from the selected version, inspect automation-created tasks, and confirm that the same
resources and errors are visible through the graphical interface.

**Acceptance Scenarios**:

1. **Given** available and unavailable image versions, **When** a template is selected, **Then** only
   compatible selectable versions are enabled and unavailable versions show a reason.
2. **Given** an image import, **When** validation completes, **Then** provenance, version, digest,
   license notes, availability, and validation outcome are visible without exposing secrets.
3. **Given** a topology mutation created through automation, **When** the user opens the laboratory or
   task center, **Then** the same resource, task status, result, and failure details are shown.

---

### User Story 5 - Work Efficiently Across Screen Sizes and Input Methods (Priority: P2)

As a user, I want the workspace to remain understandable with keyboard, mouse, and different desktop
screen sizes so that routine operations are fast and accessible.

**Why this priority**: A dense network workspace must remain usable without hiding critical state or
requiring precise pointer-only interactions.

**Independent Test**: Core topology, node operation, task, and diagnostic workflows can be completed
with keyboard navigation at common desktop widths and at the minimum supported width.

**Acceptance Scenarios**:

1. **Given** keyboard-only input, **When** the user navigates the application, **Then** all primary
   commands, dialogs, tabs, menus, forms, and topology selections have visible focus and descriptive labels.
2. **Given** a narrower viewport, **When** the workspace can no longer show all panels, **Then**
   secondary panels become drawers or tabs while the canvas and active operation remain usable.
3. **Given** color-vision differences, **When** lifecycle or traffic states are displayed, **Then** text,
   iconography, shape, or pattern communicates the state independently of color.

### Edge Cases

- The laboratory contains no nodes, hundreds of interfaces, overlapping links, or labels longer than
  the available canvas area.
- The selected resource is deleted or revised by another client while its inspector or dialog is open.
- A node created by another client has no local saved position and must receive a deterministic local
  placement without changing any other client's layout.
- An event replay gap occurs and the client must refresh the authoritative laboratory snapshot.
- A task is accepted but its resource is not yet visible, or a resource is visible while its task is
  still reconciling.
- A node exposes only Telnet, only VNC, both console modes, or temporarily no reachable console.
- A capture reaches time, byte, retention, or global quota limits while the user is viewing its stream.
- A traffic match observes loops, duplicated packets, multiple valid paths, or no packets before timeout.
- Template or image data changes while a node-creation dialog is open.
- Browser refresh or temporary network loss occurs during a mutation, console, capture, or task update.
- The browser cannot provide a requested rendering or streaming capability; the interface must provide
  a clear fallback or unsupported-state message.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The default laboratory workspace MUST use a fixed shell with a top toolbar, left device
  palette, central topology canvas, right context-sensitive inspector, and bottom task/console drawer.
  The left, right, and bottom regions MUST be collapsible and resizable without obscuring active state.
- **FR-002**: The workspace MUST use interaction placement and terminology familiar to EVE-NG users
  where doing so does not conflict with NetLab behavior, and MUST retain distinct NetLab branding and assets.
- **FR-003**: Users MUST be able to create, select, rename, duplicate, export, import, and delete
  laboratories from the graphical interface with confirmation for destructive actions.
- **FR-004**: Users MUST be able to add every supported QEMU, Docker, and lightweight node type from a
  searchable, categorized palette and choose a compatible template and image version when required.
- **FR-005**: The topology view MUST use the mandated unified charting system to display nodes,
  interfaces, links, network objects, lifecycle states, selection, pending mutations, failures, and
  traffic-path observations without relying on color alone.
- **FR-006**: Users MUST be able to pan, zoom, fit, select, multi-select, reposition, and inspect topology
  items. Node coordinates, grouping, link routing, viewport, zoom, and panel layout MUST be stored only
  in the current browser and MUST NOT alter laboratory state or another client's layout.
- **FR-007**: Users MUST be able to create, disconnect, and change live connections through direct
  topology interactions, with interface compatibility and invalid targets explained before submission.
- **FR-008**: The interface MUST show desired state, observed state, revision, runtime identity when
  available, and the latest structured problem for each selected resource.
- **FR-009**: Node actions MUST cover start, stop, delete, interface hot-add/remove, NIC-driver selection,
  guest command execution, host-port mappings, and resource configuration where supported.
- **FR-010**: Every asynchronous mutation MUST surface its durable task status, progress, result, error,
  timestamps, cancellation availability, and related resource without manufacturing a separate UI-only task.
- **FR-011**: The interface MUST prevent accidental duplicate submissions while allowing safe replay of
  an interrupted mutation and MUST handle revision conflicts without discarding user-entered changes.
- **FR-012**: Structured failures MUST display resource identity, lifecycle phase, retryability, cleanup
  progress, operator action, and retry delay in a consistent, actionable presentation.
- **FR-013**: The client MUST consume the same authoritative resources, operations, and task results as
  external automation and MUST NOT introduce mutations available only through the graphical interface.
- **FR-014**: The client MUST subscribe to shared ordered updates, apply them within the active workspace,
  detect replay gaps, and refresh authoritative state when incremental convergence is unsafe.
- **FR-015**: Multiple browser and automation clients MUST observe the same final topology, lifecycle,
  capture, traffic-filter, and task resource state without account-specific state isolation. Browser-local
  visual placement and viewport preferences are explicitly excluded from shared-state convergence.
- **FR-016**: Users MUST be able to open supported Telnet and VNC consoles in reconnectable panels or
  tabs, resize them, switch between sessions, and close them without changing node lifecycle state.
- **FR-017**: Users MUST be able to start, monitor, stop, stream, download, and inspect retained packet
  captures from node interfaces and links, including quota, truncation, retention, and completion status.
- **FR-018**: Users MUST be able to define supported traffic characteristics and see observed interfaces
  and links highlighted with packet count, direction, timing, confidence, and loop or ambiguity indicators.
- **FR-019**: The topology canvas, traffic-path visualization, time-series, distribution, resource,
  task-progress, capture-volume, and statistical views MUST all use ECharts and provide consistent
  interactive behavior with readable legends, tooltips, zoom, selection, and empty states.
- **FR-020**: Users MUST be able to view template capabilities, compatible image versions, image
  provenance, digest, license notes, validation outcome, and availability before creating a node.
- **FR-021**: The image workflow MUST not expose or persist credentials, bootstrap secrets, proprietary
  image contents, or captured traffic in browser storage, test fixtures, screenshots, or logs.
- **FR-022**: The task center MUST support filtering by laboratory, resource, operation kind, state, and
  time, and MUST retain enough context to navigate from a task to its resource and actionable error.
- **FR-023**: Forms and dialogs MUST provide field-level validation, safe defaults, explicit units,
  dependency-aware controls, unsaved-change protection, and server-error mapping to relevant fields.
- **FR-023A**: All general-purpose interface elements—including buttons, forms, menus, dialogs,
  drawers, tables, tabs, tooltips, alerts, and notifications—MUST use Shadcn/ui consistently. Specialized
  topology, terminal, packet-stream, and other domain renderers MAY use dedicated implementations while
  matching the shared design tokens and interaction conventions.
- **FR-024**: Destructive and high-impact operations MUST identify affected resources, expected cleanup,
  and whether running nodes or active streams will be interrupted before confirmation.
- **FR-025**: Primary workflows MUST be keyboard accessible, provide visible focus, descriptive labels,
  logical navigation order, and non-color state indicators.
- **FR-026**: The workspace MUST support common desktop widths and a minimum viewport of 1024 by 768.
  When space is constrained, side panels and the bottom drawer MUST collapse predictably while keeping
  topology operation, current task status, and active diagnostics accessible.
- **FR-027**: User interface preferences such as theme, panel sizes, open panels, topology viewport, and
  non-secret convenience settings MUST survive refresh without changing shared laboratory state.
- **FR-028**: Empty, loading, stale, reconnecting, unsupported, permission, quota, conflict, partial-failure,
  and terminal-error states MUST have distinct presentations and a clear next action.
- **FR-029**: Automated component tests MUST use controlled mocks to cover every reusable interaction
  pattern and all state variants used by topology, forms, tasks, consoles, captures, and errors.
- **FR-030**: Automated contract and integration tests MUST connect to a real test service and verify
  request payloads, response mapping, event updates, revision handling, task polling or streaming,
  cancellation, retries, and structured failures against the declared backend contracts.
- **FR-031**: Automated end-to-end tests MUST connect to an actual backend and exercise at least
  laboratory management, mixed-node topology creation, live linking, node lifecycle, task recovery,
  console launch, capture, traffic path, template/image selection, automation-created state, and
  destructive cleanup. Privileged runtime scenarios MUST be validated on `10.72.1.7`.
- **FR-032**: Test fixtures and simulated streams MUST preserve production contract shapes and MUST fail
  when the graphical client diverges from the backend schema or expected lifecycle semantics.
- **FR-033**: The redesigned interface MUST be deliverable without changing existing backend lifecycle,
  resource ownership, authentication boundary, or automation semantics unless a separately specified
  contract defect is discovered.

### Key Entities

- **Workspace Layout**: The browser-local fixed top/left/center/right/bottom laboratory arrangement,
  including panel collapsed state, resizable dimensions, active tabs, canvas viewport, node placement,
  link routing, task area, and diagnostics.
- **Topology View Item**: A visual representation of a node, network object, interface, link, pending
  mutation, failure, or observed traffic path associated with an authoritative resource identifier.
- **Operation Presentation**: The user-visible representation of a durable task, its related resource,
  progress, cancellation state, result, timestamps, and structured problem.
- **Diagnostic Session**: A console, capture, traffic filter, chart, or resource view with connection,
  streaming, terminal, reconnect, retention, and truncation state.
- **UI Preference**: Non-secret browser-local display preferences, including all topology placement and
  view state, that do not alter shared laboratory state or another client's layout.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: At least 90% of users familiar with EVE-NG can create a laboratory, add two nodes, connect
  them, and start them on their first attempt without documentation.
- **SC-002**: A trained user can create and start a mixed six-node topology in under 5 minutes using only
  the graphical interface.
- **SC-003**: Accepted topology and lifecycle changes become visible to two browser clients and one
  automation client within 3 seconds in at least 95% of test runs.
- **SC-004**: All supported node, link, network-object, capture, traffic-filter, image, and task operations
  exposed graphically produce the same final resource and error state as the corresponding automation flow.
- **SC-005**: Users can identify whether a failed operation is retryable and name the prescribed next
  action in at least 90% of usability-test failures.
- **SC-006**: The topology remains interactive with 100 visible nodes and 300 visible links, with pan,
  zoom, selection, and status updates completing within 1 second for at least 95% of interactions.
- **SC-007**: Console launch, capture start, and traffic-filter start can each be initiated from the
  selected topology resource in no more than three user actions.
- **SC-008**: Every primary workflow passes automated tests for success, loading, empty, conflict,
  retryable failure, terminal failure, reconnect, and cancellation states.
- **SC-009**: Contract validation detects every intentionally introduced mismatch in required request,
  response, task, event, console, capture, and structured-problem fixtures.
- **SC-010**: End-to-end tests complete the defined critical workflows at 1024×768 and 1920×1080 using
  keyboard and pointer input, with no inaccessible primary action or obscured terminal status.
- **SC-011**: No automated test artifact, browser storage record, screenshot, or client log contains a
  credential, cloud-init secret, proprietary image content, or captured payload.
- **SC-012**: After browser refresh, temporary network loss, or control-service restart, the client
  converges to authoritative shared state within 10 seconds and clearly marks sessions requiring reconnection.
- **SC-013**: Destructive cleanup initiated from the interface reaches the same terminal deletion state
  as automation, and no deleted resource remains displayed after authoritative confirmation.

## Assumptions

- The existing backend contracts and shared-state model remain authoritative; this feature primarily
  completes and reorganizes the graphical client rather than redefining runtime behavior.
- Shadcn/ui is mandatory for all general-purpose interface components. Specialized topology, terminal,
  packet-stream, and domain renderers remain dedicated components but must share the same visual tokens.
  ECharts is mandatory for every graphical visualization, including the topology canvas, traffic path,
  resource trends, and statistics.
- “Borrow from EVE-NG” means preserving familiar information architecture and interaction expectations,
  not producing a pixel-identical clone or copying code, branding, icons, stylesheets, or proprietary assets.
- The primary environment is a desktop browser on a trusted management network. Mobile-phone topology
  editing is outside scope, while narrower desktop and tablet-width viewing receives a usable fallback.
- No account or password interface is introduced. All topology coordinates, grouping, link routing,
  viewport, and panel preferences are browser-local; shared laboratory resource state continues to come
  exclusively from the server.
- English and Chinese content must fit the layout, but full localization management is outside this
  feature unless separately specified.
- Existing console, capture, event, task, topology, template, image, and automation contracts are
  available through controlled component fixtures, a real test service for contract/integration tests,
  and the designated acceptance host for privileged end-to-end validation.

## Dependencies

- Stable backend contracts for laboratories, topology snapshots, templates, images, nodes, links,
  network objects, tasks, events, consoles, captures, traffic filters, resources, and automation.
- Browser-accessible console, capture, and event streaming endpoints in the target deployment.
- Legally supplied images and existing device templates for workflows that require real runtime validation.
- A repeatable real-backend end-to-end environment capable of exercising shared clients, plus access to
  `10.72.1.7` for selected privileged scenarios without storing credentials or captured traffic in the repository.

## Out of Scope

- Changing supported runtime types, adding Cisco support, adding cluster scheduling, or adding accounts.
- Replacing or redesigning the backend desired-state, task, recovery, ownership, or API/MCP models.
- Copying EVE-NG source code, proprietary assets, branding, exact visual styling, or undocumented behavior.
- General mobile-phone topology authoring, offline laboratory editing, collaborative cursors, or chat.
- Introducing new device images or redistributing commercial appliance images as part of the frontend work.
