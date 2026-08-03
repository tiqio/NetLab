# Quickstart: Frontend Interaction Acceptance

This guide validates the acceptance infrastructure and then runs complete browser journeys. It intentionally
does not contain deployment credentials, guest secrets, packet payloads, or image download instructions.

## Prerequisites

- Node.js and npm versions compatible with `web/package-lock.json`
- Go toolchain required by the repository
- Playwright Chromium installed for the test workstation
- For target-host validation: deployed NetLab at `http://10.72.1.7:8088`, clean mutable state, and legally
  registered images for all supported template/device families
- Optional desktop Wireshark launcher configured only on a workstation where native application automation is
  safe

## 1. Install and Build

```bash
make bootstrap
make build
cd web && npx playwright install chromium
```

Expected outcome: the Vue bundle and `bin/netlabd` build successfully, and Chromium is available.

## 2. Validate Fast Feedback Layers

```bash
make lint
make test-web
make test-contract
make test-frontend-artifacts
```

Expected outcome:

- Component and state-matrix tests pass.
- Interaction inventory and evidence schemas validate.
- Frontend artifacts contain no disallowed credentials, packet captures, proprietary image data, or generated
  test evidence.

## 3. Run Disposable Real-Service Browser Tests

```bash
NETLAB_ACCEPTANCE_PROFILE=local \
NETLAB_ACCEPTANCE_TIMEOUT_SCALE=1.5 \
  ./acceptance/frontend-acceptance.sh
```

Expected outcome:

- Playwright starts the disposable service from `deploy/config/netlab.test.yaml`.
- Tests drive visible controls against the real HTTP service without `page.route` success substitution.
- Required projects run at 1920×1080 and 1024×768.
- Two browser contexts and one API request context converge on the same shared mutations.
- Privileged target-host-only scenarios are recorded as explicit environment-optional skips rather than
  failures.
- The temporary state directory is removed or contains no owned resources after completion.

## 4. Target-Host Preflight

Run before any privileged browser journey:

```bash
curl --fail --silent http://10.72.1.7:8088/healthz
curl --fail --silent http://10.72.1.7:8088/api/v1/capabilities
curl --fail --silent http://10.72.1.7:8088/api/v1/templates
curl --fail --silent http://10.72.1.7:8088/api/v1/images
```

Expected outcome:

- The service is healthy.
- QEMU, Docker, lightweight networking, Console, capture, Traffic Filter, live link changes, and other declared
  supported capabilities are available.
- At least one legal available version exists for every supported runtime/device family required by the spec.
- No user-owned laboratory is present in the designated clean acceptance environment.

A missing product-declared capability or required family is a failure, not a skip.

## 5. Run Complete Target-Host Acceptance

```bash
NETLAB_ACCEPTANCE_PROFILE=target-host \
NETLAB_ACCEPTANCE_BASE_URL=http://10.72.1.7:8088 \
NETLAB_ACCEPTANCE_TIMEOUT_SCALE=1.5 \
  ./acceptance/frontend-acceptance.sh
```

To enable safe desktop Wireshark verification:

```bash
NETLAB_ACCEPTANCE_PROFILE=target-host \
NETLAB_ACCEPTANCE_BASE_URL=http://10.72.1.7:8088 \
NETLAB_ACCEPTANCE_WIRESHARK_LAUNCHER=/absolute/path/to/approved-helper \
NETLAB_ACCEPTANCE_TIMEOUT_SCALE=1.5 \
  ./acceptance/frontend-acceptance.sh
```

Expected outcome:

- One representative version per supported runtime/device family completes its full applicable journey.
- Every other available version completes frontend-driven create, start, basic connectivity, stop when
  applicable, and delete.
- Real QEMU, Docker, lightweight nodes, live wiring, QEMU interface changes, task navigation, Telnet/VNC,
  capture, Traffic Filter, automation convergence, refresh/reconnect, and destructive cleanup pass.
- Capture evidence proves a non-empty live stream and usable Wireshark handoff. Desktop launch is either verified
  or recorded as the single permitted environment-optional skip with a concrete reason.
- Every created resource reaches `deleted` or verified `missing`; zero owned runtime resources remain.

## 6. Validate Failure Cleanup

Run the acceptance harness with its controlled failure injection profile after the normal path passes:

```bash
NETLAB_ACCEPTANCE_PROFILE=target-host \
NETLAB_ACCEPTANCE_BASE_URL=http://10.72.1.7:8088 \
NETLAB_ACCEPTANCE_FAILURE_INJECTION=after-runtime-create \
NETLAB_ACCEPTANCE_TIMEOUT_SCALE=1.5 \
  ./acceptance/frontend-acceptance.sh
```

Expected outcome:

- The run exits non-zero because the injected interaction failed.
- Cleanup still executes.
- The evidence identifies the failed interaction and cleanup trigger without retaining sensitive content.
- A second preflight confirms no run-owned laboratory, process, VM, container, interface, namespace, bridge,
  mapping, capture, filter, artifact, socket, or temporary file remains.

Exit code `1` is expected for this controlled failure. Exit code `0` means all required interactions and cleanup
passed; `2` means invalid runner configuration or evidence; `75` and `77` are reserved by the runner contract.

## 7. Review Evidence

Validate the generated JSON against:

- `specs/003-frontend-interaction-acceptance/contracts/interaction-inventory.schema.json`
- `specs/003-frontend-interaction-acceptance/contracts/acceptance-evidence.schema.json`

Review requirements:

- Every applicable inventory entry has a pass, fail, or eligible explicit skip.
- No supported capability is skipped.
- Every durable mutation includes a terminal task and refreshed authoritative final state.
- Ordinary visible feedback meets the 500 ms target; long operations expose pending identity within 2 seconds.
- Shared changes converge within 5 seconds across two browsers and one API client.
- Cleanup has `remaining_count: 0` and `baseline_restored: true`.
- Evidence contains no credentials, request authorization headers, console or guest-command output, packet
  payloads, captures, bootstrap data, proprietary images, or sensitive traces.

Each run writes to `web/test-results/acceptance/<run-id>/`. The durable run summary is `run-summary.json`, the
append-only ownership ledger is `resources.ledger.json`, and sanitized per-test evidence is under `tests/`.
Playwright's transient files are isolated under the same run's `playwright/` directory and are removed when the
artifact policy rejects them.

## 8. Repeatability Gate

Run the complete target-host acceptance twice from the same clean baseline.

Expected outcome: both runs produce the same applicable interaction inventory and version coverage, with no
unexplained result difference and no resource leakage between runs.
