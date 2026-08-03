# Topology Interaction Events

## `topology.placements_changed`

Durable ordered event emitted in the same transaction as a placement batch. It contains laboratory ID/revision,
resource IDs, coordinates, placement revisions, correlation ID, and timestamp. It contains no browser-local
viewport or manual link control points. Clients apply it only when newer than their placement revision; event gaps
trigger a full laboratory snapshot refresh.

## `link.reconnect_requested` and terminal events

Reconnect emits the standard task lifecycle events. `link.reconnected` is emitted only after runtime application
and durable endpoint commit succeed. Failure/cancellation events include a sanitized problem and compensation
outcome. Other clients keep rendering the original endpoints until `link.reconnected` arrives.
