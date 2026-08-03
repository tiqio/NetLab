# Validation: Frontend Interaction Acceptance

**Validated**: 2026-07-27  
**Target**: `http://10.72.1.7:8088`  
**Profiles**: `local`, `target-host`

No deployment credentials, guest output, packet payloads, capture files, bootstrap data, or proprietary image
bytes are retained in this document or the acceptance evidence.

## Phase 9 Convergence Status

The earlier retained runs below predate the strict Phase 9 coverage gate and therefore do not complete
T086-T102. On 2026-07-27, local validation passed after adding runtime-ownership leak verification, default-delete
browser artifact retention, aggregated evidence, strict interaction/version coverage gates, stable interaction
IDs, and a two-independent-run comparator:

- `go test ./...`: PASS.
- `npm --prefix web run test:acceptance-unit`: PASS, 7 files and 13 tests.
- `npm --prefix web run format:check`: PASS.
- `npx vue-tsc --noEmit`: PASS.
- `npm --prefix web run build`: PASS with advisory chunk-size warnings only.

A new target binary must be deployed and the expanded target-host journeys must pass twice, plus controlled
failure cleanup, before any Phase 9 task is marked complete.

## Quality Gates

| Gate | Result | Notes |
|---|---|---|
| `npm --prefix web run format:check` | PASS | All checked files match Prettier formatting. |
| `make lint` | PASS | Go vet and ESLint complete with zero errors; 559 pre-existing Vue style warnings remain. |
| `make test` | PASS | Go unit, contract, integration, recovery, and security packages pass. |
| `make test-contract` | PASS | Contract suite passes. |
| `make test-web` | PASS | 30 files and 58 frontend tests pass. |
| `make test-acceptance-schema` | PASS | Frontend acceptance schemas pass contract validation. |
| `go test ./tests/integration/... -run FrontendAPIAndMCPCreateConverge -count=1` | PASS | Frontend/API/MCP equivalent creation converges. |
| `make test-frontend-artifacts` | PASS | No prohibited credentials, images, captures, keys, or leaked runtime evidence found. |
| `make build` | PASS | Vue production bundle and `bin/netlabd` build successfully. |

The Vite build reports advisory chunk-size warnings for the console and ECharts bundles; this does not affect
the acceptance result.

## Local Disposable Run

- Run: `local-retention-check-20260727163930`
- Result: PASS; 40 tests passed and 14 privileged target-host scenarios were recorded as eligible
  environment-optional skips across the two required viewports.
- Evidence: `web/test-results/acceptance/local-retention-check-20260727163930/run-summary.json`
- Cleanup: `baseline_restored=true`, `remaining_count=0`.
- Retention check: both previously completed target-host run summaries remained present after the local run.

## Target-Host Runs

The built service/frontend containing the insecure-origin UUID fallback, laboratory selection fix, dialog Escape
handling, and Traffic Filter null-observation normalization was deployed to the designated target before these
runs. The target exposed QEMU, Docker, namespace, Telnet, VNC, MCP, capture, Traffic Filter, QMP hotplug, QGA,
port mapping, and CPU quota capabilities.

| Run | Result | Tests | Cleanup | Evidence |
|---|---|---:|---|---|
| `target-retained-pass1-20260727163501` | PASS | 54/54 | baseline restored; 0 remaining | `web/test-results/acceptance/target-retained-pass1-20260727163501/run-summary.json` |
| `target-retained-pass2-20260727163710` | PASS | 54/54 | baseline restored; 0 remaining | `web/test-results/acceptance/target-retained-pass2-20260727163710/run-summary.json` |

Both runs exercised desktop 1920×1080 and minimum 1024×768 projects against the real service. They covered
laboratory operations, template/image selection, QEMU/Docker/lightweight creation, live topology edits, node
operations, concurrent clients, Telnet/VNC, capture/Wireshark handoff, Traffic Filter, refresh/reconnect, and
destructive cleanup.

## Repeatability

- Normalized interaction signature for run 1:
  `24043d342ebbf26bf4e2876534fb732c7a9e58bdabee389bb2b39c9353efa26e`
- Normalized interaction signature for run 2:
  `24043d342ebbf26bf4e2876534fb732c7a9e58bdabee389bb2b39c9353efa26e`
- Each run recorded six unique version-coverage entries.
- No unexplained status or inventory differences were found.
- The target laboratory baseline was empty after both runs.

## Controlled Failure

- Run: `target-retained-injected-20260727164047`
- Injection: `NETLAB_ACCEPTANCE_FAILURE_INJECTION=after-runtime-create`
- Expected result: Playwright exit code `1`; evidence status `failed`.
- Cleanup result: `baseline_restored=true`, `remaining_count=0`.
- Evidence: `web/test-results/acceptance/target-retained-injected-20260727164047/run-summary.json`
- Post-run target laboratory count: `0`.

This proves the independent teardown path executes after an assertion failure and removes every run-owned
resource without treating the intentionally failed interaction as a successful run.

## Artifact Hygiene

`scripts/check-frontend-artifacts.sh` passed after all local, target-host, repeatability, and controlled-failure
runs. Screenshots, traces, videos, and HTML reports are disabled by default; transient Playwright files are
isolated per run and filtered by the artifact policy. No prohibited sensitive or proprietary content was found
in the repository or distributable.
