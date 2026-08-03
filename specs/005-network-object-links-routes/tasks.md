# Tasks: First-Class Network Object Links and Docker Routes

**Input**: Design documents from `specs/005-network-object-links-routes/`

**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/`, `quickstart.md`

**Tests**: Tests are required because this feature changes durable state, Linux networking, Docker
runtime reconciliation, HTTP/MCP contracts, packet capture, Traffic Filter attribution, recovery, and
resource cleanup. Write the listed tests first and verify that they fail for the intended missing behavior.

**Organization**: Tasks are grouped by user story and include focused local quality gates plus milestone
commits. Source changes must be completed locally before any deployment to `10.72.1.7`.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel because it changes different files and does not depend on an incomplete task
- **[Story]**: User story served by the task
- Every task includes an exact file path

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Establish compatibility notes and reusable test support before modifying established runtime
and persistence behavior.

- [X] T001 Inspect `git log` and `git blame` for the existing object-link, capture, Traffic Filter, Docker endpoint, and topology UI implementations and record compatibility constraints in `specs/005-network-object-links-routes/implementation-notes.md`
- [X] T002 [P] Add reusable three-network-object, parallel-link, traffic-generator, and namespace cleanup fixtures in `tests/testsupport/network_object_link_fixtures.go`
- [X] T003 [P] Add reusable Docker dual-stack route fixtures and exact-route assertions in `tests/testsupport/docker_route_fixtures.go`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Add shared domain types, endpoint exclusivity, persistence, and runtime boundaries required by
all four user stories.

**⚠️ CRITICAL**: User-story implementation starts only after these invariants are in place.

### Tests for Shared Foundations

- [X] T004 [P] Add failing domain tests for canonical object-link endpoint orientation, observation resource identity, and Docker IPv4/IPv6 route declarations in `internal/domain/operations_test.go` and `internal/domain/models_test.go`
- [X] T005 [P] Add failing migration and repository tests for cross-side endpoint collisions, attachment collisions, backfill conflicts, reservation release, and transaction rollback in `internal/store/sqlite/interface_reservation_test.go` and `internal/store/sqlite/network_state_repository_test.go`
- [X] T006 [P] Add failing schema tests for object-link task envelopes, capture source types, Traffic Filter object-link scopes, directional observations, and Docker routes in `tests/contract/network_object_link_route_schema_test.go`

### Shared Implementation

- [X] T007 Add typed `DockerStaticRoute`, observation resource type/direction, and object-link endpoint helpers in `internal/domain/models.go` and `internal/domain/operations.go`
- [X] T008 Add canonical route and endpoint validation with stable problem codes in `internal/domain/network_object.go` and `internal/domain/models.go`
- [X] T009 Add the endpoint-reservation, object-link lifecycle, Traffic Filter scope, observation attribution, and managed-route migration in `internal/store/sqlite/migrations/0011_network_object_link_endpoints_routes.sql`
- [X] T010 Implement transactional endpoint reservation, collision detection across endpoint sides and attachment types, object-link revision updates, and safe reservation release in `internal/store/sqlite/network_repository.go`
- [ ] T011 Create durable capture/filter observation persistence for object-link attribution and extend task, audit, and outbox storage for `link_deleted` completion in `internal/store/sqlite/runtime_observation_repository.go` and `internal/store/sqlite/automation_repository.go`
- [ ] T012 Define namespace-aware observation locator and exact managed-route runtime interfaces in `internal/app/ports/topology.go`
- [ ] T013 Update topology export/import DTOs to preserve object-link endpoint intent and Docker route declarations while excluding runtime locators and packet payloads in `internal/app/command/export.go` and `internal/app/command/import.go`
- [ ] T014 Run the foundational domain, migration, repository, and contract tests and record commands/results plus the focused milestone commit SHA in `specs/005-network-object-links-routes/implementation-notes.md`

**Checkpoint**: Endpoint occupancy is enforceable across every connection type, shared contracts can carry
object-link observations and Docker routes, and the first local milestone is committed.

---

## Phase 3: User Story 1 - Build Multi-Switch Topologies (Priority: P1) 🎯 MVP

**Goal**: Connect named ports on active network objects through one direct veth pair per durable link,
support parallel links, and recover connected intent without manual host networking commands.

**Independent Test**: Create three lightweight network objects, connect them through two object links,
attach endpoints at opposite sides, verify bidirectional traffic and parallel-link identity, restart the
service, and confirm the same path recovers.

### Tests for User Story 1

- [X] T015 [P] [US1] Replace bridge-plus-two-veth expectations with failing direct-one-veth-pair, deterministic naming, idempotent ensure, and exact cleanup tests in `internal/runtime/linuxnet/dataplane_test.go`
- [ ] T016 [P] [US1] Add failing create/list/revision/idempotency/task/outbox tests for durable object links in `internal/app/reconcile/network_object_tasks_test.go` and `internal/store/sqlite/network_state_repository_test.go`
- [ ] T017 [P] [US1] Add failing HTTP and MCP parity tests for create/list/get object-link task envelopes and occupied-port errors in `tests/contract/network_object_link_control_parity_test.go`
- [ ] T018 [P] [US1] Add a failing privileged three-object path, parallel-link isolation, bidirectional ICMP/TCP/UDP, and service-restart recovery test in `tests/integration/network_object_link_path_test.go`
- [ ] T019 [P] [US1] Add failing frontend tests for connector port selection, parallel line identity, readable endpoint labels, shared state updates, and refresh recovery in `web/src/features/topology/topologyConnectionController.test.ts`, `web/src/features/topology/TopologyCanvas.test.ts`, and `web/src/features/topology/TopologyInspector.test.ts`
- [ ] T020 [P] [US1] Add a failing Playwright journey that creates a three-object path from the browser and observes the same links in a second client in `tests/e2e/journeys/networkObjectLinks.spec.ts`

### Implementation for User Story 1

- [X] T021 [US1] Replace the per-link host bridge and two veth pairs with one owned veth pair moved directly into endpoint A and B namespaces in `internal/runtime/linuxnet/dataplane.go`
- [ ] T022 [US1] Add deterministic object-link ownership discovery, partial-pair adoption, pending-create reconciliation, and orphan cleanup in `internal/app/reconcile/host_ownership_scanners.go`, `internal/app/reconcile/recovery_coordinator.go`, and `internal/runtime/ownership/manifest.go`
- [ ] T023 [US1] Implement revisioned and idempotent object-link create/list/get commands with durable tasks and ordered outbox events in `internal/app/reconcile/network_object_tasks.go` and `internal/app/reconcile/network_objects.go`
- [ ] T024 [US1] Upgrade object-link HTTP create/list/get routes to the task-envelope and structured-conflict contract in `internal/api/http/network_handlers.go` and `internal/api/http/mutation_middleware.go`
- [ ] T025 [US1] Add `netlab.network_object_links.create` and `netlab.network_object_links.get` with the shared application handlers in `internal/api/mcp/network_tools.go`
- [ ] T026 [US1] Publish ordered object-link create/state/recovery events through the shared event stream in `internal/api/stream/events.go` and `internal/app/events/outbox.go`
- [ ] T027 [US1] Synchronize object-link task, endpoint, state, and error types and API methods in `web/src/api/generated.ts` and `web/src/api/index.ts`
- [ ] T028 [US1] Implement object-to-object connector selection, occupied-port feedback, parallel-link rendering, and `object:port ↔ object:port` presentation in `web/src/features/topology/topologyConnectionController.ts`, `web/src/features/topology/TopologyCanvas.vue`, and `web/src/features/topology/linkPresentation.ts`
- [ ] T029 [US1] Display authoritative desired/actual state, revision, lifecycle task, and actionable failures for selected object links in `web/src/features/topology/TopologyInspector.vue` and `web/src/features/topology/TopologyWorkspace.vue`
- [ ] T030 [US1] Complete object-link export/import ID remapping and transactional endpoint reservation integration in `internal/store/sqlite/import_repository.go`, `internal/app/command/export.go`, and `internal/app/command/import.go`
- [ ] T031 [US1] Run US1 unit, contract, frontend, privileged integration, recovery, and browser tests and record the focused milestone commit SHA in `specs/005-network-object-links-routes/implementation-notes.md`

**Checkpoint**: User Story 1 is independently usable as the MVP and has a committed local milestone.

---

## Phase 4: User Story 2 - Observe and Diagnose Object Links (Priority: P1)

**Goal**: Select an object link, inspect it, capture both traffic directions exactly once, and show
Traffic Filter activity only on the correct link and direction.

**Independent Test**: Start capture and a protocol-specific Traffic Filter on one link in a topology with
parallel links, generate bidirectional traffic only on the selected path, and verify counts, stream/artifact
metadata, direction, isolation, and activity decay.

### Tests for User Story 2

- [ ] T032 [P] [US2] Add failing capture-worker tests for namespace execution, cancellation, bounded output, and one-endpoint bidirectional accounting in `internal/runtime/capture/worker_test.go`
- [ ] T033 [P] [US2] Add failing capture-manager and task tests for `network_object_link` source resolution, retained metadata, stream handles, and restart behavior in `internal/app/reconcile/captures_test.go` and `internal/app/reconcile/capture_tasks_test.go`
- [ ] T034 [P] [US2] Add failing Traffic Filter tests for explicit object-link scope, endpoint-A-relative directions, ambiguity, parallel-link isolation, and decay in `internal/app/reconcile/traffic_filters_test.go` and `internal/runtime/capture/path_test.go`
- [ ] T035 [P] [US2] Add failing HTTP/MCP/event contract tests for object-link captures and observations in `tests/contract/network_object_link_observation_contract_test.go`
- [ ] T036 [P] [US2] Add a failing privileged capture/filter test covering ICMP, TCP, UDP, both directions, an unused parallel link, and sub-500 ms attribution in `tests/integration/network_object_link_observation_test.go`
- [ ] T037 [P] [US2] Add failing frontend tests for selected object-link capture, Traffic Filter scoping, inspector metadata, directional particles, and decay in `web/src/features/diagnostics/CapturePanel.test.ts`, `web/src/features/diagnostics/TrafficFilterPanel.test.ts`, `web/src/features/topology/TrafficPathOverlay.test.ts`, and `web/src/features/topology/TopologyInspector.test.ts`
- [ ] T038 [P] [US2] Add a failing Playwright journey for capture and Traffic Filter isolation across two parallel object links in `tests/e2e/journeys/networkObjectLinkObservability.spec.ts`

### Implementation for User Story 2

- [X] T039 [US2] Change the capture worker to accept a structured host-or-namespace locator and execute packet capture without shell interpolation in `internal/runtime/capture/worker.go`
- [X] T040 [US2] Resolve object-link captures to endpoint A's namespace interface and preserve stable source identity through start, recovery, stream, stop, and retention in `internal/app/reconcile/captures.go` and `internal/app/reconcile/capture_tasks.go`
- [X] T041 [US2] Extend capture HTTP and MCP inputs with `source_type=network_object_link` while keeping namespace and interface resolution server-side in `internal/api/http/capture_handlers.go` and `internal/api/mcp/tools.go`
- [ ] T042 [US2] Add `network_object_link_ids`, resource-type/resource-ID attribution, endpoint-A-relative direction, and explicit ambiguity to Traffic Filter processing in `internal/app/reconcile/traffic_filters.go` and `internal/runtime/capture/path.go`
- [ ] T043 [US2] Publish ordered capture/filter observations for object links without duplicating standard-link events in `internal/api/stream/events.go` and `internal/app/events/outbox.go`
- [ ] T044 [US2] Extend frontend capture/filter API types and source selectors with object-link identities in `web/src/api/generated.ts`, `web/src/api/index.ts`, `web/src/features/diagnostics/CapturePanel.vue`, and `web/src/features/diagnostics/TrafficFilterPanel.vue`
- [ ] T045 [US2] Render object-link capture state, packet/byte counts, completion, stream/artifact handles, and human-readable endpoint identity in `web/src/features/diagnostics/GlobalCaptureWorkspace.vue` and `web/src/features/topology/TopologyInspector.vue`
- [ ] T046 [US2] Map object-link observations to the exact topology edge and render direction-aware particles with configured decay and ambiguity presentation in `web/src/features/topology/TrafficPathOverlay.vue`, `web/src/features/topology/trafficPathTypes.ts`, and `web/src/features/topology/TopologyCanvas.vue`
- [ ] T047 [US2] Run US2 capture, filter, contract, frontend, privileged integration, and browser tests and record the focused milestone commit SHA in `specs/005-network-object-links-routes/implementation-notes.md`

**Checkpoint**: User Story 2 provides first-class troubleshooting for one selected object link without
false activity on parallel links.

---

## Phase 5: User Story 3 - Rewire and Delete Links Live (Priority: P1)

**Goal**: Delete a running object link without stopping nodes, terminate dependent observations cleanly,
remove shared UI state immediately, release ports, and recover partial deletion without leaks.

**Independent Test**: Delete a traffic-carrying link with an active capture while two clients observe the
lab, verify `link_deleted`, retained metadata, immediate line removal, stopped traffic, reusable ports, and
successful retry after an injected partial cleanup failure.

### Tests for User Story 3

- [ ] T048 [P] [US3] Add failing service/repository tests for expected revision, idempotent delete, capture-stop ordering, endpoint release, object cascade, and ordered deletion events in `internal/app/reconcile/network_object_tasks_test.go` and `internal/store/sqlite/network_state_repository_test.go`
- [ ] T049 [P] [US3] Add failing dataplane and recovery tests for live pair deletion, missing-end tolerance, partial cleanup retry, and protection of unrelated host resources in `internal/runtime/linuxnet/dataplane_test.go` and `tests/recovery/network_object_link_recovery_test.go`
- [ ] T050 [P] [US3] Add failing capture tests for `link_deleted` completion, retained packet metadata, worker cancellation timeout, and stale observation suppression in `internal/app/reconcile/captures_test.go` and `internal/app/reconcile/traffic_filters_test.go`
- [ ] T051 [P] [US3] Add failing HTTP/MCP parity tests for revisioned delete task envelopes and retries in `tests/contract/network_object_link_delete_parity_test.go`
- [ ] T052 [P] [US3] Add failing frontend tests for object-link context deletion, pending/failed state, immediate shared removal, selection clearing, and no ghost restoration in `web/src/features/topology/LinkContextMenu.test.ts`, `web/src/features/topology/TopologyWorkspace.test.ts`, and `web/src/features/topology/TopologyCanvas.test.ts`
- [ ] T053 [P] [US3] Add a failing two-browser Playwright journey for live deletion, active capture completion, port reuse, object cascade, and refresh consistency in `tests/e2e/journeys/networkObjectLinkDeletion.spec.ts`

### Implementation for User Story 3

- [ ] T054 [US3] Implement revisioned/idempotent object-link deletion tasks that mark disconnecting, stop dependent capture/filter workers, delete runtime resources, release reservations, and commit the final event in `internal/app/reconcile/network_object_tasks.go` and `internal/app/reconcile/network_objects.go`
- [ ] T055 [US3] Make direct-veth deletion idempotent and recovery-safe for absent, half-moved, or partially deleted endpoints in `internal/runtime/linuxnet/dataplane.go`
- [ ] T056 [US3] Complete active object-link captures with `link_deleted`, preserve retained metadata/artifacts, and suppress post-delete filter observations in `internal/app/reconcile/captures.go`, `internal/app/reconcile/capture_tasks.go`, and `internal/app/reconcile/traffic_filters.go`
- [ ] T057 [US3] Cascade network-object deletion through object-link tasks and endpoint reservations without stale resources in `internal/app/reconcile/network_objects.go` and `internal/store/sqlite/network_repository.go`
- [ ] T058 [US3] Expose revisioned delete task envelopes and structured failures through HTTP and MCP in `internal/api/http/network_handlers.go` and `internal/api/mcp/network_tools.go`
- [ ] T059 [US3] Publish deletion and capture-completion events in non-resurrecting order and reconcile interrupted deletion on startup in `internal/api/stream/events.go`, `internal/app/events/outbox.go`, and `internal/app/reconcile/recovery_coordinator.go`
- [ ] T060 [US3] Add object-link right-click delete, task progress, retryable failure, selection clearing, and shared event removal in `web/src/features/topology/LinkContextMenu.vue`, `web/src/features/topology/TopologyWorkspace.vue`, and `web/src/features/topology/TopologyCanvas.vue`
- [ ] T061 [US3] Run US3 unit, contract, frontend, privileged deletion, recovery, leak, and multi-client tests and record the focused milestone commit SHA in `specs/005-network-object-links-routes/implementation-notes.md`

**Checkpoint**: User Story 3 supports observable live failure simulation and cleanup without stopping
unrelated nodes or leaving ghost topology state.

---

## Phase 6: User Story 4 - Reproduce Docker L3 Routes (Priority: P1)

**Goal**: Persist, validate, apply, replace, and recover declared Docker IPv4/IPv6 routes before the node
is reported ready, with no manual namespace commands.

**Independent Test**: Configure Docker routes through the supported create/settings flow behind a
multi-object L3 path, verify routed ICMP/TCP/UDP, restart the node and service, remove a route while stopped,
and confirm exact-set convergence and actionable failures.

### Tests for User Story 4

- [X] T062 [P] [US4] Add failing domain tests for canonical CIDRs, family consistency, duplicate/conflicting prefixes, known interface, gateway reachability, metric validation, and stable error codes in `internal/domain/models_test.go`
- [X] T063 [P] [US4] Add failing node create/settings/export/import tests proving Docker `network_interfaces.routes` survive every control path in `internal/app/command/node_template_test.go`, `internal/app/command/export_test.go`, and `internal/app/command/import_test.go`
- [X] T064 [P] [US4] Add failing Docker endpoint tests for exact managed-route replacement, stale managed-route removal, unmanaged-route preservation, idempotent ensure, dual-stack support, and route-specific rollback in `internal/runtime/linuxnet/docker_endpoint_test.go`
- [X] T065 [P] [US4] Add failing Docker adapter tests proving route reconciliation runs for new, already-running, restarted, and recovered containers before readiness in `internal/runtime/docker/adapter_test.go`
- [X] T066 [P] [US4] Add failing HTTP/MCP/generated-client contract tests for Docker route declarations and actionable validation errors in `tests/contract/docker_static_route_contract_test.go`
- [X] T067 [P] [US4] Add a failing privileged multi-object L3 test for automatic IPv4/IPv6 routes, ICMP/TCP/UDP, stop/start, service recovery, and no manual `nsenter` setup in `tests/integration/docker_static_route_path_test.go`
- [X] T068 [P] [US4] Add failing frontend tests for Docker route creation/editing, readback, stopped-node update, validation feedback, and immutable unrelated credentials in `web/src/features/nodes/NodeConfigurationPanel.test.ts` and `web/src/features/topology/CreateTopologyResourceDialog.test.ts`
- [ ] T069 [P] [US4] Add a failing Playwright journey that configures and verifies Docker routes entirely through the frontend in `tests/e2e/journeys/dockerStaticRoutes.spec.ts`

### Implementation for User Story 4

- [X] T070 [US4] Implement canonical Docker route validation and normalization on typed interface settings before persistence or readiness in `internal/domain/models.go` and `internal/app/command/node.go`
- [X] T071 [US4] Preserve Docker network interface and route declarations through create, stopped-node settings, export, import, and generated task inputs in `internal/app/command/node.go`, `internal/app/command/export.go`, `internal/app/command/import.go`, and `internal/store/sqlite/node_operations_repository.go`
- [X] T072 [US4] Reconcile the exact owned route set inside the container namespace using argument vectors, remove only stale NetLab-managed routes, and return route-specific errors in `internal/runtime/linuxnet/docker_endpoint.go`
- [X] T073 [US4] Run endpoint and route reconciliation for created, already-running, restarted, recreated, service-recovered, and host-recovered containers before ready state in `internal/runtime/docker/adapter.go` and `internal/app/reconcile/recovery_coordinator.go`
- [X] T074 [US4] Expose typed Docker routes and validation problems through HTTP and MCP node create/settings operations in `internal/api/http/node_operations_handlers.go` and `internal/api/mcp/tools.go`
- [X] T075 [US4] Synchronize Docker route request/response types and API serialization in `web/src/api/generated.ts` and `web/src/api/nodeOperations.ts`
- [X] T076 [US4] Add family-aware Docker route editors, defaults, stopped-node settings support, and backend problem presentation in `web/src/features/nodes/NodeConfigurationPanel.vue` and `web/src/features/topology/CreateTopologyResourceDialog.vue`
- [X] T077 [US4] Surface route application progress and route-specific readiness failures in `web/src/features/nodes/NodeOperationsPanel.vue` and `web/src/features/topology/TopologyInspector.vue`
- [ ] T078 [US4] Run US4 domain, command, adapter, contract, frontend, privileged L3, recovery, and browser tests and record the focused milestone commit SHA in `specs/005-network-object-links-routes/implementation-notes.md`

**Checkpoint**: User Story 4 makes Docker L3 configuration reproducible without manual host intervention.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Prevent regressions, publish final contracts and operator guidance, validate a clean local
candidate, then deploy and test the immutable artifact on `10.72.1.7`.

- [ ] T079 [P] Add regression tests proving standard node-interface links, attachments, NAT, host-interface capture, standard-link capture, and existing Traffic Filter scopes retain their semantics in `tests/integration/network_compatibility_regression_test.go`
- [ ] T080 [P] Add a 100-cycle create/capture/filter/delete/recreate leak test for direct object links and Docker managed routes in `tests/integration/network_object_link_route_leak_test.go`
- [ ] T081 [P] Add security tests proving endpoint names and route values never enter interpolated shell strings and exports/evidence omit runtime identifiers, credentials, packet payloads, and target secrets in `tests/security/network_object_link_route_security_test.go`
- [ ] T082 [P] Synchronize implemented REST schemas and operation IDs with `specs/005-network-object-links-routes/contracts/openapi-delta.yaml`
- [ ] T083 [P] Synchronize implemented MCP tool inputs, outputs, errors, and capture metadata with `specs/005-network-object-links-routes/contracts/mcp-tools.md`
- [ ] T084 [P] Synchronize ordered object-link, route, capture, and Traffic Filter event semantics with `specs/005-network-object-links-routes/contracts/events.md`
- [ ] T085 [P] Document object-link creation, live deletion, capture, Traffic Filter, Docker route configuration, recovery behavior, and limitations in `README.md`
- [ ] T086 Run formatting, static analysis, Go unit/contract/security tests, frontend Vitest/build/lint/format checks, privileged integration, recovery, Playwright, and leak gates from `specs/005-network-object-links-routes/quickstart.md`
- [ ] T087 Record the final clean local gate results and focused implementation milestone commit SHA in `specs/005-network-object-links-routes/implementation-notes.md`
- [ ] T088 Build the candidate only from the clean commit and record source SHA, artifact SHA-256 digest, SQLite migration state, and rollback artifact in `compliance/evidence/current-candidate.json`
- [ ] T089 Deploy the immutable candidate without source edits using `deploy/scripts/install.sh` and record UTC deployment time and artifact identity in `compliance/deployment-authority.json`
- [ ] T090 Validate multi-switch forwarding, parallel links, live deletion, capture, Traffic Filter direction/decay, Docker IPv4/IPv6 routes, service restart, host recovery, and leak cleanup on `10.72.1.7` using `specs/005-network-object-links-routes/quickstart.md`
- [ ] T091 Record redacted target-host results and final acceptance conclusion in `compliance/evidence/current-candidate.json`; if any mandatory validation fails, execute the recorded rollback through `deploy/scripts/maintenance.sh`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 — Setup**: Starts immediately.
- **Phase 2 — Foundations**: Depends on Phase 1 and blocks every user story.
- **Phase 3 — US1**: Depends on Phase 2 and is the recommended MVP.
- **Phase 4 — US2**: Depends on US1 because capture and filtering require a connected object link.
- **Phase 5 — US3**: Depends on US1; its active-capture acceptance path also depends on US2.
- **Phase 6 — US4**: Domain, contract, and adapter work can begin after Phase 2; the full multi-object L3
  independent test depends on US1.
- **Phase 7 — Polish**: Depends on all selected stories and their focused local milestone commits.

### User Story Dependency Graph

```text
Setup -> Foundations -> US1 (MVP) -> US2 -> US3 -> Polish
                     \-> US4 -----------/
```

### Within Each User Story

1. Add the listed tests and verify expected failures.
2. Implement domain/persistence changes before application services.
3. Implement application services before HTTP, MCP, event, and frontend adapters.
4. Run focused local gates and create the milestone commit before proceeding.
5. Do not deploy until all selected stories pass the full local gate from a clean commit.

## Parallel Execution Examples

### User Story 1

- Run T015, T016, T017, T018, T019, and T020 in parallel because they cover distinct test layers.
- After T023, implement HTTP/MCP/event adapters T024, T025, and T026 in parallel.
- After T027, implement topology connection/rendering T028 while inspector work T029 proceeds separately.

### User Story 2

- Run T032 through T038 in parallel across runtime, application, contract, integration, frontend, and E2E tests.
- After T040 and T042, implement protocol adapters T041/T043 and frontend adapters T044/T045/T046 in parallel.

### User Story 3

- Run T048 through T053 in parallel across persistence, runtime, capture, contract, frontend, and browser tests.
- After T054, implement runtime cleanup T055, observation termination T056, and cascade cleanup T057 in parallel.
- After backend event ordering is stable, HTTP/MCP T058 and frontend behavior T060 can proceed in parallel.

### User Story 4

- Run T062 through T069 in parallel across validation, commands, runtime, adapter, contract, integration,
  frontend, and browser tests.
- After typed persistence T070/T071, implement runtime reconciliation T072/T073 and HTTP/MCP T074 in parallel.
- After T075, route editors T076 and lifecycle error presentation T077 can proceed in parallel.

## Implementation Strategy

### MVP First

1. Complete Setup and Foundations.
2. Complete US1 and its focused local milestone commit.
3. Demonstrate a recovered three-object path with parallel links before adding observation or route work.

### Incremental Delivery

1. Add US2 so the MVP path is selectable, capturable, and directionally observable.
2. Add US3 so the path can be safely rewired and deleted while running.
3. Add US4 in parallel after foundations, then integrate it with the US1 multi-object path.
4. Complete regression, security, leak, full local gates, clean candidate build, immutable deployment, and
   target validation only after all implementation milestones are committed.

### Completion Criteria

- Each user story passes its independent test without undocumented host commands.
- UI, HTTP, MCP, tasks, audit records, and ordered events converge on the same durable state.
- Direct object links use one owned veth pair and leave no host bridge or leaked resources.
- Docker routes converge to the exact declared managed set before readiness and survive recovery.
- The deployed target artifact is traceable to a clean local commit and has a recorded rollback path.
