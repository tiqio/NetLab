# Implementation Notes: Network Object Links and Docker Routes

## Baseline

- Baseline commit inspected: `74dadf3` (`chore: establish recovered NetLab repository baseline`).
- Feature planning commits: `09ace49` and `326d9ef`.
- The recovered repository has no earlier file-level history, so compatibility decisions are derived
  from the baseline code, public contracts, current tests, and feature design artifacts.

## Compatibility Constraints

- Preserve the public flat object-link endpoint fields: `object_a_id`, `port_a_name`, `object_b_id`,
  and `port_b_name`.
- Preserve existing node-interface links, network attachments, NAT, host-interface capture, standard
  link capture, and existing Traffic Filter scopes.
- Replace only the runtime shape of network-object links: one veth pair moved directly between the two
  namespaces instead of a host bridge plus two veth pairs.
- Keep endpoint A/B ordering stable because capture and Traffic Filter direction use endpoint A as the
  orientation reference.
- Never derive deletion ownership from a generic interface shape; use deterministic NetLab names and
  durable resource identity.
- Docker endpoint route execution already uses argument vectors through `CommandExecutor`; retain that
  boundary and carry typed route declarations through the control and persistence layers.
- Browser, HTTP, and MCP mutations must converge through shared application services and durable tasks.
- All implementation and tests run locally and receive focused commits before deployment to
  `10.72.1.7`; source files are never edited directly on the target.

## Milestone Evidence

| Milestone | Local gates | Commit SHA | Notes |
|---|---|---|---|
| Setup | Ignore rules and fixture compilation | pending | No ignore changes required |
| Foundation | Domain, SQLite migration/repository, command, ports, Linux networking, capture runtime, reconcile, and focused contract gates passed | `61cec9b` | Added durable object-link observation state, atomic `link_deleted` completion, observation/runtime ports, and object-link import/export remapping |
| US1 | pending | pending | |
| US2 | pending | pending | |
| US3 | pending | pending | |
| US4 | Focused domain, command, store, API, runtime, reconcile, contract, Vitest, build, and changed-file ESLint gates passed | `c18f223` | Canonical route declarations survive create/settings/export/import and expose route-specific readiness; ESLint reports warnings only |
| Final candidate | pending | pending | |

## Incremental Worklog

- `c3338a1` (`feat: observe and clean up object links`): added namespace-resolved object-link capture
  identity, explicit Traffic Filter object-link scope and attribution, `link_deleted` capture completion,
  pre-delete observer cleanup, generated frontend types, and scope selection fixes.
- Focused gates for `c3338a1`: `go test ./internal/runtime/capture ./internal/app/reconcile
  ./internal/api/http ./internal/api/mcp ./cmd/netlabd`; `go test -race ./internal/app/reconcile -run
  'TestCaptureStopNetworkObjectLinkUsesLinkDeletedReason|TestTrafficFilterAttributesNetworkObjectLink'
  -count=1`; `go test ./tests/contract -run
  'TestNetworkObjectLinkAndRouteContractDelta|TestGeneratedClientIncludesObjectLinkCaptureAndFilterTypes'
  -count=1`; `npm test -- --run src/features/diagnostics/TrafficFilterPanel.test.ts`; and
  `npm run build`. All passed locally. The Vite chunk-size warning remains informational.
- `26f7ab3` (`feat: reconcile owned Docker routes`): persists the exact NetLab-owned route set inside
  the container root, removes only stale recorded routes, applies IPv4 and IPv6 declarations through
  argument vectors, restores the prior set after route or ownership-write failures, and remains
  idempotent across repeated endpoint reconciliation.
- Focused gate for `26f7ab3`: `go test ./internal/runtime/linuxnet ./internal/runtime/docker
  ./internal/app/command ./internal/store/sqlite -count=1`; all passed locally.
- `96473ee` (`feat: configure Docker routes through control surfaces`): allows stopped Docker interface
  settings through HTTP, adds typed Docker route create/settings MCP schemas and mutation support,
  synchronizes generated TypeScript request types, adds family-aware route editors to node creation and
  stopped-node Settings, and restores the `listNetworkObjectLinks` generated client operation.
- Focused gates for `96473ee`: Docker route REST/MCP/generated-client contract tests, HTTP validation
  tests, `NodeConfigurationPanel` and `CreateTopologyResourceDialog` Vitest suites, frontend build, and
  changed-file ESLint all passed. Changed-file ESLint reported warnings only. The full contract suite
  still reports the unrelated built-in template-count baseline failure (`templates=8`).
- `1ac2be6` (`fix: reapply Docker routes during recovery`): makes the node reconciler invoke the
  idempotent Docker runtime ensure path for already-running containers during service/host recovery,
  rejects readiness when endpoint or route reconciliation fails, and covers stopped-container restart.
- Focused gates for `1ac2be6`: `go test ./internal/runtime/docker ./internal/app/reconcile
  ./internal/runtime/linuxnet ./internal/api/http ./internal/api/mcp ./internal/store/sqlite
  ./internal/app/command -count=1`; focused Docker recovery race tests; and Docker route/OpenAPI contract
  tests. All passed locally.
- `c18f223` (`feat: report Docker route readiness`): canonicalizes Docker interface addresses, route
  destinations, and gateways before persistence; rejects stable-coded family, metric, gateway,
  connected-prefix, duplicate-prefix, and cross-interface conflicts; preserves routes through create,
  stopped settings, export, and import; and reports pending, applying, applied, and route-specific failed
  readiness in the node operations panel and topology Inspector.
- Focused gates for `c18f223`: `go test ./internal/domain ./internal/app/command
  ./internal/store/sqlite ./internal/api/http ./internal/api/mcp ./internal/runtime/linuxnet
  ./internal/runtime/docker ./internal/app/reconcile -count=1`; focused Docker route contract tests;
  `npm test -- --run src/features/nodes/NodeOperationsPanel.test.ts
  src/features/topology/TopologyInspector.test.ts`; `npm run build`; and changed-file ESLint. All passed
  locally. Vite chunk-size output and 115 pre-existing Inspector lint warnings remain informational;
  changed-file ESLint reported zero errors.
- `3a36938` (`test: cover Docker routed traffic recovery`): adds a root-gated, digest-pinned Docker
  acceptance test with two isolated container endpoints, two bridge segments, an isolated forwarding
  namespace, exact IPv4/IPv6 route assertions, ICMP/TCP/UDP traffic, container stop/start, and a fresh
  adapter recovery pass. The test configures container routes only through the runtime adapter and never
  uses manual `nsenter` route setup.
- Focused gates for `3a36938`: `go test ./tests/integration ./tests/testsupport -run
  TestPrivilegedDockerStaticRoutePath -count=1` and `go vet ./tests/integration ./tests/testsupport` passed
  locally. Dynamic execution is intentionally gated by `NETLAB_PRIVILEGED=1` and an approved
  `NETLAB_DOCKER_ROUTE_IMAGE` pinned with `@sha256:`.
- `3703ff5` (`test: cover Docker route browser journey`): adds a two-viewport Playwright journey that
  creates canonicalized IPv4 and IPv6 routes through the Add-to-topology dialog, verifies authoritative
  readback, edits the exact stopped-node route set through Settings, and verifies the Inspector summary.
- Focused gate for `3703ff5`: `NODE_PATH=$PWD/node_modules npx playwright test --config
  playwright.config.ts --list ../tests/e2e/journeys/dockerStaticRoutes.spec.ts` discovered the desktop and
  minimum-viewport cases locally. Dynamic execution remains target-host gated because it requires the
  real Docker runtime and available BusyBox image.
- `61cec9b` (`feat: persist object link observations`): adds migration `0012` for durable capture
  lifecycle metadata, Traffic Filter scopes, and normalized directional observations; atomically commits
  `link_deleted` capture state with its task, audit record, and outbox events; defines stable namespace-aware
  observation locators and exact managed-route runtime boundaries; and preserves/remaps first-class
  network-object links through export/import without runtime locators or packet payloads. Imported active
  links reserve both ports transactionally, while deleted intent does not reoccupy endpoints.
- Focused gates for `61cec9b`: `go test ./internal/domain ./internal/store/sqlite
  ./internal/app/command ./internal/app/ports ./internal/runtime/linuxnet ./internal/runtime/capture
  ./internal/app/reconcile -count=1`; focused object-link/route/OpenAPI contract tests; and dedicated
  observation completion/import rollback tests all passed locally on August 3, 2026.
