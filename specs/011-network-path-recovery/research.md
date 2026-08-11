# Research: Network Path Recovery and Validation

## Decision 1 — Treat Named Network Namespaces as Reconstructable Runtime State

**Decision**: Durable desired state and ownership records remain authoritative. A named namespace is adopted only after an in-namespace probe succeeds and ownership/kind match; an invalid owned reference is deleted or quarantined and recreated.

**Rationale**: The target contains names listed under `/run/netns` that fail every namespace operation. Name presence alone is therefore not proof of usable backing.

**Alternatives considered**: Trust `ip netns list` output; mark all listed names active; require operators to delete and rebuild the laboratory manually.

## Decision 2 — Recover in Dependency Order

**Decision**: Reconcile object backing before ports, ports before attachments/object links, and endpoints before declaring unified connections connected.

**Rationale**: Current L2 VLAN application runs before later-created ports and silently skips them, while links can remain pending against failed L3 backing.

**Alternatives considered**: Retry all resources concurrently without dependency ordering; rely on periodic retries alone.

## Decision 3 — Classify Endpoint Backing Explicitly

**Decision**: Every connection endpoint resolves to a backing capability such as namespace port, host bridge port, QEMU tap or Docker veth. Create, inspect, compensate and delete dispatch through that capability.

**Rationale**: The known stuck bridge-to-L3 link failed because cleanup treated a plain bridge as namespace-backed.

**Alternatives considered**: Special-case the single bridge ID; infer backing from interface names during cleanup.

## Decision 4 — Make Observed State Evidence-Based

**Decision**: Network-object diagnostics return namespace usability, expected interfaces, forwarding, routes and VLAN membership. Actual state is healthy only when required observations match desired state.

**Rationale**: Objects currently report active while namespace handles are invalid, and desired IPv6 forwarding can differ from observed forwarding.

**Alternatives considered**: Preserve optimistic active state and expose diagnostics only on demand; use process/service status as the health signal.

## Decision 5 — Reconcile VLAN Membership After Port Attachment

**Decision**: Port attachment triggers idempotent L2 membership reconciliation. Desired PVID/tagged sets are cleared and reapplied, then read back for comparison.

**Rationale**: Initial switch configuration skips absent ports, leaving runtime VLAN 1 despite desired PVID 10/20/30.

**Alternatives considered**: Require deleting/recreating the switch after every connection; apply VLAN only from the UI.

## Decision 6 — Add Explicit Docker Forwarding Settings

**Decision**: Docker router-style nodes may declare IPv4 and IPv6 forwarding in their network settings. The host runtime applies and observes these settings inside the container network namespace.

**Rationale**: The service router has IPv4 forwarding enabled but IPv6 forwarding disabled, preventing a complete service-network IPv6 path.

**Alternatives considered**: Depend on image startup scripts; run untracked manual target commands; replace the Docker router with another QEMU router.

## Decision 7 — Persist Traffic Workload Aggregates

**Decision**: Model ICMP, HTTP and DNS generators as durable workloads with source capability, interval, lifecycle, success/failure totals and bounded last error. Do not persist every payload or attempt.

**Rationale**: A running shell loop currently appears healthy while all intended application exchanges fail. Aggregate outcomes are needed across restart without unbounded storage.

**Alternatives considered**: Keep shell loops as opaque node commands; infer generator success solely from Traffic Filter counts; persist every packet or response body.

## Decision 8 — Keep Vendor Guest Automation Bounded

**Decision**: NetLab owns attachments, roles, diagnostics and evidence. Device-specific configuration uses existing supported console/bootstrap mechanisms and records readiness prerequisites; no generic proprietary CLI automation framework is introduced.

**Rationale**: Appliance syntax and licensing vary, while the immediate gap is honest topology and repeatable path proof.

**Alternatives considered**: Automate all vendor CLIs; consider link-connected state sufficient; exclude vendor paths from acceptance.

## Decision 9 — Preserve Existing Traffic Filter Durability

**Decision**: Reuse durable filter packet/byte/fingerprint storage added by migration 15. Workload status and filter observations are correlated by time, source and selected connection IDs; visual decay remains presentation-only.

**Rationale**: Existing counters already satisfy durable observation needs once successful traffic actually reaches the selected path.

**Alternatives considered**: Reset counters with every animation window; duplicate packet counters in workload storage.
