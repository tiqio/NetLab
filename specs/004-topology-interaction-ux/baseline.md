# Topology Interaction Baseline

**Recorded**: 2026-07-27

## Current NetLab Behavior

- `TopologyCanvas.vue` delegates zoom and pan to ECharts graph roaming and emits graph-roam deltas.
- Node drag events update browser-local placements; confirmed positions are not server-authoritative.
- Interface click-click creates links. Reconnect disconnects before creating the replacement and can leave a gap.
- Ports are small graph nodes around their owner and can be difficult to discover in dense topologies.

## EVE-NG Behavioral Reference

- Hovering a node exposes a dedicated connector affordance.
- Dragging the connector to a target node precedes interface selection and explicit confirmation.
- Endpoint labels, selection-area behavior, link context actions, and traffic overlays are familiar patterns.

NetLab reuses only this interaction vocabulary. It does not copy EVE-NG code, assets, CSS, templates,
credentials, APIs, account-scoped state, or session behavior.

## Target Improvements

- Normalize navigation, selection, dragging, connection, and cancellation through a tested state machine.
- Persist confirmed node coordinates as shared revisioned state while keeping viewport and link routes local.
- Use connector drag, one-port automatic selection, compact multi-port selection, and keyboard parity.
- Replace disconnect-then-connect with an atomic reconnect that preserves or restores the original link.
