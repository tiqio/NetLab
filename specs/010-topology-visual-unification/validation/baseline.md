# Baseline Validation

**Recorded**: 2026-08-07 15:56–15:57 CST

## Go

```bash
go test ./internal/domain ./internal/app/command ./internal/app/query ./internal/store/sqlite
```

Result: PASS. All four packages completed successfully.

## Frontend Unit and Component Tests

```bash
cd web
npx vitest run \
  src/features/topology/TopologyCanvas.test.ts \
  src/features/topology/TopologyCanvas.a11y.test.ts \
  src/features/topology/TopologyCanvas.performance.test.ts \
  src/features/topology/TopologyWorkspace.test.ts \
  src/features/topology/topologyVisualSemantics.test.ts \
  src/features/topology/linkPresentation.test.ts \
  src/features/topology/topologyLayout.test.ts \
  src/features/topology/topologyPlacementBatch.test.ts \
  src/stores/laboratory.test.ts \
  src/stores/laboratory.runtimeTruth.test.ts
```

Result: PASS. 10 test files and 58 tests passed.

## Browser Acceptance Baseline

The initial command used a nonexistent `chromium` project; repository projects are `desktop`, `standard`, and `minimum`. The corrected command was:

```bash
cd web
npx playwright test ../tests/e2e/journeys/topologyVisualRecognition.spec.ts --project=standard
```

Result: BLOCKED before test execution because the configured local acceptance web server did not become ready within 30 seconds. This is a baseline environment/startup condition, not a topology assertion failure. Re-run after the implementation candidate is built and the acceptance profile prerequisites are available.
