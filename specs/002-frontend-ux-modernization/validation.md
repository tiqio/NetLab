# Frontend UX Modernization Validation

**Date**: 2026-07-27

## Quality Gates

- `make fmt`: PASS; Go and frontend sources are formatted.
- `make lint`: PASS; Go vet and Prettier checks pass. ESLint reports style warnings and zero
  errors because the configured Vue line-break rules differ from Prettier.
- `make build`: PASS; Vue TypeScript, Vite production assets, embedded SPA assets, and the Go server compile.
- `make test-web`: PASS; 29 test files and 47 component/unit tests.
- `make test-contract`: PASS; OpenAPI, event, MCP, workspace-preference schema, and frontend operation
  matrix contracts validate.
- `make test-integration`: PASS; real SQLite/application/HTTP service tests include the four frontend
  workflow suites.
- `make test-frontend-artifacts`: PASS; no credentials, packet captures, generated browser reports, or
  forbidden frontend artifacts are present.
- `make test-e2e`: PASS; 18 Playwright tests across desktop and 1024×768 projects.

## Quickstart Scenarios

- The five-region EVE-NG-familiar workspace renders with Shadcn Vue-style source components and ECharts
  topology, traffic-path, performance, and statistics visualizations.
- Two isolated browser contexts share server-authoritative laboratory resources while retaining independent,
  schema-validated browser-local coordinates, routes, viewport, grouping, and panel preferences.
- Laboratory, node, network-object, link, live-rewire, task, cancellation, idempotency, revision-conflict,
  event-gap recovery, and structured-problem paths use real application services and durable state.
- Template selection refreshes the generated API catalog immediately before submission and blocks removed or
  disabled versions without discarding the user's surrounding form state.
- Laboratory transfer presents durable task progress, export artifact metadata, redaction declarations, and
  missing image digests before import.
- Telnet/VNC descriptors, xterm/noVNC renderer disposal, capture state, Traffic Filter observations, task
  navigation, keyboard operation, focus visibility, and responsive drawers are covered by deterministic
  component and browser tests.
- The deterministic 100-node/300-link ECharts fixture completes tested interactions within the one-second
  component target, includes 95th-percentile update assertions, and does not retain chart instances after
  unmount.

## Phase 9 Convergence

- Laboratory rename now retries revision conflicts without discarding the operator-entered name.
- Node creation uses the live template and image catalog, validates runtime compatibility, explains disabled
  choices, and rechecks the catalog before mutation.
- Browser-local viewport, grouping, link routes, port selection, direct connect/reconnect, and keyboard
  traversal remain isolated from server-authoritative laboratory state.
- Capture and Traffic Filter collection contracts support laboratory-scoped discovery; persisted sessions
  rejoin after refresh and recover into an explicit terminal state after control-service restart.
- Node-interface and link captures expose quota, truncation, retention, stream, artifact, task, stop, and
  completion metadata; task and capture progress graphics use modular ECharts components.
- The task center now filters by laboratory scope, resource type, operation kind, state, time, and text while
  presenting timestamps, result payloads, cancellation availability, authoritative replay, and unavailable
  resource navigation.
- General-purpose feature controls use shared UI primitives, destructive operations identify cleanup and
  stream impact, and reusable state presentations cover the complete degraded/terminal state vocabulary.

## Bundle Notes

- Production build completes with ECharts and console chunks split from the main application.
- Vite reports advisory size warnings for the ECharts and console chunks; these are non-failing and remain
  isolated lazy/manual chunks rather than increasing the initial application bundle.

## Privileged Validation

The current build was deployed and validated on `10.72.1.7`. Sanitized metadata-only results are recorded in
`specs/002-frontend-ux-modernization/validation-host.md`; no credentials, image bytes, guest output, console
content, or packet payloads are included.
