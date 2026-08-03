# Event and Stream Contracts

## Ordered State Events

`GET /api/v1/events?after=<sequence>` upgrades to WebSocket. Each JSON message is:

```json
{
  "sequence": 1042,
  "type": "node.observed_state_changed",
  "occurred_at": "2026-07-23T08:00:00.123456Z",
  "laboratory_id": "019...",
  "resource_type": "node",
  "resource_id": "019...",
  "revision": 7,
  "task_id": "019...",
  "data": {"desired_state": "running", "observed_state": "running"}
}
```

- Sequences are strictly increasing for durable events.
- Clients acknowledge the last applied sequence in reconnect requests.
- If `after` predates retained events, the server sends `stream.reset_required` and closes normally;
  the client fetches a laboratory snapshot and reconnects from its returned sequence.
- Slow consumers are disconnected with a retryable reason rather than delaying reconciliation.

### Topology Interaction Events

- `topology.placements_changed` contains one bounded `placements` array and the new laboratory revision. One completed drag publishes one event.
- `link.reconnecting` contains the staged link and previous endpoint IDs.
- `link.reconnect_rolled_back` contains the restored original link after failure, cancellation, or timeout.
- Successful reconnect convergence is represented by the normal ordered link state event and terminal `link.reconnect` task result.

## Console WebSocket

`GET /api/v1/nodes/{nodeId}/consoles/{mode}/stream` uses binary frames for terminal/VNC bytes and JSON
control frames for resize, ping, close reason, and errors. One client disconnect never changes node
lifecycle. Server-side idle and bandwidth limits are configuration values returned by console discovery.

## Capture Stream

`GET /api/v1/captures/{captureId}/stream` returns chunked `application/vnd.tcpdump.pcap` or
`application/x-pcapng`. Headers include capture ID, source, effective filter, and truncation limits.
Disconnecting a stream consumer does not stop a retained capture; a non-retained capture stops after its
last consumer disconnects unless another client explicitly owns the session.

## Traffic Filter Events

`traffic_filter.observation` contains filter ID, fingerprint, interface ID, link ID, direction,
timestamp, packet length, and aggregate counters. The server may coalesce events but must preserve first
and last observation times and total counts.
