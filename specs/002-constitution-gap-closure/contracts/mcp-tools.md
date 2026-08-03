# MCP Contract Additions

## `netlab.nodes.capabilities`

Returns the same observations as `GET /api/v1/nodes/{nodeId}/capabilities`.

### Input

```json
{
  "node_id": "node-id"
}
```

### Output

```json
{
  "node_id": "node-id",
  "observations": [
    {
      "capability": "qga",
      "revision": 2,
      "state": "ready",
      "required": true,
      "details": {},
      "observed_at": "2026-07-29T00:00:00Z"
    }
  ]
}
```

### Error Semantics

- `not_found`: node does not exist.
- `capability_unavailable`: capability observation service is unavailable on the host.
- Errors use the same structured problem fields and resource identity as REST.

## Runtime and Network Tool Names

The implemented external names are stable and map to the same application handlers as REST:

- `netlab.network_objects.create|get|delete`
- `netlab.nodes.capabilities`
- `netlab.nodes.exec`
- `netlab.interfaces.add|delete`
- `netlab.links.create|delete|reconnect`
- `netlab.captures.start|get|stop`
- `netlab.traffic_filters.start|get|stop`
- `netlab.tasks.get|cancel`

`netlab.nodes.exec` returns `capability_unavailable` before creating a task unless the latest
`guest_exec` observation is `ready`. `netlab.links.reconnect` requires `expected_revision`, retains one
endpoint, replaces the other while nodes remain running, and compensates back to the original link on
failure. NAT creation accepts DHCPv4, DHCPv6, DNS, and router-advertisement configuration through the
generic `config` object; diagnostics remain a REST query because they may contain runtime-specific
structures not suitable for a stable mutation tool result.

## Existing Mutation Parity

No second MCP-only mutation path is introduced. NAT creation, node lifecycle, link/NIC changes, guest
commands, captures, traffic filters, and deletion use the same durable command handlers as REST.
Mutation tools accept `idempotency_key`; revision-sensitive tools accept `expected_revision`.
Contract tests prove identical replay, conflict, task envelope, cancellation, terminal error, audit,
and event behavior between REST and MCP.
