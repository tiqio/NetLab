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
| Target acceptance | pending | clean candidate, deployment, high-density Lab, cleanup | pending | pending |

## Recording Rules

- A milestone is complete only after its focused tests pass and a focused Git commit exists.
- Deployment records must include source SHA, artifact digest, contract digest, build time and schema state.
- Target-host failures return to the local worktree for tests, fixes and a new commit.
- Evidence must not include credentials, bootstrap secrets, terminal-sensitive output or packet contents.
