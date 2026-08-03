# Research: Frontend UX Modernization

## Shadcn Vue and Styling Foundation

**Decision**: Use `shadcn-vue` with Tailwind CSS v4, Reka UI primitives, Lucide icons, and project-owned
theme tokens. Generate or adapt primitives under `web/src/components/ui/`; compose application-specific
shell components under `web/src/components/shell/`.

**Rationale**: Shadcn Vue provides accessible source-owned components instead of a sealed runtime
library. This permits dense network-operations layouts, consistent keyboard/focus behavior, and a
NetLab visual identity while retaining upgrade control. Reka UI supplies accessible interaction
primitives, and Tailwind tokens keep state, spacing, borders, and density consistent.

**Alternatives considered**:

- Keep handwritten components: rejected because the current UI has inconsistent controls and would
  require duplicating accessibility behavior.
- Adopt a monolithic enterprise component suite: rejected because it adds a second design system and
  makes specialized topology/console composition harder.
- Copy EVE-NG styling or assets: rejected for product identity, maintainability, and provenance reasons.

## Workspace Information Architecture

**Decision**: Use a top toolbar, searchable left device palette, central topology canvas, contextual
right inspector, and bottom task/console drawer. Side and bottom regions are collapsible and resizable;
below the available desktop width, secondary panels become overlay drawers or tabs while the topology
and active operation remain visible.

**Rationale**: This arrangement is recognizable to EVE-NG users and maps directly to the user's chosen
layout. It preserves context during asynchronous operations and supports dense workflows without making
route navigation the primary control surface.

**Alternatives considered**:

- Full-page forms for each resource: rejected because topology context is lost during routine edits.
- Floating windows for all tools: rejected because keyboard order, collision handling, and small-screen
  usability become unpredictable.
- Fixed non-resizable panels: rejected because consoles and inspectors have workload-dependent space needs.

## ECharts Architecture

**Decision**: Use modular ECharts imports and a dedicated Vue wrapper/composable. Use graph series for
topology and traffic-path views, standard ECharts series for trends/distributions, stable backend resource
IDs as data IDs, `ResizeObserver` for container changes, explicit event registration/cleanup, and partial
option updates rather than recreating chart instances after every event.

**Rationale**: One visualization system satisfies the requirement for consistent interaction and visual
semantics. Stable IDs allow authoritative state changes to update existing visual objects while keeping
browser-local coordinates. Modular imports constrain bundle size, and lifecycle-managed chart instances
avoid leaks during panel/tab changes.

**Alternatives considered**:

- SVG/D3 or a separate graph library for topology: rejected because all graphical views are explicitly
  required to use ECharts and a second interaction model would increase maintenance.
- Rebuild the entire option on every event: rejected because it risks losing local selection/viewport and
  does unnecessary work at 100 nodes and 300 links.
- Store ECharts objects in Pinia: rejected because chart instances are view resources, not application state.

## Topology Interaction Model

**Decision**: Convert authoritative nodes, interfaces, links, and network objects into a view model that
joins browser-local placement data. Direct manipulation emits commands through existing API clients and
shows an optimistic pending affordance only; authoritative events or snapshot refresh determine completion.
New remote resources without local coordinates receive deterministic collision-aware placement.

**Rationale**: This preserves server authority while giving immediate feedback. Separating visual state
from resource state prevents one browser from changing another browser's workspace and avoids adding an
unnecessary backend coordinate contract.

**Alternatives considered**:

- Persist coordinates in laboratory resources: rejected by explicit clarification and multi-client semantics.
- Treat optimistic resources as completed state: rejected because durable task failure and rollback would be
  misrepresented.
- Random placement for remote resources: rejected because it makes tests and repeated sessions unstable.

## Browser-Local Persistence

**Decision**: Store only a versioned `WorkspacePreferences` document per laboratory through a storage
adapter. Validate on read, migrate known older versions, clamp unsafe dimensions/zoom values, ignore unknown
resource IDs, and fall back to defaults when storage is absent, corrupt, or quota constrained.

**Rationale**: An adapter centralizes schema changes and prevents arbitrary feature code from persisting
sensitive data. Per-laboratory keys avoid cross-lab collisions, and validation prevents stale preferences
from breaking the workspace.

**Alternatives considered**:

- Persist each component independently: rejected because migrations and cleanup become inconsistent.
- Use IndexedDB: rejected because the data is small, synchronous initialization is useful, and no binary
  artifacts belong in browser storage.
- Add server persistence: rejected because layout is explicitly local-only.

## Authoritative State and Event Convergence

**Decision**: Keep the existing laboratory Pinia store as the authoritative client projection, then split
focused stores/composables for workspace preferences, selection, panels, and task presentation. Apply ordered
events by revision/sequence; on replay gaps, invalid transitions, or reconnect uncertainty, fetch an
authoritative snapshot before resuming incremental updates.

**Rationale**: The existing store already implements shared events, task tracking, and reconnect behavior.
Refactoring its presentation responsibilities is lower risk than replacing its synchronization model.

**Alternatives considered**:

- Duplicate topology state inside the ECharts component: rejected because inspectors and task navigation
  would drift from the shared store.
- Trust all incremental events after a disconnect: rejected because missing events can create unsafe state.
- Poll as the primary update mechanism: rejected because it weakens the existing ordered event contract.

## UI-to-Backend Integration

**Decision**: Preserve generated API types and build typed feature adapters around existing operations.
Document every visible mutation in `contracts/backend-integration-matrix.md`; treat any missing operation or
error field as a contract defect requiring explicit backend contract work rather than a UI workaround.

**Rationale**: Generated types prevent frontend/backend drift. A matrix makes parity reviewable without
duplicating OpenAPI, event, or MCP schemas.

**Alternatives considered**:

- Handwrite feature-specific request/response interfaces: rejected because they can silently diverge.
- Introduce a frontend-only BFF: rejected because it creates another control path and violates parity.
- Infer success from HTTP acceptance: rejected because long operations complete through durable tasks.

## Console, Capture, and Traffic Diagnostics

**Decision**: Keep xterm.js and noVNC as dedicated renderers hosted in Shadcn-based tabs/drawers. Represent
capture streams through metadata and browser-safe streaming/download handles. Render Traffic Filter paths
through the same ECharts topology coordinate system, including direction, counts, timing, confidence, loops,
and ambiguity without exposing packet payloads in persisted state.

**Rationale**: Terminal and framebuffer protocols need specialized renderers. Shared hosting and navigation
provide a consistent experience, while bounded metadata respects the platform's capture and MCP contracts.

**Alternatives considered**:

- Render terminals or VNC inside ECharts: rejected because ECharts is not a protocol renderer.
- Load retained packet payloads into Pinia/localStorage: rejected for memory, privacy, and hygiene reasons.
- Display traffic matches as a textual list only: rejected because the requested feature is path visualization.

## Testing Strategy

**Decision**: Use three explicit layers: deterministic component tests with controlled mocks; contract and
integration tests against a real test service; and critical Playwright E2E against the actual Go backend.
Run privileged runtime scenarios on `10.72.1.7`, with sanitized artifacts and repeatable setup/cleanup.

**Rationale**: Controlled mocks make visual state and failure branches deterministic, while real-service
tests catch generated-type, serialization, task, event, and streaming mismatches. Real-backend E2E proves the
actual deployment path instead of validating only mocked behavior.

**Alternatives considered**:

- Mock every backend interaction: rejected because it cannot prove API/event compatibility.
- Use only E2E: rejected because failures would be slow and difficult to isolate.
- Run privileged scenarios in ordinary CI: rejected unless the runner supplies the required KVM, Docker,
  network, capture, and host privileges.

## Accessibility and Responsive Behavior

**Decision**: Require keyboard access and visible focus for shell controls, dialogs, menus, forms, task
navigation, and topology selection. Encode lifecycle and traffic state with text/icon/shape or pattern in
addition to color. Treat 1024×768 as the minimum supported viewport and test panel fallback behavior there.

**Rationale**: Dense operational interfaces otherwise become pointer-only and color-dependent. Shadcn/Reka
primitives cover general controls, while the ECharts adapter must add descriptive labels, keyboard selection,
tooltips, legends, and equivalent details in the inspector.

**Alternatives considered**:

- Desktop-wide-only support: rejected by the explicit minimum viewport requirement.
- Color-only state: rejected because it is inaccessible and ambiguous in screenshots or degraded displays.

## Performance and Resource Lifecycle

**Decision**: Measure interactions with fixed 100-node/300-link fixtures, debounce persistence and expensive
layout work, update chart data incrementally, virtualize long palettes/task lists where measurements justify
it, and dispose chart, observer, socket, terminal, and VNC resources on unmount or session close.

**Rationale**: The success target is interaction latency rather than maximum synthetic scale. Measurement-led
optimization avoids unnecessary complexity while lifecycle cleanup prevents reconnect and navigation leaks.

**Alternatives considered**:

- Prematurely add canvas workers or a second renderer: rejected until profiling shows a need.
- Keep hidden charts/sessions alive indefinitely: rejected because drawers and tabs are frequently reopened.

## Resolved Clarifications

All technical context fields are resolved. No `NEEDS CLARIFICATION` item remains, and the design introduces
no constitution violation.
