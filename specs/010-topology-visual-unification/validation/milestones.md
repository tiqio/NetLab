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
Commit SHA: 4365781
Focused tests: go test ./internal/domain ./internal/store/sqlite ./internal/app/reconcile ./internal/api/http ./internal/api/mcp; targeted contract parity; Vue typecheck/lint; 40 drawer/workspace/store tests
Result: PASS
Notes: Nodes and network objects now commit resource, placement, laboratory revision, and outbox audit atomically. SPA consumes returned placement and no longer issues a post-create placement batch. Dense 20-resource allocation and same-revision concurrent admission are covered. Local Playwright remains pending because the configured webServer did not become ready within 30 seconds.
```

## M4 — Control-Plane Parity and Recovery

```text
Commit SHA: 550cf74
Focused tests: Go command/query/SQLite/HTTP/MCP/stream packages; 87 focused Vitest tests; Vue typecheck; stable local Playwright visual scenarios
Result: PARTIAL PASS
Notes: Import/export preserves placements and deterministically fills only missing legacy coordinates. Atomic create now publishes placement before resource creation so other clients converge without temporary (0,0). MCP conflicts, restart persistence, stale/duplicate placement events, and local fallback stability are covered. Concurrent browser and target-host recovery acceptance remain pending.
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
