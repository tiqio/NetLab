# Frontend Testing Contract

## Test Layers

### Component Tests

- Runner: Vitest, Vue Test Utils, jsdom.
- Backend boundary: controlled typed mocks and deterministic event/task factories.
- Required coverage: loading, empty, success, permission/capability absence, structured failure, revision
  conflict, pending task, terminal task, reconnect, event-gap refresh, responsive fallback, keyboard focus,
  and renderer cleanup.
- ECharts: mock the chart factory at the module boundary; assert option semantics, stable IDs, event wiring,
  resize behavior, and disposal rather than testing ECharts internals.

### Real-Service Contract and Integration Tests

- Backend boundary: actual NetLab test service and generated API types.
- Required coverage: serialization, operation mapping, task polling/events, snapshot convergence, idempotent
  replay, conflict handling, captures/traffic-filter metadata, and redaction.
- Mock service implementations are not sufficient for this layer.

### Critical End-to-End Tests

- Runner: Playwright using the real Go server configured by `web/playwright.config.ts`.
- Required journeys: create/delete lab, add/connect/start nodes, observe a second-client or automation change,
  operate a node and task, open console, start/stop capture, visualize Traffic Filter, browse templates/images,
  and use the workspace at 1024×768 with keyboard navigation.

### Privileged Target-Host Tests

- Host: `10.72.1.7`, accessed through an operator-configured SSH mechanism without embedding credentials.
- Required journeys: real QEMU/Docker/lightweight nodes, live rewiring, NIC hot-plug, Telnet/VNC, capture,
  Traffic Filter, guest command, and port mapping.
- Runs must clean owned resources and store only sanitized reports.

## Test Data Rules

- Use synthetic template/image metadata and license-safe images/placeholders.
- Never commit proprietary image bytes, credentials, cloud-init secrets, console transcripts, guest-command
  secrets, packet payloads, `.pcap` files, Playwright traces containing sensitive data, or host snapshots.
- Screenshots and videos must use synthetic names, addresses, and payloads.

## Required Assertions

- Every graphical mutation maps to an operation listed in `backend-integration-matrix.md`.
- Accepted asynchronous operations display the durable task ID and converge to terminal server state.
- Two browsers and one automation client reach identical authoritative state within the specified target.
- Browser-local coordinates and panel layout remain independent between browser contexts.
- Event gaps trigger snapshot refresh without losing valid local placement.
- No duplicate mutation occurs from double-click, reconnect, or route navigation.
- ECharts, observers, sockets, xterm, and noVNC resources are released on teardown.
- State remains understandable without color alone and primary workflows are keyboard reachable.

## Performance Fixture

Use a deterministic laboratory snapshot with 100 visible nodes and 300 links. Measure pan, zoom, selection,
status update, and inspector-open latency. At least 95% of measured interactions must finish within 1 second
on the documented test workstation/browser profile.

## Exit Criteria

- `make test-web`, `make test-contract`, and `make test-e2e` pass.
- Target-host scenarios pass or have a documented environmental block unrelated to application behavior.
- No console errors, unhandled promise rejections, leaked renderer instances, or sensitive test artifacts remain.
- The quickstart scenarios produce the expected authoritative and browser-local outcomes.
