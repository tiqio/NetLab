# Baseline Validation — 2026-08-10

## Backend

```bash
go test ./internal/domain/... \
  ./internal/app/command/... \
  ./internal/app/reconcile/... \
  ./internal/store/sqlite/... \
  ./internal/api/http/... \
  ./internal/api/mcp/...
```

Result: PASS.

```text
internal/domain             PASS
internal/app/command        PASS
internal/app/reconcile      PASS
internal/store/sqlite       PASS
internal/api/http           PASS
internal/api/mcp            PASS
```

## Frontend

```bash
cd web
npm test -- \
  src/features/topology/topologyConnectionController.test.ts \
  src/features/topology/topologyInteractionController.test.ts \
  src/features/topology/TopologyCanvas.test.ts \
  src/features/topology/TopologyCanvas.a11y.test.ts \
  src/features/topology/TopologyCanvas.performance.test.ts \
  src/features/topology/TopologyWorkspace.test.ts \
  src/features/topology/topologyConnectionPresentation.test.ts \
  src/features/diagnostics/GlobalCaptureWorkspace.test.ts \
  src/features/diagnostics/TrafficFilterPanel.test.ts
```

Result: PASS — 9 files, 49 tests.

## Baseline Constraints

- Existing node-node and object-object click flows pass before unified interaction changes.
- Existing 010 connection presentation, accessibility, traffic overlays, and scale fixture pass.
- Existing backend command, reconciliation, SQLite, HTTP, and MCP suites pass before migration 0013.
