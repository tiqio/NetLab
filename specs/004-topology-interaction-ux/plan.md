# Implementation Plan: Topology Interaction UX

**Branch**: `004-topology-interaction-ux` | **Date**: 2026-07-27 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/004-topology-interaction-ux/spec.md`

## Summary

Replace the current chart-coupled topology gestures with a deterministic interaction controller for zoom, pan,
selection, node dragging, port-to-port connection, node-body port selection, disconnect, and atomic reconnect.
Persist confirmed node coordinates as revisioned shared laboratory state, keep viewport and link control points
browser-local, publish ordered placement events, and render locally bundled, license-tracked node symbols through
the existing ECharts graph surface. The connection journey deliberately follows the familiar EVE-NG model of a
discoverable connector handle, drag to a target node, interface selection, and explicit confirmation, while
removing account/session coupling and replacing client-side disconnect/connect with shared durable commands. All
runtime link mutations continue through application commands used by HTTP, MCP, and the SPA.

## Technical Context

**Language/Version**: Go 1.26.x; TypeScript 5.9.x; Vue 3.5.x

**Primary Dependencies**: Gin 1.12.x, SQLite WAL, existing application command/query services, ECharts 6.x,
Vue-ECharts 8.x, Pinia, Lucide Vue Next, Playwright

**Storage**: SQLite for shared revisioned node/network-object placements, tasks, events, and audit records;
browser storage for viewport, label density, reduced-motion preference, and manual link control points

**Testing**: Go unit/contract/integration/recovery tests; Vitest component/composable/state-machine tests;
Playwright pointer, wheel, keyboard, concurrency, refresh, failure, and target-host runtime acceptance

**Target Platform**: Single x86_64 Linux host; modern desktop Chromium with minimum supported viewport
1024×768 and standard mouse/trackpad input

**Project Type**: Go web service with Vue single-page application

**Performance Goals**: Visible input feedback within 100 ms for at least 95% of interactions; representative
100-node/200-link topology has no interaction freeze over 250 ms in at least 95% of measured windows; one
placement mutation is emitted per completed drag, not per pointer movement

**Constraints**: Preserve global shared resource/runtime state without authentication; node coordinates are
shared and revisioned; viewport and manual link routes are browser-local; no remote icon dependency; no shell or
runtime changes for pure layout operations; existing running-node link safety and cleanup guarantees remain

**Scale/Scope**: One host, one active topology canvas per browser tab, at least 100 nodes and 200 links per
representative acceptance topology, multiple concurrent browser/API/MCP clients

## Constitution Check

*GATE: PASS before research and PASS after design.*

- **Shared state**: Laboratories, nodes, interfaces, links, tasks, observed state, and confirmed placements are
  server-authoritative. Placement and reconnect mutations require revisions; ordered outbox events drive other
  clients and snapshot reload resolves gaps.
- **Control parity**: Placement batch update and atomic link reconnect are application commands exposed through
  HTTP and MCP. The SPA uses the same HTTP contracts. Placement is synchronous durable state; runtime reconnect
  returns a durable operation task with progress, terminal result, cancellation, and structured failure.
- **Runtime scope**: No new runtime type is introduced. Link reconnect continues to use the approved single-host
  QEMU, Docker, bridge, NAT, and namespace adapters.
- **Live operations**: Placement does not touch runtime state. Connect, disconnect, and reconnect work while
  nodes run; reconnect preserves the old link until the replacement path is accepted and restores it on failure.
- **Safety and recovery**: Drag commits are bounded and batched. Commands validate laboratory ownership,
  resource revisions, endpoint availability, idempotency keys, and coordinate bounds. Reconciliation and cleanup
  remove stale placements with deleted resources and recover interrupted reconnect tasks without orphan links.
- **Verification**: Add interaction-controller unit tests, API/MCP/event contract tests, SQLite transaction tests,
  target-host live reconnect tests, concurrent browser tests, restart recovery, and resource-leak assertions.
- **Image and secret hygiene**: Node symbols are local code-native SVG assets with recorded source/license; no
  proprietary device image, credential, packet payload, console output, or remote image URL is introduced.

## Project Structure

### Documentation (this feature)

```text
specs/004-topology-interaction-ux/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── topology-interaction.openapi.yaml
│   ├── events.md
│   ├── mcp-tools.md
│   └── ui-interaction.md
└── tasks.md
```

### Source Code (repository root)

```text
internal/domain/
internal/app/command/
internal/app/query/
internal/api/http/
internal/api/mcp/
internal/store/sqlite/
internal/runtime/

web/src/api/
web/src/features/topology/
web/src/composables/
web/src/stores/
web/src/types/
web/src/assets/topology/

tests/contract/
tests/integration/
tests/recovery/
tests/e2e/journeys/
tests/e2e/matrices/
```

**Structure Decision**: Extend the existing backend layers and topology feature directory. Interaction state and
geometry remain framework-independent TypeScript modules; Vue components adapt DOM/chart events. Shared placement
commands follow domain/application/store/API separation. Runtime reconnect reuses the existing link adapter and
durable task runner rather than introducing a frontend-only mutation path.

## Design Sequence

1. Add failing state-machine, geometry, placement contract, concurrency, and reconnect rollback tests.
2. Introduce shared `TopologyPlacement` domain/query/command and SQLite repository with outbox transaction.
3. Add placement HTTP/MCP contracts and consume placement data in topology snapshots.
4. Add atomic reconnect command/task with rollback and shared API/MCP parity.
5. Implement framework-independent interaction controller and geometry/hit-test modules.
6. Refactor ECharts canvas adapter, visual symbols, EVE-NG-familiar hover connector handle, target highlighting,
   interface chooser, link context actions, selection, and keyboard equivalents.
7. Migrate browser preferences so only viewport, density, reduced motion, and manual link routes remain local.
8. Run component, contract, integration, recovery, performance, and target-host browser acceptance gates.

## Post-Design Constitution Check

PASS. The design strengthens shared state and control-plane parity, adds explicit revision/idempotency semantics,
keeps pure layout changes outside privileged runtime code, uses durable tasks for live reconnect, and defines
rollback, recovery, cleanup, audit, and target-host validation. No constitutional exception is required.

The EVE-NG reference is behavioral only. No EVE-NG source code, minified implementation, icons, CSS, templates,
credentials, or undocumented API dependency is copied into the product.

## Complexity Tracking

No constitution violations require justification.
