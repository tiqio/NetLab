# Milestone Evidence

| Milestone | Commit SHA | Focused gates | Result | Candidate / target evidence |
|---|---|---|---|---|
| Planning and audit baseline | `d01fef6` | spec/plan/tasks/schema validation | passed | n/a |
| Shared visual foundation | `12f1237` | 53 component tests; 9 acceptance fixture tests; build; localization scan; frontend acceptance contract | passed | n/a |
| Topology clarity | `cee0087` | 27 geometry/canvas/link/traffic component tests; frontend build; localization scan; Playwright at 1024×768, 1366×768 and 1920×1080 | passed | local disposable acceptance |
| Inspector and workspaces | `1a394a8` | 40 inspector/chart/menu/workspace tests; 15 axe/focus tests; localization scan; build; three-viewport diagnostics journey | passed | local disposable acceptance |
| Full Chinese UI | `256dd7d` | 49 focused component/locales tests; strengthened interpolation scan; frontend build; six three-viewport localization journeys | passed | local disposable acceptance |
| Responsive and theme stability | `a4a2be9` | 6 shell sizing tests; 27 dual-theme/three-viewport/axe tests; 9 twenty-cycle and 125% input tests; build and localization scan | passed | local disposable acceptance |
| Continuous audit | `fc10cf4` | 5 inventory/evidence/redaction tests; duplicate and missing-matrix injection; 9 three-viewport visual audit journeys; artifact hygiene | passed | local disposable acceptance |
| Final local quality gates | `b280f6c` | Go lint/unit/contract/security; 282 frontend unit tests; 25 acceptance unit tests; localization/schema/artifact checks; 51 browser matrix scenarios with the single contrast finding corrected and 18/18 axe rerun passing | passed | `ui-overlap-final-matrix`, `ui-overlap-light-contrast` |
| Target acceptance | `d678ceb` | candidate `ui-overlap-20260806T085508Z-r2`; target three-viewport visual/localization matrix; 24/24 theme and axe rerun; 3/3 diagnostics rerun; 3/3 125% matrix; evidence schema and artifact hygiene | passed | installed SHA-256 `9db2583aaa8b1e46844f25cf0fa36d60dbd19192388d114325e50389f0ec3b7a`; owned residuals 0; user Lab baseline unchanged |

## Recording Rules

- A milestone is complete only after its focused tests pass and a focused Git commit exists.
- Deployment records must include source SHA, artifact digest, contract digest, build time and schema state.
- Target-host failures return to the local worktree for tests, fixes and a new commit.
- Evidence must not include credentials, bootstrap secrets, terminal-sensitive output or packet contents.
