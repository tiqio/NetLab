# Milestone Evidence

| Milestone | Commit SHA | Focused gates | Result | Candidate / target evidence |
|---|---|---|---|---|
| Planning and audit baseline | `d01fef6` | spec/plan/tasks/schema validation | passed | n/a |
| Shared visual foundation | pending | 53 component tests; 9 acceptance fixture tests; build; localization scan; frontend acceptance contract | passed | n/a |
| Topology clarity | pending | geometry, canvas, links, traffic overlay, three viewports | pending | n/a |
| Inspector and workspaces | pending | inspector, charts, menus, bottom workspace, axe | pending | n/a |
| Full Chinese UI | pending | localization scan, terminology, error branches, journeys | pending | n/a |
| Responsive and theme stability | pending | dual theme, three viewports, 125%, repeated transitions | pending | n/a |
| Continuous audit | pending | scenario inventory, evidence schema, redaction, injected failures | pending | n/a |
| Target acceptance | pending | clean candidate, deployment, high-density Lab, cleanup | pending | pending |

## Recording Rules

- A milestone is complete only after its focused tests pass and a focused Git commit exists.
- Deployment records must include source SHA, artifact digest, contract digest, build time and schema state.
- Target-host failures return to the local worktree for tests, fixes and a new commit.
- Evidence must not include credentials, bootstrap secrets, terminal-sensitive output or packet contents.
