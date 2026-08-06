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

## 2026-08-06 Overlap Hotfix

- Candidate: `ui-overlap-hotfix-20260806T095807Z-r2`
- Source commit: `d5cfafde226ec351c5daef24ab8ceac84835a971`
- Installed binary SHA-256: `cb2aa0fd2575dedbed0f0b7175a057a01f2726181a6ba9784873d6c859f70969`
- Deployment: the project `deploy/scripts/install.sh` prebuilt-binary path atomically synchronized the service binary, release configuration and template-readiness candidate; no target source file was edited.
- Rollback artifacts: `/usr/local/bin/netlabd.rollback-ui-overlap-hotfix-20260806T095807Z-r2`, `/etc/netlab/netlab.yaml.rollback-ui-overlap-hotfix-20260806T095807Z-r2`, and `/etc/netlab/template-readiness.json.rollback-ui-overlap-hotfix-20260806T095807Z-r2`.
- Health: `netlab.service` active and `GET /healthz` returned `{"status":"ok"}`.
- Release identity: the HTTP capability response reported the candidate, installed binary digest, contract digest and build timestamp above.
- Visual verification: at 1024×768 the topology category legend rendered at the lower-left without intersecting the top toolbar or Inspector toggle; after selecting a Docker node, the vCPU, CPU quota and memory cards remained above the resource chart and the chart no longer rendered an undeclared legend over its plot area.
- Regression gates: 69 frontend test files with 284 tests passed; 12 acceptance test files with 25 tests passed; production build, localization scan and frontend artifact hygiene passed.

## 2026-08-06 Theme and Inspector Simplification

- Candidate: `ui-simplify-theme-20260806T101629Z`
- Source commit: `1c31cc93e757fbd6afe7594f80120048d8839f00`
- Installed binary SHA-256: `343f11cea4b5d30117db2f357db4880ee05200fb6b9efbd32ffada0d7a99ccab`
- Deployment: project prebuilt-binary installation synchronized the binary, release configuration and template readiness without editing target source files.
- Health: `netlab.service` active and `GET /healthz` returned `{"status":"ok"}`.
- Empty Inspector: only the selection guidance remained; the obsolete global “创建链路” card and the disabled network-object attachment card were absent.
- Active network object behavior: the attachment workflow remains available after selecting NAT Bridge, Bridge or Lightweight switching/routing objects because it owns port, PVID and tagged-VLAN configuration that canvas links do not replace.
- Theme control: explicit light selection reported `data-resolved-theme="light"` and “当前生效：浅色主题”; the system option now includes its currently resolved theme.
- Live terminal: an already-open serial session rendered `rgb(248, 250, 252)` in light mode and changed to `rgb(5, 10, 15)` after selecting dark mode without recreating the terminal session.
- Cleanup audit: all 60 remaining Vue components had a production reference; six obsolete components and four isolated tests were removed.
- Regression gates: 65 frontend test files with 274 tests passed; 12 acceptance test files with 25 tests passed; format, lint with zero errors, production build, localization and artifact hygiene passed.
