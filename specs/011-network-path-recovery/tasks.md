# Tasks: Network Path Recovery and Validation

**Input**: Design documents from `/specs/011-network-path-recovery/`

**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/`, `quickstart.md`

**Tests**: Tests are mandatory because this feature changes runtime networking, recovery, resource ownership, API/MCP contracts, durable tasks, capture/filter observation and privileged cleanup.

**Organization**: Tasks are grouped by user story so each capability can be implemented, tested, committed and demonstrated independently.

## Phase 1: Setup and Baseline

**Purpose**: Preserve compatibility context and create repeatable local/target evidence before runtime changes.

- [X] T001 Record the current component-matrix resource, link, namespace, VLAN, route, Traffic Filter and leak baseline in `specs/011-network-path-recovery/validation/baseline.md`
- [X] T002 Inspect relevant `git log` and `git blame` history for recovery, network objects, endpoint cleanup, VLAN and Traffic Filter behavior and record compatibility constraints in `specs/011-network-path-recovery/validation/history.md`
- [X] T003 [P] Add reusable component-matrix resource identifiers and redacted target-query helpers in `tests/testsupport/network_path_recovery_fixtures.go`
- [X] T004 [P] Add privileged host baseline and leak-diff helpers for namespaces, interfaces, bridge memberships, routes, rules, sockets, processes and captures in `tests/testsupport/runtime_leak_baseline.go`
- [X] T005 [P] Add feature validation evidence templates for local milestones, target acceptance, deployment identity and rollback in `specs/011-network-path-recovery/validation/README.md`

---

## Phase 2: Foundational Runtime and Contract Primitives

**Purpose**: Establish shared backing classification, observation and structured failure primitives required by every story.

**⚠️ CRITICAL**: Complete this phase before implementing any user story.

- [X] T006 Write failing domain tests for runtime backing kinds, ownership, usability and legal recovery transitions in `internal/domain/runtime_observation_test.go`
- [X] T007 [P] Write failing domain tests for endpoint backing classification and illegal namespace operations on host bridges in `internal/domain/topology_connection_test.go`
- [X] T008 Define recoverable runtime backing and forwarding/VLAN observation value types without runtime dependencies in `internal/domain/runtime_observation.go`
- [X] T009 Extend structured runtime problems with consistent recovery, cleanup and mismatch details in `internal/app/reconcile/problems.go`
- [X] T010 Implement endpoint backing classification for namespace ports, host bridges, QEMU taps and Docker veths in `internal/app/reconcile/endpoint_backing.go`
- [X] T011 [P] Add endpoint backing classifier tests covering every supported endpoint kind in `internal/app/reconcile/endpoint_backing_test.go`
- [X] T012 Add runtime inspection interfaces for network objects and connection endpoints in `internal/app/ports/network_runtime.go`
- [X] T013 Wire runtime inspection dependencies without changing existing mutation entry points in `cmd/netlabd/main.go`
- [X] T014 Run focused foundational tests and record the passing commands and results in `specs/011-network-path-recovery/validation/foundation.md`

**Checkpoint**: Shared runtime truth and endpoint ownership primitives are ready.

---

## Phase 3: User Story 1 — Recover a Runnable Topology After Restart (Priority: P1) 🎯 MVP

**Goal**: Recover or truthfully fail namespace-backed objects and connections after restart, and clean failed bridge links without leaks.

**Independent Test**: Create mixed namespace/bridge/QEMU/Docker resources, invalidate one owned namespace, restart the service, verify valid resources recover within 30 seconds, and delete a failed bridge-to-L3 link with zero remnants.

### Tests for User Story 1

- [X] T015 [P] [US1] Write failing namespace tests for listed-but-invalid references, owned recreation, unowned refusal and idempotent deletion in `internal/runtime/linuxnet/namespace_test.go`
- [X] T016 [P] [US1] Write failing recovery coordinator tests for dependency order, partial failure isolation and checkpoint outcomes in `internal/app/reconcile/recovery_coordinator_test.go`
- [X] T017 [P] [US1] Write failing dataplane tests proving plain-bridge cleanup never enters namespace cleanup in `internal/app/reconcile/dataplane_test.go`
- [X] T018 [P] [US1] Write failing network-object task tests for retryable reconcile tasks, revision conflicts, cancellation and deletion of failed links in `internal/app/reconcile/network_object_tasks_test.go`
- [X] T019 [P] [US1] Write failing SQLite recovery tests for pending/disconnecting reservation finalization and ordered outbox events in `internal/store/sqlite/recovery_repository_test.go`
- [X] T020 [P] [US1] Write failing HTTP contract tests for object/link reconcile endpoints and structured diagnostics in `internal/api/http/network_recovery_contract_test.go`
- [X] T021 [P] [US1] Write failing MCP parity tests for object/link reconcile and diagnostics tools in `internal/api/mcp/network_recovery_tools_test.go`
- [X] T022 [P] [US1] Write failing frontend tests for honest failed state, retry/delete actions and structured warnings in `web/src/features/diagnostics/DiagnosticsPanel.test.ts`
- [X] T023 [P] [US1] Write a failing privileged service-restart test with an invalid `/run/netns` reference in `tests/recovery/network_namespace_recovery_test.go`
- [X] T024 [P] [US1] Write a failing 20-cycle bridge-to-L3 failure cleanup and leak test in `tests/integration/network_object_link_cleanup_test.go`

### Implementation for User Story 1

- [X] T025 [US1] Replace namespace name-presence checks with in-namespace usability and ownership validation in `internal/runtime/linuxnet/namespace.go`
- [X] T026 [US1] Implement safe adopt, recreate, quarantine and delete outcomes for owned namespace backing in `internal/runtime/linuxnet/namespace.go`
- [X] T027 [US1] Add inspect methods for PC, L2 and L3 runtime backing and required interfaces in `internal/runtime/linuxnet/pc.go`, `internal/runtime/linuxnet/switch_l2.go`, and `internal/runtime/linuxnet/switch_l3.go`
- [X] T028 [US1] Reorder startup reconciliation across object backing, ports, attachments/object links and unified connection state in `internal/app/reconcile/recovery_coordinator.go`
- [X] T029 [US1] Mark unusable runtime backing failed instead of active and preserve unrelated recovery progress in `internal/app/reconcile/network_objects.go`
- [X] T030 [US1] Dispatch object-link create, compensation and deletion by endpoint backing kind in `internal/app/reconcile/dataplane.go`
- [X] T031 [US1] Finalize exhausted pending/disconnecting links with structured failed outcomes and retry/delete eligibility in `internal/app/reconcile/topology_connections.go`
- [X] T032 [US1] Add durable object/link reconcile commands and task handlers in `internal/app/reconcile/network_object_tasks.go`
- [X] T033 [US1] Expose reconcile endpoints and desired-versus-observed diagnostics in `internal/api/http/network_handlers.go`
- [X] T034 [US1] Add matching MCP reconcile and diagnostics tools in `internal/api/mcp/network_tools.go`
- [X] T035 [US1] Render backing health, recovery phase, cleanup, operator hint and retry/delete actions in `web/src/features/diagnostics/DiagnosticsPanel.vue`
- [X] T036 [US1] Integrate network-object recovery tasks and warning dialogs into `web/src/features/topology/TopologyWorkspace.vue`
- [X] T037 [US1] Run US1 unit, contract, privileged recovery and leak gates, record results in `specs/011-network-path-recovery/validation/us1-recovery.md`, and commit the milestone with a focused Git commit

**Checkpoint**: User Story 1 is independently deployable as the recovery-honesty MVP.

---

## Phase 4: User Story 2 — Route Real Dual-Stack Traffic Through VyOS (Priority: P1)

**Goal**: Provide complete BusyBox service-network and Ubuntu/VyOS/downstream IPv4/IPv6 forwarding with persistent return routes.

**Independent Test**: Achieve at least 99% success over 100 probes for BusyBox service peers, Ubuntu/VyOS transit and one endpoint in each downstream network before and after node/service restart.

### Tests for User Story 2

- [ ] T038 [P] [US2] Write failing node-network validation tests for explicit IPv4/IPv6 forwarding settings in `internal/domain/node_network_test.go`
- [ ] T039 [P] [US2] Write failing Docker endpoint tests for applying and observing forwarding in the container network namespace in `internal/runtime/linuxnet/docker_endpoint_test.go`
- [ ] T040 [P] [US2] Write failing L3 runtime tests for late interfaces, dual-stack forwarding, route replacement and diagnostics mismatch in `internal/runtime/linuxnet/switch_l3_test.go`
- [ ] T041 [P] [US2] Write failing dataplane tests that rerun L3 configuration after attachments and object links appear in `internal/app/reconcile/dataplane_test.go`
- [ ] T042 [P] [US2] Write failing HTTP tests for Docker forwarding settings and requested-versus-observed diagnostics in `internal/api/http/node_operations_network_test.go`
- [ ] T043 [P] [US2] Write a failing dual-stack component-matrix path test covering BusyBox, service router, Ubuntu, VyOS, core, DMZ and management endpoints in `tests/integration/dual_stack_network_path_test.go`
- [ ] T044 [P] [US2] Write a failing node/service restart persistence test for transit addresses, forwarding and return routes in `tests/recovery/dual_stack_path_recovery_test.go`

### Implementation for User Story 2

- [ ] T045 [US2] Add validated optional IPv4/IPv6 forwarding fields to Docker node network settings in `internal/domain/models.go`
- [ ] T046 [US2] Persist forwarding settings through existing node settings commands and revisions in `internal/app/command/node_settings.go`
- [ ] T047 [US2] Apply and inspect Docker namespace forwarding through argv-based host execution in `internal/runtime/linuxnet/docker_endpoint.go`
- [ ] T048 [US2] Make L3 configuration converge after each relevant port attachment and object-link transition in `internal/app/reconcile/dataplane.go`
- [ ] T049 [US2] Reconcile desired IPv4/IPv6 forwarding, addresses and routes and return exact observed mismatches in `internal/runtime/linuxnet/switch_l3.go`
- [ ] T050 [US2] Extend network-object diagnostics with desired/observed forwarding, interfaces and routes in `internal/api/http/network_handlers.go`
- [ ] T051 [US2] Add forwarding controls and mismatch display to node/network diagnostics in `web/src/features/diagnostics/DiagnosticsPanel.vue`
- [ ] T052 [US2] Add an isolated dual-stack acceptance fixture builder without proprietary configuration in `tests/testsupport/dual_stack_path_fixture.go`
- [ ] T053 [US2] Repair the BusyBox, service router, Ubuntu and VyOS desired topology through revisioned API operations documented in `specs/011-network-path-recovery/validation/component-matrix-repair.md`
- [ ] T054 [US2] Verify QGA-visible Ubuntu routes and operator-authorized VyOS interface/route state without storing credentials in `specs/011-network-path-recovery/validation/vyos-path.md`
- [ ] T055 [US2] Run US2 unit, contract, dual-stack and restart gates, record results in `specs/011-network-path-recovery/validation/us2-dual-stack.md`, and commit the milestone with a focused Git commit

**Checkpoint**: User Story 2 proves real bidirectional routed paths rather than router-interface-only reachability.

---

## Phase 5: User Story 3 — Exercise VLAN Access and Trunk Behavior (Priority: P2)

**Goal**: Make desired PVID/tagged membership converge after ports appear and prove VLAN 10/20 isolation and trunk forwarding.

**Independent Test**: Pass 100 tagged exchanges per VLAN with at least 99% success, block all unapproved cross-VLAN exchanges, and preserve exact observed membership across 10 restarts.

### Tests for User Story 3

- [ ] T056 [P] [US3] Write failing domain validation tests for VLAN ranges, duplicates and contradictory PVID/tagged membership in `internal/domain/network_object_test.go`
- [ ] T057 [P] [US3] Write failing L2 runtime tests for late-port membership, VLAN 1 removal, tagged trunks and readback in `internal/runtime/linuxnet/switch_l2_test.go`
- [ ] T058 [P] [US3] Write failing dataplane tests proving attachment/link transitions trigger L2 membership reconciliation in `internal/app/reconcile/dataplane_test.go`
- [ ] T059 [P] [US3] Write failing HTTP contract tests for VLAN validation and observed diagnostics in `internal/api/http/network_vlan_contract_test.go`
- [ ] T060 [P] [US3] Write failing MCP parity tests for L2 updates and VLAN diagnostics in `internal/api/mcp/network_vlan_tools_test.go`
- [ ] T061 [P] [US3] Write failing frontend tests for PVID/tagged editing, preserved invalid drafts and pending/mismatch states in `web/src/features/topology/NetworkObjectEditor.test.ts`
- [ ] T062 [P] [US3] Write a failing privileged VLAN access/trunk/isolation test in `tests/integration/vlan_trunk_path_test.go`
- [ ] T063 [P] [US3] Write a failing VLAN restart persistence test in `tests/recovery/vlan_membership_recovery_test.go`

### Implementation for User Story 3

- [ ] T064 [US3] Strengthen L2 VLAN configuration validation and normalization in `internal/domain/network_object.go`
- [ ] T065 [US3] Implement idempotent clear/apply/readback of PVID, untagged and tagged membership in `internal/runtime/linuxnet/switch_l2.go`
- [ ] T066 [US3] Reapply desired VLAN membership whenever an L2 port becomes available in `internal/app/reconcile/dataplane.go`
- [ ] T067 [US3] Mark L2 objects pending or failed until required observed membership matches desired state in `internal/app/reconcile/network_objects.go`
- [ ] T068 [US3] Extend diagnostics and OpenAPI response mapping with per-port VLAN observations in `internal/api/http/network_handlers.go`
- [ ] T069 [US3] Add PVID and tagged VLAN editing with client-side validation to `web/src/features/topology/NetworkObjectEditor.vue`
- [ ] T070 [US3] Display desired/observed VLAN membership and mismatch guidance in `web/src/features/diagnostics/DiagnosticsPanel.vue`
- [ ] T071 [US3] Configure the component-matrix VLAN 10/20 access ports and a two-VLAN tagged trunk through revisioned operations documented in `specs/011-network-path-recovery/validation/vlan-topology.md`
- [ ] T072 [US3] Run US3 domain, runtime, contract, UI, privileged VLAN and restart gates, record results in `specs/011-network-path-recovery/validation/us3-vlan.md`, and commit the milestone with a focused Git commit

**Checkpoint**: User Story 3 independently proves runtime VLAN behavior matches displayed configuration.

---

## Phase 6: User Story 4 — Validate Vendor Device Management and Data Paths (Priority: P2)

**Goal**: Give all four vendor appliances explicit management/data roles and prove client-facing switching or routing instead of cable-only status.

**Independent Test**: Report honest management readiness for every vendor node and complete at least 20 successful exchanges across each required vendor data path.

### Tests for User Story 4

- [ ] T073 [P] [US4] Write failing domain tests for management, LAN, WAN, trunk and client-facing interface-role validation in `internal/domain/device_role_test.go`
- [ ] T074 [P] [US4] Write failing node settings contract tests for role metadata persistence and secret rejection in `internal/api/http/device_role_contract_test.go`
- [ ] T075 [P] [US4] Write failing MCP parity tests for role metadata and readiness diagnostics in `internal/api/mcp/device_role_tools_test.go`
- [ ] T076 [P] [US4] Write failing frontend tests separating cable, guest, management and data-path states in `web/src/features/diagnostics/DeviceReadinessPanel.test.ts`
- [ ] T077 [P] [US4] Write a vendor path fixture test using stubs or authorized target images without embedding proprietary assets in `tests/integration/vendor_device_path_test.go`
- [ ] T078 [P] [US4] Write a restart test proving NetLab-owned vendor attachments and role metadata recover without guest credentials in `tests/recovery/vendor_attachment_recovery_test.go`

### Implementation for User Story 4

- [ ] T079 [US4] Define validated device interface role metadata in `internal/domain/device_role.go`
- [ ] T080 [US4] Persist role metadata through node settings and topology export/import without exposing credentials in `internal/app/command/node_settings.go` and `internal/app/command/lab_export.go`
- [ ] T081 [US4] Derive cable, guest readiness, management reachability and data-path evidence independently in `internal/app/query/device_readiness.go`
- [ ] T082 [US4] Expose device role and readiness diagnostics through existing node endpoints in `internal/api/http/node_operations_handlers.go`
- [ ] T083 [US4] Add equivalent MCP read/update behavior in `internal/api/mcp/node_tools.go`
- [ ] T084 [US4] Implement localized role editing and four-level readiness presentation in `web/src/features/diagnostics/DeviceReadinessPanel.vue`
- [ ] T085 [US4] Attach FancyWAN/FortiGate management and LAN/WAN networks and record operator-authorized guest prerequisites in `specs/011-network-path-recovery/validation/fancywan-fortigate.md`
- [ ] T086 [US4] Attach Ruijie Switch client and Ruijie Router routed paths and record operator-authorized guest prerequisites in `specs/011-network-path-recovery/validation/ruijie-path.md`
- [ ] T087 [US4] Run US4 contract, UI, vendor path and restart gates, record results in `specs/011-network-path-recovery/validation/us4-vendor.md`, and commit the milestone with a focused Git commit

**Checkpoint**: User Story 4 independently distinguishes physical attachment from usable appliance behavior.

---

## Phase 7: User Story 5 — Generate and Observe Stable Test Traffic (Priority: P3)

**Goal**: Run durable ICMP/HTTP/DNS workloads with successful/failed aggregates and correlate successful traffic with non-zero durable filter statistics and topology highlights.

**Independent Test**: Run all protocols for 10 minutes with successful exchanges every five seconds, increasing filter counts every 10 seconds, restart recovery, and highlight decay without counter reset.

### Tests for User Story 5

- [ ] T088 [P] [US5] Write failing domain tests for traffic workload validation, lifecycle and bounded aggregates in `internal/domain/traffic_workload_test.go`
- [ ] T089 [P] [US5] Write failing migration and repository tests for workload persistence, revisions and aggregate updates in `internal/store/sqlite/traffic_workload_repository_test.go`
- [ ] T090 [P] [US5] Write failing durable-task tests for create/start/stop/delete, cancellation, idempotency and recovery in `internal/app/command/traffic_workload_test.go`
- [ ] T091 [P] [US5] Write failing runtime adapter tests for namespace, Docker and QGA workload execution allowlists, timeouts and output bounds in `internal/runtime/linuxnet/traffic_workload_test.go`
- [ ] T092 [P] [US5] Write failing HTTP contract tests for all workload endpoints and structured failures in `internal/api/http/traffic_workload_contract_test.go`
- [ ] T093 [P] [US5] Write failing MCP parity tests for workload lifecycle and aggregate reads in `internal/api/mcp/traffic_workload_tools_test.go`
- [ ] T094 [P] [US5] Write failing Traffic Filter correlation tests for successful versus failed workload attempts in `internal/app/reconcile/traffic_filters_test.go`
- [ ] T095 [P] [US5] Write failing workload restart and orphan-task recovery tests in `tests/recovery/traffic_workload_recovery_test.go`
- [ ] T096 [P] [US5] Write failing 10-minute ICMP/HTTP/DNS privileged observation test in `tests/integration/traffic_workload_filter_test.go`
- [ ] T097 [P] [US5] Write failing frontend panel tests for aggregates, degraded running state, filter counters and highlight decay in `web/src/features/topology/TrafficWorkloadPanel.test.ts`

### Implementation for User Story 5

- [ ] T098 [US5] Define traffic workload, protocol aggregate and lifecycle validation in `internal/domain/traffic_workload.go`
- [ ] T099 [US5] Add migration 0016 for durable traffic workloads and aggregate outcomes in `internal/store/sqlite/migrations/0016_traffic_workloads.sql`
- [ ] T100 [US5] Implement workload repository transactions, revisions and outbox integration in `internal/store/sqlite/traffic_workload_repository.go`
- [ ] T101 [US5] Implement durable workload create/start/stop/delete command handlers in `internal/app/command/traffic_workload.go`
- [ ] T102 [US5] Implement capability-specific namespace, Docker and QGA workload execution with safe argv and bounded output in `internal/runtime/linuxnet/traffic_workload.go`
- [ ] T103 [US5] Implement workload scheduling, aggregate updates, cancellation and startup recovery in `internal/app/reconcile/traffic_workloads.go`
- [ ] T104 [US5] Correlate successful workload observations with existing filter resources without duplicating packet counters in `internal/app/reconcile/traffic_filters.go`
- [ ] T105 [US5] Register workload repositories, tasks, reconciler and runtime adapters in `cmd/netlabd/main.go`
- [ ] T106 [US5] Add workload list/create/get/state/delete HTTP handlers in `internal/api/http/traffic_workload_handlers.go`
- [ ] T107 [US5] Add equivalent workload MCP tools and schemas in `internal/api/mcp/traffic_workload_tools.go`
- [ ] T108 [US5] Publish ordered workload lifecycle and aggregate stream events in `internal/api/stream/events.go`
- [ ] T109 [US5] Add workload API client and Pinia state convergence in `web/src/api/client.ts` and `web/src/stores/trafficWorkloads.ts`
- [ ] T110 [US5] Implement localized workload creation, lifecycle, aggregates and degraded-state UI in `web/src/features/topology/TrafficWorkloadPanel.vue`
- [ ] T111 [US5] Integrate workload statistics with Traffic Filter details and topology highlight overlays in `web/src/features/topology/TopologyWorkspace.vue`
- [ ] T112 [US5] Run US5 migration, domain, task, runtime, contract, UI, 10-minute observation and restart gates, record results in `specs/011-network-path-recovery/validation/us5-traffic.md`, and commit the milestone with a focused Git commit

**Checkpoint**: All user stories are independently functional and the component-matrix laboratory can generate meaningful observable traffic.

---

## Phase 8: Polish, Full Validation and Authoritative Deployment

**Purpose**: Validate cross-story behavior, security, cleanup, documentation, immutable deployment and rollback.

- [ ] T113 [P] Add cross-story two-browser/API/MCP concurrency acceptance for reconcile, VLAN update and workload lifecycle in `tests/e2e/network_path_recovery_concurrency.spec.ts`
- [ ] T114 [P] Add host-command injection, ownership-boundary, secret-redaction and workload allowlist tests in `tests/security/network_path_recovery_security_test.go`
- [ ] T115 [P] Add full laboratory export/import and duplicate compatibility tests for forwarding, VLAN, role and workload configuration in `tests/integration/network_path_export_import_test.go`
- [ ] T116 Add a complete temporary-lab acceptance journey covering US1–US5 and automatic cleanup in `tests/e2e/network_path_recovery.spec.ts`
- [ ] T117 Run `go test ./...`, `go vet ./...`, frontend tests, production build, lint, formatting, privileged integration, recovery and leak gates and record exact results in `specs/011-network-path-recovery/validation/final-local.md`
- [ ] T118 Commit final cross-story validation and documentation as a focused Git commit and record the clean commit SHA in `specs/011-network-path-recovery/validation/milestones.md`
- [ ] T119 Build the deployment artifact from the clean commit and record candidate ID, binary digest, contract digest and migration 16 state in `specs/011-network-path-recovery/validation/deployment.md`
- [ ] T120 Create and verify the target rollback directory with previous binary, configuration, readiness identity and online SQLite backup and record it in `specs/011-network-path-recovery/validation/deployment.md`
- [ ] T121 Deploy the identified artifact to `10.72.1.7` without editing target source and verify service health, release identity and database integrity in `specs/011-network-path-recovery/validation/deployment.md`
- [ ] T122 Remove the known stuck object link and repair the existing component-matrix desired topology through revisioned authoritative operations recorded in `specs/011-network-path-recovery/validation/target-repair.md`
- [ ] T123 Run 10 service restarts, 20 cleanup cycles, six-QEMU health, BusyBox/VyOS dual-stack, VLAN trunk, vendor path and 10-minute traffic-filter acceptance on the target and record results in `specs/011-network-path-recovery/validation/target-acceptance.md`
- [ ] T124 Run target Chromium validation for diagnostics, warnings, VLAN editor, workload panel, non-zero counters, fingerprints and highlight decay and record screenshots/evidence references in `specs/011-network-path-recovery/validation/target-browser.md`
- [ ] T125 Verify final namespace/interface/bridge/route/rule/socket/process/capture/workload leak baselines and rollback integrity in `specs/011-network-path-recovery/validation/target-cleanup.md`
- [ ] T126 Commit deployment and target-acceptance records, push `main`, and confirm the authoritative target reports the recorded clean candidate in `specs/011-network-path-recovery/validation/milestones.md`

---

## Dependencies and Execution Order

### Phase Dependencies

- **Phase 1** has no prerequisites and establishes evidence and reusable fixtures.
- **Phase 2** depends on Phase 1 and blocks all user stories.
- **US1** depends on Phase 2 and is the MVP because every later runtime path depends on honest recovery.
- **US2** depends on US1 for recovered namespaces, attachments and L3 state.
- **US3** depends on US1; it can run in parallel with US2 after the shared dataplane reconciliation contract is stable.
- **US4** depends on US1 and benefits from US2/US3 network paths, but role/readiness implementation can start after Phase 2.
- **US5** depends on US1 for recovery and on at least one working US2 path for full integration; domain/storage/API work can begin after Phase 2.
- **Phase 8** depends on all five user stories and their focused milestone commits.

### User Story Completion Order

```text
Setup → Foundation → US1 Recovery MVP
                         ├──→ US2 Dual-Stack Routing ──┐
                         ├──→ US3 VLAN/Trunk ─────────┤
                         ├──→ US4 Vendor Paths ───────┤
                         └──→ US5 Traffic Workloads ──┘
                                      ↓
                         Full Validation and Deployment
```

## Parallel Execution Examples

### User Story 1

Run T015–T024 in parallel because they create separate failing test files. After T025–T031 stabilize runtime behavior, T033–T036 can split across HTTP, MCP and frontend files.

### User Story 2

Run T038–T044 in parallel. After the domain setting is fixed, Docker forwarding work T047 and L3 runtime work T049 can proceed concurrently before dataplane integration.

### User Story 3

Run T056–T063 in parallel. Domain validation T064, runtime application T065 and frontend editor T069 have separate write scopes and can proceed concurrently before final integration.

### User Story 4

Run T073–T078 in parallel. Backend readiness T081–T083 and frontend presentation T084 can proceed concurrently after role metadata T079 is stable.

### User Story 5

Run T088–T097 in parallel. Migration/repository T099–T100, runtime adapter T102 and UI component T110 have disjoint write scopes; command/reconciler integration follows their contracts.

## Implementation Strategy

### MVP First

1. Complete Setup and Foundation.
2. Implement US1 only.
3. Prove restart recovery and failed bridge-link cleanup independently.
4. Commit the US1 milestone before adding routing, VLAN or workload scope.

### Incremental Delivery

1. Add US2 to establish complete IPv4/IPv6 paths.
2. Add US3 to prove access/trunk VLAN behavior.
3. Add US4 to convert vendor cable tests into management and client-path tests.
4. Add US5 to generate measurable successful traffic and validate filter statistics.
5. Run cross-story acceptance and deploy only from a clean identified commit.

### Task Completion Rule

Every task must leave its named files buildable and reviewable. Every user-story phase ends with focused tests, recorded evidence and a focused Git commit before dependent deployment work begins.
