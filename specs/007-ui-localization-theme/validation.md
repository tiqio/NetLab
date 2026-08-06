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
