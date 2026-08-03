# Event Contract Delta

All events use the existing ordered outbox envelope: event ID, laboratory ID, aggregate type and ID,
aggregate revision, event type, occurred-at timestamp, actor class, correlation/task ID, and payload.
The state mutation and event append must commit atomically.

## Network Object Link Events

| Event type | Required payload |
|---|---|
| `network_object_link.created` | Full durable link snapshot; reservations commit in the same transaction |
| `network_object_link.state_changed` | `observed_state` and optional structured `last_error`; desired state and revision remain in the envelope/authoritative resource |
| `network_object_link.deleted` | Empty payload; link ID, final revision, and completion task ID are carried by the ordered envelope |
| `network_object_link.recovered` | Link ID, desired/observed state, optional `last_error`, and `recovery_action=adopted_or_recreated` |

Consumers remove a topology line on `network_object_link.deleted` without requiring a full refresh.
Parallel links are keyed only by link ID, never by unordered object pair.

## Docker Route State

The implemented contract does not introduce separate route event types. Stopped-node route edits return
the updated revisioned node through the shared settings command. Start/restart/recovery route application
is observable through the existing durable node lifecycle task and authoritative node readiness/error
state. Clients must refresh the node after relevant task or topology events rather than wait for a
`node.routes_reconciled` event that the server does not publish. No event or export contains container
namespace PID, host command, credential, packet payload, or target secret.

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
