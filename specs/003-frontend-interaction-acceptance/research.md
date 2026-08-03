# Research: Frontend Interaction Acceptance

## Decision 1: Use Three Verification Layers

**Decision**: Keep Vitest for deterministic component/state matrices, use Playwright with the real Go service
for browser integration, and use a separate Playwright target-host profile for privileged runtime acceptance.

**Rationale**: Component tests are fast and can force rare states, but they cannot prove that a visible control
is connected to the backend. A disposable local service verifies HTTP/event integration without consuming real
images. The target host is required for QEMU, Docker, namespaces, consoles, capture, and Traffic Filter.

**Alternatives considered**:
- Replace all component tests with E2E: rejected because rare error, quota, stale, and conflict states become slow
  and nondeterministic.
- Treat mocked Playwright routes as E2E: rejected because a mocked success can hide a disconnected control.
- Test only on the target host: rejected because it would make ordinary feedback too slow and increase cleanup
  risk.

## Decision 2: Make the Interaction Inventory Executable

**Decision**: Store every visible interaction as a stable manifest entry with page/state applicability, role or
accessible name, pointer and keyboard methods, outcome class, API operation when applicable, required
capabilities, cleanup effects, and evidence policy.

**Rationale**: A static checklist drifts from the UI and cannot detect newly added controls. A machine-readable
inventory can be compared with role-based discovery, used to parameterize tests, and reported for every pass,
failure, or permitted skip.

**Alternatives considered**:
- Screenshot review only: rejected because screenshots cannot prove behavior or authoritative mutation.
- Ad hoc test cases without inventory IDs: rejected because omissions and duplicate coverage are hard to detect.
- Automatically click every DOM element: rejected because applicability, destructive intent, and expected
  outcomes require domain context.

## Decision 3: Verify Mutations Through Three Signals

**Decision**: A mutation passes only after visible acknowledgement, durable task terminal state when applicable,
and a refreshed authoritative resource state all agree.

**Rationale**: Request acceptance alone can mask queued, rolled-back, or cleanup-failed work. UI-only state can
be stale. Combining the three signals detects false success while preserving the frontend as the initiating
surface.

**Alternatives considered**:
- Assert toast text only: rejected as a false-positive risk.
- Poll only the API: rejected because it does not prove the browser communicated the outcome.
- Wait for an arbitrary delay: rejected as nondeterministic and unable to diagnose terminal failure.

## Decision 4: Separate Disposable and Remote Playwright Profiles

**Decision**: Keep a local profile with a disposable `netlabd` web server and add a remote profile controlled by
`NETLAB_ACCEPTANCE_BASE_URL`, with no Playwright-managed server and with the required 1920×1080 and 1024×768
projects.

**Rationale**: Remote acceptance must never start or replace the service on `10.72.1.7`. Separate profiles also
make it impossible to mistake mock/local evidence for privileged target-host evidence.

**Alternatives considered**:
- One configuration that conditionally changes behavior: rejected because accidental local fallback could make
  a required remote run appear successful.
- SSH-driven browser on the server: rejected as the default because browser evidence is easier to control from
  the test workstation; SSH remains an orchestration option for deployment and host audits.

## Decision 5: Track Run-Owned Resources in a Ledger

**Decision**: Assign each run a unique identifier and synthetic name prefix, append every created laboratory,
node, link, mapping, capture, filter, task, artifact, and known runtime owner to an in-memory and persisted
sanitized ledger, and execute cleanup from a process-level `finally`/signal trap.

**Rationale**: Browser tests can fail between creation and deletion. The ledger enables safe, idempotent cleanup
without touching pre-existing operator resources and provides exact leak evidence if cleanup cannot converge.

**Alternatives considered**:
- Delete resources only at the end of successful tests: rejected because failed runs leak state.
- Delete every laboratory on the host: rejected because it violates ownership boundaries.
- Preserve failed laboratories for debugging: rejected by clarification; sanitized evidence must be sufficient.

## Decision 6: Enforce Capability Eligibility Before Running Journeys

**Decision**: Snapshot `/api/v1/capabilities`, templates, image bindings, browser facilities, and optional desktop
tools before mutation. Missing product-declared support fails the run; only explicitly optional environmental
capabilities may produce a reasoned skip.

**Rationale**: Silent skips can turn a completely untested feature into a green run. Preflight makes the tested
surface and version selection auditable before resources are created.

**Alternatives considered**:
- Skip whenever a control is disabled: rejected because a broken capability may incorrectly disable its control.
- Fail on every missing desktop facility: rejected because external desktop Wireshark automation is explicitly
  environment-dependent while capture, stream, and handoff are product requirements.

## Decision 7: Use Sanitized Structured Evidence

**Decision**: Write one JSON run record validated by schema, Playwright HTML results, and failure screenshots or
traces after redaction checks. Store identities, actions, timings, terminal states, viewport, expected/actual
summaries, and cleanup results; do not store packet payloads, console text, guest-command output, credentials,
bootstrap data, or proprietary image content.

**Rationale**: Structured records support repeatability and coverage analysis while respecting project hygiene.
Trace and screenshot retention only on failure limits sensitive surface and storage growth.

**Alternatives considered**:
- Keep all streams and downloads: rejected for sensitivity and unbounded size.
- Keep only human-readable logs: rejected because completeness and redaction cannot be reliably validated.

## Decision 8: Model Concurrent Observation Explicitly

**Decision**: Use two isolated browser contexts and one Playwright request context. Record the mutation source,
ordered event sequence, each observer's convergence time, revision conflicts, reconnect behavior, and final
snapshot identity.

**Rationale**: The original EVE-NG problem is session/account-scoped visibility. Explicit observer records prove
that browsers and automation share authoritative state while local workspace preferences remain isolated.

**Alternatives considered**:
- Reuse tabs in one browser context: rejected because storage and session state are shared.
- Verify only final equality: rejected because lost events and excessive convergence delay would remain hidden.

## Decision 9: Validate Wireshark in Two Tiers

**Decision**: Always prove a real non-empty capture stream and valid handoff command. When
`NETLAB_WIRESHARK_LAUNCHER` identifies a safe automation helper, launch desktop Wireshark and confirm that it
attaches to the stream; otherwise record a permitted environmental skip for desktop launch only.

**Rationale**: Stream and handoff are product behavior and cannot be skipped. Native application automation is
workstation-dependent and must not block environments that cannot safely control desktop processes.

**Alternatives considered**:
- Require native launch everywhere: rejected because headless environments cannot satisfy it safely.
- Validate only a generated command string: rejected because the underlying capture could be empty or broken.

## Decision 10: Use Representative Full Journeys Plus Complete Version Smoke Coverage

**Decision**: Select one available version per supported runtime/device family for the full applicable journey.
Every remaining available version receives frontend-driven creation, startup, basic connectivity, stop when
applicable, and deletion coverage.

**Rationale**: This catches version-binding and lifecycle defects without multiplying expensive diagnostic and
concurrency scenarios across every image.

**Alternatives considered**:
- Full matrix for every version: rejected as unnecessarily expensive for equivalent frontend behavior.
- One QEMU and one Docker image total: rejected because device-family-specific forms and capabilities would be
unverified.

## Decision 11: Use Bounded Condition Waits, Never Sleeps as Proof

**Decision**: Centralize waits for visible feedback, task identity, task terminal state, resource convergence,
event convergence, stream readiness, and cleanup. Fixed sleeps may only debounce UI mechanics and never prove
success.

**Rationale**: Bounded condition polling produces deterministic timeouts with useful last-observed evidence.

**Alternatives considered**:
- Per-test arbitrary delays: rejected as slow and flaky.
- Unlimited polling: rejected because hung runtime operations would stall the suite and bypass cleanup.
