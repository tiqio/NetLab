# Visual Layout Contract

## Purpose

定义 NetLab 前端在支持的主题、视口、显示比例和高密度状态下必须保持的布局、命中区域、覆盖层和可读性行为。本合同不改变任何服务器资源或操作结果。

## Supported Matrix

- Themes: `light`, `dark`
- Viewports: `1024×768`, `1366×768`, `1920×1080`
- Display scale: all scenarios at `100%`; critical create/connect/inspect/menu/terminal/capture journeys at `125%`
- Density: empty, normal, dense
- Interaction: default, hover, focus, selected, disabled, loading, running, failed, overlay-open

## Global Invariants

1. `documentElement.scrollWidth - clientWidth <= 1` except inside an explicitly identified local horizontal scroller.
2. A critical or primary layout region MUST NOT intersect an unrelated region by more than one device pixel.
3. Two interactive hit targets MUST NOT overlap; transparent decoration MUST use non-intercepting pointer behavior.
4. Keyboard focus MUST remain visible and MUST NOT be clipped by an overflow container.
5. Fixed headers, tabs and action bars MUST NOT cover the first or last reachable item in their associated scroll region.
6. Theme switching MUST update foreground, background, border, chart and topology semantic colors without stale white-on-white or dark-on-dark states.
7. Loading, animation and live traffic effects MUST NOT move stable node, panel or control bounds after the initial transition settles.

## Intentional Overlay Rule

Menus, tooltips, dialogs, drag previews and connection previews may overlap other content only when all conditions hold:

- the overlay has a higher declared layer;
- it can be dismissed by an expected pointer and keyboard action;
- focus is managed for modal content;
- the active task target remains visible or is intentionally contained by the overlay;
- the overlay stays inside the viewport or uses collision-aware repositioning;
- the audit record identifies the overlap as intentional.

## Topology Contract

### Node Regions

Each topology node provides separate regions for:

- resource graphic;
- name and kind;
- desired/actual state;
- selection and warning indication;
- port track;
- port label;
- connection entry.

### Port Geometry

- Representative nodes with 1, 2, 4, 8 and 16 interfaces MUST render every available port with a unique center and hit target.
- A port hit target and its connection entry MUST remain independently selectable.
- Hover, selection, dragging and reconnection MUST NOT change the underlying port coordinates.
- At reduced zoom, secondary labels may collapse according to priority, but port markers, selection state and connection entry remain present.

### Links and Traffic

- Parallel links MUST use separate paths or offsets and expose readable endpoint names.
- Link labels MUST use `node:interface ↔ node:interface` or the equivalent network-object endpoint name.
- Direction markers and particles MUST reflect observed direction; inactive parallel links MUST NOT inherit the active link highlight.
- Traffic Filter effects may overlay link paths but MUST NOT cover node names, ports or connection controls.
- When matching traffic stops, particles stop promptly and persistent direction decoration decays according to the presentation timeout without indefinite flashing.

## Inspector and Chart Contract

- The header separates identity, state and actions; actions wrap or enter an overflow group before overlapping identity.
- Resource chart, metrics, uplink/downlink text, forms and lifecycle controls occupy separate layout regions.
- Charts render only with positive container dimensions and resize after panel, viewport or theme changes.
- Legends and numeric labels remain outside the primary plot area when the inspector is at minimum supported width.
- Long IDs and errors use wrap, truncation with full-value access, or a local scroller; they never expand the page width.

## Context Menu Contract

- Node, link, network-object and canvas menus use the same collision-aware positioning behavior.
- Menu text, icon, shortcut and status regions do not overlap.
- Disabled, dangerous, selected and normal items remain distinguishable in both themes using at least one non-color cue.
- Menus opened near the right or bottom viewport edge remain fully reachable.

## Bottom Workspace Contract

- 任务、终端、抓包、流量过滤和诊断 tabs remain present when content switches, hides or restores.
- Resource/session selectors occupy a separate row from per-resource session tabs and add actions.
- Switching tabs MUST hide inactive content without destroying persistent console/capture state unless existing lifecycle rules require cleanup.
- Empty, loading, failed and large-history states provide a stable local scroll region and do not collapse the topology canvas.

## Verification Interface

Automated checks may expose stable `data-testid` or `data-layout-region` attributes. These identifiers are test contracts only and MUST NOT become API, MCP or persistent laboratory fields.
