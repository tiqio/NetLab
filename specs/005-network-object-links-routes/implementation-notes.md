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
| US1 | Durable task, repository, HTTP/MCP parity, outbox ordering, and race gates passed | `d542d2f` | Object-link create/list/get now share idempotent durable commands and synchronous occupied-port admission |
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
- `d542d2f` (`feat: add durable object link controls`): routes first-class object-link creation through
  the shared durable task runner, returns the same predicted link/task envelope for repeated idempotency
  keys across REST and MCP, adds authoritative single-link GET, validates laboratory membership and port
  occupancy before enqueue, and maps occupied endpoints to HTTP 409 while invalid topology is HTTP 422.
  Repository events retain revision 1 and precede the terminal task update in ordered outbox sequence.
- Focused gates for `d542d2f`: core domain/store/task/reconcile/HTTP/MCP/stream tests, the object-link
  control parity contract suite, and `go test -race ./internal/app/reconcile -run
  TestNetworkObjectLinkCreateIsDurableIdempotentAndObservable -count=1` all passed locally on August 3,
  2026.
- `d3bf1d8` (`feat: improve object link topology UX`): synchronizes the SPA object-link create task
  envelope and authoritative GET method, adds readable `object:port ↔ object:port` labels and distinct
  parallel curves, filters occupied named object ports, and presents desired/actual state, revision,
  structured runtime failures, and the submitted lifecycle task ID in the Inspector. This milestone
  completes T027; shared refresh recovery, canvas-driven object-port connection, and durable task
  follow-up remain open under T019, T028, and T029.
- Focused gates for `d3bf1d8`: 19 topology controller/canvas/Inspector/presentation Vitest cases passed;
  `npm run build` passed; and changed-file ESLint completed with zero errors. Existing Vue style/default
  warnings and the Vite chunk-size warning remain informational.
- `38bbf37` (`feat: connect shared network object ports`): reconciles first-class object-link create,
  state, and delete events into every active Pinia client; preserves authoritative links when reopening
  a laboratory; renders configured PC/L2/L3 object ports around selected or hovered objects; prevents
  occupied endpoint selection; creates object links by selecting two canvas ports; records the returned
  durable task globally; and shows the latest task state, progress, and structured failure in the
  selected-link Inspector. This completes T019, T028, and T029.
- Focused gates for `38bbf37`: 37 store/controller/presentation/canvas/Inspector Vitest cases passed;
  `npm run build` passed; and changed-file ESLint completed with zero errors. Existing Vue style/default
  warnings and the Vite chunk-size warning remain informational.
- `a926dfb` (`feat: publish object link state events`): commits each object-link observed-state and
  structured-error update atomically with a revisioned `network_object_link.state_changed` outbox event,
  allowing every connected client to observe pending, connected, and failed transitions. T026 remains
  open until recovery-specific event publication is implemented with the ownership recovery work.
- Focused gates for `a926dfb`: full `internal/store/sqlite` and `internal/app/reconcile` package tests and
  `go vet` passed locally on August 3, 2026.
- `a02861f` (`test: cover network object link paths`): adds a root-gated three-object L3 netns
  acceptance test covering two isolated parallel links, routed three-object traffic, bidirectional
  ICMP/TCP/UDP, live deletion of one parallel pair, and service-restart-style idempotent re-ensure with a
  fresh DataPlane instance. This completes T018.
- Focused gates for this milestone: the integration package discovered and cleanly skipped the dynamic
  test without `NETLAB_PRIVILEGED=1`, and `go vet ./tests/integration` passed locally on August 3, 2026.
- `2152d65` (`feat: recover direct object links`): scans every owned network namespace for explicitly
  aliased direct-veth endpoints, records deterministic two-endpoint manifests, safely removes previously
  observed orphans, replaces half-created namespace or host pairs, reconfigures adopted complete pairs,
  and publishes task-correlated `network_object_link.recovered` events after state convergence. The shared
  event stream preserves create, state, recovery, and delete ordering. This completes T022 and T026.
- The earlier `61cec9b` export/import milestone already remaps object IDs, preserves named ports, reserves
  both active endpoints transactionally, avoids reservations for deleted intent, and rolls back conflicts;
  the focused command/repository tests were rerun in this milestone, completing T030.
- Focused gates for `2152d65`: full ownership, Linux dataplane, reconciliation, SQLite, event publisher,
  and stream tests passed; relevant import/export tests passed; `go vet` passed; and focused recovery tests
  passed under the race detector locally on August 3, 2026.
- `462fbcb` (`test: cover shared object link browser path`): adds keyboard access to configured
  network-object ports, includes readable object-link names in the topology accessibility summary, and
  adds a Playwright journey that creates a three-object path entirely through the canvas before verifying
  both links in a second browser before and after refresh. This completes T020.
- Focused gates for `462fbcb`: 40 topology/store Vitest cases and the production SPA build passed; the
  browser journey passed against the final embedded SPA in both the 1920x1080 desktop and 1024x768 minimum
  viewport projects locally on August 3, 2026.
- `c9adc05` (`test: preserve surviving object link path`): corrects the privileged recovery scenario so
  deleting one parallel link moves the routed path onto the surviving pair and a fresh DataPlane adopts
  only links that still have connected intent. The root-gated ICMP/TCP/UDP path test then passed against
  real Linux namespaces and veth pairs in 8.46 seconds. This is the focused US1 milestone SHA.
- US1 gates completed locally on August 3, 2026: domain/application/runtime/store/API unit packages,
  object-link HTTP/MCP/event contracts, recovery tests, `go vet`, focused race tests, export/import tests,
  40 focused frontend tests, production SPA build, both Playwright viewports, and the privileged path test
  passed. Two repository-wide baseline gates remain unrelated to US1: `TestBuiltInTemplatesLoad` expects a
  template count other than the current 8, and `PortMappingsPanel.test.ts` cannot find its QGA preset
  button; repository-wide ESLint/Prettier also report pre-existing errors and unformatted files outside
  this milestone's change set.
