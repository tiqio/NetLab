# UI Visual and Localization Audit Inventory

**Feature**: `008-ui-overlap-remediation`
**Status**: local implementation complete; target-host acceptance pending

## Severity

- **Blocking**: prevents a required action, hides authoritative state, or causes destructive misclicks.
- **Serious**: makes primary content unreadable or a primary control unreliable.
- **Moderate**: requires avoidable scrolling, resizing, or repeated attempts.
- **Cosmetic**: alignment issue without task or readability impact.

## Required Matrix

- Themes: light, dark
- Viewports: 1024×768, 1366×768, 1920×1080
- Display scale: 100%; critical journeys also 125%
- States: empty, normal, dense, hover, focus, selected, disabled, loading, running, failed, overlay-open

## Surface Inventory

| Surface | Primary files | Required states | Initial risks | Result |
|---|---|---|---|---|
| Laboratory navigation | `web/src/features/laboratories/LaboratoryToolbar.vue` | long names, context menu, delete, theme switch | wrapping, menu contrast | passed: wrapping and Chinese context actions remain readable in both themes |
| Add resource drawer | `web/src/features/topology/CreateTopologyResourceDrawer.vue` | long forms, routes, 125% | field/action overlap, unreachable footer | passed: local scrolling preserves fields and footer actions at 125% |
| Topology nodes | `web/src/features/topology/TopologyCanvas.vue` | 1/2/4/8/16 ports, hover, drag, zoom | port/label/connector overlap and drift | passed: deterministic four-side tracks, separated hit targets and zoom-aware labels |
| Topology links | `web/src/features/topology/linkPresentation.ts` | parallel links, failed links, selection | path and label collision | passed: parallel paths, labels and failure state remain distinguishable |
| Traffic overlay | `web/src/features/topology/TopologyCanvas.vue` | forward, reverse, idle, reduced motion | wrong-link highlight, persistent flashing | passed: exact-link matching, pointer transparency and reduced-motion behavior are integrated into the authoritative canvas renderer |
| Inspector | `web/src/features/topology/TopologyInspector.vue` | all resource kinds, long errors, narrow width | chart/metrics/actions overlap | passed: fixed header/actions with independently scrolling content |
| Resource charts | `web/src/features/analytics/ResourceCharts.vue` | resize, theme switch, long metrics | invalid size, legend overlap | passed: separated metrics/chart regions and size-aware initialization |
| Context menus | `web/src/features/topology/TopologyWorkspace.vue`, `web/src/features/topology/LinkContextMenu.vue` | four viewport edges, disabled, danger | clipping and low contrast | passed: viewport collision handling and semantic readable states |
| Tasks | `web/src/features/tasks/TaskCenter.vue` | empty, filter, large history | English paging, local overflow | passed: Chinese states and bounded list scrolling |
| Console | `web/src/features/diagnostics/GlobalConsoleWorkspace.vue` | empty, multiple nodes/sessions, reconnect | English guidance and lost tabs | passed: Chinese guidance and stable node/session tab layout |
| Capture | `web/src/features/diagnostics/GlobalCaptureWorkspace.vue`, `web/src/features/diagnostics/CapturePanel.vue` | no source, running, helper failure | English errors, mixed selectors | passed: Chinese source selection, helper guidance and local scrolling |
| Traffic Filter | `web/src/features/diagnostics/TrafficFilterPanel.vue` | session list, search, running, stopped | chart/list overlap, status wording | passed: bounded history controls, Chinese states and topology-integrated overlays |
| Templates | `web/src/features/templates/TemplateCatalog.vue`, `web/src/features/topology/CreateTopologyResourceDrawer.vue` | unavailable versions, long image names | untranslated generic labels | passed: the active catalog and add drawer provide Chinese availability states and constrained long image names; the unused legacy picker was removed |
| Image import | `web/src/features/templates/ImageImportDialog.vue` | validation and errors | untranslated form labels | passed: Chinese field labels, validation and error summaries |
| Automation | `web/src/views/AutomationView.vue` | long audit rows, theme switch | local horizontal overflow | passed: wrapping header and shrink-safe content columns |
| Shared controls | `web/src/components/ui/`, `web/src/components/common/` | focus, disabled, danger, loading | hit target and contrast inconsistency | passed: semantic colors, focus visibility and consistent action sizing |

## Allowed Intentional Overlays

| Overlay | Conditions | Evidence required |
|---|---|---|
| Dialog/Sheet | dismissible, focus controlled, active content contained | keyboard and viewport-edge result |
| Context menu | collision-aware and fully reachable | four-edge screenshot/geometry result |
| Tooltip | non-interactive content only and does not block target | hover/focus result |
| Connection preview | pointer-transparent except target endpoints | port hit-target result |
| Traffic overlay | pointer-transparent and limited to link path | active/inactive parallel-link result |

## Initial Localization Findings

- Console empty guidance and session controls.
- Capture/Wireshark helper error branches.
- Tasks/Console/Capture workspace labels.
- Template selection and image import generic labels.
- Interface chooser, link menu and add-drawer actions.
- Traffic chart status, topology hover descriptions and hidden accessibility headings.

## Coverage Summary

- Inventory: 14 named browser scenarios spanning topology, inspector, context menu, console, capture, Traffic Filter, templates and automation.
- Matrix: light and dark themes at 1024×768, 1366×768 and 1920×1080; critical input journeys also run at 125% page zoom.
- Component coverage: topology geometry, parallel links, traffic matching, charts, inspector, menus, workspaces, localization scanner, theme continuity and evidence redaction.
- Accessibility coverage: keyboard focus and axe checks for the primary workspace, overlays and diagnostics surfaces.
- Continuous gate: `tests/e2e/matrices/uiVisualAudit.spec.ts` samples named layout regions and rejects blocking or serious overlap findings.

## Fixed Findings

- Replaced dynamic port placement with stable four-side tracks and separated port labels, hit targets and connection controls.
- Prevented connection previews and Traffic Filter particles from consuming pointer input or highlighting inactive parallel links.
- Split inspector headers, metrics, charts, actions and long content into explicit responsive layout regions.
- Added delayed ECharts initialization and resize/theme reflow to avoid zero-size charts and legend collisions.
- Added viewport-edge collision handling and readable normal, disabled and dangerous context-menu states.
- Bounded scrolling for task, terminal, capture and Traffic Filter content so controls remain reachable.
- Removed remaining English product guidance from terminal, capture, templates, tasks, topology and automation flows.
- Added three-viewport, dual-theme, reduced-motion, 125% zoom and 20-cycle route/theme continuity regression coverage.

## Approved Overlay Rationale

- Dialogs and sheets isolate temporary tasks without resizing the topology; focus containment and dismissal are required.
- Context menus remain attached to the pointer target but are clamped to the viewport.
- Tooltips expose truncated values and never contain required interactive controls.
- Connection previews and Traffic Filter effects communicate transient topology state and use pointer-transparent layers.
- No overlap is approved when it hides authoritative state, blocks a required action, intercepts an unrelated pointer target or reduces text contrast below the shared semantic theme rules.
