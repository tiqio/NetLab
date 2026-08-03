# Event Contract Delta

All events use the existing ordered outbox envelope: event ID, laboratory ID, aggregate type and ID,
aggregate revision, event type, occurred-at timestamp, actor class, correlation/task ID, and payload.
The state mutation and event append must commit atomically.

## Network Object Link Events

| Event type | Required payload |
|---|---|
| `network_object_link.created` | Full durable link snapshot and endpoint reservations |
| `network_object_link.state_changed` | `desired_state`, `actual_state`, `revision`, optional structured error |
| `network_object_link.deleted` | Link ID, final revision, both endpoint identities, completion task ID |
| `network_object_link.recovered` | Link ID, adopted/recreated action, resulting actual state |

Consumers remove a topology line on `network_object_link.deleted` without requiring a full refresh.
Parallel links are keyed only by link ID, never by unordered object pair.

## Docker Route Events

| Event type | Required payload |
|---|---|
| `node.network_configuration_changed` | Node ID, revision, interface summaries, redacted route declarations |
| `node.routes_reconciled` | Node ID, revision, applied/removed counts, resulting readiness |
| `node.route_reconciliation_failed` | Node ID, revision, interface ID/name, destination, error code/message |

Route event payloads contain no container namespace PID, host command, credential, or secret.

## Capture and Traffic Events

- Capture completion adds `completion_reason`; object-link deletion uses `link_deleted`.
- Traffic observations add `resource_type`, `resource_id`, and deterministic or `ambiguous` direction.
- Observation updates for a deleted link are ignored after the link deletion revision is observed.
- The server preserves global event ordering across link state, capture completion, reservation release,
  and deletion so clients cannot resurrect a ghost line.

## Compatibility

Existing standard-link and interface events remain valid. New fields are additive. Clients that do not
recognize `network_object_link` must ignore the unknown resource type rather than treating it as a
standard link.
