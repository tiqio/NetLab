# Compatibility History

## Reviewed Changes

- `74dadf3` established the original namespace and lightweight network-object runtime. Namespace readiness was defined by an in-namespace loopback probe, but deletion trusted namespace list membership.
- `3813579`, `c3338a1`, `2152d65` established direct network-object links, observation and recovery.
- `b74f6ce` and `4045f73` prevented deletion resurrection and recovered interrupted object-link deletion.
- `8961c3a` added compensation for failed unified connection runtime creation.
- `1162f4d` and `90fbea8` added Traffic Filter history and durable session statistics.

## Constraints Preserved

- Durable desired state and explicit ownership remain authoritative.
- Existing resource IDs, coordinates, connection categories and outbox ordering remain stable.
- Recovery must not adopt unowned resources or silently replace a resource of another kind.
- Deletion remains revisioned and idempotent and must release durable port reservations.
- Traffic Filter counters remain independent from visual highlight decay.
- Host commands continue to use validated argv rather than interpolated shell strings.

## Root Compatibility Risks

- Namespace list membership is not sufficient when a bind reference is stale or invalid.
- L2 configuration currently skips ports that do not exist yet, so later attachment leaves VLAN 1.
- Shared object-link cleanup assumes both objects are namespace-backed, which is false for a plain bridge.
- Recovery can publish connected/active projections before runtime dependencies are usable.
