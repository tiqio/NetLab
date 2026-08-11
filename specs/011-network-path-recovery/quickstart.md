# Quickstart: Network Path Recovery and Validation

## Prerequisites

- Clean local worktree and identified feature milestone commit.
- Go, Node.js/npm and existing NetLab test dependencies.
- Privileged Linux validation host with network namespaces, bridges, VLANs, Docker, KVM/QEMU, nftables and capture tools.
- Authorized access to `10.72.1.7`; proprietary images and credentials remain target-local.
- Online database backup and runtime/resource baseline before modifying the existing component-matrix laboratory.

## Local Quality Gates

```bash
go test ./...
go vet ./...
cd web
npm test
npm run build
npm run lint
npm run format:check
```

Run the focused privileged suites for namespace recovery, endpoint cleanup, VLAN membership, IPv6 forwarding, traffic workloads, Traffic Filter attribution and leak detection. Any skipped privileged test must record the exact target procedure below.

## Scenario 1 — Invalid Namespace Recovery

1. Create an isolated lab with a PC, L2 switch, L3 switch, plain bridge and mixed attachments.
2. Record desired/actual states, runtime ownership and resource counts.
3. Stop the service while a link mutation is pending; invalidate one owned namespace reference.
4. Restart the service.
5. Expect valid backing to be adopted, stale owned backing to be recreated, unrelated resources to recover, and no invalid resource to appear active.
6. Repeat for 10 restarts and verify recovery completes within 30 seconds each time.

## Scenario 2 — Failed Bridge Link Cleanup

1. Reproduce a failed bridge-to-L3 object link in a temporary lab.
2. Delete it through UI, HTTP and MCP retry/delete paths using revisions and idempotency keys.
3. Expect the bridge endpoint to use host-bridge cleanup, the L3 endpoint to use namespace cleanup, and the durable reservation to disappear.
4. Repeat 20 create/delete failure-injection cycles.
5. Compare namespaces, interfaces, bridge membership, routes, rules, sockets, processes and database rows with baseline; all deltas must be zero.

## Scenario 3 — BusyBox and Dual-Stack Routing

Use the component-matrix lab or an equivalent fixture:

- Service network: BusyBox, traffic generator and Docker router with IPv4/IPv6 forwarding.
- Transit: Ubuntu QEMU `172.16.0.2/30`, `fd16::2/64`; VyOS `172.16.0.1/30`, `fd16::1/64`.
- Downstream: core, DMZ and management IPv4/IPv6 networks with complete return routes.

Verify 100 probes per destination:

- BusyBox ↔ service router and traffic generator.
- Ubuntu ↔ VyOS transit.
- Ubuntu ↔ one endpoint in each downstream network.
- At least one IPv6 path crossing two or more subnets.

Each required path must achieve at least 99% success before and after node restart and service restart.

## Scenario 4 — VLAN Access and Trunk

1. Configure VLAN 10 and VLAN 20 access endpoints.
2. Configure one trunk carrying tagged VLANs 10 and 20.
3. Verify diagnostics show desired and observed membership equal on every port.
4. Send 100 exchanges per VLAN across the trunk; require at least 99% success.
5. Send 100 unapproved cross-VLAN exchanges with routing disabled; require all blocked.
6. Restart the service 10 times and repeat membership and forwarding checks.

## Scenario 5 — Vendor Management and Data Paths

1. Attach FancyWAN, FortiGate, Ruijie Router and Ruijie Switch to VLAN 30 management access ports.
2. Assign LAN/WAN roles to FancyWAN/FortiGate and client/routed roles to the Ruijie pair.
3. Use operator-authorized guest configuration without recording secrets.
4. From the management PC, verify each declared management address or an explicit guest prerequisite.
5. Run at least 20 successful exchanges across each required vendor data path.
6. Confirm UI distinguishes cable connected, guest ready, management reachable and data path proven.

## Scenario 6 — Traffic Workloads and Filters

1. Create ICMP IPv4/IPv6, HTTP and DNS workloads with intervals no greater than five seconds.
2. Create matching Traffic Filters on the intended connections.
3. Run for 10 minutes.
4. Expect successful exchanges in every five-second window and workload aggregates to distinguish failures.
5. Expect filter packet/byte counts and fingerprints to become non-zero and increase in every 10-second observation window.
6. Stop traffic; expect highlights to decay within the visual window while durable counters remain.
7. Restart the service during the run and verify workload lifecycle and aggregates recover idempotently.

## Concurrent Control Validation

- Run two browsers plus HTTP and MCP clients against the same lab.
- Race reconcile retry, VLAN update, workload start/stop and delete with stale and current revisions.
- Expect one committed result per idempotency key, ordered events, explicit revision conflicts and convergence without duplicate runtime resources.

## Target Deployment

1. Commit each local milestone after its focused gates pass.
2. Build from a clean identified commit and record candidate ID, commit SHA, artifact digest, contract digest and migration state.
3. Create and verify a rollback directory containing the previous binary/configuration/readiness metadata and an online SQLite backup.
4. Deploy without editing target source files.
5. Run health, migration, runtime ownership, six-QEMU, scenarios 1–6, browser and leak checks.
6. Record deployment time, exact target results, database integrity and rollback path under this feature's validation documentation.

## Expected Completion State

- No invalid namespace references.
- No resources indefinitely pending or disconnecting.
- No failed resource displayed healthy.
- BusyBox, VyOS, VLAN, vendor and IPv6 paths meet the specified success rates.
- Traffic workloads show successful exchanges and matching filters show non-zero durable counters.
- Existing resource coordinates remain unchanged.
- Cleanup returns every temporary resource class to baseline.
