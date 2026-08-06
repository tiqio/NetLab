# Validation: 全页面中文化与明暗主题

## Local milestones

- Planning baseline: `ba6040f`.
- Localization/theme foundation: `02f9e42`.
- Candidate implementation: pending final commit in this validation cycle.

## Local validation — 2026-08-06

- `scripts/check-ui-localization.sh`: PASS.
- `npm run format:check`: PASS.
- `npm run lint`: PASS with zero errors; repository baseline Vue style warnings remain.
- `npm test`: PASS, 68 files and 257 tests.
- Focused topology performance test: PASS.
- `npm run build`: PASS.
- `go test ./...`: PASS, including contract, integration, recovery and security packages.
- New Playwright minimum-view project: PASS, 9/9 tests covering Chinese routes, theme persistence, client isolation, keyboard focus, two themes and 1024×768 / 1366×768 / 1920×1080.

## Contract and state evidence

- No backend domain, application command, API, MCP, event or SQLite source was changed.
- Theme preference remains browser-local under `netlab.appearance.v1`.
- Existing machine-readable statuses remain unchanged; only presentation is localized.
- `go test ./...` confirms HTTP/MCP and generated contract compatibility.

## Security and artifact hygiene

- No credentials, packet captures, proprietary images or bootstrap secrets were added.
- Generated SPA assets are embedded through the existing `make build` pipeline.
- SQLite migration state is unchanged.

## Target validation

Pending deployment of the clean candidate to `10.72.1.7`.

## Candidate and target evidence — 2026-08-06

- Source commit: `9009df58aea1256c2ef7203f7efda790440feda6`.
- Candidate: `ui-zh-theme-9009df58aea1`, version `0.1.0-ui-zh-theme`.
- Contract digest: `sha256:fab929cd33f598834af85510cbd8577928c7e3725a9671335799ac3827e86bfe`.
- Embedded provenance binary digest: `sha256:aee51f886160e73712623d63947f91749b4c2d7b8862abed45089d4db0ace072`.
- Installed artifact digest: `sha256:908c8f2e685403ab0ccf21862b5bf6523c66fb9ef97de4c4ef77c58c623ac255`.
- Build time: `2026-08-06T03:04:07Z`.
- Target service: active on `10.72.1.7:18082`.
- Target HTML: `lang=zh-CN` and pre-paint `netlab.appearance.v1` bootstrap confirmed.
- Target Playwright UI matrix: PASS 9/9 for Chinese routes, two themes, three viewports, refresh persistence and two-client isolation.
- Target baseline after cleanup: `[]`.
- First deployment attempt was rejected safely by authoritative release validation; service was restored with correctly prefixed SHA-256 identities and matching template-readiness candidate metadata.
- Focused QEMU Console and Docker Traffic Filter journeys remain pending because legacy page objects still locate English-only accessible names after the UI contract changed to Chinese.
