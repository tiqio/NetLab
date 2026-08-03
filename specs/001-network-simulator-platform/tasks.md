---
description: "Dependency-ordered implementation tasks for NetLab"
---

# Tasks: NetLab Network Simulator Platform

**Input**: Design documents from `specs/001-network-simulator-platform/`

**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/`, `quickstart.md`

**Tests**: Runtime, networking, lifecycle, API, MCP, console, capture, and resource-management tests are mandatory under the project constitution.

**Organization**: Tasks are grouped by user story. Complete Setup and Foundational phases first; each story phase ends with an independently executable checkpoint.

## Phase 1: Setup

**Purpose**: Initialize the monorepo, toolchains, generated-contract workflow, and deployment skeleton.

- [X] T001 Create the Go module and backend directory skeleton from the implementation plan in `go.mod` and `cmd/netlabd/main.go`
- [X] T002 [P] Create the Vue 3 TypeScript workspace with Vite, Pinia, Vue Router, Vitest, and Playwright in `web/package.json`
- [X] T003 [P] Add backend dependency pins including Gin, go-qemu commit, Docker SDK, MCP SDK, SQLite, netlink, and WebSocket libraries in `go.mod`
- [X] T004 [P] Configure Go formatting, vet, static analysis, and test targets in `Makefile`
- [X] T005 [P] Configure TypeScript, ESLint, formatting, Vitest, and Playwright in `web/tsconfig.json`
- [X] T006 [P] Add generated SPA embedding entry and placeholder assets in `internal/api/http/webassets.go`
- [X] T007 [P] Add configuration defaults for listener, paths, startup concurrency, capture quotas, and retention in `deploy/config/netlab.example.yaml`
- [X] T008 [P] Add systemd unit with required ordering, capabilities, restart policy, and state directories in `deploy/systemd/netlab.service`
- [X] T009 [P] Add install, uninstall, and host-prerequisite checks in `deploy/scripts/install.sh`
- [X] T010 [P] Add built-in template manifest directories and JSON/YAML schemas in `templates/schema/template.schema.json`
- [X] T011 [P] Add repository ignore rules for images, runtime state, seed ISOs, captures, exports, secrets, and frontend build output in `.gitignore`
- [X] T012 Add CI entry points for unit, contract, frontend, and non-privileged integration suites in `.github/workflows/ci.yml`

## Phase 2: Foundational

**Purpose**: Build blocking domain, persistence, task, event, configuration, ownership, and reconciliation infrastructure used by every story.

**Critical**: No user-story implementation begins until this phase passes its unit and migration tests.

### Foundational Tests

- [X] T013 [P] Add domain ID, revision, state-transition, and validation tests in `internal/domain/domain_test.go`
- [X] T014 [P] Add SQLite migration, foreign-key, WAL, idempotency, and outbox atomicity tests in `internal/store/sqlite/store_test.go`
- [X] T015 [P] Add operation-task cancellation, retry, checkpoint, and compensation tests in `internal/app/task/runner_test.go`
- [X] T016 [P] Add configuration validation and unsafe-listener warning tests in `internal/support/config/config_test.go`
- [X] T017 [P] Add host-resource ownership and deterministic naming tests in `internal/runtime/ownership/manifest_test.go`

### Foundational Implementation

- [X] T018 Implement UUIDv7 IDs, revision values, timestamps, problem details, and shared enums in `internal/domain/types.go`
- [X] T019 [P] Implement laboratory, node, interface, link, task, audit, artifact, and outbox domain types in `internal/domain/models.go`
- [X] T020 Implement legal node, task, capture, port-mapping, and deletion state transitions in `internal/domain/transitions.go`
- [X] T021 Create the initial forward-only SQLite schema for all entities and indexes from the data model in `migrations/0001_initial.sql`
- [X] T022 Implement SQLite connection setup with WAL, foreign keys, busy timeout, migration checksums, and serialized writes in `internal/store/sqlite/database.go`
- [X] T023 Implement transactional repositories for laboratories, nodes, interfaces, links, tasks, audit, artifacts, idempotency, and outbox in `internal/store/sqlite/repositories.go`
- [X] T024 Implement command/query transaction boundaries and optimistic-revision enforcement in `internal/app/transaction/service.go`
- [X] T025 Implement durable task enqueueing, bounded workers, cancellation, checkpoints, retry policy, and terminal results in `internal/app/task/runner.go`
- [X] T026 Implement transactional outbox publication, ordered sequence retention, and replay-gap detection in `internal/app/events/outbox.go`
- [X] T027 Implement structured JSON logging, correlation IDs, redaction, health, readiness, and metrics registry in `internal/support/observability/observability.go`
- [X] T028 Implement validated configuration loading and startup security warnings for all-interface unauthenticated listening in `internal/support/config/config.go`
- [X] T029 Define runtime adapter interfaces for QEMU, Docker, Linux networking, cgroups, captures, consoles, images, and artifacts in `internal/app/ports/runtime.go`
- [X] T030 Implement deterministic host-resource names, ownership manifests, and safe unowned-resource guards in `internal/runtime/ownership/manifest.go`
- [X] T031 Implement single-instance locking and process-level lifecycle coordination in `internal/app/reconcile/instance_lock.go`
- [X] T032 Implement the generic desired-versus-observed reconciliation scheduler and drift reporting in `internal/app/reconcile/coordinator.go`
- [X] T033 Implement Gin server assembly, error envelopes, request IDs, CORS/origin policy, health endpoints, and graceful shutdown in `internal/api/http/server.go`
- [X] T034 Wire configuration, database, task runner, outbox publisher, reconciler, HTTP server, and embedded SPA in `cmd/netlabd/main.go`

**Checkpoint**: Migrations apply on an empty and existing database; tasks and outbox events survive restart; unsafe network exposure is prominently reported.

## Phase 3: User Story 1 - Build and Operate a Shared Topology (Priority: P1)

**Goal**: Multiple browser and automation clients create and operate one shared topology without session invalidation or account-scoped state.

**Independent Test**: Two browsers and one API client concurrently create, edit, start, stop, and delete a laboratory using a basic namespace-backed endpoint; every accepted mutation converges within 3 seconds and stale revisions are rejected.

### Tests for User Story 1

- [X] T035 [P] [US1] Add laboratory and topology command tests for revisions, conflicts, deletion, and recovery policy in `internal/app/command/laboratory_test.go`
- [X] T036 [P] [US1] Add laboratory repository and snapshot consistency tests in `internal/store/sqlite/laboratory_repository_test.go`
- [X] T037 [P] [US1] Add REST contract tests for lab, node, link, task, and event endpoints in `tests/contract/us1_shared_topology_test.go`
- [X] T038 [P] [US1] Add ordered event replay, reset-required, and slow-consumer tests in `internal/api/stream/events_test.go`
- [X] T039 [P] [US1] Add Vue store tests for snapshot hydration, event application, revision conflicts, and reconnect in `web/src/stores/laboratory.test.ts`
- [X] T040 [P] [US1] Add Playwright concurrent-client acceptance coverage in `tests/e2e/us1_shared_topology.spec.ts`
- [X] T041 [P] [US1] Add service-restart adoption and host-restart policy integration tests for namespace endpoints in `tests/recovery/us1_recovery_test.go`

### Implementation for User Story 1

- [X] T042 [P] [US1] Implement laboratory aggregate commands for create, rename, duplicate, recovery policy, and delete in `internal/app/command/laboratory.go`
- [X] T043 [P] [US1] Implement topology snapshot and list queries with event sequence cursors in `internal/app/query/laboratory.go`
- [X] T044 [US1] Implement generic node and interface creation, desired-state changes, and deletion commands in `internal/app/command/node.go`
- [X] T045 [US1] Implement link connect/disconnect commands with endpoint validation and conflict detection in `internal/app/command/link.go`
- [X] T046 [US1] Implement a basic supervised namespace endpoint runtime for shared-topology lifecycle validation in `internal/runtime/linuxnet/endpoint.go`
- [X] T047 [US1] Implement laboratory/node/link reconciliation and owned-resource cleanup workflows in `internal/app/reconcile/topology.go`
- [X] T048 [US1] Implement startup adoption of namespace endpoints and automatic/remain-stopped host recovery policies in `internal/app/reconcile/recovery.go`
- [X] T049 [US1] Implement lab, node, link, task, and snapshot handlers from the OpenAPI contract in `internal/api/http/topology_handlers.go`
- [X] T050 [US1] Implement ordered `/api/v1/events` WebSocket replay and snapshot reset signaling in `internal/api/stream/events.go`
- [X] T051 [P] [US1] Implement typed frontend API clients and problem/revision handling in `web/src/api/client.ts`
- [X] T052 [US1] Implement shared laboratory Pinia state, event replay, reconnect, and conflict refresh in `web/src/stores/laboratory.ts`
- [X] T053 [US1] Implement laboratory list, topology canvas, node/link editor, task progress, loading, empty, and error states in `web/src/features/topology/TopologyWorkspace.vue`
- [X] T054 [US1] Integrate shared topology routes and acceptance-host restart scenarios in `web/src/router/index.ts`

**Checkpoint**: US1 passes with one shared authoritative topology, durable task/event state, concurrent mutation protection, and restart recovery using the basic endpoint runtime.

## Phase 4: User Story 2 - Use Versioned Device Templates (Priority: P1)

**Goal**: Users create QEMU and Docker nodes from immutable, versioned, capability-aware templates and operator-supplied images.

**Independent Test**: Register two versions for each required family, instantiate selected versions, start Ubuntu QEMU and BusyBox Docker nodes, reject invalid images, and verify existing nodes stay pinned when defaults change.

### Tests for User Story 2

- [X] T055 [P] [US2] Add template manifest schema, capability, default, and immutability tests in `internal/domain/template_test.go`
- [X] T056 [P] [US2] Add image staging, archive traversal, checksum, qcow2, OCI digest, and license-gate tests in `internal/runtime/image/importer_test.go`
- [X] T057 [P] [US2] Add QEMU command-line, overlay, socket, cgroup, start/stop, and adoption fixture tests in `internal/runtime/qemu/adapter_test.go`
- [X] T058 [P] [US2] Add Docker API negotiation, labels, resource limits, lifecycle, and adoption tests in `internal/runtime/docker/adapter_test.go`
- [X] T059 [P] [US2] Add template/image REST contract tests in `tests/contract/us2_templates_test.go`
- [X] T060 [P] [US2] Add privileged QEMU and Docker create/start/stop/delete integration tests in `tests/integration/us2_runtime_test.go`
- [X] T061 [P] [US2] Add Vue template/version selection and unsupported-capability component tests in `web/src/features/templates/TemplatePicker.test.ts`

### Implementation for User Story 2

- [X] T062 [P] [US2] Implement device template, template version, and image version domain validation in `internal/domain/template.go`
- [X] T063 [US2] Add SQLite repositories and immutable-reference constraints for templates and images in `internal/store/sqlite/template_repository.go`
- [X] T064 [P] [US2] Add built-in QEMU manifests for FancyWAN, Ubuntu, FortiGate, and VyOS in `templates/qemu/manifest.yaml`
- [X] T065 [P] [US2] Add built-in Docker manifests for BusyBox and Ubuntu in `templates/docker/manifest.yaml`
- [X] T066 [US2] Implement manifest loading, schema validation, capability resolution, and default-version selection in `internal/app/query/templates.go`
- [X] T067 [US2] Implement streamed upload/local import staging, SHA-256 content addressing, safe archive extraction, qcow2 validation, and atomic publication in `internal/runtime/image/importer.go`
- [X] T068 [US2] Implement OCI reference resolution to immutable digest and Docker image availability checks in `internal/runtime/image/oci.go`
- [X] T069 [US2] Implement QEMU runtime directories, qcow2 overlays, deterministic sockets, cgroup placement, direct process launch, stop, query, and adoption in `internal/runtime/qemu/adapter.go`
- [X] T070 [US2] Implement go-qemu QMP connection, capability query, event consumption, and raw command wrapper in `internal/runtime/qemu/qmp.go`
- [X] T071 [US2] Implement Docker create/start/stop/delete/inspect/adopt with `network=none`, ownership labels, and resource limits in `internal/runtime/docker/adapter.go`
- [X] T072 [US2] Extend node reconciliation to dispatch QEMU and Docker adapters and enforce image/license availability in `internal/app/reconcile/nodes.go`
- [X] T073 [US2] Implement template, image import, image status, and capability HTTP handlers in `internal/api/http/template_handlers.go`
- [X] T074 [P] [US2] Implement template/image typed frontend clients in `web/src/api/templates.ts`
- [X] T075 [US2] Implement template family/version picker, image status, license warnings, and capability display in `web/src/features/templates/TemplatePicker.vue`
- [X] T076 [US2] Integrate template-based node creation and pinned version display into the topology editor in `web/src/features/topology/NodeEditor.vue`
- [X] T077 [US2] Add image import and template administration views with progress/error handling in `web/src/views/TemplatesView.vue`

**Checkpoint**: Required template families and versions are data-driven; Ubuntu QEMU and BusyBox Docker nodes run; invalid or unreviewed images cannot start.

## Phase 5: User Story 3 - Automate Topologies through API and MCP (Priority: P1)

**Goal**: Programs and LLMs discover capabilities and fully orchestrate the same laboratories, nodes, links, tasks, captures, and exports visible in the browser.

**Independent Test**: An MCP client creates a mixed topology, observes tasks, retries safely, exports/imports it, and deletes it while an open browser sees every state change.

### Tests for User Story 3

- [X] T078 [P] [US3] Add complete OpenAPI validation and UI-mutation coverage tests in `tests/contract/us3_openapi_parity_test.go`
- [X] T079 [P] [US3] Add idempotency replay, mismatch, expiry, and concurrent-request tests in `internal/app/command/idempotency_test.go`
- [X] T080 [P] [US3] Add MCP tool schema, error, task-envelope, and HTTP transport tests in `tests/contract/us3_mcp_test.go`
- [X] T081 [P] [US3] Add lab export schema, redaction, missing dependency, and round-trip tests in `tests/contract/us3_export_test.go`
- [X] T082 [P] [US3] Add end-to-end browser/API/MCP shared-state parity tests in `tests/e2e/us3_automation.spec.ts`

### Implementation for User Story 3

- [X] T083 [P] [US3] Implement idempotency-key persistence, request fingerprinting, replay, and conflict handling in `internal/app/command/idempotency.go`
- [X] T084 [P] [US3] Implement task status, progress, result, error, and cancellation query services in `internal/app/query/tasks.go`
- [X] T085 [US3] Complete versioned OpenAPI handlers for every durable resource and mutating browser action in `internal/api/http/routes.go`
- [X] T086 [US3] Implement HTTP precondition, idempotency, resource-exhaustion, and retry headers in `internal/api/http/middleware.go`
- [X] T087 [P] [US3] Implement MCP server initialization, Streamable HTTP transport, origin validation, and typed error mapping in `internal/api/mcp/server.go`
- [X] T088 [US3] Implement MCP discovery, lab, node, interface, link, port mapping, console, capture, filter, and task tools in `internal/api/mcp/tools.go`
- [X] T089 [US3] Implement laboratory export generation with schema versioning and mandatory redaction report in `internal/app/command/export.go`
- [X] T090 [US3] Implement laboratory import preflight, dependency resolution, atomic creation, and no-substitution errors in `internal/app/command/import.go`
- [X] T091 [US3] Implement artifact metadata, opaque download handles, expiry, and byte streaming in `internal/api/http/artifact_handlers.go`
- [X] T092 [P] [US3] Implement redacted audit event creation for browser, API, MCP, and system actions in `internal/app/audit/service.go`
- [X] T093 [P] [US3] Generate TypeScript API types from `contracts/openapi.yaml` into `web/src/api/generated.ts`
- [X] T094 [US3] Replace handwritten mutation calls with generated contract clients across `web/src/api/index.ts`
- [X] T095 [US3] Add task history, audit visibility, export, import, and automation status UI in `web/src/views/AutomationView.vue`

**Checkpoint**: REST, SPA, and MCP use identical application commands; retries create no duplicates; redacted exports round-trip without images, secrets, or captures.

## Phase 6: User Story 4 - Diagnose Traffic Live (Priority: P2)

**Goal**: Users open Telnet/VNC consoles, stream live captures to Wireshark, retain bounded artifacts, and see where matching packets traverse the topology.

**Independent Test**: Generate a known three-hop flow, view the console, stream packets to Wireshark, retain/download the capture, and identify every traversed link for 100 flows.

### Tests for User Story 4

- [X] T096 [P] [US4] Add serial/VNC proxy framing, reconnect, idle, and bandwidth-limit tests in `internal/runtime/console/proxy_test.go`
- [X] T097 [P] [US4] Add capture worker filter, stream, tee, quota, truncation, cancellation, and cleanup tests in `internal/runtime/capture/worker_test.go`
- [X] T098 [P] [US4] Add traffic-filter compilation, fingerprint, correlation-window, loop, and ambiguity tests in `internal/runtime/capture/path_test.go`
- [X] T099 [P] [US4] Add console, capture, artifact, and traffic-filter contract tests in `tests/contract/us4_diagnostics_test.go`
- [X] T100 [P] [US4] Add privileged three-hop Wireshark stream and path-observation integration tests in `tests/integration/us4_diagnostics_test.go`
- [X] T101 [P] [US4] Add console and capture UI component tests in `web/src/features/diagnostics/DiagnosticsPanel.test.ts`

### Implementation for User Story 4

- [X] T102 [US4] Implement QEMU serial Unix-socket console and allocated Telnet-compatible listener management in `internal/runtime/console/telnet.go`
- [X] T103 [US4] Implement VNC Unix-socket WebSocket proxy and noVNC session descriptors in `internal/runtime/console/vnc.go`
- [X] T104 [US4] Implement console discovery and WebSocket handlers with reconnect-safe lifecycle in `internal/api/stream/console.go`
- [X] T105 [US4] Implement supervised pcap/pcapng capture workers with BPF, cancellation, counters, tee, and backpressure handling in `internal/runtime/capture/worker.go`
- [X] T106 [US4] Implement capture quotas, artifact retention, expiry cleanup, and global ceiling enforcement in `internal/app/reconcile/captures.go`
- [X] T107 [US4] Implement capture start/status/stop/stream and artifact handlers from the OpenAPI contract in `internal/api/http/capture_handlers.go`
- [X] T108 [US4] Implement traffic-match validation and BPF compilation for address, protocol, and port combinations in `internal/runtime/capture/filter.go`
- [X] T109 [US4] Implement packet fingerprint correlation and observed interface/link aggregation in `internal/runtime/capture/path.go`
- [X] T110 [US4] Implement traffic-filter task lifecycle, bounded observations, and event publication in `internal/app/reconcile/traffic_filters.go`
- [X] T111 [US4] Implement Telnet terminal, noVNC, live capture, download, and truncation UI in `web/src/features/diagnostics/DiagnosticsPanel.vue`
- [X] T112 [US4] Implement topology link highlighting, counts, timing, and ambiguous-loop presentation in `web/src/features/topology/TrafficPathOverlay.vue`

**Checkpoint**: Consoles reconnect without lifecycle effects; Wireshark receives live packets within 5 seconds; traffic filters identify observed links and report ambiguity explicitly.

## Phase 7: User Story 5 - Change Running Nodes Safely (Priority: P2)

**Goal**: Users rewire live nodes, hot-add/remove QEMU NICs, select NIC drivers, map host ports, execute guest commands, and enforce CPU-time quotas.

**Independent Test**: On a running QEMU node, rewire an existing NIC, hot-add and remove another NIC, establish an SSH/ZTP port mapping, execute a bounded guest command, and verify 2 vCPUs constrained to one core of CPU time.

### Tests for User Story 5

- [X] T113 [P] [US5] Add Linux tap/veth live bridge-membership and rollback tests in `internal/runtime/linuxnet/link_test.go`
- [X] T114 [P] [US5] Add QMP netdev/device add/delete event and partial-failure tests in `internal/runtime/qemu/hotplug_test.go`
- [X] T115 [P] [US5] Add guest-exec polling, timeout, cancellation, decoding, and output-limit tests in `internal/runtime/qemu/guest_test.go`
- [X] T116 [P] [US5] Add cgroup v2 CPU quota, cleanup, and Docker normalization tests in `internal/runtime/cgroup/manager_test.go`
- [X] T117 [P] [US5] Add nftables port mapping conflict, DNAT/SNAT, ownership, and cleanup tests in `internal/runtime/linuxnet/portmap_test.go`
- [X] T118 [P] [US5] Add live-link, hot-plug, guest-exec, port-map, and quota contract tests in `tests/contract/us5_live_changes_test.go`
- [X] T119 [P] [US5] Add privileged running-node mutation and failure-injection integration tests in `tests/integration/us5_live_changes_test.go`

### Implementation for User Story 5

- [X] T120 [US5] Implement tap/veth creation, owned bridge membership changes, link-state observation, and compensating rollback in `internal/runtime/linuxnet/link.go`
- [X] T121 [US5] Update link reconciliation to rewire existing interfaces without node restart in `internal/app/reconcile/links.go`
- [X] T122 [US5] Implement QMP `netdev_add`, `device_add`, `device_del`, event wait, `netdev_del`, and reconciliation queries in `internal/runtime/qemu/hotplug.go`
- [X] T123 [US5] Implement interface hot-add/remove commands with template driver validation and task checkpoints in `internal/app/command/interface.go`
- [X] T124 [US5] Implement QGA `guest-exec` and `guest-exec-status` with timeout, cancellation, decoding, and bounded output in `internal/runtime/qemu/guest.go`
- [X] T125 [US5] Implement guest-command application service and redacted audit records in `internal/app/command/guest_exec.go`
- [X] T126 [US5] Implement cgroup v2 subtree creation, process attachment, `cpu.max`, memory limits, metrics, and cleanup in `internal/runtime/cgroup/manager.go`
- [X] T127 [US5] Apply independent vCPU count and CPU-time quotas to QEMU and normalized limits to Docker in `internal/app/reconcile/resources.go`
- [X] T128 [US5] Implement NetLab-owned nftables table, host-port conflict checks, DNAT/SNAT rules, and cleanup in `internal/runtime/linuxnet/portmap.go`
- [X] T129 [US5] Implement port-mapping commands, repository operations, and reconciliation in `internal/app/command/port_mapping.go`
- [X] T130 [US5] Implement live-interface, guest-exec, port-mapping, and resource handlers in `internal/api/http/node_operations_handlers.go`
- [X] T131 [P] [US5] Add frontend clients for interfaces, guest commands, mappings, and resource metrics in `web/src/api/nodeOperations.ts`
- [X] T132 [US5] Implement live wiring and NIC-driver controls in `web/src/features/topology/InterfaceEditor.vue`
- [X] T133 [US5] Implement guest command, host-port mapping, CPU quota, and resource status UI in `web/src/features/nodes/NodeOperationsPanel.vue`

**Checkpoint**: Running-node changes remain observable and reversible; port collisions are explicit; guest output is bounded; CPU time is host-enforced independently from vCPU count.

## Phase 8: User Story 6 - Add Lightweight Network and PC Nodes (Priority: P3)

**Goal**: Users build dual-stack PC, layer-2 switch, layer-3 switch, bridge, and NAT scenarios without full virtual machines.

**Independent Test**: A topology with dual-stack PCs, L2/L3 switches, DHCPv4, DHCPv6, SLAAC, routes, and NAT passes addressing and connectivity checks and cleans up all host resources.

### Tests for User Story 6

- [X] T134 [P] [US6] Add PC static IPv4/IPv6, DHCPv4, DHCPv6, SLAAC, timeout, and diagnostics tests in `internal/runtime/linuxnet/pc_test.go`
- [X] T135 [P] [US6] Add namespace L2 bridge, VLAN membership, and forwarding tests in `internal/runtime/linuxnet/switch_l2_test.go`
- [X] T136 [P] [US6] Add namespace L3 addressing, route, forwarding, and failure tests in `internal/runtime/linuxnet/switch_l3_test.go`
- [X] T137 [P] [US6] Add NAT bridge prefix, allocation, masquerade, uplink, overlap, and cleanup tests in `internal/runtime/linuxnet/nat_test.go`
- [X] T138 [P] [US6] Add privileged dual-stack and NAT end-to-end integration tests in `tests/integration/us6_lightweight_nodes_test.go`
- [X] T139 [P] [US6] Add lightweight-node editor and diagnostics component tests in `web/src/features/nodes/LightweightNodeEditor.test.ts`

### Implementation for User Story 6

- [X] T140 [P] [US6] Implement network-object schemas for bridge, NAT, PC, L2 switch, and L3 switch configuration in `internal/domain/network_object.go`
- [X] T141 [US6] Implement network-object and attachment repositories with overlap and ownership constraints in `internal/store/sqlite/network_repository.go`
- [X] T142 [US6] Implement supervised PC namespace runtime with static IPv4/IPv6, DHCPv4, DHCPv6, SLAAC, routes, DNS, and diagnostics in `internal/runtime/linuxnet/pc.go`
- [X] T143 [US6] Implement L2 switch namespace bridge, port membership, VLAN configuration, and state observation in `internal/runtime/linuxnet/switch_l2.go`
- [X] T144 [US6] Implement L3 switch namespace interfaces, routes, forwarding, and optional policy in `internal/runtime/linuxnet/switch_l3.go`
- [X] T145 [US6] Implement bridge and NAT bridge allocation, nftables masquerade, uplink handling, and cleanup in `internal/runtime/linuxnet/nat.go`
- [X] T146 [US6] Implement network-object commands, lifecycle reconciliation, and attachment workflows in `internal/app/reconcile/network_objects.go`
- [X] T147 [US6] Implement network-object HTTP handlers and topology snapshot serialization in `internal/api/http/network_handlers.go`
- [X] T148 [US6] Add MCP network-object creation and diagnostics tools in `internal/api/mcp/network_tools.go`
- [X] T149 [US6] Implement PC, switch, bridge, NAT, addressing, route, and lease-status UI in `web/src/features/nodes/LightweightNodeEditor.vue`

**Checkpoint**: Lightweight dual-stack topologies demonstrate static addressing, DHCPv4, DHCPv6, SLAAC, L2/L3 forwarding, NAT, capture visibility, and complete cleanup.

## Phase 9: Polish and Cross-Cutting Concerns

**Purpose**: Complete production hardening, packaging, documentation, usability, performance, and full-system verification.

- [X] T150 [P] Add OpenAPI linting, generated-client drift checks, MCP schema validation, and export-schema validation in `tests/contract/contracts_test.go`
- [X] T151 [P] Add audit redaction and repository secret/image/capture scanning tests in `tests/security/redaction_test.go`
- [X] T152 [P] Add API request limits, WebSocket slow-consumer limits, task queue backpressure, and quota error tests in `tests/integration/limits_test.go`
- [X] T153 [P] Add service restart during every lifecycle transition and owned-orphan quarantine tests in `tests/recovery/lifecycle_matrix_test.go`
- [X] T154 [P] Add full-host automatic/restricted recovery orchestration and bounded startup concurrency tests in `tests/recovery/host_restart_test.go`
- [X] T155 Add 100-cycle QEMU, Docker, namespace, bridge, nftables, cgroup, socket, capture, seed, and artifact leak suite in `tests/integration/leak_cycle_test.go`
- [X] T156 [P] Add 10-node/4-QEMU performance, 3-second event convergence, 10-second link change, and 5-second capture benchmarks in `tests/integration/acceptance_scale_test.go`
- [X] T157 [P] Add accessibility keyboard navigation, focus management, color-independent state, and responsive topology tests in `tests/e2e/accessibility.spec.ts`
- [X] T158 Harden systemd sandboxing, Linux capabilities, writable paths, file permissions, and log rotation in `deploy/systemd/netlab.service`
- [X] T159 Add database backup, artifact cleanup, image-store maintenance, and disaster-recovery utilities in `deploy/scripts/maintenance.sh`
- [X] T160 Add deployment, firewall, image licensing, template authoring, MCP, Wireshark, recovery, and troubleshooting documentation in `docs/operations.md`
- [X] T161 Add architecture decisions, lifecycle diagrams, ownership rules, and contributor test guidance in `docs/architecture.md`
- [X] T162 Execute every scenario in the feature quickstart and record acceptance-host evidence in `specs/001-network-simulator-platform/validation-report.md`
- [X] T163 Run formatting, static analysis, unit, contract, frontend, privileged integration, recovery, leak, and end-to-end suites through `Makefile`

## Dependencies and Execution Order

### Phase Dependencies

- Phase 1 has no dependencies.
- Phase 2 depends on Phase 1 and blocks every user story.
- US1 depends on Phase 2 and establishes shared topology, tasks, events, and basic runtime reconciliation.
- US2 depends on US1 domain/reconciliation contracts and provides production QEMU/Docker nodes.
- US3 depends on US1; its full mixed-topology acceptance uses US2, while API/MCP infrastructure can proceed after US1.
- US4 depends on US1 events/artifacts and US2 QEMU console/runtime sockets.
- US5 depends on US1 links/tasks and US2 QEMU/Docker adapters.
- US6 depends on US1 topology and foundational Linux ownership; it can run in parallel with US2 after US1.
- Phase 9 depends on all selected user stories.

### User Story Dependency Graph

```text
Setup -> Foundational -> US1 Shared Topology
                          ├─> US2 Templates/Runtimes ─┬─> US4 Diagnostics
                          │                          ├─> US5 Live Changes
                          │                          └─> US3 Full Automation Acceptance
                          ├─> US3 API/MCP Foundation
                          └─> US6 Lightweight Nodes

US2 + US3 + US4 + US5 + US6 -> Polish/Acceptance
```

### Within Each Story

- Write and run story tests first; confirm they fail for the intended missing behavior.
- Implement domain/store changes before application commands and reconcilers.
- Implement runtime adapters before handlers that expose their capabilities.
- Implement HTTP/MCP contracts before wiring SPA interactions.
- Complete each story checkpoint before relying on it from a later story.

## Parallel Execution Examples

### US1

- T035–T041 can run in parallel because they target distinct test layers.
- T042 and T043 can run in parallel after foundational repositories are available.
- T051 can proceed while backend handlers T049–T050 are implemented against the frozen OpenAPI contract.

### US2

- T055–T061 can run in parallel as failing tests.
- T064 and T065 are independent template-manifest work.
- QEMU tasks T069–T070 and Docker task T071 can proceed in parallel after T062–T068 define templates/images.
- Frontend client T074 can proceed against the contract while runtime adapters are implemented.

### US3

- T078–T082 can run in parallel.
- Idempotency T083, task queries T084, MCP transport T087, audit T092, and generated types T093 use disjoint files.
- Export T089 and import T090 can be split after the export schema and artifact service are stable.

### US4

- T096–T101 can run in parallel.
- Telnet T102, VNC T103, capture T105, and filter compilation T108 have disjoint runtime ownership.
- UI tasks T111–T112 can proceed after contract fixtures are available.

### US5

- T113–T119 can run in parallel.
- QMP hot-plug T122, guest execution T124, cgroups T126, and nftables mapping T128 are independent adapter slices.
- Frontend client T131 can proceed while backend commands are completed.

### US6

- T134–T139 can run in parallel.
- PC T142, L2 T143, L3 T144, and NAT T145 adapters own separate files and can proceed after T140–T141.

## Implementation Strategy

### MVP

1. Complete Setup and Foundational phases.
2. Complete US1 for shared topology, tasks, events, conflicts, and recovery.
3. Complete the Ubuntu QEMU and BusyBox Docker subset of US2 tasks T055–T076.
4. Validate one browser-driven mixed topology before expanding all template families.

US1 alone proves the shared control-plane design, but US1 plus the minimal US2 slice is the smallest
production-useful MVP because it runs real supported nodes.

### Incremental Delivery

1. Shared topology and basic runtime reconciliation.
2. Versioned QEMU/Docker templates and images.
3. Full API/MCP automation and portable exports.
4. Consoles, Wireshark capture, and traffic-path observation.
5. Live NIC changes, guest commands, port mappings, and CPU quotas.
6. Dual-stack lightweight PC/switch/NAT nodes.
7. Full recovery, leak, scale, accessibility, and operations hardening.

## Notes

- `[P]` means the task can run concurrently with adjacent tasks because it owns different files and has no unmet dependency.
- User-story labels map directly to the six stories in `spec.md`.
- Commercial images remain operator-supplied and are never committed or automatically downloaded.

## Phase 10: Convergence

- [X] T164 CRITICAL redact guest-command arguments and retain only bounded non-secret audit metadata with regression tests per Constitution VII and FR-030 (contradicts)
- [X] T165 Implement template-version-pinned node creation, immutable image resolution, template defaults/capabilities, and pre-start checksum, compatibility, availability, and license validation per FR-016–FR-023 and US2 (partial)
- [X] T166 Implement laboratory- and node-isolated cloud-init seed ISO generation, QEMU attachment, secret-safe task handling, and lifecycle cleanup for Ubuntu, VyOS, and FancyWAN per FR-024–FR-025 and US2/AC3 (missing)
- [X] T167 Implement reconciled tap/veth endpoint creation, bridge and network-object attachment, live link connect/disconnect, operational status, rollback, and owned cleanup across QEMU, Docker, and namespace nodes per FR-012–FR-015 and US1 (partial)
- [X] T168 Implement laboratory duplication and a terminal deletion reconciler that stops and removes owned nodes, links, network objects, mappings, captures, artifacts, and database rows with retryable cleanup state per FR-008, FR-015, and FR-035 (partial)
- [X] T169 Implement service-restart adoption and full-host recovery orchestration for nodes, links, network objects, namespaces, port mappings, captures, tasks, and recovery policies, including bounded startup and client-visible progress per FR-006 and US1/AC4–AC5 (partial)
- [X] T170 Replace unsupported MCP placeholders with typed, structured tools for templates, laboratory import/delete, nodes, interfaces, links, guest commands, port mappings, consoles, captures, traffic filters, and task operations per FR-054 and US3 (partial)
- [X] T171 Route long-running topology and lifecycle mutations through durable cancellable tasks, enforce replay-safe idempotency for every retriable mutation, and record redacted audit events for every required action per FR-055–FR-059 and US3 (partial)
- [X] T172 Implement a browser topology workspace for placing versioned nodes and network objects, wiring interfaces, controlling lifecycle, viewing task and operational state, resolving conflicts, and deleting resources per US1 and SC-001 (missing)
- [X] T173 Align OpenAPI schemas and paths with actual HTTP behavior, include every durable/browser operation, regenerate the Vue client, and add executable request/response and route drift checks per FR-053 and plan: API contract (contradicts)
- [X] T174 Complete capture lifecycle metadata and state semantics, count packets, register retained captures as downloadable expiring artifacts, schedule cleanup, and recover or terminate abandoned capture processes across restarts per FR-045–FR-048 and FR-051–FR-052 (partial)
- [X] T175 Connect Traffic Filter sessions to live packet observers on selected interfaces and links, parse supported match fields, correlate packet fingerprints across hops, and publish counts and observation times per FR-049–FR-050 and SC-006 (missing)
- [X] T176 Enforce host-capacity admission and declared CPU, memory, storage, interface, and process limits at node start, automatically apply cgroups/runtime limits, expose observed metrics, and validate the 10-node/4-QEMU acceptance workload per FR-037–FR-038 and SC-015 (partial)
- [X] T177 Complete lightweight PC, L2, L3, bridge, and NAT attachments and end-to-end forwarding, expose DHCPv4/DHCPv6/SLAAC lease and advertisement status plus NAT/forwarding diagnostics, and verify cleanup per FR-039–FR-044 and US6 (partial)
- [X] T178 Implement reconnect-safe Telnet/VNC console backends for every template that advertises them, including Docker where declared, or remove unsupported capability declarations with contract tests per FR-026–FR-027 (partial)

## Phase 11: Convergence

- [X] T179 CRITICAL add explicit storage, interface, and process ownership fields to node/resource models, enforce declared CPU, memory, storage, process, and interface limits across QEMU, Docker, and namespace runtimes, expose normalized metrics, and run a real 10-node/4-QEMU acceptance workload per Constitution V, FR-037–FR-038, and SC-015 (contradicts)
- [X] T180 Implement a unified durable recovery coordinator for service restart and host reboot that adopts or restores nodes, links, network objects, namespaces, port mappings, captures, and queued tasks with bounded concurrency, policy handling, progress checkpoints, actionable failures, and lifecycle-matrix tests per FR-006 and US1/AC4–AC5 (partial)
- [X] T181 Route laboratory deletion/duplication/import/export, node lifecycle, link changes, captures, traffic filters, and other long-running mutations through durable cancellable operation tasks with status, progress, result, error, and timestamps across REST and MCP per FR-055 and US3/AC3 (partial)
- [X] T182 Enforce replay-safe idempotency fingerprints and response replay for every retriable REST and MCP create/mutation operation, including conflict tests for reused keys with different payloads per FR-056 and US3/AC2 (partial)
- [X] T183 Record redacted operational audit events for topology mutations, node lifecycle actions, laboratory operations, port mappings, captures, traffic filters, task cancellation, and their success/failure outcomes with regression tests per FR-059 (missing)
- [X] T184 Implement reconciled veth/namespace attachment plumbing for PC, L2 switch, and L3 switch network objects, propagate attachment state/errors, verify VLAN/routing/DHCPv4/DHCPv6/SLAAC/NAT forwarding and owned cleanup on a privileged host per FR-039–FR-044 and US6/AC1–AC4 (partial)
- [X] T185 Derive ingress/egress direction and ordered hop observations from selected interface/link capture points, preserve packet fingerprints across hops, report ambiguity correctly, and add multi-hop live-traffic tests per FR-049–FR-050 and SC-006 (partial)
- [X] T186 Align every OpenAPI request/response schema with current handlers, generate a complete typed Vue client for all documented operations, replace substring parity checks with executable route and schema validation, and cover capture/task/console/network-object envelopes per FR-053 and plan: API contract (contradicts)
- [X] T187 Subscribe the topology workspace to shared event streams, incrementally reconcile nodes/interfaces/links/network objects without full reloads, and display/cancel durable operation tasks with conflict and reconnect handling per FR-057, US1, and SC-001 (partial)
- [X] T188 Deploy the current build to the acceptance host, rerun privileged runtime, recovery, capture, traffic-path, lightweight-network, cleanup, and scale scenarios, and replace stale July 23, 2026 evidence and capture-state assertions in `specs/001-network-simulator-platform/validation-report.md` per Constitution VI and SC-015 (contradicts)

## Phase 12: Convergence

- [X] T189 CRITICAL fix resource application after fresh node start, enforce declared CPU-time, memory, storage, interface, and process limits across QEMU, Docker, and namespace runtimes, and expose normalized configured and observed metrics per Constitution V and FR-037–FR-038 (contradicts)
- [X] T190 CRITICAL extend recovery to persist per-resource checkpoints, continue independent recovery work after individual failures, report node/link/object/mapping/capture outcomes, and validate active lifecycle restoration after a full host reboot per Constitution VI, FR-006, SC-007, and SC-013 (partial)
- [X] T191 Route laboratory duplicate/import/export/delete, node lifecycle, link and network-object changes, captures, and traffic filters through durable task-runner handlers whose cancellation controls the underlying work and whose REST/MCP responses expose task status, progress, result, error, and timestamps per FR-055 and US3/AC3 (partial)
- [X] T192 Implement PC DNS configuration, supervised bounded DHCPv4/DHCPv6 clients, acquired-route and SLAAC advertisement status, actionable timeout diagnostics, and NAT allocation/external/translation/cleanup status with one repeatable privileged dual-stack scenario per FR-039–FR-044, SC-011, and US6/AC3–AC4 (partial)
- [X] T193 Add a validated recursive Traffic Filter expression model supporting AND, OR, and NOT combinations across address, protocol, and port predicates, propagate it through REST, MCP, OpenAPI, and the capture compiler, and add rejection and live-match tests per FR-049 and US4/AC3 (partial)
- [X] T194 Align interface operational-state serialization across the Go domain, SQLite event payloads, OpenAPI, generated Vue types, and the topology store, remove the divergent handwritten API type path, and add executable response/event tests that fail on property-name drift per FR-014, FR-053, and plan: API contract (contradicts)
- [X] T195 Integrate reconnectable Telnet/VNC consoles, live capture start/stream/stop/download controls, Traffic Filter creation, observation refresh, ambiguity display, and topology path highlighting into routed Vue views using the shared API and event state per US4/AC1–AC3 and SC-005 (partial)
- [X] T196 Integrate the lightweight-node editor into the topology workspace with valid PC/L2/L3/bridge/NAT configuration forms, interface-to-object attachment controls, VLAN/routing/address diagnostics, and browser tests for the complete lightweight workflow per US6/AC1–AC4 and SC-011 (partial)
- [X] T197 Build a repeatable acceptance harness that consumes only operator-registered legal images and verifies four simultaneous QEMU nodes, QGA execution, QMP NIC hot-plug, Telnet/VNC, SSH/ZTP port mappings, CPU-time quota behavior, live capture, cleanup, and current-build host-reboot recovery per SC-015, US5/AC2–AC5, and Constitution VI (partial)

## Phase 13: Convergence

- [X] T198 CRITICAL move managed QEMU, Docker, namespace, capture, and helper runtimes outside the control-service systemd kill domain, preserve them across `netlab.service` restarts, adopt them without process replacement, and add a real acceptance-host restart test that proves stable PIDs, sockets, links, and client-visible state per FR-006, US1/AC4, and Constitution VI (contradicts)
- [X] T199 CRITICAL enforce the declared node lifecycle state machine in persistence and reconciliation, persist provisioning/starting/stopping/deleting transitions, wait for QEMU process and QMP readiness before reporting running, map early process exit and timeout logs to actionable terminal failures, and add real failure-path lifecycle tests per FR-007, FR-058, and Constitution VI (contradicts)
- [X] T200 CRITICAL replace post-hoc `http.mutation` shadow tasks with durable task-runner handlers for laboratory duplicate/import/export/delete, node lifecycle, link and network-object changes, captures, and traffic filters so cancellation interrupts or compensates the underlying operation and recovery resumes the same task with complete progress/result/error/timestamps across REST and MCP per FR-055, US3/AC3, and Constitution IV (partial)
- [X] T201 CRITICAL enforce `InterfaceLimit` before interface persistence and hot-plug, align admitted limits with available QEMU PCIe hot-plug capacity and Docker/namespace adapters, reject unsatisfiable starts or additions before side effects, and add boundary plus rollback tests for running nodes per FR-037–FR-038 and Constitution V (contradicts)
- [X] T202 CRITICAL implement startup and periodic owned-resource discovery across QEMU runtime directories, Docker labels, network namespaces, interfaces, bridges, nftables rules, cgroups, captures, and helper processes; quarantine and audit unknown owned orphans and safely remove validated abandoned resources without touching unowned host state per FR-015, FR-052, Constitution V–VI, and plan: reconciliation and recovery (missing)
- [X] T203 standardize REST, MCP, task-runner, and reconciler failures as structured problems containing resource, lifecycle phase, retryability, cleanup progress, and operator action; preserve typed runtime errors instead of generic `task_failed`, and map conflicts, exhaustion, and temporary unavailability to the planned HTTP status and retry headers per FR-058 and plan: API errors (partial)
- [X] T204 complete the supervised PC network helper for bounded DHCPv4/DHCPv6 renewal and cancellation, DNS and acquired-route management, SLAAC advertisement state, detailed lease/time-out diagnostics, and NAT allocation/external/translation/cleanup reporting, then replace the skipped privileged test with one repeatable dual-stack L2/L3/NAT scenario per FR-039–FR-044, SC-011, and US6/AC3–AC4 (partial)
- [X] T205 extend operator-image acceptance to one 10-node mixed laboratory containing four concurrent QEMU guests plus Docker and lightweight nodes, validate selectable legal template images and Ubuntu/VyOS/FancyWAN cloud-init, open real Telnet/VNC sessions, prove SSH or ZTP traffic through host mappings, measure CPU-time throttling under load, exercise live rewiring/capture/shared-state convergence, and verify cleanup and host-reboot recovery per SC-010, SC-015, US2/AC3, and US5/AC2–AC5 (partial)

## Phase 14: Convergence

- [X] T206 CRITICAL replace `MutationAutomation` post-hoc `http.mutation` records with durable task-runner commands for laboratory duplicate/import/export/delete, node lifecycle, link and network-object mutation, capture, and traffic-filter operations; make cancellation interrupt or compensate the real operation, recover the same task after restart, and return identical task envelopes through REST and MCP per Constitution IV, FR-055, and US3/AC3 (contradicts)
- [X] T207 CRITICAL wire startup and periodic owned-resource discovery for QEMU runtime directories and sockets, Docker labels, namespaces, veth/TAP/bridge devices, nftables rules, cgroups, capture workers, console proxies, and helper processes; quarantine and audit unknown owned resources and remove only validated abandoned resources with explicit unowned-host safety tests per FR-015, FR-052, Constitution V–VI, and plan: reconciliation and recovery (missing)
- [X] T208 CRITICAL replace `COUNT` plus `MAX(slot)+1` interface admission with a transactional lowest-free-slot allocator bounded by each runtime's actual capacity, serialize concurrent additions, and delete the interface row plus TAP/QMP artifacts when asynchronous hot-add fails; cover deleted-slot reuse, slot 63/64 boundaries, concurrent requests, and rollback on a running QEMU node per FR-037–FR-038 and Constitution V (contradicts)
- [X] T209 CRITICAL complete node lifecycle persistence and reconciliation for provisioning, starting, running, stopping, stopped, deleting, and failed phases; attach bounded timeouts, QEMU log diagnostics, cleanup progress, and operator actions to terminal failures, and run real invalid-image, early-exit, QMP-timeout, stop-failure, and deletion-failure tests on the acceptance host per FR-007, FR-058, and Constitution VI (partial)
- [X] T210 extend service-restart adoption validation to simultaneously running Docker containers, namespace nodes, active links, capture/helper processes, console proxies, and client streams; prove stable runtime identifiers and uninterrupted client-visible state while the control PID changes, then verify terminal owned cleanup per FR-006, US1/AC4, and Constitution VI (partial)
- [X] T211 consolidate REST, MCP, task-runner, and reconciler error conversion around one wrapped/pointer-safe structured `Problem` contract carrying resource, task, phase, retryability, cleanup status, operator action, and retry metadata; add parity tests for conflict, exhaustion, temporary unavailability, cancellation, and runtime failure responses per FR-058 and plan: API errors (partial)
- [X] T212 implement a supervised cancellable PC networking helper with bounded DHCPv4/DHCPv6 acquisition and renewal, lease-derived DNS and route installation/removal, SLAAC advertisement and timeout state, and actionable diagnostics; replace the unconditional privileged skip with an isolated repeatable dual-stack L2/L3/NAT scenario that verifies translation and owned cleanup per FR-039–FR-044, SC-011, and US6/AC3–AC4 (partial)
- [X] T213 build one operator-image acceptance laboratory with ten mixed nodes including four concurrent QEMU guests, immutable Docker nodes, and lightweight nodes; validate selectable legal template versions, Ubuntu/VyOS/FancyWAN cloud-init, real Telnet and VNC sessions, SSH or ZTP through host mappings, measured one-core CPU-time throttling under load, live rewiring/capture/shared-state convergence, cleanup, service restart, and host-reboot recovery per SC-010, SC-015, US2/AC3, and US5/AC2–AC5 (partial)

## Phase 15: Convergence

- [X] T214 CRITICAL remove synchronous post-hoc `http.mutation` shadow tasks and implement durable task-runner command handlers for laboratory duplicate/import/export/delete, node lifecycle, link and network-object mutation, captures, and traffic filters; ensure cancellation interrupts or compensates actual work, restart recovery resumes the same task, and REST/MCP expose identical task envelopes per Constitution IV, FR-055, and US3/AC3 (contradicts)
- [X] T215 CRITICAL integrate startup and periodic discovery of owned QEMU directories/sockets/processes, Docker labels, namespaces, veth/TAP/bridges, nftables rules, cgroups, capture workers, console proxies, and helper processes; quarantine and audit unknown owned orphans, validate abandonment before removal, and prove unowned host resources are never modified per FR-015, FR-052, Constitution V–VI, and plan: reconciliation and recovery (missing)
- [X] T216 CRITICAL persist and reconcile the complete provisioning, starting, running, stopping, stopped, deleting, and failed node lifecycle with bounded phase timeouts and structured cleanup/operator diagnostics; execute real invalid-image, early-exit, QMP-readiness-timeout, stop-failure, and deletion-failure cases on the acceptance host per FR-007, FR-058, and Constitution VI (partial)
- [X] T217 CRITICAL add a controlled running-QEMU hot-add failure acceptance that reaches TAP creation and QMP mutation, verifies QMP/netdev/TAP/interface-row compensation, retries successfully into the same lowest free slot, and proves zero leaked devices or ownership records per FR-037–FR-038 and Constitution V–VI
- [X] T218 extend service-restart adoption acceptance to concurrent QEMU and Docker nodes, namespace nodes, live links, capture/helper processes, console proxies, and active browser/API streams; prove stable runtime identifiers and uninterrupted shared state across the control PID replacement, followed by terminal owned cleanup per FR-006, US1/AC4, and Constitution VI (partial)
- [X] T219 migrate all node, topology, network-object, capture, deletion, mutation, and recovery reconciler failures through the shared wrapped/pointer-safe structured `Problem` normalizer, filling resource/task, phase, retryability, cleanup status, operator action, and retry delay; add cross-surface REST/MCP/task/reconciler parity tests for every terminal failure class per FR-058 and plan: API errors (partial)
- [X] T220 implement a supervised cancellable PC networking helper with bounded DHCPv4/DHCPv6 acquisition and renewal, lease-derived DNS/routes, SLAAC advertisement and timeout state, and cleanup diagnostics; replace the unconditional privileged skip with an isolated repeatable dual-stack L2/L3/NAT forwarding and translation test per FR-039–FR-044, SC-011, and US6/AC3–AC4 (partial)
- [X] T221 run one repeatable operator-image acceptance laboratory with ten mixed nodes including four concurrent QEMU guests, immutable Docker nodes, and lightweight nodes; verify selectable legal versions, Ubuntu/VyOS/FancyWAN cloud-init, real Telnet/VNC sessions, SSH or ZTP through host mappings, measured CPU-time throttling under load, live rewiring/capture/shared-state convergence, cleanup, service restart, and host reboot recovery per SC-010, SC-015, US2/AC3, and US5/AC2–AC5 (partial)

## Phase 16: Convergence

- [X] T222 CRITICAL replace the synchronous `http.mutation` middleware shadow-task pattern with durable task-runner command handlers for laboratory duplicate/import/export/delete, node lifecycle, links, network objects, captures, and traffic filters; ensure cancellation interrupts or compensates the underlying operation, restart recovery resumes the original task, and REST/MCP return identical task envelopes per Constitution IV, FR-055, and US3/AC3 (contradicts)
- [X] T223 CRITICAL wire startup and periodic discovery across owned QEMU directories/processes/sockets, Docker labels, namespaces, TAP/veth/bridges, nftables rules, cgroups, capture workers, console proxies, and helper processes; quarantine and audit unknown owned resources, validate abandonment before removal, persist ownership for hot-added TAPs, and prove unowned host resources are never modified per Constitution V–VI, FR-015, FR-052, and plan: reconciliation and recovery (missing)
- [X] T224 CRITICAL persist complete provisioning, starting, running, stopping, stopped, deleting, and failed lifecycle checkpoints with bounded per-phase deadlines and structured cleanup/operator diagnostics; run repeatable acceptance-host invalid-image, early-exit, QMP-readiness-timeout, stop-failure, and deletion-failure cases with terminal leak checks per Constitution VI, FR-007, and FR-058 (partial)
- [X] T225 extend service-restart adoption to one concurrent laboratory containing QEMU and Docker nodes, namespace nodes, live links and attachments, capture/helper processes, console proxies, and active browser/API streams; prove stable runtime identifiers, resumed durable tasks, shared state across control PID replacement, and terminal owned cleanup per FR-006, US1/AC4, and Constitution VI (partial)
- [X] T226 finish structured failure parity by routing every node, topology, network-object, capture, deletion, mutation, and recovery terminal error through the wrapped/pointer-safe normalizer and adding table-driven REST/MCP/task/reconciler tests for resource/task identity, phase, retryability, cleanup state, operator action, HTTP status, and retry delay per FR-058 and plan: API errors (partial)
- [X] T227 replace one-shot PC `dhclient -1` execution with a supervised cancellable helper that performs bounded DHCPv4/DHCPv6 acquisition and renewal, lease-derived DNS/routes, SLAAC advertisement and timeout tracking, restart adoption, and cleanup diagnostics; replace the unconditional privileged skip with an isolated repeatable dual-stack L2/L3/NAT forwarding and translation test per FR-039–FR-044, SC-011, and US6/AC3–AC4 (partial)
- [X] T228 create and run one repeatable operator-image acceptance laboratory with ten mixed nodes including four concurrent QEMU guests, immutable Docker nodes, and lightweight nodes; verify legal selectable versions, Ubuntu/VyOS/FancyWAN cloud-init, Telnet/VNC, SSH or ZTP mappings, measured CPU-time throttling under load, live rewiring/capture/shared-state convergence, cleanup, service restart, and host reboot recovery per SC-010, SC-015, US2/AC3, and US5/AC2–AC5 (partial)

## Phase 17: Convergence

- [X] T229 finish table-driven structured failure parity across every node, topology, network-object, capture, deletion, ownership, data-plane, mutation, and recovery terminal path; normalize wrapped runtime errors and verify REST/MCP/task/reconciler resource/task identity, phase, retryability, cleanup, operator action, HTTP status, and retry delay per FR-058 and plan: API errors (partial)
- [X] T230 replace one-shot PC DHCP execution with a supervised cancellable and restart-adoptable helper for bounded DHCPv4/DHCPv6 acquisition and renewal, lease-derived DNS/routes, SLAAC advertisement/timeout state, and cleanup diagnostics; replace the unconditional privileged skip with an isolated repeatable dual-stack L2/L3/NAT forwarding and translation test per FR-039–FR-044, SC-011, and US6/AC3–AC4 (partial)
- [X] T231 create and run one repeatable operator-image acceptance laboratory with ten mixed nodes including four concurrent QEMU guests, immutable Docker nodes, and lightweight nodes; verify legal selectable versions, Ubuntu/VyOS/FancyWAN cloud-init, real Telnet/VNC, SSH or ZTP mappings, measured one-core CPU-time throttling under load, live rewiring/capture/shared-state convergence, cleanup, service restart, and host-reboot recovery per SC-010, SC-015, US2/AC3, and US5/AC2–AC5 (partial)
