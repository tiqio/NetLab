# Implementation Plan: Frontend Interaction Acceptance

**Branch**: `[003-frontend-interaction-acceptance]` | **Date**: 2026-07-27 | **Spec**: `specs/003-frontend-interaction-acceptance/spec.md`

**Input**: Feature specification from `specs/003-frontend-interaction-acceptance/spec.md`

## Summary

Replace shallow frontend checks with a layered interaction-acceptance system that proves every enabled control
has an observable outcome and that every mutation converges to authoritative server state. Playwright drives
pointer and keyboard journeys against both a disposable local service and the deployed service on
`10.72.1.7`; Vitest retains deterministic component-state coverage but cannot satisfy real-function acceptance.
An explicit interaction inventory, capability gate, run-owned resource ledger, durable-task observer, sanitized
evidence bundle, and unconditional cleanup coordinator make failures reproducible without leaving laboratories,
runtime processes, captures, or temporary artifacts behind.

## Technical Context

**Language/Version**: TypeScript 5.8.x and Vue 3.5.x for browser acceptance support; Bash for target-host
orchestration; existing backend remains Go 1.26.x

**Primary Dependencies**: Playwright 1.54.x, Vitest 3.2.x, Vue Test Utils 2.4.x, axe-core 4.12.x, existing Vue
SPA/API client, xterm.js 5.5.x, noVNC 1.6.x, ECharts 6.1.x, and existing NetLab acceptance scripts

**Storage**: No new product database storage; versioned JSON acceptance manifests and sanitized JSON/HTML
evidence under ignored test-result directories, plus the existing SQLite-backed authoritative product state

**Testing**: Vitest component and state-matrix tests, Playwright interaction and multi-client tests, Go contract
and integration suites, and privileged real-runtime acceptance on `10.72.1.7`

**Target Platform**: Chromium on Linux at 1920×1080 and 1024×768; SPA served by the single-host NetLab service
on x86_64 Linux with QEMU/KVM, Docker, network namespaces, capture tools, and optional desktop Wireshark

**Project Type**: Existing Vue SPA plus Go HTTP/event backend with browser, HTTP automation, and MCP control
surfaces

**Performance Goals**: 95% of ordinary controls show feedback within 500 ms; every accepted long operation
exposes a task or pending identity within 2 seconds; shared mutations converge across two browsers and one
automation client within 5 seconds

**Constraints**: Real acceptance cannot use route interception or mocked success; enabled controls cannot be
silent no-ops; supported capabilities cannot be skipped; every run cleans owned resources after success,
failure, timeout, or interruption; evidence excludes credentials, console/packet payloads, proprietary images,
and bootstrap contents; target-host tests may modify only run-owned synthetic resources

**Scale/Scope**: Complete inventory of all visible interactions and applicable UI states; full representative
journey for each supported runtime/device family; lifecycle and connectivity coverage for every other available
image version; two viewports; two browser contexts plus one API client; QEMU, Docker, lightweight nodes,
topology wiring, tasks, Telnet/VNC, capture, Traffic Filter, automation, and cleanup

## Constitution Check

*GATE: Passed before Phase 0 research and re-checked after Phase 1 design.*

- **Shared state — PASS**: Tests treat laboratory snapshots, desired/observed states, tasks, captures, filters,
  and ordered events as server-authoritative. Two browser contexts and one HTTP client verify shared convergence;
  only documented workspace preferences remain browser-local.
- **Control parity — PASS**: The interaction inventory maps every state-changing frontend control to the existing
  versioned API operation and durable task behavior. API/MCP parity is checked through authoritative final state,
  not by creating a separate test-only mutation path.
- **Runtime scope — PASS**: Real journeys use only approved QEMU, Docker, bridge/NAT, and network-namespace
  resources and data-driven device template versions. The feature adds acceptance infrastructure, not a new
  runtime primitive.
- **Live operations — PASS**: Running-node connect/disconnect, QEMU interface mutation, Telnet/VNC, live capture,
  Wireshark handoff, and Traffic Filter observations are driven through visible controls and verified through
  terminal tasks, streams, refreshed snapshots, and rollback/cleanup state.
- **Safety and recovery — PASS**: Every run has a unique owner prefix and resource ledger. Cleanup runs from a
  `finally` path after success or failure, uses bounded waits and idempotent deletion, reports remaining owners,
  and prevents contaminated state from being reused.
- **Verification — PASS**: Component matrices remain in Vitest; API schemas and evidence manifests receive
  contract tests; Playwright covers complete browser journeys, accessibility, concurrent clients, refresh and
  reconnect; target-host runs cover privileged runtime behavior and leak detection.
- **Image and secret hygiene — PASS**: The harness discovers operator-registered images but never downloads or
  commits them. Evidence stores identifiers, versions, checksums, states, timings, and redacted errors only; it
  excludes packet payloads, console transcripts, credentials, and proprietary image content.

## Project Structure

### Documentation (this feature)

```text
specs/003-frontend-interaction-acceptance/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── acceptance-evidence.schema.json
│   ├── interaction-inventory.schema.json
│   └── runner-contract.md
└── tasks.md
```

### Source Code (repository root)

```text
web/
├── package.json
├── playwright.config.ts
└── src/
    ├── api/
    ├── components/
    ├── features/
    ├── stores/
    └── test/

tests/e2e/
├── fixtures/                 # run identity, capability gate, resource ledger, evidence
├── pages/                    # role-based workspace drivers
├── journeys/                # real-service end-to-end workflows
├── matrices/                 # inventory, viewport, version, and state coverage
└── *.spec.ts                 # acceptance entry points

acceptance/
├── frontend-acceptance.sh    # target-host orchestration and cleanup trap
└── README.md

scripts/
└── check-frontend-artifacts.sh

internal/api/http/            # existing authoritative APIs and streams
tests/contract/               # schema/control parity checks
tests/integration/            # runtime and cleanup verification
```

**Structure Decision**: Extend the existing web application and root Playwright suite instead of introducing a
new test project. Browser-driving code stays under `tests/e2e`; privileged orchestration stays under
`acceptance`; product fixes remain in their existing `web/src` or Go package. Contracts live with this feature
and are validated from the existing test commands.

## Phase 0: Research Outcomes

Research decisions are recorded in `specs/003-frontend-interaction-acceptance/research.md`. All technical
unknowns are resolved.

## Phase 1: Design Outcomes

- Acceptance entities, validation rules, ownership relationships, and state machines are defined in
  `specs/003-frontend-interaction-acceptance/data-model.md`.
- The interaction inventory and evidence bundle have machine-readable JSON Schema contracts in
  `specs/003-frontend-interaction-acceptance/contracts/`.
- Runner environment, exit behavior, cleanup obligations, and Wireshark handling are defined in
  `specs/003-frontend-interaction-acceptance/contracts/runner-contract.md`.
- Local and target-host validation sequences are defined in
  `specs/003-frontend-interaction-acceptance/quickstart.md`.

## Post-Design Constitution Check

**PASS**. The design preserves shared authoritative state and control-plane parity, limits privileged actions to
existing runtime boundaries, exercises live operations rather than simulating them, assigns explicit ownership
to all acceptance resources, and retains only sanitized evidence. No constitutional exception is required.

## Complexity Tracking

No constitution violations or complexity exceptions are required.
