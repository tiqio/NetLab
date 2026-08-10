# Final Local Validation — 2026-08-10

## Source

```text
Final local validation commit: d4bbd8ca1da97ae8155889eb061d2deb64c01178
Branch: main
Remote: git@github.com:tiqio/NetLab.git
Worktree before candidate documentation: clean
Contract digest: sha256:2e1a4279f16d7e6c282b741f06c52e75f1db210e91dbf4b99180bea5c9da808b
Latest migration: 0014_network_attachment_revision.sql
```

The final implementation does not add an external API shape after the checked-in OpenAPI/MCP/client
types. `contracts/openapi.yaml`, `contracts/mcp-tools.json`, and `web/src/api/generated.ts` remain
synchronized through the passing HTTP/MCP contract suites and frontend build.

## Go Gates

```bash
make lint
go test ./... -count=1 -timeout=180s
make test-contract
make test-integration
make test-recovery
make test-leaks CYCLES=20
```

Result: PASS. `go vet ./...`, all Go packages, contract tests, non-privileged integration and recovery
suites, and the 20-cycle resource-leak suite completed successfully. The focused transaction tests
cover task, backing record, endpoint reservation, audit/outbox, revision, cancellation, and
operation-owned cleanup failure points.

## Frontend Gates

```bash
cd web
npm run format:check
npm run lint
npm run build
npm test
npm run test:acceptance-unit
NO_PROXY=127.0.0.1,localhost no_proxy=127.0.0.1,localhost npm run test:e2e:local
```

Result: PASS.

- Prettier: all files formatted.
- ESLint: 0 errors; 898 pre-existing warnings.
- Vite production build: PASS; only the existing large-chunk warning.
- Vitest: 78 files, 375 tests passed.
- Acceptance-unit: 12 files, 25 tests passed.
- Local Playwright: 159 passed, 57 target-only skipped, 0 failed in 13.7 minutes across 1920×1080,
  1366×768, and 1024×768 projects.

The final focused rerun of authoritative placement and navigation executed six project cases and
passed 6/6 after using project-specific idempotency keys and the rebuilt embedded frontend.

## Interaction, Accessibility, Performance, and Visual Evidence

- Keyboard and focus behavior passed in `TopologyCanvas.a11y.test.ts`,
  `topologyKeyboardController.test.ts`, `topologySelectionKeyboard.spec.ts`, and the laboratory input
  matrix.
- Axe checks passed for light/dark themes at 1024×768, 1366×768, and 1920×1080 in
  `frontend_localization_theme_accessibility.spec.ts`; component axe checks also passed.
- Fifty connection drags completed inside the deterministic local interaction budget in
  `TopologyCanvas.performance.test.ts`.
- Parallel links remained independently keyboard-selectable at 50%, 100%, and 200% zoom in all three
  Playwright projects.
- 010 visual regression coverage passed for resource symbols, dense lifecycle labels, endpoint hover,
  mixed connection states, semantic legend, direction-guide decay, and `/`, `/templates`, and
  `/automation` visual audits.
- Local object-link observability passed for capture and Traffic Filter isolation across parallel
  links. Hardware-backed capture/filter validation remains target-host work.

## Privileged Gate Boundary

The local machine ran every applicable non-privileged integration, recovery, and leak gate. Tests that
require KVM/QEMU, Docker runtime ownership, host namespaces, TAP/veth, nftables, packet capture, or
service restart authority are intentionally deferred to `10.72.1.7`.

Run on the target after deploying the recorded candidate:

```bash
NETLAB_ACCEPTANCE_PROFILE=target-host \
NETLAB_ACCEPTANCE_BASE_URL=http://10.72.1.7:8088 \
NO_PROXY=10.72.1.7,127.0.0.1,localhost \
no_proxy=10.72.1.7,127.0.0.1,localhost \
  ./acceptance/frontend-acceptance.sh

ssh root@10.72.1.7 \
  'NETLAB_BASE_URL=http://127.0.0.1:8088 /opt/netlab/acceptance/t225-service-restart.sh'

ssh root@10.72.1.7 \
  'cd /opt/netlab && NETLAB_PRIVILEGED=1 CYCLES=20 go test ./tests/integration/... -run Leak -count=1'
```

Target evidence must record the exact candidate SHA/digest, runtime capability snapshot, Playwright
output directory, restart/recovery output, leak counts, laboratory deletion cleanup, and rollback
decision.
