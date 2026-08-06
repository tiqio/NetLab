# UI Visual and Localization Audit Inventory

**Feature**: `008-ui-overlap-remediation`
**Status**: baseline

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
| Laboratory navigation | `web/src/features/laboratories/LaboratoryToolbar.vue` | long names, context menu, delete, theme switch | wrapping, menu contrast | pending |
| Add resource drawer | `web/src/features/topology/CreateTopologyResourceDrawer.vue` | long forms, routes, 125% | field/action overlap, unreachable footer | pending |
| Topology nodes | `web/src/features/topology/TopologyCanvas.vue` | 1/2/4/8/16 ports, hover, drag, zoom | port/label/connector overlap and drift | pending |
| Topology links | `web/src/features/topology/linkPresentation.ts` | parallel links, failed links, selection | path and label collision | pending |
| Traffic overlay | `web/src/features/topology/TrafficPathOverlay.vue` | forward, reverse, idle, reduced motion | wrong-link highlight, persistent flashing | pending |
| Inspector | `web/src/features/topology/TopologyInspector.vue` | all resource kinds, long errors, narrow width | chart/metrics/actions overlap | pending |
| Resource charts | `web/src/features/analytics/ResourceCharts.vue` | resize, theme switch, long metrics | invalid size, legend overlap | pending |
| Context menus | `web/src/features/topology/TopologyWorkspace.vue`, `web/src/features/topology/LinkContextMenu.vue` | four viewport edges, disabled, danger | clipping and low contrast | pending |
| Tasks | `web/src/features/tasks/TaskCenter.vue` | empty, filter, large history | English paging, local overflow | pending |
| Console | `web/src/features/diagnostics/GlobalConsoleWorkspace.vue` | empty, multiple nodes/sessions, reconnect | English guidance and lost tabs | pending |
| Capture | `web/src/features/diagnostics/GlobalCaptureWorkspace.vue`, `web/src/features/diagnostics/CapturePanel.vue` | no source, running, helper failure | English errors, mixed selectors | pending |
| Traffic Filter | `web/src/features/diagnostics/TrafficFilterPanel.vue` | session list, search, running, stopped | chart/list overlap, status wording | pending |
| Templates | `web/src/features/templates/TemplatePicker.vue`, `web/src/features/templates/TemplateCatalog.vue` | unavailable versions, long image names | untranslated generic labels | pending |
| Image import | `web/src/features/templates/ImageImportDialog.vue` | validation and errors | untranslated form labels | pending |
| Automation | `web/src/views/AutomationView.vue` | long audit rows, theme switch | local horizontal overflow | pending |
| Shared controls | `web/src/components/ui/`, `web/src/components/common/` | focus, disabled, danger, loading | hit target and contrast inconsistency | pending |

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
