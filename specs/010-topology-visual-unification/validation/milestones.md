# Milestone Evidence

## M1 — Unified Connection Presentation

```text
Commit SHA: 550cf74
Focused tests: TopologyCanvas unit/a11y/performance, GlobalCaptureWorkspace, TrafficFilterPanel, desktop mocked Playwright
Result: PASS
Notes: Traffic Filter and capture overlays are connection-ID scoped, time bounded, and preserve the underlying connection state. All connection kinds expose consistent inspector/context capabilities and explicit disabled reasons.
```

## M2 — Semantic Markers and Dynamic Legend

```text
Commit SHA: 550cf74
Focused tests: TopologyConnectionLegend component tests, Vue typecheck, focused ESLint, desktop mocked Playwright in light/dark themes
Result: PASS
Notes: State and semantic text is centralized in Chinese terminology. The dynamic legend supports collapse, scroll, keyboard focus, accessible names, and non-destructive connection highlighting.
```

## M3 — Authoritative Collision-Free Placement

```text
Commit SHA: 4365781, 95097ee
Focused tests: Go domain/SQLite/reconcile/HTTP/MCP; authoritative HTTP contract; 20-resource Playwright; drawer/workspace/store tests
Result: PASS
Notes: Nodes and network objects commit resource, placement, laboratory revision, and ordered outbox events atomically. SPA consumes the returned placement without a second position write. Twenty mixed resources created at one center remain footprint-safe and stable after refresh.
```

## M4 — Control-Plane Parity and Recovery

```text
Commit SHA: 550cf74, 95097ee
Focused tests: Go command/query/SQLite/HTTP/MCP/stream; 334 Vitest; 25 acceptance-unit; two-browser/API/MCP concurrency; real service restart and deletion cleanup
Result: PASS
Notes: Import/export preserves placements and fills only missing legacy coordinates. Creation publishes placement before resource events, preventing temporary (0,0). Ten concurrent four-client groups converge within two seconds using revision refresh and new idempotency keys. Exact placements and connection summaries survive restart, then terminal laboratory deletion removes owned state.
```

## Deployment Candidate

```text
Candidate ID: topology-visual-010-20260811T013126Z
Product source SHA: 2a8a7961e1e3341d5c778f325f5a570272068506
Final acceptance harness SHA: ce1eca0
Artifact digest: sha256:ac275806eb8bbb9f06aa1cf0f9949c3ad5b908ec2ab996506a13ed338bd8c100
Migration state: 14
Contract version: sha256:2e1a4279f16d7e6c282b741f06c52e75f1db210e91dbf4b99180bea5c9da808b
Build time: 2026-08-11T01:31:26Z
Deployment time: 2026-08-11T01:42:02Z
Target validation result: PASS — 27 Playwright, 30 acceptance-unit, 20-cycle leak gate, zero cleanup remainder
Previous artifact: unified-link-009-20260810T162640Z-r8, sha256:3d3511d1ee4c77baffa0386ab6a136b04654c8fc40c43543447faccda5f6efae
Rollback directory: /var/lib/netlab/rollback/topology-visual-010-20260811T013126Z-predeploy
```

## Post-Acceptance r2

```text
Candidate ID: topology-visual-010-20260811T021524Z-r2
Commit SHA: f509048c832f8644534077de0c64e8321c1bc0a7
Artifact digest: sha256:e776eed9b0ef5193137c7f6baec29ac74121d6ff36c43e93a3332286006584f3
Deployment time: 2026-08-11T02:16:14Z
Result: PASS — default unaddressed L3 active, L3↔L2 connected, invalid CIDR rejected before persistence
Three-backing recovery: 3 passed across desktop, standard and minimum viewports
Cleanup: baseline_restored=true, remaining_count=0
Previous candidate rollback: /var/lib/netlab/rollback/topology-visual-010-20260811T021524Z-r2-predeploy
```

## Post-Acceptance r3

```text
Candidate ID: topology-visual-010-20260811T024306Z-r3
Commit SHA: 0e3c528951186bee8a3cb59c38ffd864cb29499a
Artifact digest: sha256:03c4cf394b57a59098afa29b5a1c9f281127d73c4001e150556c85f8d031120f
Deployment time: 2026-08-11T02:53:19Z
Migration state: 15
Result: PASS — durable Traffic Filter statistics schema installed and Ctrl hold-to-pan verified in target Chromium
Target state: netlab active, health ok, no NetLab error-level journal entries, user resource counts unchanged
Rollback: 1.6G online SQLite backup integrity ok and all recorded SHA-256 checks passed
Previous candidate rollback: /var/lib/netlab/rollback/topology-visual-010-20260811T024306Z-r3-predeploy
```

## Post-Acceptance r4

```text
Candidate ID: topology-visual-010-20260811T031446Z-r4
Commit SHA: e6bac120b24774851ccef7c5d036200283dd6e62
Artifact digest: sha256:39ccf99e66f71df82bee5b215432f71f56310d239615d4ce73a72e0e28ffcbb7
Deployment time: 2026-08-11T03:16:28Z
Migration state: 15 (unchanged)
Result: PASS — QEMU serial limitation clarified and two VNC tabs repeatedly connected and displayed
Target state: netlab active, health ok, zero NetLab error-level journal entries, user resources unchanged
Rollback verification: all recorded r3 file SHA-256 checks passed
Previous candidate rollback: /var/lib/netlab/rollback/topology-visual-010-20260811T031446Z-r4-predeploy
```

## Post-Acceptance Unlimited QEMU Capacity

```text
Candidate ID: qemu-unlimited-20260811T071624Z-r2
Commit SHA: 3b7f4121f1cf3faf270f957b2fdd7834837f4fe5
Feature commit: efe2fbf6489c4d94cc05494d5b3e14f7cd3c6c92
Artifact digest: sha256:d2bfd22b133dde62677f94b9013f282ff0907a43aafa66e25eac6eeb75575b1d
Deployment date: 2026-08-11
Migration state: 15 (unchanged)
Result: PASS — max_running_qemu=0 accepted six simultaneous QEMU nodes without errors
Target state: netlab active, health ok, three tested QEMU links connected
Safety: host memory, disk, node-shape, interface and cgroup protections retained
Failure UX: structured resource-creation and node-operation warning dialogs deployed
Rollback: /var/lib/netlab/rollback/qemu-unlimited-20260811T071624Z-r2-predeploy
```
