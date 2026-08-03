# Implementation Plan: First-Class Network Object Links and Docker Routes

**Branch**: `main` | **Date**: 2026-08-03 | **Spec**: `specs/005-network-object-links-routes/spec.md`
**Input**: Feature specification from `specs/005-network-object-links-routes/spec.md`

## Summary

Promote network-object-to-network-object connections into durable, revisioned topology resources and
replace the current bridge-plus-two-veth runtime with one owned veth pair whose endpoints are moved
directly into the two network-object namespaces. Extend capture and Traffic Filter addressing with a
namespace-aware locator so these links remain selectable, observable, directionally highlighted, and
live-deletable. Complete Docker static-route support by carrying typed IPv4/IPv6 route declarations
through domain, HTTP, MCP, persistence, frontend settings, reconciliation, readiness, and recovery.

## Technical Context

**Language/Version**: Go 1.26.x; TypeScript 5.8.x; Vue 3.5.x
**Primary Dependencies**: Gin 1.12.x, modernc SQLite, Docker Engine Go SDK, Linux netlink/iproute2
adapters, packet capture tools, Vue, Pinia, ECharts, shadcn-vue, Vitest, Playwright
**Storage**: SQLite in WAL mode with migrations, durable tasks, revisions, audit records, and ordered
outbox events; retained packet artifacts remain under the existing bounded artifact policy
**Testing**: Go unit and contract tests, privileged Linux integration tests, recovery and resource-leak
tests, Vitest component/store tests, and Playwright browser journeys
**Target Platform**: Single x86_64 Linux host with cgroup v2, Docker, network namespaces, veth,
iproute2/netlink, nftables, and packet-capture tooling; production validation target `10.72.1.7`
**Project Type**: Single Go web service plus embedded Vue SPA and versioned MCP surface
**Performance Goals**: Shared create/delete convergence within 2 seconds for 99% of operations;
Traffic Filter link/direction visibility within 500 ms for 95% of observations; no duplicate capture
accounting from observing both ends of a direct pair
**Constraints**: Live mutation without stopping nodes, one occupied topology endpoint per port, no
shell interpolation of untrusted values, no manual `nsenter` route setup, no cluster behavior, and no
source edits on the target host
**Scale/Scope**: One trusted single-host deployment, multiple simultaneous browser/API/MCP clients,
parallel object links on distinct ports, IPv4 and IPv6 Docker routes, and compatibility with existing
node links, attachments, NAT, capture, and Traffic Filter workflows

## Constitution Check

*GATE: Passed before research and passed again after design.*

- **Shared state — PASS**: `NetworkObjectLink` and Docker interface routes are server-authoritative,
  revisioned SQLite state. Mutations require expected revision where applicable, support idempotency,
  return durable tasks, and publish ordered outbox events consumed by every client.
- **Control parity — PASS**: Create/delete/inspect/capture/filter link actions and Docker route edits
  are represented in the same application handlers and exposed through HTTP, MCP, and SPA adapters.
  Runtime work returns typed task state rather than browser-local completion assumptions.
- **Runtime scope — PASS**: The design uses only approved single-host network namespaces, one veth pair
  per object link, Docker namespace configuration, and existing capture primitives. It adds no cluster,
  external queue, or unsupported emulator.
- **Live operations — PASS**: Links reconcile while objects and attached nodes remain running. Capture
  uses one namespace endpoint, observes both directions, and stops with `link_deleted` during deletion.
  Traffic observations identify the exact link and decay through the existing visualization policy.
- **Safety and recovery — PASS**: Every veth endpoint, namespace locator, capture worker, task, and
  managed route has explicit ownership. Creation/deletion are idempotent, partial work is reconciled,
  startup adopts only owned resources, and cleanup never matches unrelated host interfaces by shape.
- **Verification — PASS**: The plan includes domain, migration, repository, HTTP/MCP contract, frontend,
  privileged dataplane, recovery, failure-path, and leak tests, followed by a repeatable target-host
  validation matrix.
- **Image and secret hygiene — PASS**: No image, credential, bootstrap secret, packet payload, or target
  password enters source control. Test images use approved public references pinned by digest; capture
  artifacts remain outside Git and are bounded by existing retention controls.
- **Local-first delivery — PASS**: Five independently testable implementation milestones are committed
  locally. Deployment is built from a clean commit, records commit SHA, artifact digest, migration
  state, deployment time, and target results, and rolls back to a previously recorded artifact.

## Project Structure

### Documentation (this feature)

```text
specs/005-network-object-links-routes/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── events.md
│   ├── mcp-tools.md
│   └── openapi-delta.yaml
└── tasks.md
```

### Source Code (repository root)

```text
cmd/netlab/
internal/
├── domain/                  # Link, endpoint, route, capture, and observation types
├── app/
│   ├── command/             # Shared revisioned/idempotent mutations
│   ├── query/               # Authoritative topology and diagnostics reads
│   ├── reconcile/           # Link, capture, filter, Docker route, recovery orchestration
│   └── ports/               # Runtime and repository boundaries
├── api/
│   ├── http/                # Versioned HTTP contracts
│   ├── mcp/                 # Equivalent typed MCP tools
│   └── stream/              # Ordered shared-state updates
├── runtime/
│   ├── linuxnet/            # Direct veth pair and namespace route operations
│   ├── docker/              # Endpoint reconciliation and readiness
│   ├── capture/             # Namespace-aware packet capture
│   └── ownership/           # Adoption and orphan cleanup
└── store/sqlite/            # Migrations, repositories, outbox, audit, tasks

web/src/
├── api/                     # Generated and handwritten contract adapters
├── stores/                  # Shared topology/task/filter state
└── features/topology/       # Object-link rendering, inspector, capture, filters, routes

tests/
├── contract/
├── integration/
├── recovery/
├── security/
└── e2e/
```

**Structure Decision**: Extend the existing Go module and Vue workspace. Domain and application
packages stay independent of Gin, SQLite, Docker, and host commands; privileged namespace operations
remain in runtime adapters. HTTP, MCP, and SPA mutations converge through the same application
services and durable task model.

## Design Phases

### Phase 0 — Research

Resolve runtime topology, endpoint exclusivity, capture addressing, direction attribution, deletion
semantics, route validation, and recovery decisions in `research.md`. No unresolved clarification is
allowed before implementation tasks are generated.

### Phase 1 — Durable Model and Contracts

1. Add a canonical endpoint-occupancy representation that prevents a port from being reused regardless
   of whether it appears as endpoint A or B or is occupied by another attachment type.
2. Model a topology observation target as `{resource_type, resource_id, namespace, interface}` without
   persisting transient namespace PIDs or host-specific names in exports.
3. Add typed Docker route declarations to node interface settings and validate family, destination,
   gateway, interface, metric, duplicates, conflicts, and ambiguous egress.
4. Extend HTTP, MCP, events, generated frontend types, export/import, and audit payloads as specified in
   `contracts/`.

### Phase 2 — Runtime and Recovery

1. Reconcile one deterministic, owned veth pair per object link, move one end into each namespace, and
   assign the requested port names without a host bridge.
2. Make ensure/delete/adopt operations idempotent and recover partial pairs, stale names, and durable
   state interruptions without touching unowned resources.
3. Resolve capture/filter requests to endpoint A's namespace and interface, capturing both directions
   exactly once. Stop affected active captures before final deletion and preserve retained metadata.
4. Apply the exact managed Docker route set on every endpoint reconciliation before readiness; remove
   stale previously managed routes while preserving connected, kernel, and explicitly unmanaged routes.

### Phase 3 — Control Surfaces and UX

1. Expose identical object-link lifecycle, inspector, capture, Traffic Filter, and Docker route settings
   through HTTP, MCP, and frontend actions.
2. Render parallel links independently, label them `object:port ↔ object:port`, display desired/actual
   state and failures, and remove links immediately from shared state when deletion commits.
3. Add route editing with family-aware validation and actionable backend errors; stopped nodes may be
   edited, while running-node changes follow existing lifecycle restrictions.

### Phase 4 — Verification and Delivery

1. Run focused tests after each milestone, then full Go, frontend, contract, E2E, recovery, and leak
   suites using the validation order in the base quickstart.
2. Build from a clean committed worktree and record `git rev-parse HEAD`, SHA-256 artifact digest,
   migration version, UTC deployment time, and target test report.
3. Deploy only that artifact to `10.72.1.7`; validate multi-object forwarding, parallel-link isolation,
   live deletion, capture, direction, restart recovery, Docker IPv4/IPv6 routes, and cleanup.
4. On failure, redeploy the prior recorded artifact; all fixes return to the local worktree and receive
   tests plus a new focused commit before another deployment.

## Local Milestones

| Milestone | Deliverable | Minimum local gate | Required commit intent |
|---|---|---|---|
| M1 | Schema, domain types, occupancy and route validation | domain/repository/migration/contract tests | `feat: model object links and docker routes` |
| M2 | Direct veth lifecycle, adoption, cleanup, route application | privileged integration, recovery, leak tests | `feat: reconcile object links and docker routes` |
| M3 | Capture and Traffic Filter object-link support | capture/filter unit, contract, direction, deletion tests | `feat: observe network object links` |
| M4 | HTTP/MCP/frontend parity and topology UX | generated contract, Vitest, Playwright shared-client tests | `feat: expose object link controls` |
| M5 | End-to-end validation and deploy tooling/evidence | full local quality gate and clean artifact build | `test: validate object links and docker routes` |

## Agent Context Update

The installed Spec Kit distribution does not contain `.specify/scripts/bash/update-agent-context.sh`
or an equivalent CLI command. The equivalent update is applied manually to repository-root `AGENTS.md`:
the active feature points to this specification, plan, and contracts while retaining the applicable
Go/Vue stack, architectural boundaries, target host, test obligations, and local-first delivery rules.

## Complexity Tracking

No constitution violations require justification. The namespace-aware capture locator and canonical
endpoint occupancy are required invariants for first-class links, not new deployment tiers or services.
