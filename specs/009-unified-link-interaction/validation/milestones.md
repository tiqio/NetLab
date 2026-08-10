# Milestone Validation Ledger

## Milestone 1 — Unified Endpoint and Command Foundation

```text
Commit SHA: fd2c6e9bb4d109dae01dd9c25a6a9c3c29b2f618
Focused tests: go test ./internal/domain ./internal/store/sqlite ./internal/app/command ./internal/app/reconcile ./internal/api/http ./internal/api/mcp ./cmd/netlabd; npm test -- topologyEndpointCompatibility.test.ts; npm run build
Result: PASS
Notes: Migration 0013, cross-backing endpoint reservations, unified domain/command projection, HTTP/MCP routes and frontend client types are implemented. Vite reports only the existing large-chunk warning.
```

## Milestone 2 — Unified Canvas Interaction

```text
Commit SHA: 5d28414070c18f42c5b24c3d7222059e6fe76ade
Focused tests: go test ./internal/api/http ./internal/api/mcp ./internal/app/command ./internal/app/reconcile ./internal/store/sqlite; npm test -- topologyConnectionController topologyInteractionController topologyGeometry topologyEndpointCompatibility PortChooser TopologyCanvas TopologyWorkspace topologyConnectionPresentation topologyVisualSemantics laboratory; npm run format:check; npm run lint; npm run build; Playwright unifiedPortConnection.spec.ts --list
Result: LOCAL PASS; TARGET JOURNEY PENDING (T037 remains open)
Notes: Direct pointer-captured port drag now emits normalized endpoints for node links, node/object attachments, object links, Bridge access, and NAT access. The source-anchored preview, compatible target feedback, endpoint chooser, revision-conflict refresh, authoritative task merge, 50-drag budget, HTTP/MCP final-state symmetry, and 010 visual presentation reuse are covered locally. ESLint reports zero errors and the repository's existing warnings; Vite reports the existing large-chunk warning. The target-host Playwright journey is registered for three viewports but was not executed without the target runtime.
```

### Milestone 2B — Unified Plus and Keyboard Entry

```text
Commit SHA: f13352ba9f09197e97576484b99d5f9b5946c3d7
Focused tests: npm test -- TopologyCanvas TopologyCanvas.a11y TopologyWorkspace PortChooser topologyKeyboardController topologySelection topologyConnectionController topologyEndpointCompatibility laboratory; npm run format:check; npm run lint; npm run build; Playwright unifiedPlusConnection.spec.ts --list
Result: LOCAL PASS; TARGET JOURNEY PENDING (T048 remains open)
Notes: Node, lightweight L2/L3, Bridge, and NAT resources now expose the same top-right capacity-gated connector. Source selection, target selection, cancellation, selection restoration, live status, keyboard entry, and authoritative unified submission reuse the US1 flow. Vitest passed 77 tests; ESLint reported zero errors with existing warnings; Vite reported the existing large-chunk warning. The target-host Playwright journey is registered for three viewports but was not executed without the target runtime.
```

## Milestone 3 — Four-Port Defaults and Live Lifecycle

```text
Commit SHA: 736add1d04e462a2a8dc5a9d3a485016e589a103; compatibility tests a4f75b2
Focused tests: go test ./internal/domain ./internal/api/http ./internal/api/mcp ./internal/app/command ./internal/app/reconcile; npm test -- lightweightSwitchConfig.test.ts topologyResourceDraft.test.ts CreateTopologyResourceDrawer.test.ts; npm run format:check; npm run lint; npm run build; Playwright lightweightFourPorts.spec.ts --list
Result: LOCAL PASS; TARGET JOURNEY PENDING (T060 remains open)
Notes: New lightweight L2/L3 create requests default to exactly eth0–eth3 in the server-authoritative create path, while explicit configurations and legacy single-port import, export, update, and recovery paths preserve their saved port sets. HTTP and MCP default creation and idempotent replay are covered locally; the SPA/HTTP/MCP Playwright journey is registered for three viewports but was not executed without the target runtime.
```

## Milestone 4 — Final Candidate and Target Validation

```text
Commit SHA:
Focused tests:
Result:
Candidate ID:
Artifact digest:
Target result:
Rollback artifact:
```
