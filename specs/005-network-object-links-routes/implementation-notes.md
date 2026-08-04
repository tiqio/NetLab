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
- `cf430d4` (`test: cover namespaced capture workers`): verifies namespace-wrapped capture commands,
  cancellation of the capture subprocess, retained-stream byte bounds, and exact counting of packets in
  both directions from one observed endpoint. Full capture-runtime tests and `go vet` passed, completing
  T032 locally on August 3, 2026.
- `97fb185` (`test: cover object link capture lifecycle`): verifies endpoint-A namespace resolution,
  source metadata and resolved locator persistence, live stream replay, durable capture task identity,
  and non-streamable retained metadata after service restart for `network_object_link` captures. Full
  reconciliation tests and `go vet` passed, completing T033 locally on August 3, 2026.
- `fe92f64` (`feat: attribute object link traffic direction`): starts separate ingress and egress
  Traffic Filter workers on endpoint A, maps them to `b_to_a` and `a_to_b`, keeps parallel object links
  isolated by stable resource ID, reports unknown direction explicitly as ambiguous, and removes stale
  observations after the configured correlation window. Full capture and reconciliation tests plus
  `go vet` passed, completing T034 and T042 locally on August 3, 2026.
- `6642b83` (`feat: publish object link observation events`): adds dynamic HTTP/MCP parity coverage for
  object-link capture envelopes and direction-attributed Traffic Filter observations, plus distinct
  ordered event constructors for capture state/completion and traffic observations. Focused stream,
  contract, and `go vet` gates passed, completing T035 and T043 locally on August 3, 2026.
- `e806804` (`feat: observe network object links in topology`): treats a network-object link as a
  first-class capture and Traffic Filter source throughout the SPA, preserves exact stable identity for
  parallel links, renders endpoint-A-relative particles and a longer-lived direction guide, and exposes
  readable endpoint names, capture counters, completion metadata, live stream, and retained artifact
  handles in the global workspace and Inspector. This completes T037 and T044 through T046.
- Focused gates for `e806804`: 29 capture/filter/topology/Inspector Vitest cases passed; changed-file
  Prettier passed; changed-file ESLint completed with zero errors; and both the production SPA build and
  embedded `netlabd` build passed locally on August 3, 2026. Existing Vue style warnings and the Vite
  chunk-size warning remain informational.
- `cb08bd3` (`test: verify live object link observations`): adds a root-gated two-parallel-link test that
  verifies one-endpoint bidirectional capture accounting, ICMP/TCP/UDP attribution in both endpoint-A-
  relative directions, exact stable-link isolation, and observation availability within 500 ms. The
  dynamic test exposed and fixed two production races: filters now receive their pcap header while still
  pending, and capture chunks are isolated by owning filter and capture ID so concurrent filters cannot
  corrupt each other's decoder streams. This completes T036.
- Focused gates for `cb08bd3`: the real namespace/veth/tcpdump test passed as root in 3.15 seconds;
  focused capture/filter/reconciliation tests, a three-run asynchronous cleanup check, the focused race
  detector, and `go vet` all passed locally on August 3, 2026.
- `0b87558` (`test: isolate object link browser observations`): creates two owned two-port lightweight
  switches and two connected parallel object links through real local APIs, selects one stable link from
  the keyboard topology surface, and verifies that both the Capture workspace and Traffic Filter persist
  only that link ID. It also registers the new diagnostics interaction in the acceptance inventory. This
  completes T038.
- Focused gates for `0b87558`: the Playwright journey and acceptance coverage/cleanup gates passed in
  both the 1920x1080 desktop and 1024x768 minimum viewport projects locally on August 3, 2026.
- `673152d` (`fix: wait for capture stop persistence`) makes capture stop wait for worker completion and
  terminal metadata persistence with bounded timeouts. This removes the asynchronous temporary-directory
  cleanup race and ensures callers receive authoritative stopped metadata rather than a transient state.
- US2 focused milestone SHA: `673152d`. Final local gates passed on August 3, 2026: full capture runtime,
  reconciliation, event, and stream packages; focused object-link observation/control contracts; focused
  race and `go vet`; 29 frontend capture/filter/topology/Inspector tests; production SPA and embedded Go
  build; the root-gated ICMP/TCP/UDP bidirectional attribution test in 3.32 seconds; and the two-project
  Playwright isolation journey with acceptance coverage and cleanup gates. The repository-wide contract
  baseline still includes the previously recorded built-in template count mismatch; a broad shared-memory
  contract run can also expose test-state reuse, so US2 uses isolated focused contract gates.
- `21cb1ed` (`feat: delete object links through durable tasks`): replaces synchronous object-link HTTP
  deletion with a revisioned, idempotent durable task shared by REST and MCP; publishes disconnecting
  before terminal deletion; stops dependent observation workers before runtime cleanup; atomically checks
  revision, releases both endpoint reservations, and publishes the task-correlated delete event; allows
  completed requests to replay after the link is gone; cascades network-object deletion through owned
  links; and cleans a surviving veth endpoint when the first endpoint is already absent. This completes
  T048, T050, T051, and T054 through T058.
- Focused gates for `21cb1ed`: task/repository/runtime/HTTP/MCP/contract tests, object-link create/delete
  parity, stale-observation suppression, focused race tests, and `go vet` passed locally on August 3,
  2026.
## US3 Frontend Object-Link Deletion Milestone

- Commit: `b2b1aec` (`feat: delete object links from topology workspace`)
- Added revisioned durable delete calls from the topology toolbar, right-click menu, and Inspector.
- Object links are hidden immediately, selection is cleared, stale refresh/event upserts are tombstoned, and failed submissions restore authoritative state with an explicit retry action.
- Focused validation passed on August 3, 2026:
  - `cd web && npm test -- --run src/features/topology/LinkContextMenu.test.ts src/features/topology/TopologyWorkspace.test.ts src/features/topology/TopologyCanvas.test.ts src/stores/laboratory.test.ts` (28 tests)
  - `cd web && npm run build`
- `b74f6ce` (`fix: avoid resurrecting deleting object links`): makes the generic DataPlane reconciler leave `disconnecting` object links under durable delete-task ownership during service recovery and reports `pending_task_recovery` rather than publishing a misleading recovered outcome. This closes the identified restart resurrection race; T059 remains open for the full interrupted-cleanup recovery and event-ordering journey.
- Focused combined gates for `b74f6ce` passed locally on August 3, 2026: reconcile/store/linuxnet/event/stream packages, focused object-link delete/capture/Traffic Filter contracts, focused `go vet`, 28 frontend deletion/store/canvas tests, and the production SPA build.
- `4045f73` (`feat: recover interrupted object link deletions`): makes durable task recovery an explicit startup recovery participant, resumes running object-link deletion tasks before normal convergence, preserves the existing capture-completion-before-link-deletion outbox order, verifies terminal task-correlated delete events, tolerates missing endpoints, and proves cleanup targets only explicitly owned namespace ports. This completes T049 and T059.
- Focused gates for `4045f73` passed locally on August 3, 2026: reconciliation, linuxnet, SQLite, event stream, recovery, and object-link contract tests; focused race tests; and `go vet` for the daemon, recovery, and runtime packages.
- `9f7081a` (`test: verify shared live object link deletion`): adds a two-browser Playwright journey that observes the same live object link, starts an object-link capture, deletes through the revisioned durable API, verifies immediate removal in both clients and after refresh, confirms `link_deleted` capture completion, reuses the released named ports, and verifies network-object cascade cleanup. This completes T053.
- Focused gates for `9f7081a` passed locally on August 3, 2026 against a candidate built from the current source: both the 1920x1080 desktop and 1024x768 minimum viewport projects passed with acceptance evidence and cleanup coverage.
- US3 focused milestone SHA: `4d0f46d`. Final US3 gates passed locally on August 3, 2026: reconciliation, SQLite, linuxnet, HTTP/MCP, event stream, focused object-link contracts and recovery tests; focused race and `go vet`; both privileged three-object path and bidirectional capture/Traffic Filter attribution tests; 28 frontend deletion/store/canvas tests; production SPA build; changed-file ESLint with zero errors; changed-file Prettier; and the two-browser live-delete journey in desktop and minimum viewports.
- Repository-wide baseline exceptions remain outside US3: `TestBuiltInTemplatesLoad` expects a template count different from the authoritative 8, full Vitest has one existing `PortMappingsPanel` QGA preset DOM lookup failure, and repository-wide ESLint reports the established warning/error backlog. These are not masked by the focused US3 gate and remain for the final polish phase.

## US4 Docker Static Route Milestone

- `f7d7a30` (`fix: stabilize Docker dual-stack route recovery`) waits for IPv6 duplicate-address
  detection to complete before route application, uses authoritative container inspection rather than
  stale list summaries, waits for a usable container namespace PID, cleans owned veth endpoints before
  stop, and gives Docker operations enough time to complete graceful container shutdown. Regression
  coverage includes tentative and duplicate IPv6 addresses, delayed namespace readiness, stale running
  summaries, and endpoint cleanup ordering.
- The approved public test image was used only from the local Docker store as
  `busybox@sha256:73aaf090f3d85aa34ee199857f03fa3a95c8ede2ffd4cc2cdb5b94e566b11662`;
  no image, credential, bootstrap secret, or packet capture was added to Git.
- The privileged Docker IPv4/IPv6 route path passed three consecutive runs locally on August 3, 2026.
  Each run created a real BusyBox container, waited for IPv6 DAD, verified direct IPv4/IPv6 gateway
  reachability and routed traffic, restarted the container, and confirmed route recovery without manual
  host namespace configuration.
- `fff7632` (`fix: preserve Docker route settings feedback`) keeps successful node-settings feedback
  visible across the authoritative node revision refresh and makes the Docker route Playwright journey
  capability-driven rather than target-profile-only. The interaction is registered in the acceptance
  inventory, while authoritative backend readback remains the stable end-to-end acceptance assertion.
- US4 focused milestone SHA: `fff7632`. Final focused gates passed locally on August 3, 2026: domain,
  command, Docker adapter, Linux networking, reconciliation, and focused contract tests; race tests and
  `go vet`; the three-run privileged Docker dual-stack route path; 21 focused frontend configuration,
  creation, lifecycle, and Inspector tests; production SPA build; changed-file ESLint with zero errors;
  changed-file Prettier; and the Docker route Playwright journey in both 1920x1080 desktop and 1024x768
  minimum viewport projects.

## Cross-Cutting Regression and Contract Milestones

- `aed80b7` (`test: harden object link route regressions`) adds compatibility coverage for standard
  links, bridge/NAT attachments, interface/link capture locators, and existing Traffic Filter scopes; a
  configurable 100-cycle object-link/capture/filter/route-ownership cleanup gate; and malicious endpoint,
  route, export, and evidence tests. The new export gate exposed and fixed transient container PID,
  namespace locator, runtime-interface, and packet-payload leakage while retaining declared routes and
  credential redaction.
- `9163e41` (`docs: synchronize object link route contracts`) aligns REST operation IDs and schemas with
  implemented `observed_state`, node settings, observation counters/timestamps, and cancelling task state;
  aligns MCP inputs/outputs and capture metadata; documents the ordered event payloads actually emitted;
  and adds operator guidance for live links, observation, Docker routes, recovery, and limitations.
- Focused gates for these milestones passed locally on August 3, 2026: contract, integration, and security
  regression selections; the default 100-cycle leak test; export command tests; `go vet` for command,
  contract, integration, and security packages; and repository whitespace validation.

## Final Local Candidate Gate

- Final locally validated implementation SHA: `215198e74ae4450ca3215674bc13e24b636205dd`.
- Formatting and static analysis passed on August 3, 2026: repository `gofmt` produced no paths,
  `git diff --check` passed, `go vet ./...` passed, frontend ESLint completed with zero errors, and the
  complete frontend Prettier check passed.
- `go test ./... -count=1` passed, including all domain, command, adapter, SQLite, HTTP, MCP, stream,
  contract, integration, recovery, and security packages. The obsolete six-template assertion was
  corrected to the authoritative eight built-ins, and the systemd unit now explicitly grants the bounded
  `/var/lib/netlab`, `/run/netlab`, and `/run/netns` write paths required by its security contract.
- Frontend Vitest passed all 59 files and 199 tests; the production SPA and embedded Go binary build
  passed. The stale port-mapping QGA test now opens Advanced Settings, uses the current labels, and expects
  backend-assigned host port `0`, matching the implemented contract. The Vite large-chunk message remains
  informational.
- Privileged validation passed with the pinned public BusyBox digest: real Docker IPv4/IPv6 route
  application and restart recovery, exact object-link ICMP/TCP/UDP capture and direction attribution,
  the three-object routed path, service recovery, the 100-cycle leak gate, and focused race tests for
  reconciliation, SQLite, Linux networking, Docker, and capture. Runtime tests requiring operator QEMU
  media or `dhclient` were explicitly skipped and remain target-host capability checks rather than local
  failures.
- Full local Playwright passed both 1920x1080 desktop and 1024x768 minimum viewport projects: 62 tests
  passed and 24 target/runtime-capability cases skipped, with no failures. The run covered shared control
  state, laboratory lifecycle, parallel object-link creation/deletion/observation, topology navigation,
  selection, visual recognition, keyboard parity, refresh recovery, cleanup, and accessibility.
- Frontend artifact hygiene passed. The compliance validator still described the previous
  `workspace-2026-07-29` candidate as not ready; T088 replaces that stale evidence with the immutable
  candidate built from this clean feature history.

## Immutable Candidate

- Candidate ID: `object-links-routes-ca072b6-20260803`; source SHA:
  `ca072b625b2a781e31855a0a66c528c261c0f15d`; build time: August 3, 2026 at 11:58:09 UTC.
- The immutable source/deployment archive is
  `sha256:7b040a226e9d5771409331cbd938cfbdfd422ee34d012c1c9a350fcdb1c403ce`;
  the embedded prebuilt Linux binary is
  `sha256:d8e8d0b374a3fa18ceb0a1a076d9d485b50f2fdce9c2446592a5550868dca924`;
  and the combined contract digest is
  `sha256:3eaffde190cf88ab4db67a6d273d1fc336383865898d0fc9630277f3710adbc0`.
- The target database advanced from migrations 1 through 6 to migrations 1 through 12. Before deployment,
  the existing 7.1 GiB SQLite database passed `PRAGMA integrity_check`
  after online backup, and the existing binary and database were copied into the target's protected
  `/var/lib/netlab/backups` directory with SHA-256 sidecars.
- The first target candidate, `object-links-routes-6fec815-20260803`, was rolled back after the genuine
  three-object path exposed that a newly hot-added L3 veth port did not inherit the object's forwarding
  policy. Commit `580d4d9` applies IPv4 and IPv6 forwarding to each arriving L3 port. Commit `abe9a14`
  restores the acceptance fixture's global L3 forwarding initialization so the gate models the real
  lifecycle while still proving per-port forwarding. Focused unit, privileged three-object ICMP/TCP/UDP,
  capture/Traffic Filter, Docker dual-stack route, full Go test, and Go vet gates pass locally.
- Commit `ca072b6` makes the laboratory page object tolerate bounded live-menu redraws. The three
  object-link browser journeys then passed on the target in both desktop and minimum viewports, including
  shared-client convergence, parallel-link isolation, live deletion, immediate port reuse, refresh, and
  acceptance-ledger cleanup back to the production baseline.

## Target Acceptance

- The immutable candidate was deployed to the authoritative `netlab.service` on `10.72.1.7:18082` at
  August 3, 2026 11:59:21 UTC. The installed binary and capability endpoint report the recorded candidate,
  binary digest, contract digest, and build time, and the service is active with migrations 1 through 12.
- Target privileged gates passed for Docker IPv4/IPv6 static routes, three-object L3 ICMP/TCP/UDP traffic,
  exact object-link capture and Traffic Filter direction attribution, parallel-link isolation, 100 cleanup
  cycles, interrupted deletion recovery, ownership recovery, and authoritative service restart.
- The final target browser run passed all six selected object-link journeys across desktop and minimum
  viewports. It created and observed shared paths in two clients, isolated parallel-link capture/filter
  scope, deleted a live link, reused its ports immediately, refreshed both clients, and reported zero
  remaining acceptance-owned resources with the production baseline restored.
- The feature acceptance conclusion is `passed`. The broader constitution ledger remains `not_ready`
  because unrelated operator-owned credential rotation and genuine VyOS/FancyWAN/FortiGate media gates
  remain blocked; those items are outside this feature's implementation scope.

## Phase 8 Authority and Acceptance Milestone

- On August 4, 2026, the convergence implementation added a runtime listener invariant and an install/
  systemd authority guard that rejects a second externally reachable `netlabd`, while allowing a
  loopback-only validation instance and providing an explicit retirement path for the known legacy
  `/opt/netlab/netlabd` process.
- Target browser acceptance now requires the real `target-host` profile, validates immutable release
  identity and the expected host, and supports an explicit `preserve` baseline mode for production hosts
  that already contain laboratories and runtime ownership records.
- Runtime ownership responses now classify managed, acceptance-tagged, and foreign-observed records.
  Acceptance cleanup and constitution leak snapshots enforce managed ownership stability without treating
  persistent foreign EVE-NG resources as NetLab-owned baseline resources.
- Focused gates: `go test ./internal/compliance ./internal/app/reconcile ./internal/store/sqlite`,
  `go test ./tests/security -run Authority`, and the target-policy/cleanup acceptance-unit tests passed.

## Phase 8 Atomic Deployment Milestone

- The service applies embedded immutable build identity before authoritative configuration validation and
  exposes `release` plus `validate-config` command modes for deployment preflight.
- The prebuilt installer now stages the binary and configuration, preserves operator-owned settings,
  validates the candidate before replacement, installs through atomic file moves, and restores the prior
  binary/configuration/readiness files if restart or authority verification fails.
- Installation generates a candidate-bound readiness document for all eight built-in templates. The
  server rejects readiness evidence from another candidate instead of silently publishing stale truth.
- Focused gates: netlabd/config/template query tests, deployment security tests, script syntax checks, and
  release-configuration/readiness generation tests passed locally on August 4, 2026.

## Phase 8 UX and Isolated Traffic Validation Milestone

- The laboratory switcher remains mounted while open state and live laboratory props change, so a click
  no longer destroys its own option/button DOM. The Playwright-only retry workaround was removed.
- Task Center renders the newest 30 matching tasks initially and expands in bounded batches, preserving
  responsive interaction with the target's 100-record task history.
- Privileged test runs now receive a separate ownership domain. Namespace/interface names and link aliases
  no longer match the authoritative reconciler's production prefixes, while production naming remains
  byte-compatible when no validation domain is configured.
- The target candidate browser journey configures two BusyBox interfaces through their real Telnet
  WebSockets, generates ICMP, TCP, and UDP, verifies directional Traffic Filter particles within 500 ms,
  and verifies particle and direction-guide decay. Existing privileged object-link tests retain exact
  parallel-link isolation and bidirectional attribution coverage.
- Focused gates: ownership/linuxnet tests, topology/laboratory/task Vitest, acceptance policy/cleanup unit
  tests, and Prettier passed locally on August 4, 2026.
- The complete acceptance-unit gate additionally exposed missing operation-registry entries for
  first-class object-link create/delete and node settings update, plus incorrect handling of a composite
  capture/filter interaction. The registry and inventory validator now enforce each real operation.
- The first target deployment attempt on August 4, 2026 rolled back cleanly because immediate authority
  verification ran before the HTTP listener became ready. Target inspection also identified
  `netlab-preview.service` as the parent that restarted the legacy process. The local fix adds a bounded
  listener wait and explicitly disables that known preview unit when legacy retirement is requested.
- The second target attempt disabled the preview unit but encountered host storage distress: an older
  `netlabd` remained in uninterruptible `D` state, the replacement waited on the 7.1 GiB SQLite state,
  `vmstat` reported sustained blocked I/O, and kernel history contained `sda`/EXT4 write errors from July
  30, 2026. Deployment verification now allows the documented 180-second recovery window and fails fast
  when systemd becomes inactive, but further target replacement is suspended until host I/O recovers.
- A reversible recovery attempt on August 4, 2026 retained hard-linked copies of the checksum-verified
  predeployment database, moved the inode held by the stuck process aside, and restored the verified
  predeployment database without deleting state. The old uninterruptible process exited and systemd
  restarted the authoritative service, but the replacement still blocked before opening `:18082` in
  `ext4_sync_file`/`rq_qos_wait`; host I/O pressure remained above 90 percent and the API stayed
  unreachable. The retired preview unit remains disabled and no second NetLab listener is present.
- Clean source `498291003f9d1d0c024f08f47fa24055fcc17970` produced candidate
  `object-links-routes-4982910-20260804` with binary digest
  `sha256:6af50bc8bf2910ee67f02413e8ba4719d2213810c7fda8785a26761cdbe67482`.
  Frontend production build, embedded artifact hygiene, and release identity checks passed locally. This
  candidate was deliberately not installed while the target storage path remained unhealthy; the target
  binary digest is still the previously accepted `ca072b6` candidate.

## Phase 9 Traffic Generation Correction

- The focused clean-target journey initially started Traffic Filter successfully but reported no matching
  observations. A direct target-host reproduction on August 4, 2026 proved that the BusyBox node's
  `ip addr replace` command failed with `RTNETLINK answers: Operation not permitted`; the browser helper
  had previously closed each Telnet WebSocket after 600 ms without reading output or checking command
  completion, so the acceptance run silently continued without configured IPv4 addresses or generated
  ICMP/TCP/UDP traffic.
- Docker network nodes now receive only `NET_ADMIN` and `NET_RAW` in addition to Docker's default
  capability set and remain non-privileged. This permits users and automation to configure interfaces,
  routes, ICMP, and packet generators inside the node while avoiding unrestricted host privileges.
- The target acceptance console helper now waits for a unique shell completion marker, preserves command
  output, rejects non-zero exit status, and waits for the first authoritative Traffic Filter observation
  before measuring the 500 ms browser update requirement. Local `go vet ./...`, `go test ./... -count=1`,
  the 17-test acceptance-unit suite, frontend production build, Prettier, and ESLint with zero errors passed.
- The first capability-enabled target reproduction then exposed a separate command-compatibility error:
  BusyBox 1.36 supports `ip addr add` and `ip addr flush` but not `ip addr replace`. The acceptance traffic
  setup now uses the portable flush/add sequence rather than relying on the fuller iproute2 syntax.
- The target Traffic Filter read endpoint returns an envelope whose live observations are under
  `traffic_filter.observations`; the focused polling predicate now follows that contract instead of looking
  for an unwrapped top-level collection. Direct target traffic delivered ICMP successfully and delivered
  the `tcpudp` payloads to the peer container before the expected two-second observation decay.
- The first focused browser rerun reached the console step but showed that `crypto.randomUUID()` is not
  available inside a non-secure HTTP page context. Completion markers are now generated by the Node-based
  Playwright process and passed into `page.evaluate`, preserving compatibility with the approved HTTP
  management endpoint without weakening command-result validation.
- The authoritative restart acceptance script previously honored `NETLAB_BASE_URL` for REST calls but
  hard-coded port 8088 for its raw console and event WebSockets. Both socket checks now derive host and
  port from the same validated HTTP base URL, so the script exercises the actual `10.72.1.7:18082`
  authority rather than a stale development endpoint.
- Fresh-reset template data keeps enabled template versions and image versions as separate selectable
  resources. The restart journey no longer requires `template_versions.image_version_id`; it selects an
  enabled `ubuntu-qemu` version plus an available Ubuntu QEMU image and sends both IDs during node create,
  matching the production API and browser workflow.
- The production systemd sandbox gives `netlabd` its own mount namespace because of the configured writable
  path protections, even though `PrivateMounts` reports `no`. Named network-namespace bind mounts are
  therefore valid inside the service mount view but appear as invalid handles to an unrelated host shell.
  Restart acceptance now runs namespace checks through the current `netlab.service` MainPID with `nsenter
  --mount`, including after the MainPID changes across restart.
