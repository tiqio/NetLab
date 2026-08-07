# MCP Tool Contract

MCP is served over Streamable HTTP at `/mcp`. Tools use the same validation, revisions, idempotency,
tasks, and errors as `/api/v1`. Every mutating tool accepts optional `idempotency_key` and
`expected_revision` fields and returns either a resource or an `operation_task` envelope.

| Tool | Purpose | Principal Inputs | Result |
|---|---|---|---|
| `netlab.capabilities` | Discover limits and supported operations | none | server capabilities and quotas |
| `netlab.templates.list` | List template/image variants | filters | template summaries |
| `netlab.labs.list` | List shared labs | pagination | lab summaries |
| `netlab.labs.get` | Read complete topology snapshot | `lab_id` | lab snapshot and event sequence |
| `netlab.labs.create` | Create lab | name, recovery policy | laboratory |
| `netlab.labs.delete` | Delete lab and owned resources | `lab_id` | operation task |
| `netlab.labs.export` | Create redacted export | `lab_id` | operation task and artifact metadata |
| `netlab.labs.import` | Import validated bundle | artifact handle | operation task |
| `netlab.nodes.create` | Add node from template or lightweight kind with authoritative initial placement | lab, revision, idempotency key, template/version, resources, optional placement intent | node, interfaces, placement assignment, updated lab revision |
| `netlab.network_objects.create` | Add PC, Bridge, NAT, L2, or L3 object with authoritative initial placement | lab, revision, idempotency key, kind, config, optional placement intent | network object, task, placement assignment, updated lab revision |
| `netlab.nodes.set_state` | Start or stop node | node, desired state | operation task |
| `netlab.nodes.delete` | Delete node | node | operation task |
| `netlab.nodes.exec` | Execute guest-agent command | node, argv, timeout, output limit | operation task |
| `netlab.interfaces.add` | Hot-add QEMU NIC | node, driver | operation task |
| `netlab.interfaces.remove` | Hot-remove QEMU NIC | interface | operation task |
| `netlab.links.connect` | Connect two interfaces live | endpoints | operation task |
| `netlab.links.disconnect` | Disconnect link live | link | operation task |
| `netlab.links.reconnect` | Replace one endpoint atomically with rollback | link, revision, retained endpoint, replacement endpoint | operation task |
| `netlab.topology.set_positions` | Manually move already-created node/network-object coordinates; never performs initial placement | lab, revision, 1-100 placements | updated revision and authoritative placements |
| `netlab.port_mappings.create` | Publish guest endpoint | node, protocol, host/guest ports | operation task |
| `netlab.port_mappings.delete` | Remove mapping | mapping | operation task |
| `netlab.consoles.get` | Discover Telnet/VNC session | node, mode | bounded session descriptor |
| `netlab.captures.start` | Start live/retained capture | source, filter, limits, retain | capture/task |
| `netlab.captures.get` | Read capture status | capture | metadata, bounded summary, stream/artifact handle |
| `netlab.captures.stop` | Stop capture | capture | operation task |
| `netlab.traffic_filters.start` | Observe matching packet path | lab, scope, match, limits, optional six-digit hex `color` | filter/task |
| `netlab.traffic_filters.get` | Read observed path | filter | link/interface observations and counts |
| `netlab.traffic_filters.stop` | Stop path observation | filter | operation task |
| `netlab.tasks.get` | Read operation status | task | task envelope |
| `netlab.tasks.cancel` | Request cancellation | task | task envelope |

## Error Result

Tool errors include `code`, `message`, `retryable`, `resource`, optional `task_id`, and bounded
`details`. Expected codes include `revision_conflict`, `idempotency_conflict`, `capability_unsupported`,
`resource_exhausted`, `image_unavailable`, `port_conflict`, `invalid_transition`, and `reconciliation_failed`.

Capture and export handles are opaque references suitable for a follow-up HTTP resource fetch. MCP
responses never embed packet-capture bytes, image bytes, secrets, or unbounded guest-command output.
