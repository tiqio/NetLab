# Feature Specification: Network Path Recovery and Validation

**Feature Branch**: `011-network-path-recovery`

**Created**: 2026-08-11

**Status**: Draft

**Input**: User description: "修复复杂组件矩阵实验室中尚未完成的 BusyBox 服务网接入、VyOS 业务路由、厂商设备管理与数据网络、VLAN/Trunk、完整 IPv6 路径，以及持续有效流量与流量统计；同时修复服务重启后轻量网络对象命名空间失效、链路 pending/disconnecting 和运行状态失真的根因。"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Recover a Runnable Topology After Restart (Priority: P1)

As a laboratory operator, I can restart the NetLab service or host and recover every owned lightweight network object and connection into an honest, usable state without manually rebuilding the laboratory.

**Why this priority**: Namespace-backed PCs, L2 switches and L3 switches currently become unusable after restart while some resources still appear active. All end-to-end routing, VLAN and traffic validation depends on trustworthy recovery.

**Independent Test**: Build a laboratory containing namespace-backed PCs, L2 and L3 objects, a non-namespace bridge, Docker nodes and QEMU nodes; restart the service and verify that all valid resources return to their pre-restart connectivity, while any unrecoverable resource reports a specific failure and can be deleted cleanly.

**Acceptance Scenarios**:

1. **Given** an active laboratory with owned namespace-backed objects and connected links, **When** the service restarts, **Then** each resource is adopted or safely recreated from durable desired state and returns to an accurate active or connected state.
2. **Given** a stale or invalid namespace reference, **When** startup reconciliation runs, **Then** the affected resource is not reported active, its failure identifies the affected resource and recovery phase, and unrelated resources continue reconciling.
3. **Given** a failed or partially created connection involving a plain bridge, **When** the operator deletes it, **Then** cleanup completes without treating the bridge as a namespace-backed endpoint and no owned runtime resource remains.
4. **Given** multiple clients retry the same recovery-sensitive mutation, **When** reconciliation and the mutation overlap, **Then** the final desired state is applied once and all clients converge on the same revision and actual state.

---

### User Story 2 - Route Real Dual-Stack Traffic Through VyOS (Priority: P1)

As a network tester, I can use the existing Ubuntu QEMU and VyOS nodes as a dual-stack routed path between the point-to-point transit network and the core, DMZ and management networks.

**Why this priority**: The point-to-point link currently works, but routed traffic stops before reaching real endpoints, so the laboratory cannot validate network behavior beyond a single link.

**Independent Test**: From Ubuntu QEMU, reach the VyOS transit address and at least one IPv4 and IPv6 endpoint in each configured downstream network, then repeat after node stop/start and service restart.

**Acceptance Scenarios**:

1. **Given** Ubuntu and VyOS are running, **When** Ubuntu sends IPv4 and IPv6 traffic over the transit link, **Then** both VyOS transit addresses respond without packet loss beyond the allowed acceptance threshold.
2. **Given** VyOS has downstream LAN and management interfaces, **When** Ubuntu sends traffic to downstream clients, **Then** VyOS forwards the traffic and return routes deliver replies to Ubuntu.
3. **Given** the configured dual-stack routes are working, **When** VyOS or Ubuntu is stopped and started, **Then** the same interface addressing and route behavior returns without manual reconfiguration.

---

### User Story 3 - Exercise VLAN Access and Trunk Behavior (Priority: P2)

As a network tester, I can place access ports into distinct VLANs and carry multiple tagged VLANs over a trunk, with the displayed configuration matching the actual forwarding behavior.

**Why this priority**: PVID values alone do not prove VLAN isolation or tagged forwarding. A real trunk is required to validate switch behavior and topology presentation.

**Independent Test**: Configure VLAN 10 and VLAN 20 access endpoints plus one trunk carrying both VLANs; verify same-VLAN reachability, cross-VLAN isolation without routing, routed reachability when enabled, and persistence after restart.

**Acceptance Scenarios**:

1. **Given** access ports assigned to VLAN 10 and VLAN 20, **When** endpoints exchange untagged traffic, **Then** traffic remains in the assigned VLAN and cannot cross VLAN boundaries without a routed path.
2. **Given** a trunk configured for VLANs 10 and 20, **When** tagged traffic traverses it, **Then** both VLANs pass independently and unapproved VLANs do not pass.
3. **Given** the UI or control plane reports PVID and tagged VLAN membership, **When** runtime diagnostics are inspected, **Then** the actual forwarding membership matches the authoritative configuration.

---

### User Story 4 - Validate Vendor Device Management and Data Paths (Priority: P2)

As an appliance tester, I can connect FancyWAN, FortiGate, Ruijie Router and Ruijie Switch to a reachable management network and meaningful LAN/WAN or client-facing data networks instead of only testing pairwise links.

**Why this priority**: Running appliances and connected virtual cables do not demonstrate management access, forwarding, switching or client behavior.

**Independent Test**: Reach each appliance management address from the management PC and prove at least one appliance-routed or appliance-switched client path per vendor pair without relying on host-side shortcuts.

**Acceptance Scenarios**:

1. **Given** a vendor appliance is running and attached to the management network, **When** the management PC probes its declared management address, **Then** the appliance is reachable or reports a device-specific configuration prerequisite rather than a false connected result.
2. **Given** FancyWAN and FortiGate have management, LAN and WAN roles, **When** traffic is sent between their attached test endpoints, **Then** the intended path traverses the appliances and can be observed on the corresponding connections.
3. **Given** Ruijie Switch has an attached PC and Ruijie Router has a routed-side attachment, **When** the PC sends same-subnet and routed traffic, **Then** switching and routing behavior can be independently verified.

---

### User Story 5 - Generate and Observe Stable Test Traffic (Priority: P3)

As a UI and traffic-analysis tester, I can run continuous ICMP, HTTP and DNS traffic over working IPv4 and IPv6 paths and see non-zero, increasing Traffic Filter statistics and topology highlights for matching traffic.

**Why this priority**: A traffic process that only sends to unreachable targets does not validate packet matching, counters, byte totals or visual highlighting.

**Independent Test**: Start a traffic generator against reachable endpoints, create filters for ICMP, HTTP and DNS, and verify packet counts, byte counts, traffic fingerprints and topology highlights while traffic runs and after it stops.

**Acceptance Scenarios**:

1. **Given** reachable IPv4 and IPv6 endpoints, **When** the generator runs, **Then** successful ICMP, HTTP and DNS exchanges occur repeatedly at a documented interval.
2. **Given** a matching Traffic Filter, **When** successful traffic crosses the selected path, **Then** matching packet and byte counters become non-zero and continue increasing.
3. **Given** matching traffic is active, **When** the topology is viewed, **Then** only the matching path is highlighted and its fingerprint describes the observed traffic.
4. **Given** the generator or destination fails, **When** successful exchanges stop, **Then** the UI distinguishes generator activity from successful matched traffic and does not imply end-to-end success.

### Edge Cases

- A service restart occurs while a namespace-backed object or link is being created or deleted.
- A durable namespace owner exists but its namespace reference is stale, points to the wrong owner, or cannot be safely adopted.
- One endpoint recovers while the opposite endpoint remains failed or stopped.
- A plain Linux bridge participates in cleanup even though it has no network namespace.
- A VLAN trunk is configured with an empty allowed list, duplicate VLANs, VLAN 1, or a PVID also present as tagged membership.
- IPv4 forwarding succeeds while IPv6 forwarding is disabled or lacks a return route.
- A vendor appliance is booted but its operating system has not completed initialization or lacks configured management addressing.
- A traffic generator process is alive while every application exchange fails.
- Concurrent retries use stale laboratory or resource revisions during startup reconciliation.
- Deletion is requested for the known connection stuck in `disconnecting` or another partially created connection.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST maintain authoritative desired and actual states for namespace-backed PCs, L2 switches, L3 switches, plain bridges, node attachments, object links and unified connections.
- **FR-002**: Legal recovery transitions MUST include pending or failed resources moving to active or connected after successful reconciliation, and active or connected resources moving to failed when their required runtime backing is absent or invalid.
- **FR-003**: Startup reconciliation MUST verify ownership and usability of every persisted namespace-backed runtime resource before reporting it active.
- **FR-004**: Startup reconciliation MUST safely adopt an owned usable runtime resource or recreate it from durable desired state when adoption is impossible and recreation is safe.
- **FR-005**: Recovery failure of one resource MUST NOT prevent unrelated laboratories or resources from completing reconciliation.
- **FR-006**: Every recovery failure MUST expose the affected resource, lifecycle phase, cleanup state, retryability and an operator action through the UI and all applicable control-plane surfaces.
- **FR-007**: Deleting a failed, pending or disconnecting connection MUST clean each endpoint according to its actual runtime ownership model and MUST NOT assume that a plain bridge owns a namespace.
- **FR-008**: Successful deletion MUST remove all owned interfaces, bridge memberships, namespace references, routes, rules, sockets and durable connection records associated with the deleted resource.
- **FR-009**: The system MUST make repeated reconciliation and deletion requests idempotent and MUST reject stale concurrent mutations with revision information that allows clients to refresh and retry.
- **FR-010**: Long-running recovery, creation and deletion operations MUST expose durable progress, timeout, cancellation and partial-failure results without losing the final authoritative state.
- **FR-011**: UI, HTTP API and applicable MCP operations MUST report equivalent desired state, actual state, revisions, structured failures and final recovery outcomes.
- **FR-012**: The acceptance laboratory MUST connect the BusyBox client to a service network containing a reachable gateway and at least one reachable peer.
- **FR-013**: BusyBox MUST have working IPv4 and IPv6 addressing, default routes and bidirectional reachability to the service router and traffic generator.
- **FR-014**: Ubuntu QEMU and VyOS MUST retain a dual-stack point-to-point transit network after node restart, service restart and host recovery according to the laboratory recovery policy.
- **FR-015**: VyOS MUST provide active downstream interfaces and bidirectional IPv4 and IPv6 forwarding between the transit network and the declared core, DMZ and management test networks.
- **FR-016**: Every routed endpoint MUST have a valid forward path and return path; a locally reachable router interface alone MUST NOT satisfy routed-path acceptance.
- **FR-017**: L3 network objects MUST apply declared IPv4 and IPv6 forwarding and route state, and diagnostics MUST distinguish requested forwarding from observed forwarding.
- **FR-018**: L2 network objects MUST support access-port PVID membership and explicit tagged VLAN membership without silently falling back to VLAN 1.
- **FR-019**: The acceptance topology MUST include VLAN 10 and VLAN 20 access paths plus at least one trunk carrying both tagged VLANs.
- **FR-020**: VLAN configuration shown to users MUST match observed forwarding membership after creation, restart and recovery.
- **FR-021**: Invalid VLAN combinations MUST be rejected before they change durable or runtime state, with the draft configuration preserved for correction.
- **FR-022**: FancyWAN and FortiGate MUST each have a declared management attachment and distinct data-side roles sufficient to test a LAN-to-WAN or equivalent forwarding path.
- **FR-023**: Ruijie Switch MUST have at least one client-facing attachment, and Ruijie Router MUST participate in at least one routed client path.
- **FR-024**: Vendor-device connectivity MUST distinguish cable state, guest readiness, management reachability and proven data forwarding instead of representing them as one success state.
- **FR-025**: The acceptance topology MUST provide at least one complete multi-hop IPv6 terminal path with successful return traffic across more than one subnet.
- **FR-026**: IPv6 forwarding, neighbor discovery and route diagnostics MUST expose where a failed path stops without requiring direct host database inspection.
- **FR-027**: A stable traffic generator MUST repeatedly attempt ICMP, HTTP and DNS exchanges over declared IPv4 and IPv6 test paths and MUST expose successful and failed exchange totals separately.
- **FR-028**: Traffic Filter packet counts, byte counts and fingerprints MUST become non-zero for matching successful traffic and MUST remain durable independently of short-lived topology animation.
- **FR-029**: Topology highlighting MUST identify only connections carrying matching traffic and MUST decay after matching traffic stops without resetting durable counters.
- **FR-030**: Packet capture or other binary evidence used for acceptance MUST be bounded, explicitly owned, retrievable with truncation status, retained only for the configured evidence period and removed during laboratory cleanup.
- **FR-031**: Runtime creation and recovery MUST enforce existing resource, privilege and ownership limits and MUST record audit-visible outcomes for adoption, recreation, cleanup and failure.
- **FR-032**: No proprietary appliance image, credential, bootstrap secret or captured user payload MUST be added to source control or acceptance documentation.
- **FR-033**: Existing laboratory coordinates and connection presentation MUST remain stable while runtime resources are recovered or repaired.
- **FR-034**: The acceptance workflow MUST prove that the known stuck object link can be deleted and that no connection remains indefinitely pending or disconnecting after its timeout and retry policy is exhausted.

### Key Entities

- **Recoverable Runtime Resource**: An owned namespace, bridge, interface, route, rule, socket or process with durable identity, desired state, observed state, owner and last reconciliation outcome.
- **Network Path**: An ordered set of endpoints and connections with address family, expected forwarding roles, forward route, return route and observed reachability result.
- **VLAN Membership**: A port's PVID, tagged VLAN set, allowed forwarding membership and observed runtime membership.
- **Device Role Assignment**: A vendor-device interface assignment describing management, LAN, WAN, trunk or client-facing purpose and its reachability status.
- **Traffic Workload**: A repeatable ICMP, HTTP or DNS exchange definition with source, destination, address family, interval and success/failure totals.
- **Traffic Observation**: Durable matched packet count, byte count, fingerprint, affected connections and last-match time for a Traffic Filter.
- **Recovery Outcome**: The result of adopting, recreating, failing or deleting a runtime resource, including phase, cleanup, retryability and operator guidance.

## Assumptions

- The existing component-matrix laboratory remains the primary target acceptance fixture and may be repaired in place after a verified backup, while automated tests use isolated temporary laboratories.
- `0` continues to mean no count ceiling for running QEMU nodes; host-resource admission and per-node isolation protections remain in force.
- Vendor appliance guest configuration may require device-specific console actions, but NetLab remains responsible for correct interface attachment, honest readiness reporting, persistence of NetLab-owned state and repeatable acceptance evidence.
- The management network uses VLAN 30 as an access network; VLAN 10 and VLAN 20 are the minimum required tagged trunk pair.
- Passing a routed path requires bidirectional endpoint traffic, not only successful probes to router-owned addresses.
- Existing Traffic Filter retention policy remains unchanged unless planning identifies a conflict with durable acceptance evidence.

## Dependencies

- The base platform resource ownership, durable task, reconciliation and recovery guarantees defined by feature 001 remain authoritative.
- Existing network-object links, routes and traffic filtering behavior from feature 005 remains available and is corrected rather than replaced.
- Unified connection interaction and connection-state presentation from features 009 and 010 remain the user-facing control model.
- The target host continues to provide the virtualization, container, namespace, bridge, routing, filtering and capture capabilities required by the existing single-host product boundary.
- Required vendor appliance images and operator-authorized guest configuration access remain available on the target without being added to source control.

## Out of Scope

- Multi-host scheduling or distributed network emulation.
- Automated configuration of every possible vendor appliance operating-system feature.
- Supplying, redistributing or modifying proprietary vendor images.
- Replacing existing topology coordinates with automatic global relayout.
- Treating unrestricted QEMU count as unrestricted CPU, memory, storage or privilege consumption.

## Delivery and Deployment Constraints *(mandatory)*

- **Local milestone**: Deliver independently testable slices for recovery honesty and cleanup, dual-stack routed paths, VLAN/trunk behavior, vendor-device paths, and traffic observation; each slice passes focused unit, contract, integration, recovery and leak checks locally where privileges permit.
- **Commit evidence**: Record each independently testable slice in a focused Git commit and record its SHA with the associated validation result before deployment.
- **Deployment artifact**: Build a reproducible artifact from a clean identified commit and record its digest, contract identity and migration state before replacing the target service.
- **Target validation**: Deploy to `10.72.1.7`, validate the repaired component-matrix laboratory with privileged restart, connectivity, VLAN, IPv4, IPv6, traffic-filter and cleanup tests, and record the deployment time and results.
- **Rollback**: Preserve and verify the previous binary, configuration, readiness metadata and online database backup; rollback selects those recorded artifacts and verifies database integrity and laboratory recovery.
- **Target immutability**: Source fixes MUST NOT be made directly on `10.72.1.7`; failed validation returns to the local worktree for tests, a new commit, and redeployment.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: After 10 consecutive service restarts, 100% of valid namespace-backed objects and connections in the acceptance laboratory return to their expected active or connected state within 30 seconds, with zero invalid namespace references.
- **SC-002**: The known stuck object link and 20 additional create-delete failure-injection cycles complete cleanup with zero owned runtime or durable connection remnants.
- **SC-003**: Ubuntu and VyOS exchange 100 consecutive IPv4 packets and 100 consecutive IPv6 packets across the transit network with at least 99% success in each address family.
- **SC-004**: Ubuntu reaches at least one endpoint in each required downstream network over IPv4 and IPv6 with at least 99% success, including successful reply-path validation.
- **SC-005**: BusyBox exchanges IPv4 and IPv6 traffic with both the service router and traffic generator with at least 99% success over 100 probes per destination.
- **SC-006**: VLAN 10 and VLAN 20 each pass 100 tagged test exchanges over the trunk with at least 99% success, while 100 unapproved cross-VLAN exchanges are blocked unless an explicit routed path is enabled.
- **SC-007**: Observed VLAN membership matches configured PVID and tagged membership for every acceptance port after creation and after 10 service restarts.
- **SC-008**: All four vendor appliance nodes expose an honest management-readiness result, and each required vendor data path completes at least 20 consecutive successful client exchanges.
- **SC-009**: At least one multi-hop IPv6 path crossing two or more subnets completes 100 request-response exchanges with at least 99% success.
- **SC-010**: During a 10-minute traffic run, successful ICMP, HTTP and DNS exchanges occur at least once every 5 seconds on their declared paths, and matching packet and byte counters increase in every 10-second observation window.
- **SC-011**: Traffic fingerprints identify the expected address family and protocol for all acceptance filters, topology highlighting appears only on matching paths, and highlights decay within the configured visual window after traffic stops while durable counters remain unchanged.
- **SC-012**: Every injected recovery, routing, VLAN or traffic failure is visible to operators with a specific resource, phase and corrective action; no failed resource is displayed as healthy.
- **SC-013**: Local and target validation complete with zero leaked namespaces, interfaces, bridge memberships, rules, sockets, processes, captures or temporary laboratories.
