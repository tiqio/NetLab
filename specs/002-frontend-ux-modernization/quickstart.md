# Quickstart: Frontend UX Modernization Validation

## 1. Prerequisites

- Go, Node.js/npm, SQLite, and host dependencies from the platform quickstart are installed.
- The repository has been bootstrapped with `make bootstrap`.
- Privileged acceptance scenarios use an operator-configured connection to `10.72.1.7`; do not place
  passwords or private keys in the repository, shell history, test configuration, or reports.

## 2. Build and Static Validation

```bash
make fmt
make lint
make build
```

Expected: Shadcn Vue/Tailwind assets and modular ECharts imports compile without TypeScript, lint, or bundle
errors. No generated API type is replaced by a handwritten duplicate.

## 3. Component Tests

```bash
make test-web
```

Expected: shell, workspace preferences, ECharts adapters, forms, task/problem presentation, console hosting,
capture, Traffic Filter, responsive, accessibility, and cleanup tests pass with deterministic mocks.

## 4. Contract Validation

```bash
make test-contract
```

Validate `contracts/workspace-preferences.schema.json` with one valid fixture and fixtures containing an
unsupported version, invalid zoom, and forbidden extra fields. Expected: only the valid fixture is accepted;
invalid persisted data falls back to safe defaults in component/integration tests.

## 5. Start the Real Test Service

```bash
make build
rm -rf /tmp/netlab-frontend-test
./bin/netlabd -config deploy/config/netlab.test.yaml
```

Open `http://127.0.0.1:8080` in two isolated browser contexts. Expected: both receive the same authoritative
laboratory state while each retains independent topology placement and panel layout.

## 6. Workspace MVP Scenario

1. Create a laboratory from the top toolbar.
2. Add supported QEMU, Docker, and lightweight nodes from the searchable palette.
3. Choose required template/image versions and place nodes on the ECharts canvas.
4. Create links and a network object through direct canvas interactions.
5. Start nodes and inspect desired/observed state, revision, task progress, and failures.
6. Move nodes and resize/collapse panels, then reload only the first browser.

Expected: resource mutations converge through durable tasks; layout survives locally; the second browser sees
the same topology resources but not the first browser's positions or panel sizes.

## 7. Convergence and Conflict Scenario

1. Change the topology through the second browser or an API client.
2. Confirm the first browser updates within 3 seconds without reload.
3. Edit a revision-sensitive form, mutate the same resource from the second client, then submit the stale form.
4. Interrupt the event stream and force a replay gap.

Expected: remote resources receive deterministic local placement; the conflict preserves entered values and
offers refresh/retry; the replay gap triggers an authoritative snapshot and keeps valid local layout.

## 8. Node Operations and Task Center

Exercise start, stop, interface hot-add/remove, resource update, guest command, port mapping, cancellation,
and an injected failure. Expected: every action shows one durable task with operation/resource identity,
progress, timestamps, terminal result, retryability, cleanup state, and operator guidance.

## 9. Console and Diagnostics

1. Open available Telnet and VNC sessions in the bottom drawer and reconnect them.
2. Start, inspect, stream/download, and stop a capture.
3. Start a known Traffic Filter flow and inspect ECharts path direction, counts, timing, confidence, loops,
   or ambiguity.
4. Close renderer tabs and reopen them.

Expected: closing a console does not stop its node; capture limits/truncation/retention remain visible; no packet
payload enters storage; chart, terminal, VNC, observer, and transport resources are disposed on close.

## 10. Responsive and Accessibility Validation

Run the workspace at 1024×768 and a wider desktop viewport using keyboard-only navigation. Expected: palette,
inspector, and bottom content fall back to usable drawers/tabs; canvas and active operations remain visible;
focus is visible; topology selection and primary actions have descriptive labels; state is not color-only.

## 11. Performance Validation

Load the deterministic 100-node/300-link fixture described in `contracts/testing-contract.md`. Record pan,
zoom, selection, status-update, and inspector-open latency. Expected: 95% of interactions complete within one
second and repeated mount/unmount cycles do not grow active chart, observer, socket, or renderer counts.

## 12. End-to-End Suite

```bash
make test-e2e
```

Expected: Playwright starts or reuses the real Go service and verifies critical user journeys without mocked
backend lifecycle behavior.

## 13. Privileged Acceptance Host

After deploying the current build through the project's normal mechanism, run the relevant privileged suites
and manually verify QEMU/Docker/lightweight node operations, live rewiring, consoles, capture, Traffic Filter,
guest commands, and port mappings on `10.72.1.7`.

Expected: the SPA reflects the same authoritative results as REST/MCP clients, and cleanup leaves no owned
runtime, network, capture, or renderer resource behind. Reports contain metadata only and no credentials,
proprietary images, bootstrap secrets, console secrets, or packet payloads.
