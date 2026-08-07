# Implementation Notes

## History Reviewed

- `git log` inspected connection rendering, topology workspace creation, layout, and placement repository history through 2026-08-07.
- `git blame` inspected `TopologyCanvas.vue` graph configuration, `TopologyWorkspace.vue` creation placement, and `topology_placement_repository.go` revision handling.

## Compatibility Constraints

1. Keep ECharts graph layout fixed with `layout: "none"`; traffic, hover, selection, theme, and edge updates must never enable force layout or rewrite persisted node coordinates.
2. Preserve the stabilized interaction split: graph roaming handles scale only, explicit pan mode controls canvas movement, and node dragging remains separate from port connection gestures.
3. Preserve persisted coordinates for every existing node and network object. Deterministic fallback is only for resources that genuinely have no placement.
4. Replace the current create-then-`PUT /placements` workflow without breaking manual drag/group placement updates.
5. Keep `Link`, `NetworkAttachment`, and `NetworkObjectLink` as distinct runtime ownership models; normalize only their presentation and user-facing capabilities.
6. Keep Traffic Filter activity scoped to the exact connection ID and interface pair. Parallel connections must not inherit one another's observations.
7. Keep resource category colors separate from connection status colors; previous commits intentionally separated topology/resource chart legends and applied theme-aware symbol colors.
8. Keep Chinese labels, light/dark theme tokens, accessibility names, and reduced-motion behavior introduced by the recent UI stabilization work.
9. Preserve laboratory and placement revision checks. Concurrent creation must serialize through the authoritative SQLite transaction rather than client-side collision prediction.
10. Do not modify target-host source directly. Each visual, legend, placement, and recovery milestone requires local tests and a focused commit before deployment.
