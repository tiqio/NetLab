# Validation Record: 拓扑添加抽屉

## Planning and Setup

- Date: 2026-08-05
- Feature: `006-topology-add-drawer`
- Checklist: `requirements.md` 16/16 complete
- Pre-implementation worktree: specification, plan, contracts, tasks, Agent context pending commit
- Compatibility history: recorded in `research.md`

## Milestone 1 — Shared Sheet

- Commit SHA: `6bdee8c`
- Focused tests: PASS — `Sheet.test.ts` and `LaboratoryShell.test.ts`, 9 tests
- Type check: PASS — `npm run build`
- Lint/format: PASS for changed files; ESLint reported 14 warnings and 0 errors in focused files
- Repository baseline note: full `npm run format:check` remains blocked by 7 pre-existing unrelated files; changed files pass focused Prettier check
- Worktree clean after commit: yes

## Milestone 2 — Resource Drawer MVP

- Commit SHA: `3524e17`
- Focused tests: PASS — Catalog, Drawer, Workspace and Toolbar, 22 tests
- Playwright: scenario compiles and discovers desktop/minimum projects; runtime execution deferred to deployed candidate because no local acceptance service is active
- Build: PASS — Vue type check and Vite production build
- Worktree clean after commit: yes

## Milestone 3 — Long Form Reliability

- Commit SHA: `713105b`
- Previous US1 SHA: `3524e17`
- Draft/validation tests: PASS — draft model 7 tests; Drawer 12 tests; shared Sheet 7 tests; axe 1 test
- Playwright: PASS discovery — 5 relevant tests discovered for desktop/minimum viewport; runtime execution reserved for candidate service
- Build: PASS — Vue type check and Vite production build
- Worktree clean after commit: yes

## Milestone 4 — Close and Workspace Recovery

- Commit SHA: `d6104db`
- Previous US2 SHA: `713105b`
- Accessibility/keyboard tests: PASS — Sheet close reasons, discard alertdialog, Drawer dirty switch/close, axe, Shell and Toolbar; 40 focused tests passed before milestone polish
- Playwright: PASS discovery — keyboard open/Escape/focus restoration and dirty discard scenarios discovered in both configured viewport projects
- Build: PASS — Vue type check and Vite production build
- Worktree clean after commit: yes

## Final Local Validation

- Source commit SHA: `c10a33c194aae3093bdebad6248277a0d0c631a3`
- Previous US3 SHA: `d6104db`
- Lint: PASS — 0 errors; repository baseline remains 957 warnings
- Format: BASELINE FAIL — only 5 pre-existing unrelated files remain unformatted: `GlobalConsoleWorkspace.test.ts`, `GuestCommandPanel.vue`, `LightweightNodeEditor.vue`, `TopologyInspector.test.ts`, `TopologyInspector.vue`; all feature files pass focused Prettier
- Unit tests: FEATURE PASS — 40 focused tests; full Vitest 235/236 with the sole unrelated failure in `LightweightNodeEditor.test.ts` expecting removed English labels while the component displays Chinese labels
- Go contract tests: PASS — `go test ./...`, including contract, integration, recovery and security packages
- Browser matrix: PASS — two consecutive rounds at 1024×768, 1366×768 and 1920×1080; 12/12 drawer journeys passed with right alignment, fixed actions, draft preservation, authoritative creation and unchanged canvas bounds
- Build: PASS — `make build` synchronized `web/dist` into embedded `internal/api/http/webdist` and produced `bin/netlabd`

## Candidate Artifact

- Version: `0.1.0-006.c10a33c1`
- Candidate ID: `006-topology-add-drawer-c10a33c1-20260805`
- Contract digest: `sha256:30b394d5d1210614064cf732f714679f3f98072c4bc0b4fed6f2d22476201984`
- Binary digest: `sha256:18309d9330492453970ad8050fe12b11a1fef13639f5a02a2313716cd5c4948a`
- Built at: `2026-08-05T18:53:52+08:00`
- Candidate directory: `/var/tmp/netlab-candidates/006-topology-add-drawer`
- Candidate metadata: `/var/tmp/netlab-candidates/006-topology-add-drawer/release.json`

## Target Deployment

- Target: `10.72.1.7`
- Deployment time: `2026-08-05T18:55:08+08:00`
- Previous candidate: `pc-dhcp-helper-c1e4b73-20260805`
- Release verification: PASS — `/api/v1/capabilities` reports the candidate ID, version, contract digest, build time and binary digest matching the uploaded artifact
- Service health: PASS — `GET /healthz` returned `{"status":"ok"}` after `deploy/scripts/install.sh`
- Deployment evidence: `/var/tmp/netlab-add-drawer-acceptance/deployment.json`

## Target Acceptance

- Acceptance laboratory: PASS — `Topology Add Drawer Acceptance` created through the real Chromium UI
- Focused Playwright: PASS — 4/4 desktop and 1024×768 target drawer journeys; the earlier non-focused subset run also passed all tests but correctly failed the global full-suite coverage gate
- Ubuntu QEMU creation: PASS — `Acceptance Ubuntu QEMU` created from the Ubuntu QEMU catalog entry with an authoritative node and two interfaces
- BusyBox Docker creation: PASS — `Acceptance BusyBox Docker` created through the drawer
- PC creation: PASS — `Acceptance PC` created through the drawer
- Network object creation: PASS — `Acceptance L2` created as a Lightweight L2 network object
- Multi-client isolation: PASS — the second Chromium context saw all four successful resources, did not see the first client drawer or `browser-local-unsaved-draft`, and the draft was absent from the API
- Failure-path recovery: PASS — disabling the selected Ubuntu template version after the drawer opened produced the stale catalog message, preserved `stale-template-must-not-create`, created no ghost resource, and the version was restored immediately afterward
- Cleanup: PASS — laboratory deletion task `019fd196aceb-3acdfb4a85f5efc32be2` succeeded; zero laboratories, zero active runtime ownership records and zero nonterminal tasks remained
- Evidence files: `/var/tmp/netlab-add-drawer-acceptance/resource-creation.json`, `/var/tmp/netlab-add-drawer-acceptance/multi-client.json`, `/var/tmp/netlab-add-drawer-acceptance/failure-path.json`, `/var/tmp/netlab-add-drawer-acceptance/cleanup.json`
- Rollback required: no
- Deployed source remains T049 commit `c10a33c194aae3093bdebad6248277a0d0c631a3`; this final commit changes evidence documents only
