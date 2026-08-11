# US1 Recovery Validation

- Validation time: `2026-08-11T16:41:35+08:00`
- Scope: User Story 1 recovery-honesty MVP
- Local branch: `main`
- Predecessor commit: `ab9b924 feat(recovery): expose durable network reconcile controls`

## Recovery Behavior

- Startup recovery runs in explicit dependency order: nodes, network-object backing and ports, durable tasks, endpoint reservations, data plane, port mappings, captures.
- A participant failure remains structured and does not prevent later participants from reconciling.
- Failed or cancelled topology-connection tasks leave `pending` or `disconnecting` resources in a structured `failed` state while retaining endpoint reservations for retry or delete.
- Queued or running tasks protect their resources from premature failure finalization.
- Network-object and object-link reconcile operations are revisioned, durable, idempotent, and exposed through HTTP and MCP.
- Diagnostics expose desired versus observed state, backing ownership/usability, recovery phase, cleanup guidance, and operator hints.
- The SPA renders honest failed state and offers retry or confirmed deletion without presenting the resource as healthy.

## Validation Gates

| Gate | Result | Notes |
|------|--------|-------|
| `git diff --check` | PASS | No whitespace errors. |
| `go test ./...` | PASS | Unit, contract, integration, recovery, security and leak suites passed. |
| `go vet ./...` | PASS | No findings. |
| `NETLAB_PRIVILEGED_TESTS=1 go test -v ./tests/recovery -run TestServiceRestartRecreatesInvalidOwnedNamespaceReference -count=1` | PASS | Invalid `/run/netns` reference was safely recreated and inspected usable. |
| `go test ./tests/integration -run TestBridgeToL3CleanupCompletesTwentyCyclesWithoutNamespaceLeak -count=1` | PASS | Included in full Go suite; 20 cleanup cycles issued only the L3 namespace endpoint deletion. |
| `npm test` | PASS | Full Vitest suite passed. |
| `npm run lint` | PASS | Zero errors; 924 pre-existing style warnings remain. |
| `npm run format:check` | PASS | Prettier check passed. |
| `npm run build` | PASS | Vue type-check and Vite production build passed; existing large-chunk warning remains. |

## Evidence

- Recovery coordinator tests cover dependency order, partial-failure isolation, durable checkpoints and recovered-link event publication.
- SQLite recovery tests cover failed/cancelled terminal tasks, active-task protection, structured errors, reservation retention, ordered outbox events and idempotent reruns.
- Frontend tests cover failed backing visibility, recovery phase/cleanup/operator hints, retry actions and failed-link deletion actions.
- Privileged recovery test creates a namespace, invalidates its `/run/netns` mount reference, configures the owned L3 object through a fresh runtime instance, and verifies the recreated backing is owned and usable.
