# Data Model: First-Class Network Object Links and Docker Routes

## NetworkObjectLink

| Field | Type | Rules |
|---|---|---|
| `id` | ID | Stable, server-generated |
| `laboratory_id` | ID | Both endpoints must belong to this active laboratory |
| `endpoint_a` | NetworkObjectEndpoint | Stable orientation used for direction |
| `endpoint_b` | NetworkObjectEndpoint | Must differ from endpoint A owner |
| `revision` | integer | Starts at 1; increments on durable mutation |
| `desired_state` | enum | `connected`, `deleted` |
| `actual_state` | enum | `pending`, `connecting`, `connected`, `disconnecting`, `failed` |
| `last_error` | Problem/null | Structured actionable failure |
| `created_at`, `updated_at` | timestamp | UTC audit fields |

### NetworkObjectEndpoint

| Field | Type | Rules |
|---|---|---|
| `object_id` | ID | Existing network object in same laboratory |
| `port_name` | string | Valid connectable Linux interface name for that object |

Endpoint A/B ordering is persisted and stable. It is not recomputed from display names. Parallel links
are separate rows with separate endpoint port pairs.

## TopologyEndpointReservation

| Field | Type | Rules |
|---|---|---|
| `laboratory_id` | ID | Reservation scope |
| `owner_type` | enum | `node`, `network_object` |
| `owner_id` | ID | Endpoint owner |
| `port_name` | string | Canonical port key |
| `resource_type` | enum | `node_link`, `attachment`, `network_object_link` |
| `resource_id` | ID | Resource holding the reservation |

Primary key: `(laboratory_id, owner_type, owner_id, port_name)`. Reservation creation and link durable
creation occur in one transaction. Deletion releases reservations only after runtime deletion succeeds
or reconciliation confirms absence.

## RuntimeObservationLocator

Application-to-runtime value object; never exported as topology configuration.

| Field | Type | Rules |
|---|---|---|
| `resource_type` | enum | `interface`, `link`, `network_object_link` |
| `resource_id` | ID | Durable source identity |
| `namespace_name` | string/null | Owned namespace name, resolved at execution time |
| `interface_name` | string | Interface inside host or namespace |
| `orientation` | enum/null | `endpoint_a`, `endpoint_b`, or not applicable |

For object links, capture resolves endpoint A and observes one interface only.

## Capture Changes

Existing capture records retain their bounded lifecycle fields and add/accept:

- `source_type = network_object_link`
- `source_id = <object-link-id>`
- `completion_reason = link_deleted` when deletion terminates an active capture
- runtime locator is resolved, not persisted as exported intent

## TrafficFilter Changes

| Field | Type | Rules |
|---|---|---|
| `network_object_link_ids` | ID[] | Explicit object-link scope; distinct from standard `link_ids` |

### TrafficObservation Attribution

| Field | Type | Rules |
|---|---|---|
| `resource_type` | enum | `link`, `network_object_link`, `interface` |
| `resource_id` | ID | Exact traversed topology resource |
| `direction` | enum | `a_to_b`, `b_to_a`, `ambiguous`, or existing interface direction |
| `first_seen_at`, `last_seen_at` | timestamp | Observation time window |
| `packets`, `bytes` | integer | Monotonic within the observation/session |

Compatibility fields such as existing `link_id` remain populated for standard links during migration.
The UI keys highlights by `(resource_type, resource_id)` so unused parallel links remain unmarked.

## DockerStaticRoute

Stored as an ordered child of `NodeNetworkInterfaceSettings`.

| Field | Type | Rules |
|---|---|---|
| `destination` | CIDR | Canonical IPv4 or IPv6 prefix; default routes use `0.0.0.0/0` or `::/0` |
| `gateway` | IP/null | Same family as destination; optional for valid on-link route |
| `metric` | integer/null | Nonnegative; absent uses platform default |

The containing interface supplies egress identity. Duplicate canonical route keys and conflicting
same-prefix declarations are rejected. Gateway-based routes require the gateway to be reachable through
an address configured on the containing interface unless explicitly supported by a future on-link flag.

## ManagedDockerRoute

Durable or reconstructable ownership record used to remove stale managed routes safely.

| Field | Type | Rules |
|---|---|---|
| `node_id` | ID | Docker node owner |
| `interface_id` | ID | Interface owner |
| `destination` | CIDR | Canonical prefix |
| `gateway` | IP/null | Applied gateway |
| `metric` | integer/null | Applied metric |
| `configuration_revision` | integer | Revision that declared the route |

Only routes proven to be NetLab-managed may be removed during exact-set reconciliation.

## State Transitions

### Object Link

```text
create accepted -> pending -> connecting -> connected
                                 |              |
                                 v              v
                               failed <--- disconnecting -> deleted
```

- Retry may move `failed` back to `connecting` or `disconnecting` according to desired state.
- Runtime absence with desired `connected` triggers ensure.
- Owned runtime presence with no durable owner triggers orphan cleanup after recovery classification.

### Docker Readiness with Routes

```text
container running -> endpoints reconciling -> routes applying -> ready
                                              |
                                              v
                                            failed
```

The node cannot report ready until every declared route is confirmed. A route-specific problem includes
interface, destination, gateway, and a stable error code without exposing secrets.

## Persistence Migration

1. Add timestamps if absent and preserve existing `network_object_links` IDs and endpoint ordering.
2. Create canonical endpoint reservation storage and backfill existing attachments and object links in
   one migration transaction; abort on collisions instead of silently choosing a winner.
3. Add typed route JSON/schema support or normalized route rows consistent with the repository's current
   node-settings persistence strategy.
4. Add object-link scope/attribution fields to traffic filters and observations additively.
5. Add indexes for laboratory listing, resource attribution, active captures, and reservation lookup.

## Export and Import

- Export link IDs, endpoint object IDs/port names, desired intent, and Docker route declarations.
- Do not export actual state, namespace names/PIDs, kernel interface names, task internals, capture
  payloads, or host-specific ownership markers.
- Import validates all endpoint references first, reserves ports transactionally, and then queues runtime
  reconciliation. ID remapping updates both endpoints and observation scopes consistently.
