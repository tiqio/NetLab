# Event Contract Additions

All events use the existing ordered state-event envelope and durable outbox sequence.

## `network_object.observed_state_changed`

Published in the same transaction that updates the network object.

```json
{
  "sequence": 1201,
  "type": "network_object.observed_state_changed",
  "laboratory_id": "lab-id",
  "resource_type": "network_object",
  "resource_id": "network-object-id",
  "revision": 3,
  "data": {
    "observed_state": "active",
    "last_error": null
  }
}
```

Allowed observed states: `provisioning`, `active`, `degraded`, `failed`, `deleting`.

## `node.capability_changed`

Published in the same transaction that upserts one runtime capability observation.

```json
{
  "sequence": 1202,
  "type": "node.capability_changed",
  "laboratory_id": "lab-id",
  "resource_type": "node",
  "resource_id": "node-id",
  "revision": 2,
  "data": {
    "node_id": "node-id",
    "capability": "qga",
    "state": "unavailable",
    "required": true,
    "problem": {
      "code": "guest_agent_unavailable",
      "message": "QEMU guest agent did not become ready",
      "retryable": true,
      "phase": "capability_probe",
      "cleanup": "node remains running; no cleanup required",
      "operator_hint": "install and enable qemu-guest-agent in the selected image"
    },
    "observed_at": "2026-07-29T00:00:00Z"
  }
}
```

## Client Rules

- Events with a sequence not greater than the local cursor are ignored.
- Deletion events remain terminal even when their resource revision is lower than a cached revision.
- Capability revisions are compared within `(node_id, capability)`, not against the node revision.
- Capability events share the node resource identity; clients MUST use `data.capability` as the
  secondary key and MUST NOT replace node lifecycle state with capability state.
- A sequence gap or reset event requires a full authoritative snapshot reload.
- UI labels must map network-object `active` to a healthy active state rather than stopped/unknown.
- `network_object.deleted`, `node.deleted`, `link.deleted`, and `interface.deleted` are tombstones;
  later stale snapshots or lower-revision events MUST NOT resurrect those resources.
