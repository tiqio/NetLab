# Implementation Plan: Frontend UX Modernization

**Branch**: `[002-frontend-ux-modernization]` | **Date**: 2026-07-27 | **Spec**: `specs/002-frontend-ux-modernization/spec.md`

**Input**: Feature specification from `specs/002-frontend-ux-modernization/spec.md`

## Summary

Modernize the existing Vue SPA into an EVE-NG-familiar network-lab workspace built from Shadcn Vue
primitives, with ECharts powering topology, traffic-path, trend, distribution, and statistical
visualizations. The implementation preserves the existing REST, event, task, console, capture, and
automation contracts; shared laboratory state remains server-authoritative, while coordinates,
grouping, link routing, viewport, and panel layout are versioned browser-local preferences keyed by
laboratory. The current monolithic topology workspace will be decomposed into a shell, feature panels,
ECharts adapters, operation presenters, and tested integration boundaries.

## Technical Context

**Language/Version**: TypeScript 5.8.x, Vue 3.5.x; existing backend remains Go 1.26.x

**Primary Dependencies**: Vite 7.x, Pinia 3.x, Vue Router 4.x, shadcn-vue 2.8.x, Tailwind CSS 4.x,
Reka UI 2.10.x, Lucide Vue Next 1.x, Apache ECharts 6.1.x, xterm.js 5.5.x, noVNC 1.6.x

**Storage**: Existing SQLite-backed server state; browser `localStorage` through a versioned storage
adapter for non-secret, laboratory-scoped visual preferences only

**Testing**: Vitest 3.2.x, Vue Test Utils 2.4.x, jsdom, Playwright 1.54.x, existing Go test service,
and privileged acceptance validation on `10.72.1.7`

**Target Platform**: Modern desktop browsers at 1024×768 or larger; SPA served by the single-host
NetLab service on x86_64 Linux

**Project Type**: Existing web application with Vue SPA and Go HTTP/event backend

**Performance Goals**: At 100 visible nodes and 300 links, 95% of pan, zoom, selection, status, and
inspector interactions complete within 1 second; authoritative changes converge across clients within
3 seconds in 95% of runs; reconnect convergence completes within 10 seconds

**Constraints**: No authentication UI; no UI-only mutations; no backend state redesign without a
verified contract defect; no proprietary EVE-NG assets; no credentials, bootstrap secrets, packet
payloads, or proprietary image data in browser storage, fixtures, screenshots, or logs; all graph views
use ECharts; specialized terminal, VNC, and packet streaming renderers remain dedicated components

**Scale/Scope**: One SPA with five primary workspace regions, laboratory management, all supported node
and network-object workflows, tasks, consoles, captures, traffic filters, templates/images, automation
views, and responsive/accessibility behavior for the existing single-host product

## Constitution Check

*GATE: Passed before Phase 0 research and re-checked after Phase 1 design.*

- **Shared state — PASS**: Laboratory resources, desired/observed lifecycle state, tasks, captures,
  traffic filters, and ordered events remain server-authoritative. Only visual layout preferences are
  browser-local, namespaced by laboratory, versioned, and excluded from shared convergence semantics.
- **Control parity — PASS**: Every state-changing control maps to an existing versioned REST operation
  and its durable task result. The SPA does not create private lifecycle state or UI-only operations;
  external automation and MCP clients observe the same authoritative resources and errors.
- **Runtime scope — PASS**: This is a frontend-only modernization. It introduces no runtime primitive
  beyond the approved QEMU, Docker, bridge, NAT, and network-namespace capabilities already exposed.
- **Live operations — PASS**: Live link mutation, NIC hot-plug, Telnet/VNC, capture, and traffic-filter
  controls present existing backend stages, task status, errors, and rollback/cleanup outcomes rather
  than simulating success locally.
- **Safety and recovery — PASS**: The UI treats long operations as durable tasks, prevents accidental
  duplicate submission, uses revisions/idempotency where supplied, recovers after event gaps by fetching
  authoritative snapshots, and exposes cleanup and retry guidance from structured problems.
- **Verification — PASS**: Component behavior uses deterministic mocks; API/event contracts and feature
  integration use a real test service; critical E2E uses the actual Go backend; privileged console,
  capture, traffic-filter, and live-runtime scenarios are repeatable on `10.72.1.7`.
- **Image and secret hygiene — PASS**: Browser preferences contain display state only. Test contracts
  prohibit proprietary images, credentials, cloud-init content, packet payloads, and secrets in source,
  fixtures, screenshots, traces, videos, or logs; image metadata remains provenance/digest based.

### Post-Design Re-check

The data model separates authoritative projections from browser-local preferences, the UI contract
forbids local mutation authority, the integration matrix maps graphical actions to backend operations,
and the testing contract includes real-service, convergence, failure, recovery, and redaction checks.
No constitution exception is required.

## Project Structure

### Documentation (this feature)

```text
specs/002-frontend-ux-modernization/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── backend-integration-matrix.md
│   ├── testing-contract.md
│   ├── ui-state-contract.md
│   └── workspace-preferences.schema.json
└── tasks.md
```

### Source Code (repository root)

```text
web/
├── src/
│   ├── api/                    # Existing generated types and REST/event clients
│   ├── components/
│   │   ├── ui/                 # Generated/adapted Shadcn Vue primitives
│   │   └── shell/              # Toolbar, palette, inspector, and drawer shell
│   ├── features/
│   │   ├── topology/           # ECharts topology and topology interactions
│   │   ├── diagnostics/        # Console, capture, and traffic-filter UI
│   │   ├── laboratories/       # Laboratory management workflows
│   │   ├── nodes/              # Node forms, operations, resources, and mappings
│   │   ├── tasks/              # Durable task center and problem presentation
│   │   ├── templates/          # Template/image browsing and selection
│   │   └── analytics/          # Shared ECharts trend/statistical components
│   ├── composables/            # ECharts, resize, keyboard, and persistence adapters
│   ├── stores/                 # Authoritative projections and local UI state
│   ├── styles/                 # Tailwind theme tokens and application layout
│   ├── views/                  # Route-level workspace and management views
│   └── test/                   # Shared component-test setup, factories, and mocks
└── tests/                      # Frontend integration tests where colocating is unsuitable

tests/e2e/                      # Playwright tests against the real Go service
internal/ and cmd/              # Existing backend; changed only for verified contract defects
specs/001-network-simulator-platform/contracts/
                                # Existing OpenAPI, events, MCP, and export contracts
```

**Structure Decision**: Retain the existing `web/` SPA and backend layout. Add a reusable component
layer and split feature-level concerns out of `TopologyWorkspace.vue`; do not introduce a second
frontend package, a separate browser backend, or duplicate API models.

## Implementation Strategy

1. Establish Tailwind/Shadcn Vue tokens and primitives without changing backend behavior.
2. Introduce the resizable five-region application shell and responsive drawer fallbacks.
3. Separate authoritative laboratory projections from versioned browser-local workspace preferences.
4. Replace form/list topology rendering with an ECharts graph adapter and direct manipulation flows.
5. Build consistent node, link, task, structured-problem, console, capture, and traffic-filter panels.
6. Complete laboratory, template/image, automation, trend, and statistical views with shared primitives.
7. Add accessibility, performance, convergence, real-service integration, and real-backend E2E coverage.

## Complexity Tracking

No constitution violations or additional architectural layers require justification.
