# Data Model: Network Path Recovery and Validation

## Recoverable Runtime Resource

Represents the relationship between a durable resource and its host backing.

- `resource_type`, `resource_id`, `laboratory_id`
- `backing_kind`: namespace, host bridge, QEMU tap, Docker veth, route/rule, socket or process
- `owner_identity`, expected runtime name and optional parent owner
- `desired_state`, `observed_state`
- `usable`, `owned`, `adoptable`, `recreatable`
- `last_checked_at`, `last_action`, structured `problem`

This is primarily a derived observation backed by existing ownership manifests, resource state and audit records rather than a new standalone table.

### State transitions

`unknown → inspecting → adopted|recreated|failed`; `adopted|recreated → missing|failed`; `failed → inspecting`; any owned state may move to `deleting → deleted|failed`.

## Endpoint Backing

Derived classification used by connection operations.

- endpoint resource and port identity
- backing kind and owner
- namespace name when applicable
- host interface and peer interface when applicable
- create/inspect/delete capabilities
- current attachment and master membership

Plain bridges never carry a namespace name. Namespace operations are illegal when the backing kind is a host bridge.

## VLAN Membership

- network object ID and port name
- desired `pvid`
- desired `tagged_vlans`
- observed `pvid`, `untagged_vlans`, `tagged_vlans`
- `matches_desired`, `observed_at`, structured mismatch problem

Validation rejects VLAN IDs outside 1–4094, duplicates, a zero PVID when an access VLAN is required, and contradictory untagged/tagged membership.

## Forwarding Observation

- object or Docker node identity
- desired and observed IPv4 forwarding
- desired and observed IPv6 forwarding
- desired and observed routes
- interface addresses and link state
- timestamp and structured problem

An object is not healthy when required forwarding or routes are absent even if its namespace exists.

## Device Role Assignment

Stored within existing node/interface configuration.

- node ID and interface ID
- role: management, LAN, WAN, trunk or client-facing
- expected address family and optional address/gateway
- cable state, guest readiness, management reachability and data-path result

Role metadata is descriptive and must not expose credentials.

## Traffic Workload

Durable entity stored in `traffic_workloads`.

- `id`, `laboratory_id`, `name`, `revision`
- source kind/resource/interface
- protocol: ICMP, HTTP or DNS
- destination address/URL/name and address family
- interval, timeout, enabled lifecycle and output limit
- `desired_state`: running or stopped
- `observed_state`: queued, starting, running, stopping, stopped or failed
- total attempts, successes, failures, bytes received
- first/last success, last attempt, last structured error
- created/updated/finished timestamps

### State transitions

`stopped → starting → running`; `running → stopping → stopped`; `starting|running|stopping → failed`; `failed → starting|stopped`; deletion is allowed from stopped or failed and first stops a running workload.

## Traffic Workload Protocol Aggregate

Bounded aggregate associated with a workload.

- protocol and address family
- attempt, success, failure and received-byte totals
- current consecutive failure count
- last latency and last result time
- last bounded diagnostic message

No response body, DNS payload or packet payload is persisted.

## Network Path Observation

Derived acceptance/diagnostic result.

- source, destination and address family
- ordered expected connection IDs
- forward and return route summaries
- workload or probe result
- first failing resource/connection when known
- observed time and evidence reference

## Relationships

- A laboratory owns many runtime resources, VLAN memberships, role assignments, workloads and path observations.
- A network object owns many ports and VLAN memberships.
- A connection resolves two endpoint backings.
- A traffic workload uses one source endpoint and may correlate with many Traffic Filter observations and connection IDs.
- Recovery outcomes update actual state and publish ordered events without changing resource placement.
