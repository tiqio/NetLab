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
Candidate ID:
Commit SHA:
Artifact digest:
Migration state:
Contract version:
Build time:
Deployment time:
Target validation result:
Previous artifact:
Rollback command:
```
