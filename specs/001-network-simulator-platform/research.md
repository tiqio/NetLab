# Research: NetLab Network Simulator Platform

**Date**: 2026-07-23

## Target Host Baseline

**Decision**: Treat the designated acceptance server as an x86_64 Ubuntu 24.04 single-host target with
KVM, QEMU 8.2.2, Docker 29.5.2, cgroup v2, iproute2, nftables, and packet-capture tools. Install the
NetLab binary, frontend assets, `xorriso`, and any missing capture helper without modifying EVE-NG
application code.

**Rationale**: Read-only inspection confirmed 60 logical CPUs, 125 GiB RAM, available `/dev/kvm`, and
ample storage, which exceeds the clarified 10-node/4-QEMU acceptance baseline. Reusing the host avoids
premature portability work while keeping the product independent of EVE-NG internals.

**Alternatives considered**:

- Develop only in nested local virtualization: rejected because QMP hot-plug, KVM, netns, cgroups, and
  packet capture require realistic privileged validation.
- Extend EVE-NG directly: rejected because shared-state and automation semantics conflict with the core
  product goals and would inherit unwanted templates and account behavior.

## Go and Web Stack

**Decision**: Use Go 1.26.x, Gin 1.12.x, Vue 3 with TypeScript and Vite, and build the SPA into the
production Go binary.

**Rationale**: Go provides direct Linux system integration, strong concurrency primitives, a small
deployment footprint, and the user-mandated Gin stack. Vue 3 provides a current SPA platform; embedding
the compiled application preserves the single-service deployment model.

**Alternatives considered**:

- Separate frontend server: rejected because it adds deployment and version-skew risk without product
  value on a single host.
- Server-rendered UI: rejected because topology editing, live events, consoles, and captures benefit
  from a stateful SPA.

## QEMU Process and QMP Control

**Decision**: Launch QEMU directly, one process per node, and use
`github.com/digitalocean/go-qemu/qmp.SocketMonitor` over a node-owned Unix socket. Pin the dependency by
commit and wrap it behind a local adapter. Use raw QMP commands when generated helpers do not cover a
command required by the installed QEMU version.

**Rationale**: The required library supports socket monitors, arbitrary command execution, and event
streams. QMP provides machine-readable capabilities, device add/remove, state queries, and asynchronous
events. Direct process ownership is simpler than introducing libvirt and makes cgroup placement,
runtime directories, adoption, and cleanup explicit.

**Alternatives considered**:

- Libvirt: rejected because it adds a second desired-state system and an unnecessary daemon boundary.
- QEMU human monitor: rejected for automation because its text output is not a stable machine contract.
- Shelling out to `qmp-shell`: rejected because it weakens typed errors, event handling, and testing.

**Sources**:

- `https://github.com/digitalocean/go-qemu`
- `https://qemu.readthedocs.io/en/v8.2.10/interop/qmp-spec.html`
- `https://qemu.readthedocs.io/en/v8.2.10/interop/qemu-qmp-ref.html`

## QEMU Guest Agent

**Decision**: Expose guest commands through QEMU Guest Agent `guest-exec` and `guest-exec-status`, with
template capability flags, strict timeout/cancellation, output decoding, configurable output bounds,
and redacted audit records.

**Rationale**: The guest agent exposes structured process execution and status polling. Treating it as
an optional template capability prevents unsupported images from receiving impossible operations.

**Alternatives considered**:

- SSH-only guest execution: rejected because credentials and network readiness would become mandatory.
- Serial-console scripting: rejected because prompt parsing is appliance-specific and fragile.

**Source**: `https://qemu.readthedocs.io/en/v8.2.10/interop/qemu-ga-ref.html`

## QEMU Hot-Plug and Live Rewiring

**Decision**: Distinguish cable changes from interface-count changes. Rewire an existing interface by
changing the host tap/veth bridge membership. For new QEMU interfaces, run `netdev_add`, `device_add`,
wait for QMP confirmation, and attach the tap. For removal, detach the link, run `device_del`, wait for
`DEVICE_DELETED`, then `netdev_del` and remove the tap.

**Rationale**: Most topology edits do not require guest PCI changes. Separating the two operations makes
routine rewiring fast and lowers partial-failure risk while preserving the explicit hot-plug capability.

**Alternatives considered**:

- Restart QEMU for every cable change: rejected by the live-topology requirement.
- Always hot-remove and re-add NICs: rejected because it causes avoidable guest churn and interface-name
  changes.

## CPU-Time Quotas

**Decision**: Place each QEMU process in a dedicated cgroup v2 subtree and express CPU-time limits with
`cpu.max`; configure guest vCPU count independently with QEMU `-smp`. Map Docker limits to equivalent
Engine CPU settings and record normalized quota values in the domain model.

**Rationale**: cgroup v2 directly supports quota/period controls for all threads in a QEMU process and
matches the requirement that two virtual CPUs may share one host-core time budget.

**Alternatives considered**:

- CPU affinity only: rejected because affinity limits placement, not consumed CPU time.
- QEMU throttling options alone: rejected because host-enforced cgroups provide a consistent ownership
  and cleanup boundary.

**Source**: `https://docs.kernel.org/admin-guide/cgroup-v2.html`

## Docker Runtime

**Decision**: Use the official Docker Engine Go SDK with API version negotiation. Create containers with
network mode `none`, resource limits, and immutable ownership labels; move an owned veth peer into the
container network namespace and configure it through the Linux networking adapter.

**Rationale**: Engine API negotiation tolerates target-host daemon updates. Bypassing Docker-managed
networks gives QEMU, containers, and namespace nodes one uniform topology model and enables live cable
changes without replacing containers.

**Alternatives considered**:

- Docker bridge networks per link: rejected because mixing Docker and QEMU endpoints becomes indirect
  and cleanup semantics differ by node type.
- Docker CLI subprocesses: rejected because structured errors, events, and API negotiation are weaker.

**Sources**:

- `https://docs.docker.com/reference/api/engine/`
- `https://docs.docker.com/reference/api/engine/sdk/`

## Linux Networking and NAT

**Decision**: Manage interfaces, addresses, routes, bridges, and namespaces through netlink APIs, with a
small validated command runner only for capabilities not exposed reliably by the selected library.
Use one NetLab-owned nftables table for NAT and host-port mappings. Name every host object from a stable
resource ID and persist the full ownership manifest.

**Rationale**: A shared Linux networking model supports QEMU tap devices, container veth devices,
lightweight PCs, layer-2 switches, layer-3 switches, and NAT without runtime-specific topology rules.
Dedicated ownership prevents deletion of unrelated host networking.

**Alternatives considered**:

- Open vSwitch: rejected because Linux bridges satisfy the requested scope and reduce dependencies.
- iptables-only rules: rejected because nftables is the current host-native rule model and supports an
  isolated named table.

## Bootstrap Seed Images

**Decision**: Build per-node NoCloud-compatible seed ISOs containing `user-data`, `meta-data`, and
optional `network-config`, using template-specific renderers and `xorriso`. Attach the generated ISO as
read-only removable media and delete it according to the node/bootstrap lifecycle.

**Rationale**: NoCloud supports local seed media and makes the bootstrap artifact explicit and testable.
Template flags limit the workflow to image versions known to consume the supplied data.

**Alternatives considered**:

- HTTP metadata service: deferred because local ISO is deterministic and works before guest networking.
- Mutating the base qcow2: rejected because it breaks image immutability and repeatability.

**Source**: `https://cloudinit.readthedocs.io/en/latest/reference/datasources/nocloud.html`

## SQLite Consistency Model

**Decision**: Use SQLite WAL mode with foreign keys enabled, a busy timeout, short write transactions,
and an application-level serialized write path. Store an outbox event in the same transaction as each
durable mutation. Use resource revisions for optimistic concurrency and unique idempotency records for
retry safety.

**Rationale**: WAL allows readers to continue during writes, while one serialized writer matches the
single-control-process architecture and avoids avoidable lock contention. The transactional outbox
prevents durable state from diverging from client event delivery.

**Alternatives considered**:

- PostgreSQL: rejected as unnecessary operational complexity for a single host.
- In-memory event bus without outbox: rejected because clients could miss state changes after crashes.
- Event sourcing as the primary store: rejected because it adds reconstruction complexity beyond the
  required audit and task history.

**Source**: `https://sqlite.org/wal.html`

## HTTP, WebSocket, and MCP Contracts

**Decision**: Publish REST under `/api/v1`, ordered events and console streams over WebSocket, capture
bytes over cancellable HTTP streams, and MCP Streamable HTTP at `/mcp`. MCP tools call the same
application commands as REST and return typed resource/task envelopes. Capture tools return metadata,
bounded summaries, and opaque API handles rather than binary payloads.

**Rationale**: One application layer guarantees UI/API/MCP parity. Streamable HTTP is the current MCP
transport and supports resumability patterns; WebSocket remains appropriate for high-frequency UI and
console duplex streams.

**Alternatives considered**:

- Separate MCP service: rejected because it risks state and behavior divergence.
- Returning base64 captures through MCP: rejected because capture size is unbounded relative to model
  context and transport limits.
- GraphQL: rejected because long-running tasks, byte streams, and explicit lifecycle resources fit REST
  and event streams more directly.

**Sources**:

- `https://modelcontextprotocol.io/specification/2025-06-18/basic/transports`
- `https://modelcontextprotocol.io/specification/2025-06-18/server/tools`
- `https://github.com/modelcontextprotocol/go-sdk`

## Capture and Traffic-Path Observation

**Decision**: Use supervised packet-capture workers with compiled BPF filters. Stream pcap/pcapng bytes
and optionally tee to bounded artifacts. For traffic-path filters, observe topology-facing interfaces,
compute a normalized packet fingerprint, correlate observations in a short configurable time window,
and publish link/interface match sets with counts and timestamps.

**Rationale**: This design works with ordinary Linux capture tooling, supports Wireshark streaming, and
can explain observed traversal without requiring an eBPF-only kernel path. It also handles loops by
showing every observed link rather than claiming a single route.

**Alternatives considered**:

- One capture per entire host: rejected because unrelated traffic and namespace boundaries weaken
  attribution.
- Mandatory eBPF: deferred as a later optimization because it raises kernel/toolchain requirements.
- Inferring paths from topology alone: rejected because forwarding policy and guest behavior may differ
  from intended topology.

## Capture Limits and Retention

**Decision**: Default to 4 concurrent captures, 15 minutes and 256 MiB per capture, 24-hour artifact
retention, and a 10 GiB global capture/export artifact ceiling. Use oldest-expired-first cleanup and
reject new retained captures when safe reclamation cannot restore headroom. All values are configurable.

**Rationale**: The defaults make accidental unbounded capture impossible while supporting ordinary
debugging sessions. Explicit truncation and quota errors satisfy API and MCP observability requirements.

**Alternatives considered**:

- Unlimited streaming with no artifacts: rejected because it prevents later download and risks slow
  consumer backpressure.
- Permanent retention: rejected because captures may contain sensitive traffic and exhaust disk.

## Image Ingestion and Exports

**Decision**: Support streamed operator upload and server-local import into a content-addressed image
store. Compute SHA-256 while staging, safely unpack supported archives, validate qcow2 metadata, and
atomically publish only after provenance/license fields are recorded. Docker references resolve to an
immutable digest. Lab exports use a versioned JSON bundle containing topology, pinned template/image
references, and redacted non-secret configuration only.

**Rationale**: Content addressing prevents accidental duplication and version drift. Excluding images,
secrets, bootstrap-sensitive data, and captures preserves licensing and privacy boundaries.

**Alternatives considered**:

- Automatically downloading commercial images: rejected by the constitution and licensing risk.
- Embedding qcow2 files in lab exports: rejected because exports become huge and may redistribute
  proprietary software.

## Observability

**Decision**: Emit structured JSON logs with resource/task correlation IDs, persist redacted audit
events, expose health/readiness and OpenMetrics-compatible operational counters, and surface reconciler
drift in both API status and metrics. Distributed tracing is not required for the first single-process
release.

**Rationale**: Logs, audit, health, metrics, and drift status cover operational diagnosis without adding
an external tracing backend.

**Alternatives considered**:

- Full distributed tracing: deferred until there are multiple services or demonstrated debugging need.

