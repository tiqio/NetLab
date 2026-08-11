# Implementation Plan: Network Path Recovery and Validation

**Branch**: `011-network-path-recovery` | **Date**: 2026-08-11 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/011-network-path-recovery/spec.md`

## Summary

Repair the component-matrix laboratory by making lightweight network-object recovery truthful and repeatable, dispatching connection cleanup by endpoint ownership, reconciling VLAN membership after ports exist, and proving complete IPv4/IPv6 paths through VyOS and vendor appliances. Add durable traffic workloads so successful ICMP, HTTP and DNS exchanges can be distinguished from failed attempts and correlated with Traffic Filter counters and topology highlights.

## Technical Context

**Language/Version**: Go 1.26.x; TypeScript with Vue 3

**Primary Dependencies**: Gin 1.12.x, SQLite WAL, Linux `ip`/`bridge`/network namespaces, Docker Engine SDK, QEMU QMP/QGA, nftables, dumpcap/tcpdump, Pinia, ECharts

**Storage**: Existing SQLite repositories and outbox; one migration for durable traffic-workload definitions and aggregate outcomes; network-object desired configuration remains JSON-backed

**Testing**: Go unit/contract/integration tests, Vitest component/store tests, Playwright acceptance, privileged Linux runtime tests, restart/recovery and resource-leak gates

**Target Platform**: Single x86_64 Linux host with KVM/QEMU, Docker, cgroup v2, netlink, nftables and capture tools; authoritative deployment at `10.72.1.7`

**Project Type**: Go web service plus Vue single-page application and MCP control surface

**Performance Goals**: Recover valid laboratory runtime resources within 30 seconds; update traffic-workload aggregates within one observation interval; keep topology interaction responsive while statistics stream

**Constraints**: No shell interpolation of untrusted values; preserve resource ownership and coordinates; no direct source edits on target; support partial recovery; retain existing QEMU resource admission; no proprietary assets or credentials in source control

**Scale/Scope**: One authoritative host, mixed laboratories with dozens of nodes/objects and connections, at least six simultaneous QEMU nodes, repeated 10-restart recovery and 20-cycle cleanup validation

## Constitution Check

*GATE: Passed before research and passed again after design.*

- **Shared state — PASS**: SQLite remains authoritative for desired state, revisions, task envelopes, aggregate workload statistics and ordered outbox events. Runtime inspection determines actual state; invalid backing cannot remain reported healthy.
- **Control parity — PASS**: Reconcile retry, diagnostics, VLAN configuration and traffic workloads have matching UI, HTTP and applicable MCP contracts. Mutations use revisions, idempotency keys and durable task results.
- **Runtime scope — PASS**: Design stays within the approved single-host QEMU, Docker, bridge, namespace, routing, filtering and capture primitives.
- **Live operations — PASS**: Existing hot link/NIC behavior remains; repair adds endpoint-aware rollback, observed VLAN state and Traffic Filter correlation without changing console ownership.
- **Safety and recovery — PASS**: Recovery validates ownership before adoption, recreates only owned resources, isolates partial failures, bounds commands and traffic output, supports cancellation, and audits adoption/recreation/deletion.
- **Verification — PASS**: Unit, contract, privileged adapter, restart, failure-injection, concurrency, target-browser and leak tests are defined in `quickstart.md`.
- **Image and secret hygiene — PASS**: Existing target images are referenced only by non-secret identifiers. Credentials, proprietary images and packet payloads remain outside source control; evidence is redacted and bounded.
- **Local-first delivery — PASS**: Four focused local milestones precede clean-commit builds and deployment. Candidate SHA, digest, migration, deployment time, target results and rollback backup are recorded.

## Project Structure

### Documentation (this feature)

```text
specs/011-network-path-recovery/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── openapi-delta.yaml
│   ├── mcp-tools.md
│   └── ui-contract.md
├── checklists/requirements.md
└── tasks.md
```

### Source Code (repository root)

```text
internal/
├── domain/
│   ├── network_object.go
│   ├── topology_connection.go
│   ├── runtime_observation.go
│   └── traffic_workload.go
├── app/
│   ├── command/
│   │   ├── topology_connection*.go
│   │   └── traffic_workload.go
│   └── reconcile/
│       ├── boot.go
│       ├── recovery*.go
│       ├── network_objects.go
│       ├── dataplane.go
│       ├── topology_connections.go
│       ├── traffic_filters.go
│       └── traffic_workloads.go
├── runtime/
│   └── linuxnet/
│       ├── namespace.go
│       ├── endpoint.go
│       ├── link.go
│       ├── switch_l2.go
│       ├── switch_l3.go
│       └── traffic_workload.go
├── api/
│   ├── http/
│   └── mcp/
└── store/sqlite/
    ├── migrations/
    └── traffic_workload_repository.go

web/src/
├── api/
├── stores/
└── features/topology/
    ├── TopologyWorkspace.vue
    ├── NetworkObjectDiagnostics.vue
    ├── TrafficWorkloadPanel.vue
    └── topologyVisualSemantics.ts

tests/
├── contract/
├── integration/
├── recovery/
├── security/
└── e2e/
```

**Structure Decision**: Preserve the existing domain/application/runtime/API/store layering and the Vue topology feature. Linux command details remain behind runtime adapters; domain state and contracts do not depend on Gin, SQLite or command execution.

## Design Phases

### Phase A — Truthful Runtime Recovery and Cleanup

- Add inspectable runtime-backing observations for namespace-backed objects and plain bridges.
- Validate namespace usability, ownership and expected object kind before adoption.
- Recreate stale owned namespaces in dependency order: object backing, endpoint ports, attachments/object links, then observed connection state.
- Dispatch deletion and compensation by endpoint backing kind so bridges never enter namespace cleanup.
- Convert exhausted pending/disconnecting resources to structured failed outcomes and permit idempotent retry or deletion.
- Verify restart, concurrent mutation, failure isolation and leak behavior before the milestone commit.

### Phase B — VLAN and Dual-Stack Dataplane Convergence

- Reconcile L2 port master, PVID and tagged membership whenever an endpoint appears, not only during initial switch creation.
- Read back observed VLAN membership and compare it with desired membership in diagnostics and state evaluation.
- Reconcile L3 forwarding and routes after interfaces are present; expose requested versus observed IPv4/IPv6 forwarding and route failures.
- Add optional Docker-node forwarding settings for router workloads, applied through the existing safe runtime boundary.
- Repair the acceptance topology: BusyBox service attachment, VyOS transit/downstream paths, core/DMZ/management return routes and one VLAN 10/20 trunk.

### Phase C — Vendor Paths and Durable Traffic Workloads

- Represent vendor interface roles and readiness as configuration/diagnostic metadata without automating proprietary guest features.
- Attach management and data networks, then prove FancyWAN/FortiGate and Ruijie client paths with supported guest configuration procedures.
- Add durable ICMP, HTTP and DNS traffic workloads with source adapter capability checks, interval, cancellation, success/failure aggregates and bounded errors.
- Correlate successful workload traffic with existing durable Traffic Filter observations; keep highlight decay independent of counters.

### Phase D — Recovery, Acceptance and Deployment

- Run 10 service restarts, 20 failure-injection cleanup cycles, dual-stack path tests, VLAN isolation/trunk tests, vendor path probes and a 10-minute traffic observation.
- Validate UI/API/MCP parity, structured warnings, task cancellation, revision conflicts and browser recovery.
- Commit each independently testable milestone before building a clean candidate.
- Deploy to `10.72.1.7`, record source SHA, artifact digest, migration, deployment time, target evidence and rollback directory, and restore the component-matrix laboratory without changing coordinates.

## Persistence and Transactions

- Existing network-object and connection rows remain authoritative; no migration is needed for namespace recovery or VLAN observation.
- Add `traffic_workloads` for definition, lifecycle, aggregate attempts/success/failure, protocol totals, timestamps and last structured error.
- Traffic-workload create/update/delete and associated outbox events commit atomically with laboratory revision changes where topology-visible state changes.
- Reconciliation writes actual state and structured problems using compare-and-swap revisions or dedicated observation updates, avoiding accidental desired-state mutation.
- Startup recovery resumes running/stopping traffic workloads idempotently and finalizes orphaned task reservations using existing durable-task conventions.

## Privilege and Command Boundaries

- All namespace, interface, bridge, VLAN, route and sysctl operations use argv-based executors with validated owned names and numeric VLAN/address inputs.
- Recovery may remove only resources whose owner identity maps to a durable NetLab resource or an explicitly quarantined stale owner.
- Traffic workloads run only through capability-specific adapters with fixed executable families, bounded arguments, timeouts and output limits.
- Captures and workload evidence remain owned by the laboratory and follow existing retention and cleanup policy.

## Testing Strategy

- **Domain/unit**: state transitions, VLAN validation, workload validation, endpoint backing classification and aggregate counters.
- **Contract**: OpenAPI/MCP parity, revisions, idempotency, structured problems, diagnostics and event ordering.
- **Runtime adapter**: invalid namespace handles, recreate/adopt, late port VLAN application, IPv6 forwarding, bridge cleanup and workload command bounds.
- **Privileged integration**: real namespaces, veths, bridges, tagged VLANs, routes, nftables, Docker netns and capture/filter attribution.
- **Recovery**: service restart during create/delete/workload execution, stale `/run/netns` entries, partial endpoint recovery and host restart policy.
- **Frontend**: diagnostics truthfulness, retry/delete warnings, workload success/failure display, non-zero filters and highlight decay.
- **Leak gate**: namespace/interface/bridge/rule/socket/process/capture/workload baselines restored after every temporary laboratory.

## Agent Context Update

The installed Spec Kit distribution has no `.specify/scripts/bash/update-agent-context.sh` or equivalent command. Update the repository guidance manually so the active feature points to `specs/011-network-path-recovery/`, retains 009/010 as interaction dependencies, and adds the recovery-honesty, endpoint-aware cleanup, observed VLAN and durable traffic-workload rules.

## Complexity Tracking

No constitution exception is required. The durable traffic-workload entity is justified by the specification's requirement to distinguish a running generator from successful application exchanges and to expose restart-safe aggregate results across UI, API and MCP.
