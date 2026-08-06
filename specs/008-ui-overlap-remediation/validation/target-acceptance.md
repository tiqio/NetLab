# Target Acceptance

- Target: `10.72.1.7:18082`
- Candidate: `ui-overlap-20260806T085508Z-r2`
- Installed binary SHA-256: `9db2583aaa8b1e46844f25cf0fa36d60dbd19192388d114325e50389f0ec3b7a`
- Deployment method: uploaded the locally built binary, backed up `/usr/local/bin/netlabd`, synchronized `/etc/netlab/template-readiness.json`, and restarted `netlab.service`; no target source files were edited.
- Rollback: `/usr/local/bin/netlabd.rollback-ui-overlap-r1` and `/etc/netlab/template-readiness.json.rollback-ui-overlap-r1`.
- Service result: `active`; `GET /healthz` returned `{"status":"ok"}`.

## Acceptance Results

- Three viewport target matrix: 1024×768, 1366×768 and 1920×1080.
- Themes and accessibility: final focused rerun passed 24/24; light/dark contrast, keyboard focus and separator ARIA passed.
- Visual/localization matrix: 49/51 passed in the combined run; the two laboratory-menu timing failures were rerun independently and passed 3/3 across all viewports.
- Visual audit routes: topology, templates and automation passed at all three target viewports.
- Responsive keyboard flows: command palette and right-side add drawer passed.
- 125% flow: add-resource drawer, close action, command palette, keyboard focus and horizontal-overflow checks passed 3/3.
- Localization gate: unapproved English product text count is 0; technical names remain unchanged.
- Regression found and fixed on target: resizable separators now expose required ARIA values; target baseline preservation and asynchronous focus helpers are deterministic.

## Cleanup

- Existing user Lab before: 1, ID `019fd5b6ed98-323749f52653bd5cecf1` (`ddtest`).
- Existing user Lab after: 1, same ID and revision.
- Acceptance-owned residual resources: 0.
- Nonterminal tasks after acceptance: 0.
- Each passing run reports `baseline_restored: true` and `remaining_count: 0`.
- Final evidence: `specs/008-ui-overlap-remediation/validation/acceptance-evidence.json`.
