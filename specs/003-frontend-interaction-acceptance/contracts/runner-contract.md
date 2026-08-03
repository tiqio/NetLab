# Frontend Acceptance Runner Contract

## Purpose

The runner executes real browser interactions, produces schema-valid sanitized evidence, and restores the
starting resource baseline after every outcome. It is not a product API and must not introduce a test-only
mutation path.

## Profiles

### `local`

- Starts the disposable test configuration on `127.0.0.1:8080`.
- Uses empty temporary state and no privileged runtime claims.
- May run browser integration, contract, accessibility, error-state, and concurrency scenarios.
- Cannot satisfy QEMU, Docker, namespace, Console, capture, Traffic Filter, or full target-host acceptance.

### `target-host`

- Uses `NETLAB_ACCEPTANCE_BASE_URL`, expected to be `http://10.72.1.7:8088` for the designated environment.
- Never starts, stops, replaces, or reconfigures the deployed service through Playwright.
- Requires a clean preflight baseline and operator-registered, license-safe images.
- Exercises real QEMU, Docker, lightweight nodes, Console, capture, Traffic Filter, topology changes, and cleanup.

## Environment Variables

| Name | Required | Meaning |
|---|---|---|
| `NETLAB_ACCEPTANCE_PROFILE` | yes | `local` or `target-host` |
| `NETLAB_ACCEPTANCE_BASE_URL` | target-host | Browser origin without embedded credentials |
| `NETLAB_ACCEPTANCE_OUTPUT_DIR` | no | Defaults to ignored `web/test-results/acceptance/<run-id>` |
| `NETLAB_ACCEPTANCE_RUN_ID` | no | Generated when absent; must be unique |
| `NETLAB_ACCEPTANCE_WIRESHARK_LAUNCHER` | no | Executable helper for safe desktop-launch verification |
| `NETLAB_ACCEPTANCE_TIMEOUT_SCALE` | no | Positive multiplier for slower legal operator images |
| `NETLAB_ACCEPTANCE_CLEAN_BASELINE` | no | Defaults to `required`; complete target-host runs cannot disable it |
| `NETLAB_ACCEPTANCE_FAILURE_INJECTION` | no | Named controlled failure point used only to prove unconditional cleanup |

Secrets, SSH passwords, guest credentials, bootstrap data, and proprietary image paths must not be passed as
runner variables or written to evidence. Deployment access is configured out of band.

Run `bash acceptance/frontend-acceptance-repeat.sh` for the repeatability gate. It executes two independent
complete runs against the selected profile and fails when normalized inventory outcomes, capability decisions,
version coverage, or cleanup results differ.

## Required Phases

1. **Preflight**: Verify health, capabilities, templates/images, browser requirements, clean baseline, and
   evidence output permissions before creating resources.
2. **Inventory validation**: Load the interaction inventory, reject duplicate IDs, verify required metadata,
   and compare applicable enabled controls with the manifest.
3. **Execution**: Run pointer and keyboard journeys at 1920×1080 and 1024×768. Use unique synthetic names and
   append created identities to the resource ledger immediately.
4. **Authoritative verification**: For mutations, verify visible feedback, terminal task where applicable, and
   refreshed final state. For shared-state scenarios, record observations from two browser contexts and one API
   client.
5. **Cleanup**: Always execute after success, assertion failure, timeout, or termination signal. Prefer the
   frontend's destructive flow for the journey, then use bounded idempotent API cleanup for remaining run-owned
   resources. Host-level remediation is diagnostic and must never target resources outside the run ledger.
6. **Evidence finalization**: Redact, validate against `acceptance-evidence.schema.json`, verify zero owned
   runtime resources, and only then emit a passing exit status.

## Capability Rules

- A product-declared supported capability must run. Missing, disabled, unavailable, or nonfunctional behavior
  fails preflight or the relevant interaction.
- A skip is permitted only for an inventory entry that names an `environment-optional` capability and records a
  verified missing prerequisite.
- Desktop Wireshark launch is environment-optional. Real capture creation, non-empty stream data, artifact or
  handoff metadata, and capture cleanup are product-supported and cannot be skipped.
- A complete target-host gate cannot pass if all versions of a supported runtime/device family are unavailable.

## Evidence and Redaction

- Validate interaction inventory with `interaction-inventory.schema.json`.
- Validate terminal run evidence with `acceptance-evidence.schema.json`.
- Failure screenshots and traces may be retained only after automated filename/content policy checks.
- Never retain packet payloads, capture files, console transcripts, guest-command output, credentials, bootstrap
  data, proprietary image bytes, or unredacted request headers.
- HTTP evidence records method, route template, status, correlation/task/resource IDs, duration, and sanitized
  problem codes only.

## Exit Codes

| Code | Meaning |
|---|---|
| `0` | All required interactions passed and cleanup restored the baseline |
| `1` | Product behavior, coverage, timeout, unsupported skip, or cleanup failure |
| `2` | Invalid runner configuration, inventory, or evidence schema |
| `75` | Supervised external host-restart continuation requested by an existing privileged scenario |
| `77` | Optional sub-suite lacks an explicitly optional environmental prerequisite; never valid for the complete target-host gate |

## Completion Invariants

- Every applicable interaction has exactly one terminal result per required context.
- Every created resource has a terminal cleanup state.
- `remaining_count` is zero and the baseline is restored before exit `0`.
- Every supported runtime/device family has representative full-journey coverage and every remaining available
  version has lifecycle/connectivity coverage.
- No supported product capability is reported as skipped.
