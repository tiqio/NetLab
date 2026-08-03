# Research: Topology Interaction UX

## Decision 1: Keep ECharts as the renderer, move interaction policy outside it

**Decision**: Continue using the existing ECharts graph with fixed layout, but disable reliance on its implicit
gesture policy. Normalize pointer, wheel, keyboard, graph-roam, and drag signals through a framework-independent
interaction controller and geometry module. Vue-ECharts remains the rendering adapter.

**Rationale**: The repository already uses ECharts and Vue-ECharts. Official ECharts graph capabilities cover
fixed positions, roaming, draggable graph data, curved lines, symbols, and event delivery, while Vue-ECharts
supports explicit/manual update control and event binding. A separate controller makes gesture precedence,
thresholds, cancellation, port targeting, and evidence deterministic and unit-testable.

**Alternatives considered**: Replace ECharts with a dedicated node editor; use only built-in graph `roam` and
`draggable`; implement a custom canvas renderer. Replacement has unnecessary migration risk, built-in behavior is
insufficient for the specified port workflow, and a custom renderer duplicates mature rendering work.

## Decision 2: Persist node positions as shared revisioned resources

**Decision**: Add one `TopologyPlacement` per laboratory resource. Node/network-object drag previews stay local
until pointer release; release submits one batch mutation with expected revisions. The transaction updates
placements and ordered outbox events together.

**Rationale**: Confirmed node coordinates are explicitly shared across browsers/API/MCP. Keeping them in browser
preferences contradicts that requirement. A distinct entity avoids hiding concurrency semantics in untyped node
configuration and supports group moves atomically.

**Alternatives considered**: Store coordinates inside node config; update every pointer movement; keep coordinates
browser-local. These respectively couple presentation to runtime configuration, create excessive writes/events,
and violate the clarified shared-layout requirement.

## Decision 3: Use an explicit interaction state machine

**Decision**: Model idle, pressing, panning, box-selecting, dragging-resources, connecting, choosing-target-port,
and cancelling states. A movement threshold separates click from drag. Escape cancels the most transient state
first. Pointer capture guarantees cleanup after pointer leave or focus loss.

**Rationale**: Current behavior mixes chart events with workspace mutations, making gesture conflicts difficult to
reason about. Explicit legal transitions support deterministic pointer and keyboard parity tests.

**Alternatives considered**: Independent event handlers with shared booleans; browser drag-and-drop API. Both make
multi-gesture precedence and cancellation less reliable.

## Decision 4: Make reconnect one atomic durable operation

**Decision**: Add a reconnect command that receives link revision, retained endpoint, and replacement endpoint.
The old runtime link remains authoritative until validation succeeds; the task applies runtime changes with
compensation and commits the new endpoint plus outbox event only on success.

**Rationale**: The existing frontend sequence disconnects then connects, which can leave a half-connected topology.
The specification requires preservation or restoration of the original link on failure, conflict, timeout, or
cancellation.

**Alternatives considered**: Continue client-side disconnect/connect; create a second temporary persisted link.
The first is non-atomic and the second complicates ownership and duplicate-link constraints.

## Decision 5: Use locally bundled neutral SVG symbols

**Decision**: Build a consistent topology symbol set from code-native SVG paths already covered by an approved
permissive icon source, with text labels for QEMU, Docker, PC, bridge, NAT, L2, and L3. Record source, version, and
license in the asset directory. Do not dynamically load internet images or rely on vendor logos.

**Rationale**: Neutral symbols avoid trademark ambiguity, work offline, can be themed, and satisfy non-color-only
identification. Official icon-library guidance permits local SVG usage under its license; brand icon collections
also warn that brand use may require separate permission.

**Alternatives considered**: Download arbitrary “good-looking” images; use vendor logos; use emoji. These create
provenance/trademark, consistency, or cross-platform rendering risks.

## Decision 6: Keep manual link routes browser-local

**Decision**: Auto-route every link from current node/port geometry. Optional control points remain sanitized local
preferences keyed by link ID and are deleted when the link disappears. They are not exposed through API or MCP.

**Rationale**: This matches the clarification: endpoints are shared topology, visual route adjustments are local.

**Alternatives considered**: Shared route records; no route editing. Shared routes create needless concurrent
presentation state, while no editing limits dense-topology usability.

## Decision 7: Validate at component, contract, runtime, and real-browser levels

**Decision**: Unit-test transitions and geometry; component-test event adaptation and symbols; contract-test
placement/reconnect schemas and MCP parity; integration-test transactions and rollback; Playwright-test mouse,
wheel, keyboard, two-browser/API convergence, refresh/reconnect, dense topology, and target-host live links.

**Rationale**: Mock-only happy paths cannot prove gesture correctness, shared convergence, or live-link safety.

**Alternatives considered**: Screenshot-only tests or manual target checks. Neither provides deterministic
regression evidence.

## Decision 8: Adopt EVE-NG's recognizable connection journey, not its implementation

**Decision**: Use this primary pointer journey: hovering or selecting a connectable node reveals a dedicated
connector handle; dragging the handle to another node highlights valid/invalid targets; dropping on the target
resolves interfaces; one available interface is selected automatically and multiple interfaces open a compact
chooser; the user confirms with Connect or cancels without mutation. Direct port-to-port drag and keyboard paths
remain available for experienced and accessibility users.

**Rationale**: The official EVE-NG cookbook documents a hover connector, drag-and-drop to the second node, then a
connection window where interfaces are chosen and saved. Inspection of the designated EVE-NG 6.5 frontend on
2026-07-27 also showed endpoint/interface identity, selection-area handling, link repainting, endpoint labels,
link context actions, and traffic overlays. This is a familiar network-simulator mental model, but NetLab can
reduce unnecessary dialogs and provide stronger shared-state feedback.

**Behavior retained**:

- A visually distinct connector handle appears only when useful, avoiding permanent port clutter.
- The target node is chosen spatially before detailed interface selection.
- Interface names are shown at both link endpoints.
- Existing links expose contextual actions without requiring selection of tiny line segments alone.
- Background drag selection and topology-element drag use different gesture origins.

**Behavior deliberately changed**:

- Browser, HTTP, and MCP clients share one authoritative topology and do not invalidate each other's sessions.
- A single valid target port bypasses the chooser; multiple ports use a compact chooser rather than a large form.
- Connect/reconnect immediately exposes task identity and authoritative terminal state.
- Reconnect is atomic with rollback instead of a browser-driven disconnect followed by connect.
- Link visual style/control points remain local presentation preferences, while endpoints and runtime state are
  shared.
- Error, conflict, timeout, and cancellation keep the source gesture recoverable and never leave a half-link.

**Alternatives considered**: Copy EVE-NG frontend behavior exactly; expose all ports permanently; use only
click-click connection. Exact copying retains known UX/session problems and risks proprietary coupling, permanent
ports add clutter, and click-only flow is slower for users familiar with network simulators.

## Primary Sources Consulted

- Apache ECharts official handbook and graph-series documentation.
- Vue-ECharts official repository documentation.
- Lucide official license/package documentation.
- Simple Icons official license and brand-use disclaimer, used only to reject unreviewed vendor-logo adoption.
- EVE-NG Professional Cookbook connection workflow and official feature comparison.
- Behavioral inspection of the designated EVE-NG 6.5 static frontend assets; no source code or assets retained.
