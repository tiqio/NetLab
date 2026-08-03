# Research: First-Class Network Object Links and Docker Routes

## 1. Runtime Shape for Object Links

**Decision**: Represent each object link at runtime as one Linux veth pair. Move endpoint A directly
into network object A's namespace and endpoint B directly into object B's namespace, then rename each
end to its durable port name.

**Rationale**: A point-to-point network-object link does not need a host bridge. One pair has fewer
resources, clearer ownership, no accidental third attachment point, no duplicate capture surface, and
matches the requested network semantics.

**Alternatives considered**:

- Host bridge plus two veth pairs: already partially implemented, but creates five runtime resources
  for a two-endpoint link and complicates capture, cleanup, and directional attribution.
- One shared bridge for all object links: rejected because it collapses distinct links into one L2
  domain and breaks parallel-link isolation.

## 2. Endpoint Occupancy

**Decision**: Introduce canonical durable endpoint occupancy keyed by laboratory, owner type, owner ID,
and port name. Link creation reserves both endpoints transactionally before runtime work begins.

**Rationale**: Separate uniqueness constraints for endpoint A and endpoint B do not prevent the same
port from appearing once on each side. A canonical record also allows object links and existing node
attachments to share one exclusivity rule.

**Alternatives considered**:

- Cross-side application query only: vulnerable to concurrent transactions unless serialized with an
  explicit durable reservation.
- Lexically sorting endpoint A/B: improves stable ordering but does not cover occupancy by other
  attachment resource types.

## 3. Lifecycle, Concurrency, and Tasks

**Decision**: Persist desired state before runtime work, enforce expected revision on update/delete,
accept idempotency keys, and execute ensure/delete as durable cancellable tasks. Publish the state and
outbox event in the same transaction.

**Rationale**: Every browser, API client, and MCP client must observe the same final state. Durable
tasks allow retries and recovery when the service stops between database and host changes.

## 4. Namespace-Aware Capture

**Decision**: Replace the capture worker's plain host-interface input with a structured runtime locator
containing stable source identity plus an optional namespace and interface. For an object link, capture
only endpoint A's interface inside object A's namespace.

**Rationale**: A direct veth pair has no host bridge. One endpoint observes ingress and egress and avoids
double-counting packets. The durable capture remains associated with the object-link ID rather than a
transient namespace PID or kernel-generated name.

**Alternatives considered**:

- Keep a host-side veth end solely for capture: impossible for a direct two-namespace pair without an
  extra forwarding resource.
- Capture both endpoints: duplicates packet and byte counts and can produce conflicting observations.

## 5. Traffic Filter Scope and Direction

**Decision**: Add explicit `network_object_link_ids` filter scope. Observations identify resource type
and resource ID while retaining the existing standard-link field for compatibility. Endpoint A is the
stable orientation reference: traffic leaving A is `a_to_b`, traffic entering A is `b_to_a`.

Packet direction should use capture metadata when the selected Linux capture implementation exposes
reliable ingress/egress direction. Otherwise, resolve direction from learned endpoint MAC ownership;
packets that cannot be resolved are marked `ambiguous` and never assigned an invented direction.

**Rationale**: Stable orientation makes frontend arrows deterministic and parallel links independently
addressable. Explicit ambiguity is safer than visually reversing unknown traffic.

## 6. Deletion During Observation

**Decision**: Mark the link disconnecting, stop captures and internal filter workers that target it,
record capture completion reason `link_deleted`, delete the owned veth resources, release endpoint
reservations, then durably remove or tombstone the link according to existing event semantics.

**Rationale**: Deletion must not hang on a capture worker, retained data must survive, and ports must not
be reusable until the runtime pair is absent.

## 7. Recovery and Ownership

**Decision**: Derive deterministic temporary veth names from link ID and label owned resources using the
existing ownership mechanism. Recovery enumerates durable links, adopts exact owned pairs, completes
pending moves/renames, and removes only orphaned resources carrying NetLab ownership evidence.

**Rationale**: Kernel interface names and namespace PIDs are transient. Durable identity must remain the
link ID and endpoint intent, with exact ownership checks preventing unrelated host changes.

## 8. Docker Route Contract

**Decision**: Add ordered typed routes under each `NodeNetworkInterfaceSettings` entry:
`destination`, optional `gateway`, optional `metric`, and the containing interface as the egress owner.
The API and frontend no longer omit `network_interfaces` for Docker nodes.

**Rationale**: The runtime adapter already understands route-like JSON, but the typed domain and control
contracts currently discard it. Keeping routes on the interface makes egress unambiguous and preserves
the topology definition through create, settings, export, and import.

## 9. Route Validation and Reconciliation

**Decision**: Validate canonical CIDR destination, gateway family, known interface, nonnegative metric,
duplicates, conflicting same-prefix declarations, and gateway reachability through the selected
interface's configured subnet. Apply routes with argument-vector host operations inside the container
namespace on every endpoint reconciliation before readiness.

Maintain an owned managed-route set per node/interface. Replace the exact declared set and delete stale
previously managed routes, but never remove connected, kernel-created, Docker-managed, or otherwise
unowned routes.

**Rationale**: Repeated ensure calls become idempotent, stopped-node edits converge on next start, and
recovery does not depend on undocumented manual `nsenter` commands.

## 10. Compatibility

**Decision**: Add object-link source types and route fields additively. Existing node-interface links,
network attachments, NAT, host-interface capture, standard-link capture, and filter scopes retain their
current semantics. Export/import excludes transient runtime locators and packet payloads.

**Rationale**: The feature closes a missing topology primitive without redefining established resources.
