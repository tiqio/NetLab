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
