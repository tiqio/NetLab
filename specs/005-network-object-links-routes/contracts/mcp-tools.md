# MCP Tool Contract Delta

MCP mutations invoke the same application commands as HTTP and return the same durable resource/task
envelopes. Every mutation accepts an optional `idempotency_key`; revision-sensitive deletion requires
`expected_revision`.

## `netlab.network_object_links.create`

**Input**

```json
{
  "laboratory_id": "lab-id",
  "object_a_id": "object-a",
  "port_a_name": "swp1",
  "object_b_id": "object-b",
  "port_b_name": "swp1",
  "idempotency_key": "optional-key"
}
```

**Output**: `{ "network_object_link": NetworkObjectLink, "task": OperationTask }`

Reject cross-laboratory, missing, non-connectable, or occupied endpoints with a structured conflict or
validation problem and no partial reservation.

## `netlab.network_object_links.get`

Input: `{ "link_id": "..." }`
Output: authoritative `NetworkObjectLink`, including endpoint IDs/names, `desired_state`,
`observed_state`, revision, and optional structured `last_error`. Human-readable labels are derived by
clients from the authoritative endpoint objects and names.

## `netlab.network_object_links.delete`

Input:

```json
{
  "link_id": "link-id",
  "expected_revision": 3,
  "idempotency_key": "optional-key"
}
```

Output: `{ "network_object_link": NetworkObjectLink, "task": OperationTask }`. Active captures are
completed with `link_deleted`; deletion is asynchronous and observable.

## Existing Capture Tools

`netlab.captures.start` extends `source_type` with `network_object_link`. The caller supplies the durable
link ID as `source_id`; it never supplies a namespace or host interface. Responses retain bounded
metadata, stream URL, optional artifact reference, packet/byte counters, state, and completion reason.

## Existing Traffic Filter Tools

`netlab.traffic_filters.start` adds:

```json
{
  "network_object_link_ids": ["link-a", "link-b"]
}
```

Observations return `resource_type`, `resource_id`, `direction`, `first_seen`, `last_seen`, `count`, and
`bytes` (plus protocol/address metadata when available).
Direction may be `ambiguous`; MCP clients must not infer a direction when the server marks ambiguity.

## Existing Node Create/Settings Tools

Docker node `network_interfaces` accept ordered `routes`:

```json
{
  "id": "interface-id",
  "name": "eth0",
  "driver": "virtio",
  "modes": ["static"],
  "addresses": ["192.0.2.10/24"],
  "routes": [
    { "destination": "0.0.0.0/0", "gateway": "192.0.2.1", "metric": 100 },
    { "destination": "2001:db8:2::/64", "gateway": "2001:db8:1::1" }
  ]
}
```

`netlab.nodes.update_settings` requires `node_id`, `expected_revision`, the complete stopped-node resource
settings, and the complete ordered `network_interfaces` collection. It returns the authoritative updated
node. The mutation returns stable validation problems before start for malformed declarations. Runtime
route failures are returned through the node lifecycle task and authoritative readiness/error state.
