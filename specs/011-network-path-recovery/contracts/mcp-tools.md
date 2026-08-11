# MCP Contract Delta

## `netlab_reconcile_network_object`

Inputs: `object_id`, `expected_revision`, `idempotency_key`.

Returns a durable task envelope. Final results and structured problems match `POST /api/v1/network-objects/{id}/reconcile`.

## `netlab_reconcile_network_object_link`

Inputs: `link_id`, `expected_revision`, `idempotency_key`.

Retries a pending or failed link without allocating a second reservation. Deletion remains a separate revisioned tool.

## `netlab_get_network_object_diagnostics`

Inputs: `object_id`.

Returns desired/observed backing, interfaces, forwarding, routes and VLAN membership with the same schema and redaction as HTTP.

## `netlab_create_traffic_workload`

Inputs: laboratory revision, name, source endpoint, protocol, address family, destination, interval, timeout and idempotency key.

Returns the durable workload plus its creation task. Unsupported source capabilities fail before runtime execution.

## `netlab_set_traffic_workload_state`

Inputs: workload ID, expected revision, desired state and idempotency key.

Start/stop semantics, cancellation and final aggregates match HTTP and UI behavior.

## `netlab_get_traffic_workload` / `netlab_list_traffic_workloads`

Returns aggregate attempt, success, failure, byte and timestamp fields. Response bodies, DNS payloads, credentials and packet payloads are never returned.

## `netlab_delete_traffic_workload`

Stops and deletes the workload idempotently. Active captures and Traffic Filters are not implicitly deleted, but laboratory deletion cleans all owned resources.

## Parity Rules

- All mutations require revision and idempotency semantics equivalent to HTTP.
- Task, problem, audit and event fields use the same names and lifecycle values across MCP, HTTP and SPA.
- MCP cannot bypass ownership validation, resource limits, workload command allowlists or recovery gates.
