# NetLab Validation Report

**Date**: 2026-07-24  
**Feature**: `001-network-simulator-platform`  
**Acceptance endpoint**: `http://10.72.1.7:8088`

## Result

The current build passes the complete repository test entry point and the privileged acceptance
scenarios run on the target host. The July 24 run exercised a ten-node namespace topology, live links,
PC/L2/L3/NAT network objects, four simultaneous Ubuntu QEMU guests, QGA commands, QMP NIC hot-plug,
Telnet/VNC endpoints, nftables port mappings, Wireshark-compatible pcap streaming, Traffic Filter
correlation, service and host restart recovery, and terminal owned-resource cleanup.

The result remains a **partial acceptance** only for image-dependent scenarios not covered by the
registered Ubuntu image: immutable Docker images, Ubuntu/VyOS/FancyWAN cloud-init boot, real guest SSH
or ZTP connectivity through a mapped port, and measured CPU throttling under load. No commercial image
was downloaded or committed.

## Phase 12 Current-Build Evidence

The official Ubuntu 24.04 cloud image was downloaded by the operator outside the repository, verified
against SHA-256 `ffe6203da54deeb6db5d2a98a83f9ec8e55f149d3f7ba622e1abe5fa966ee3d6`, and imported through the
image API. A local acceptance derivative with the Ubuntu `qemu-guest-agent` package installed was also
checksummed, imported, and bound to the Ubuntu template. Neither image nor any capture artifact was
copied into the workspace.

Four QEMU nodes using independent persistent qcow2 overlays ran simultaneously. Each node exposed QMP,
QGA, serial, and VNC sockets; executed a command through QGA; added and removed a virtio NIC through a
pre-created PCIe Root Port; created and removed an nftables port mapping; and started a live pcap stream.
Each node declared 2 vCPUs but a one-core CPU-time quota, 1024 MiB memory, eight interfaces, and 256
processes. The resource API reported live process metrics, and cgroup v2 contained `cpu.max` =
`100000 100000`, `memory.max` = `1073741824`, and `pids.max` = `256` with the correct QEMU PID.

The four-node `auto_restore` laboratory remained active across a full host reboot. The host and API
returned in approximately 14 seconds. The durable `system.recovery` task succeeded at 15/15 and
persisted a `recovered` checkpoint for all four QEMU nodes and each recovery participant. All four
overlays were reused and all four processes returned to their node cgroups. The isolated acceptance
laboratory was deleted after verification.

## Acceptance Host

| Item | Observed value |
|------|----------------|
| Operating system | Ubuntu 24.04.4 LTS |
| Kernel | `7.1.1-eve-ksm+` |
| QEMU | 8.2.2 |
| Docker | 29.5.2 |
| Hardware virtualization | `/dev/kvm` available |
| Control groups | cgroup v2 |
| Capacity | 60 CPUs, 125 GiB memory, 636 GiB free disk |
| Runtime tooling | `iproute2`, `bridge`, `nft`, `tcpdump`, `xorriso`, and `sqlite3` available |
| Service | `netlab.service` enabled and active on the current binary |
| Listener | `0.0.0.0:8088`; startup emits the unauthenticated trusted-network warning |

## Local Quality Gates

`make test-all` completed successfully on 2026-07-24. It ran:

- `go vet ./...`
- ESLint and Prettier checks; ESLint reports warnings and zero errors
- all Go unit, contract, integration, recovery, leak, and security packages
- Vue unit tests: 4 files and 6 tests passed
- production frontend and Go binary builds
- Playwright includes the shared-state, accessibility, automation, and lightweight create/attach/diagnose workflows

Executable HTTP route/schema parity, REST and MCP idempotency replay, different-payload conflict,
mutation audit, task cancellation, event reconnect/reset, recovery progress, namespace restoration,
multi-hop correlation, and 100-cycle ownership tests passed. The repository hygiene scan found no
proprietary images, packet captures, private keys, or embedded bootstrap credentials.

## Remote Acceptance Evidence

### July 24 Service-Restart and Capacity Convergence

- Installed the current binary and systemd unit without replacing the acceptance host's registered
  template/image manifest.
- Added `KillMode=process` and `DelegateSubgroup=manager`; corrected the cgroup manager so managed node
  cgroups live at `netlab.service/nodes` as a sibling of the control process subgroup rather than below
  it.
- Started an Ubuntu QEMU node and restarted `netlab.service`. The control PID changed from `633654` to
  `635820`, while the QEMU PID remained `611213`; QMP inode `4850004`, serial inode `4850006`, both TAP
  names, and the API-visible `running` state were unchanged.
- Deleted the acceptance laboratory and verified that the preserved QEMU process stopped and its runtime
  directory disappeared.
- Created a QEMU node with `InterfaceLimit=65`; admission moved it to `failed` with structured
  `resource_exhausted` diagnostics before creating a launch manifest, QEMU process, or TAP interface.
- Re-ran `make test-all`: all Go, contract, Vue unit, security, integration, recovery, leak, build, and
  Playwright tests passed. ESLint reported the existing 219 warnings and zero errors.
- Replaced non-atomic interface counting and `MAX(slot)+1` allocation with a serialized SQLite
  lowest-free-slot reservation. Unit/integration coverage verifies deleted-slot reuse, the 63/64
  boundary, concurrent additions, and persistence/TAP rollback after hot-add failure.
- On the acceptance host, a running Ubuntu QEMU node reached slots `0,1,2,3`, hot-removed slot `1`,
  then hot-added a replacement into slot `1` without stopping; QMP deletion now falls back to verified
  PCI absence when the host QEMU does not emit `DEVICE_DELETED`.
- REST, MCP, durable tasks, and reconciliation now share wrapped/pointer-safe structured problem
  normalization, including lifecycle phase, cleanup status, operator action, and retry delay metadata.

| Scenario | Result |
|----------|--------|
| Deployment | PASS: the July 24 binary and hardened systemd unit were installed; `/run/netns` is explicitly writable inside the sandbox |
| Recovery task | PASS: two consecutive service restarts completed a durable `system.recovery` task at `6/6` with no new reconciliation errors |
| Scale baseline | PASS: 10 namespace PC nodes restored to `running`; one live link restored to `connected` |
| Network objects | PASS: PC, L2 switch, L3 switch, and NAT bridge restored to `active`; all five final attachments reconciled to `active` |
| Direct link | PASS: IPv4 and IPv6 ping completed with zero packet loss between two PC nodes |
| PC object | PASS: the attached node reached the PC object static IPv4 address |
| L2 VLAN | PASS: two attached nodes in PVID 10 forwarded IPv4 traffic with zero packet loss |
| L3 routing | PASS: the attached node reached the L3 object over IPv4 and IPv6; forwarding diagnostics were available |
| NAT | PASS: the attached node reached the NAT gateway and an uplink address; exactly one owned masquerade rule existed after repeated restart |
| Wireshark capture | PASS: HTTP streaming returned pcap media type and capture headers; 708 bytes were streamed, metadata reported 936 bytes and 8 packets, and explicit stop converged to `cancelled` |
| Traffic Filter | PASS: live ICMP generated 16 correlated observations across the selected interface and link, with packet fingerprints, ingress direction, and `ambiguous=false` |
| Cleanup | PASS: laboratory deletion removed the lab, nodes, links, network objects, attachments, captures, nft rule, namespaces, interfaces, and cgroup matches; SQLite reported zero orphan interfaces |
| QEMU image | PASS: operator-imported official Ubuntu 24.04 image checksum was verified; an Ubuntu-package QGA derivative was registered without adding images to the repository |
| Four QEMU nodes | PASS: four guests using the same template version ran concurrently from independent per-node qcow2 overlays |
| QGA and QMP | PASS: commands completed through QGA on all four guests; virtio NIC add/remove completed through QMP using deterministic hot-plug buses |
| Consoles | PASS: each guest exposed reconnectable Telnet and VNC descriptors backed by serial and VNC Unix sockets |
| QEMU port mapping | PASS: valid dynamic host ports produced owned IPv4 DNAT and masquerade rules and were deleted through durable tasks |
| QEMU resources | PASS: normalized metrics were available and each QEMU PID was placed in a node cgroup with the declared CPU-time, memory, and process limits |
| Four-QEMU reboot | PASS: host/API returned in about 14 seconds; `system.recovery` succeeded at `15/15`; all four overlays and cgroups were restored |

The acceptance runs found and fixed deployment-level defects in namespace mount permissions, cgroup v2
delegation, QEMU device visibility, QGA cancellation, template image rebinding, per-node qcow2 overlays,
q35 PCIe hot-plug buses, `inet` DNAT address-family syntax, capture source resolution, nft diagnostics,
and mixed interface/link Traffic Filter scopes. Regression tests cover the corrected behavior.

One older, unrelated acceptance laboratory remains in `delete_failed` state from a previous run. Its
pre-existing retained capture files were not removed because the July 24 cleanup intentionally touched
only resources owned by the fresh acceptance laboratory.

## Quickstart Coverage

| Quickstart section | Coverage |
|--------------------|----------|
| 1. Host prerequisites | Verified on the acceptance host |
| 2. Build and install | Current binary and systemd unit deployed and restarted successfully |
| 3. Images and template versions | Official Ubuntu checksum, provenance, import, validation, and template binding passed; commercial images were not fetched |
| 4. Shared state and idempotency | REST, MCP, Vue, executable contract, and Playwright coverage passed |
| 5. Reference topology | Ten lightweight nodes and four simultaneous Ubuntu QEMU nodes passed remotely |
| 6. Cloud-init, console, guest commands | Real Telnet/VNC endpoints and QGA commands passed; multi-platform cloud-init boot remains deferred |
| 7. Live rewiring and NIC hot-plug | Namespace live links and real QMP virtio NIC add/remove passed |
| 8. Port mapping and CPU quota | Real QEMU nftables mappings and cgroup enforcement passed; guest connectivity and load-based throttling measurement remain deferred |
| 9. Capture and Traffic Filter | Privileged pcap streaming and live multi-point correlation passed |
| 10. Service restart adoption | Current build restored nodes, link, objects, attachments, and NAT ownership twice |
| 11. Full-host recovery | Current build restored four active QEMU guests and completed durable recovery at 15/15 after a full host reboot |
| 12. Export/import/redaction | Contract, round-trip, and secret-redaction tests passed |
| 13. Failure and leak validation | Recovery, quarantine, limits, cleanup, and 100-cycle suites passed; image-dependent privileged cycles remain deferred |
| 14. Contract and UI validation | OpenAPI route/schema parity, typed client, Vue tests, build, event synchronization, and Playwright passed |

## Deferred Operator Validation

The remaining image-dependent acceptance requires operator-approved immutable Docker images and legal
VyOS/FancyWAN images. The next privileged pass should cover Ubuntu/VyOS/FancyWAN cloud-init, real guest
SSH or ZTP connectivity through host port mappings, Docker runtime metrics, and load-based CPU-time
throttling measurement. No unofficial commercial image source should be used.

## Runtime Ownership Discovery — July 24, 2026

- Added startup and two-second periodic inventory before lifecycle reconciliation for QEMU runtime
  directories, validated QEMU PIDs and Unix sockets, Docker containers carrying NetLab labels, Linux
  namespaces and explicitly aliased TAP/veth/bridge devices, NetLab-commented nftables rules, delegated
  node cgroups, capture metadata/workers, active in-process console proxies, and marked helper processes.
- Unknown QEMU directories are moved only within the configured QEMU runtime root into a restrictive
  quarantine directory and audited. Unknown host networking, process, cgroup, capture, and console
  objects are recorded and audited without mutation. Active records that disappear transition to
  `missing_validation_required`; discovery never treats absence alone as abandonment or deletion proof.
- Hot-added TAP creation now persists ownership before QMP device insertion. Ownership, TAP, and
  interface rows are compensated together on persistence or QMP failure, and successful hot removal
  deletes all three layers.
- QEMU, capture, DHCP helper, Docker, Linux link, and console creation paths now emit explicit ownership
  evidence. Repeated discovery does not promote an unknown observation into an owned active record.
- `make test-all` passed: Go vet and all Go suites, security/integration/recovery and 100-cycle leak
  tests, Vue unit tests, formatting, production builds, and Playwright 4/4. ESLint remains at the
  existing baseline of 219 warnings and zero errors.

## Lifecycle Failure Matrix — July 24, 2026

- Node reconciliation now durably records `provisioning`, `starting`, `running`, `stopping`, `stopped`,
  and `failed` transitions. Inspection, provisioning, startup, resource application, and stopping use
  explicit phase deadlines with retryable structured diagnostics.
- QEMU provisioning is separated from launch, making invalid image/overlay failures terminal in the
  provisioning phase before a PID or launch manifest is created. Early QEMU exit and QMP readiness
  timeout errors have distinct codes, cleanup descriptions, retry delays, and operator actions.
- QEMU launch failure tests verify that `launch.json` and QMP, QGA, serial, and VNC sockets do not
  remain. Durable overlays and bounded logs remain available for diagnosis.
- Single-node deletion now records `deleting`, removes the owned runtime, cgroup and ownership-backed
  hot-added TAPs before deleting the SQLite node row, and records `failed` with the row retained when
  runtime, resource, interface, or persistence cleanup fails.
- Added `acceptance/lifecycle-failures.sh`. The invalid-image, early-exit, QMP-timeout, start-timeout,
  stop-failure, deletion-failure, and successful terminal-cleanup matrix passed locally and as isolated
  test binaries on the acceptance host. The active service, database, image bindings, and laboratories
  were not modified; temporary test binaries were removed afterward.
- `make test-all` passed after the lifecycle changes: Go vet, all Go and contract suites,
  security/integration/recovery and 100-cycle leak tests, Vue tests, formatting, production builds, and
  Playwright 4/4. ESLint remains at 219 warnings and zero errors.

## Controlled QEMU Hot-Add Failure — July 24, 2026

- Added `acceptance/qemu-hotadd-rollback.sh`, which fills the unused slot-1 QEMU PCIe Root Port with
  temporary QMP devices so the application completes TAP creation and `netdev_add` before the real
  `device_add` fails because the bus has no free slot/function.
- The failed durable task reported `interface_hot_add_failed`, phase `hot_add`, and confirmed that the
  interface row and TAP were removed. Direct checks also proved the TAP, runtime ownership rows, and
  QMP netdev were absent.
- Restarting only the acceptance node cleared the injected QMP devices. Retrying the same API operation
  succeeded and reserved slot 1, proving transactional lowest-free-slot reuse after compensation.
- The first cleanup pass exposed a successful-hot-add TAP leak during laboratory deletion. The deletion
  reconciler now removes deterministic hot-add TAP names before finalizing the laboratory, with a unit
  regression test. The repeated remote acceptance passed and left zero matching laboratories or TAPs.
- The final remote result was `PASS`, the service remained active, and no operator image or template
  registration was modified.
- The post-fix `make test-all` validation passed Go vet, all Go/unit/contract/security/integration/
  recovery/leak suites, Vue unit tests, production builds, and all four Playwright scenarios. ESLint
  reported the existing 219 warnings and zero errors.
- Reconciler terminal errors for node lifecycle, topology, data-plane attachments, network objects,
  captures, mutation observation, port-mapping recovery, and laboratory deletion now pass through the
  wrapped/pointer-safe normalizer with resource, phase, cleanup, operator action, and retry-delay
  context. Focused metadata regression tests pass; the broader T219 cross-surface matrix remains open.

## July 24 Durable Mutation Migration

- Added runner-backed `laboratory.export`, `laboratory.import`, and `laboratory.duplicate` handlers with
  persisted checkpoints, fixed target resource identifiers, request-fingerprint replay protection,
  cancellation propagation, and restart-safe import/export replay.
- Added runner-backed `network_object.create` and `network_object.delete` handlers with fixed object
  identifiers, persisted request state, create cancellation cleanup, delete cancellation restoration,
  and restart adoption of the original task identifier.
- REST and MCP now return matching task envelopes for laboratory automation and network-object
  mutations. The mutation middleware skips shadow `http.mutation` creation for these routes while
  retaining audit association with the real durable task.
- `go test ./...`, Vue unit tests, and the production Vue build pass. T222 remains open because
  laboratory deletion, capture lifecycle, Traffic Filter lifecycle, and remaining generic mutation
  middleware paths still require migration before the shadow-task pattern can be removed.

### Capture and Traffic Filter Tasks

- Added runner-backed `capture.start`, `capture.stop`, `traffic_filter.start`, and
  `traffic_filter.stop` handlers with fixed resource identifiers, persisted request fingerprints,
  restart replay, bounded capture-stop convergence, and cancellation cleanup or restart compensation.
- Capture metadata persistence now retains the normalized start request so an interrupted durable task
  can resume the same capture identifier after control-service restart.
- REST, MCP, and Vue now use matching task envelopes for capture and Traffic Filter mutations while
  preserving live capture stream URLs and Wireshark media metadata.
- Focused idempotency, stop, restart-recovery, middleware-classification, and REST/MCP parity tests pass.
  `go test ./...`, Vue unit tests, and the production Vue build also pass after this migration.
- T222 remains open for laboratory deletion and the remaining generic `http.mutation` paths before the
  shadow-task mechanism and its legacy mutation observer can be removed completely.

### Durable Mutation Completion

- Added runner-backed `laboratory.delete` with persisted cleanup checkpoints, restart convergence,
  idempotent REST/MCP envelopes, structured `delete_failed` propagation, and an explicit cancellation
  boundary that permits queued cancellation but rejects unsafe cancellation after cleanup is committed.
- Removed generic `http.mutation` task creation and the legacy mutation-task reconciler. Synchronous
  mutations retain request replay and audit behavior without manufacturing operation tasks, while all
  long-running mutation classes named by T222 use real runner handlers.
- Fixed nested idempotency handling for `POST /api/v1/labs` by allowing the route's existing
  idempotency implementation to own that request, eliminating the re-entrant lock that blocked browser
  automation.
- Updated the lightweight-node Playwright mock for the asynchronous network-object task envelope.
- `make test-all` passes, including Go vet, all Go suites, 100-cycle leak coverage, Vue unit tests,
  production builds, formatting checks, and all four Playwright scenarios. ESLint reports the existing
  219 warnings and zero errors.
- T222 is complete. No `http.mutation`, `MutationTaskReconciler`, or `mutation-tasks` references remain
  in the implementation.

## Mixed Runtime Service-Restart Adoption — July 24, 2026

- Added `acceptance/t225-service-restart.sh`, which creates one auto-restore laboratory containing an
  Ubuntu QEMU guest, a BusyBox Docker node, a namespace PC, a live QEMU-to-Docker link, a PC network
  object attachment, an active capture, a Telnet WebSocket, an injected recoverable durable task, and
  an event-stream replay cursor before replacing the `netlab.service` control PID.
- Docker start now preserves JSON-decoded command arrays, uses a BusyBox-compatible bounded default,
  rebuilds endpoints for an already-running owned container, and compensates a newly started or
  created container when endpoint setup fails. The acceptance host required `CAP_SYS_PTRACE` in
  addition to `CAP_SYS_ADMIN` for access to Docker process network namespace handles.
- Named network namespaces are created and deleted through bounded `systemd-run` oneshot units in PID
  1's mount namespace. This preserves namespace handles across control-service mount namespace
  replacement while retaining the service's `ProtectSystem` and `PrivateTmp` sandboxing.
- Recovery checkpoints now persist QEMU PIDs, complete Docker container IDs, namespace names, runtime
  details, reconnect-required capture/console ownership outcomes, and the original durable task ID.
  The event WebSocket replayed only events newer than the pre-restart cursor.
- The acceptance run proved unchanged QEMU PID, Docker ID, namespace name, live link and attachment
  names; capture termination as `failed:service_restart`; Console ownership as
  `missing_validation_required`; successful task replay; and shared node state after control PID
  replacement.
- QEMU ownership discovery now reads the runtime's `launch.json` with legacy `manifest.json` fallback,
  preventing a live runtime directory from being quarantined. Cgroup removal is idempotent when the
  node cgroup is already absent.
- Laboratory deletion now retries `delete_failed` rows to terminal removal and transactionally deletes
  owned node, interface, link, attachment, network-object, capture, traffic-filter, and laboratory
  ownership records. Capture ownership cleanup also resolves the laboratory through durable task
  input after in-memory capture purge removes the capture row.
- The acceptance host returned `PASS T225`; its temporary laboratory, QEMU process/runtime directory,
  Docker container, namespace, link/attachment devices, cgroups, and ownership rows were absent after
  cleanup. Existing image and template bindings were not changed, and `netlab.service` remained active
  on build `t225-20260724-r8`.
- `make test-all` passed after the final changes: Go vet, all Go/unit/contract/security/integration/
  recovery/leak suites, Vue tests, formatting checks, production builds, and Playwright 4/4. ESLint
  retained the existing baseline of 219 warnings and zero errors.

## Phase 17 Convergence Evidence — July 24, 2026

- Structured failure parity now normalizes terminal errors at node, data-plane, ownership, deletion,
  network-object, and live-mutation boundaries while preserving wrapped runtime/task identity. Table-driven
  reconcile tests plus HTTP, MCP, and task-runner suites verify phase, retryability, cleanup guidance,
  operator action, retry delay, and retryable HTTP 503 mapping.
- PC DHCPv4/DHCPv6 now runs as stable systemd-supervised foreground helpers with restart adoption,
  bounded acquisition, cancellation compensation, per-node lease/PID state, SLAAC timeout markers,
  DNS/route diagnostics, and cleanup. The privileged integration scenario builds isolated namespaces,
  DHCPv4/DHCPv6/RA services, L2 bridging, L3 forwarding, and observed nftables NAT translation.
- Official `busybox:1.36.1` and `ubuntu:24.04` OCI images were pulled on the acceptance host and recorded
  by digest through the implemented `oci_registry` importer. No proprietary or unofficial image was fetched.
- `acceptance/operator-image-acceptance.sh` passed one laboratory with 10 running nodes: four QEMU,
  two digest-pinned Docker, and four namespace PCs. The four QEMU nodes exercised the Ubuntu, FancyWAN,
  and VyOS template cloud-init/seed paths with the legal operator-supplied Ubuntu QGA image, real Telnet
  endpoints, QGA commands, live linking/capture, and shared-state service restart.
- A real ZTP-style guest service was reached through a host nftables port mapping (`ztp_port=55079`).
  Under two concurrent guest CPU workers, the 2-vCPU/one-core quota reported a 7,204,278 microsecond
  CPU-time increase across the bounded measurement interval, consistent with one-core throttling.
- A supervised full host reboot restored all 10 marker-recorded nodes. The durable `host_restart`
  recovery task contained 16 recovered resource outcomes; terminal laboratory deletion completed with
  zero remaining ownership rows. Acceptance boot ID: `ba2c3dbe-25ab-45f0-91a1-773609fb0c78`.
- The isolated privileged DHCPv4/DHCPv6/SLAAC, L2 bridge, IPv4 L3 forwarding, IPv6 adjacency, and
  observed NAT translation test passed on `10.72.1.7`. `make test-all` then passed Go vet, all Go,
  contract, security, integration, recovery, leak, Vue/Vitest, production build, and Playwright 4/4.
