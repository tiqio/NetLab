# Implementation Notes: Unified Link Interaction

## Compatibility History

- `38bbf37b` introduced separate node-interface and network-object-port click flows. The current
  object-port path deliberately rejects a pending node interface and redirects users to Inspector;
  009 replaces that UI split but must preserve the existing attachment runtime and cleanup path.
- `4365781` and `550cf74` established the 010 `ConnectionPresentation`, stable parallel routing,
  status-first visual semantics, ordered topology convergence, and authoritative placement. 009
  must submit authoritative connections into those existing projections rather than create a fourth
  visual edge model or relayout resources.
- `cee0087` and related topology interaction fixes separated resource drag, box selection, viewport
  roam, keyboard selection, and SVG port overlays. Port connection gestures must use pointer capture
  on the overlay and must not let one pointer sequence trigger any other canvas gesture.
- `f23d453` and network-object-link migrations established explicit runtime ownership and endpoint
  reservations for object links. Unified occupancy must extend this mechanism to node interfaces and
  attachments without changing which reconciler owns each backing resource.
- Existing specialized HTTP and MCP routes are compatibility surfaces. They may adapt into a shared
  command, but response IDs, durable task observation, capture sources, Traffic Filter matching, and
  deletion behavior must remain usable by old clients.

## Preserved Invariants

- Keep `Link`, `NetworkAttachment`, and `NetworkObjectLink` as distinct persisted/runtime models.
- Keep 010 connection colors, non-color state markers, route grouping, labels, capture overlays, and
  Traffic Filter decay semantics unchanged.
- Keep existing laboratory placements byte-for-byte stable unless the user explicitly moves them.
- Treat connection drafts, pointer positions, hover candidates, and port choosers as browser-local.
- Validate concrete endpoint occupancy in SQLite, not only in Vue state or process-local locks.
- Apply `eth0` through `eth3` defaults only to newly created lightweight L2/L3 objects whose create
  request omits an explicit port set; never expand update, import, or recovery data.
- Preserve running-resource mutation, durable task, idempotency, revision, audit, recovery, and
  owned-resource cleanup requirements across all connection entry points.

## History Inspected

```text
git log -- web/src/features/topology/TopologyWorkspace.vue
git log -- web/src/features/topology/TopologyCanvas.vue
git log -- web/src/features/nodes/lightweightSwitchConfig.ts
git log -- internal/app/command internal/app/reconcile internal/store/sqlite
git blame -L 812,945 -- web/src/features/topology/TopologyWorkspace.vue
git blame -L 27,42 -- web/src/features/nodes/lightweightSwitchConfig.ts
```
