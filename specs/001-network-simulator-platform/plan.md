# Implementation Plan: NetLab Network Simulator Platform

**Branch**: `001-network-simulator-platform` | **Date**: 2026-07-23 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/001-network-simulator-platform/spec.md`

## Summary

Build a single-host network simulator whose server owns all topology and runtime state. A Go control
plane exposes one versioned HTTP contract to the Vue SPA, external automation, and MCP tools. Durable
desired state and operation history live in SQLite; reconcilers manage QEMU processes, Docker
containers, Linux bridges, namespaces, nftables rules, consoles, captures, and traffic filters.

Each node receives an owned runtime directory and host-resource manifest. Control-service restarts
adopt still-running resources; full host restarts restore the prior running set according to the
laboratory recovery policy. Runtime adapters are isolated behind domain ports so QEMU, Docker, Linux
networking, capture, and console behavior can be tested independently and reconciled uniformly.

## Technical Context

**Language/Version**: Go 1.26.x for backend and host agent; TypeScript 5.x with Vue 3 for frontend

**Primary Dependencies**: Gin 1.12.x, `github.com/digitalocean/go-qemu` pinned to commit
`ee9b0668d242374af3071f72347d7c741ca4300c`, Docker Engine Go SDK with API negotiation,
Model Context Protocol Go SDK v1, `modernc.org/sqlite`, `vishvananda/netlink`, WebSocket support,
Vue Router, Pinia, Vite, xterm.js, and noVNC

**Storage**: SQLite in WAL mode for desired state, revisions, tasks, audit, and artifact metadata;
filesystem object store under `/var/lib/netlab` for qcow2 layers, seed ISOs, runtime sockets, exports,
and bounded capture artifacts

**Testing**: Go unit and contract tests, QMP socket fixtures, SQLite migration/repository tests,
Vitest component tests, Playwright browser tests, OpenAPI/schema validation, and privileged integration
tests on the designated Linux/KVM acceptance host

**Target Platform**: Single x86_64 Linux server with systemd, cgroup v2, KVM, QEMU >= 8.2, Docker
Engine with API negotiation, `iproute2`, nftables, tcpdump/dumpcap-compatible capture tooling, and
`xorriso`; the current acceptance host is Ubuntu 24.04 with QEMU 8.2.2 and Docker 29.5.2

**Project Type**: Monorepo web application with one Go control-plane binary, embedded Vue SPA,
versioned HTTP/WebSocket API, and MCP endpoint

**Performance Goals**: Browser and automation clients converge accepted state changes within 3
seconds; valid live-link changes complete within 10 seconds in at least 95% of attempts; live capture
delivers first matching traffic within 5 seconds; service restart reconciliation completes within 60
seconds; host-restart recovery reaches a terminal result within 5 minutes

**Constraints**: Single-host only; no application authentication; listens on all interfaces by
default with prominent network-boundary warnings; no proprietary images or secrets in source control;
live rewiring cannot stop nodes; QEMU vCPU count and CPU-time quota are independent; exported labs omit
images, secrets, bootstrap-sensitive data, and captures

**Scale/Scope**: One acceptance laboratory with 10 nodes including up to 4 running QEMU nodes;
bounded startup concurrency defaults to 2 QEMU nodes and 4 non-QEMU nodes per host; capture defaults
to 4 concurrent sessions, 15-minute duration, 256 MiB per session, 24-hour artifact retention, and a
10 GiB global artifact ceiling, all configurable

## Constitution Check

*GATE: Passed before Phase 0 research and re-checked after Phase 1 design.*

- **Shared state — PASS**: SQLite stores revisions and desired state; all clients consume the same
  application services and ordered event stream. Mutations require resource revisions and support
  idempotency keys.
- **Control parity — PASS**: The SPA calls `/api/v1`; MCP tools call the same application commands and
  return the same task/resource envelopes. There are no UI-only mutations.
- **Runtime scope — PASS**: Adapters are limited to QEMU, Docker, Linux bridge/NAT, and network
  namespaces. Device behavior is selected from data-driven template manifests.
- **Live operations — PASS**: Cable changes move owned tap/veth ports between per-link or shared
  bridges. Interface-count changes use QMP `netdev_add`/`device_add` and `device_del`/`netdev_del` with
  event confirmation and reconciliation. Console, capture, and filter streams have explicit contracts.
- **Safety and recovery — PASS**: Every host object has an owner tag/name and cleanup manifest. QEMU
  processes use cgroup v2; Docker uses equivalent engine limits. Restart adoption, host restore,
  rollback, timeouts, output bounds, and redacted audit events are designed explicitly.
- **Verification — PASS**: Unit, API/MCP contract, frontend, privileged runtime, failure injection,
  restart recovery, and 100-cycle leak tests are part of the validation strategy.
- **Image and secret hygiene — PASS**: Image ingestion verifies checksum and provenance, commercial
  images are operator-supplied, exports are redacted, seed data is node-scoped, and artifact directories
  are excluded from source control.

### Post-Design Re-check

The data model gives every mutable resource a revision and every host resource an owner. The OpenAPI,
event, MCP, and export contracts preserve control parity and bounded artifacts. The quickstart includes
restart adoption, host recovery, live NIC/link changes, captures, traffic filters, and leak checks.
No constitution exception is required.

## Architecture

### Control Plane

1. **HTTP/API layer** validates transport data and maps requests to application commands and queries.
2. **MCP layer** publishes typed tools that invoke the same commands and queries as HTTP handlers.
3. **Application layer** enforces revisions, idempotency, lifecycle rules, transactions, and task
   orchestration without issuing host commands directly.
4. **Domain layer** defines laboratories, templates, nodes, interfaces, links, operations, captures,
   traffic filters, artifacts, and state-machine invariants.
5. **Reconciler layer** compares desired state with adapter-observed actual state, executes bounded
   steps, records checkpoints, and publishes ordered events.
6. **Runtime adapters** own QEMU/QMP/QGA, Docker Engine, netlink/netns/nftables, console proxies,
   capture workers, cgroups, image ingestion, and filesystem artifacts.
7. **Store layer** serializes writes through short SQLite transactions, uses WAL for concurrent reads,
   and writes an outbox event in the same transaction as each durable mutation.

### Runtime Ownership

- QEMU runtime: `/var/lib/netlab/runtime/qemu/<node-id>/` contains pid metadata, QMP/QGA/serial/VNC
  sockets, launch manifest, overlay disk references, and reconciliation checkpoints.
- Docker runtime: containers carry `io.netlab.*` ownership labels and use network mode `none`; owned
  veth interfaces are moved into the container network namespace after creation.
- Linux network runtime: host objects use deterministic `nl`-prefixed names plus persisted mappings.
  Each point-to-point link is an owned bridge; shared network nodes own a multi-access bridge.
- NAT runtime: NetLab owns one nftables table and chains, with rules keyed by resource ID. Host-port
  mappings use validated DNAT/SNAT rules rather than QEMU user-mode networking.
- Process limits: each QEMU/capture/helper process is placed in a node- or task-specific cgroup v2
  subtree. `cpu.max` implements CPU-time quota independently from QEMU `-smp` vCPU count.

### Reconciliation and Recovery

- The service acquires a single-host instance lock before accepting mutations.
- On control-service startup it scans QEMU runtime directories, QMP sockets, Docker labels, netns,
  links, nftables rules, and helper processes; owned live resources are adopted rather than restarted.
- Unknown unowned resources are reported but never deleted. Owned orphan resources are quarantined,
  audited, and removed only after state validation.
- After a full host restart, labs with automatic recovery recreate the previously running set in
  dependency order: network objects, containers/lightweight nodes, QEMU nodes, links/port mappings,
  captures only when explicitly configured as resumable.
- Every reconciliation step is retryable and checkpointed. Partial hot-plug uses QMP events and a
  compensating action before the task reaches a terminal state.

### Networking and Live Changes

- QEMU interfaces use tap devices. Docker and namespace nodes use veth pairs. One end is permanently
  owned by the node interface; changing a cable changes bridge membership and does not stop the node.
- Adding a new QEMU interface creates a tap and QMP netdev first, then adds the guest device, waits for
  confirmation, and attaches the tap to the desired link. Removal detaches the link, requests device
  deletion, waits for `DEVICE_DELETED`, deletes the netdev, then removes the tap.
- Layer-2 switches use a bridge inside an owned namespace. Layer-3 switches use namespace interfaces,
  routes, forwarding, and optional nftables policy. PC nodes run a supervised network helper supporting
  static addressing, DHCPv4, DHCPv6, and SLAAC.

### Console, Capture, and Traffic Filters

- QEMU serial consoles terminate at Unix sockets and are exposed through bounded WebSocket streams;
  optional allocated TCP listeners provide Telnet-compatible external access.
- VNC remains on an owned Unix socket; a WebSocket proxy feeds noVNC in the browser. Console endpoint
  discovery never exposes filesystem socket paths to clients.
- Capture workers stream pcap/pcapng bytes through HTTP with cancellation and optionally tee to a
  bounded artifact file. Wireshark can consume `curl ... | wireshark -k -i -` or a small extcap helper.
- Traffic filters compile supported fields to BPF. Workers observe selected topology ports, compute a
  normalized packet fingerprint, aggregate link/interface matches in short time windows, and publish
  highlighted paths with counts and timestamps. Ambiguous loops are represented as observed link sets,
  not an invented single route.

### API, Events, and MCP

- REST resources live under `/api/v1`; long operations return `202 Accepted` plus an `OperationTask`.
- Mutating requests accept `Idempotency-Key` and `If-Match` revision headers. Revision conflicts return
  `409`, stale preconditions return `412`, and resource exhaustion returns `429` or `503` with retry data.
- `/api/v1/events` is an ordered WebSocket stream with monotonically increasing sequence numbers and
  replay from `after=<sequence>`. Clients recover gaps with a resource snapshot.
- MCP uses Streamable HTTP at `/mcp`, validates `Origin`, and exposes typed tools backed by the same
  application commands. Binary capture data is represented by metadata plus API artifact/stream handles.

## Project Structure

### Documentation (this feature)

```text
specs/001-network-simulator-platform/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── openapi.yaml
│   ├── events.md
│   ├── mcp-tools.md
│   └── lab-export.schema.json
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
└── netlabd/
    └── main.go

internal/
├── api/
│   ├── http/
│   ├── mcp/
│   └── stream/
├── app/
│   ├── command/
│   ├── query/
│   ├── reconcile/
│   └── task/
├── domain/
├── runtime/
│   ├── capture/
│   ├── cgroup/
│   ├── console/
│   ├── docker/
│   ├── image/
│   ├── linuxnet/
│   └── qemu/
├── store/
│   └── sqlite/
└── support/

migrations/
templates/
├── qemu/
└── docker/

web/
├── src/
│   ├── api/
│   ├── components/
│   ├── features/
│   ├── router/
│   ├── stores/
│   └── views/
└── tests/

tests/
├── contract/
├── integration/
├── recovery/
└── e2e/

deploy/
├── systemd/
├── config/
└── scripts/
```

**Structure Decision**: Use one Go module and one Vue workspace. The production Go binary embeds the
built SPA and serves HTTP, WebSocket, and MCP endpoints from one process. Domain/application packages
remain independent of Gin, SQLite, QEMU, Docker, and host commands. Privileged behavior is restricted
to runtime adapters so it can be isolated in integration tests and audited consistently.

## Delivery Sequence

1. Establish domain IDs, revisions, state machines, SQLite migrations, outbox events, task semantics,
   and read-only topology UI.
2. Add Linux network ownership, bridges, namespaces, PC nodes, live link changes, NAT, and cleanup.
3. Add QEMU launch/adoption, overlays, QMP/QGA, cgroups, Telnet/VNC, port mapping, and cloud-init seeds.
4. Add Docker node lifecycle and template/image version management.
5. Add HTTP parity, MCP tools, event replay, idempotency, exports/imports, and redacted audit history.
6. Add live capture, Wireshark streaming, traffic-filter path observation, artifact retention, and quotas.
7. Complete restart/host-recovery workflows, failure injection, leak tests, usability validation, and
   acceptance-host packaging.

## Agent Context Update

The installed Spec Kit distribution does not provide an `update-agent-context` script or CLI command.
The equivalent project context is recorded in the repository-root `AGENTS.md`, covering the selected
stack, architecture boundaries, validation commands, and image/secret safety constraints for subsequent
planning and implementation agents.

## Complexity Tracking

No constitution violations require justification. The adapter count reflects the explicitly required
runtime types and privileged host integrations; no cluster, external queue, distributed database,
plugin system, or application authentication subsystem is introduced.
