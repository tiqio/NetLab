# Phase 9 Convergence Validation — 2026-08-12

## Candidate and Deployment

- Authoritative target: `10.72.1.7:18082`.
- Deployed candidate: `network-path-011-20260812T093618Z-convergence-r25`.
- Source commit: `0d5016ca5972ab5860c4a7cbe9b3c83b4ed1530a`.
- Binary digest: `sha256:dbb286ce4ec2d22e1eab25adde46c02bafaf8189685b29d26eeca9aeb6cd8842`.
- Contract digest: `sha256:a61b993c5a9a96443705cb912115eb8ee357824d3c4e39273e30524ae59a5901`.
- Built at: `2026-08-12T09:36:18Z`.
- Deployment role remained `authoritative`; `/readyz`, release identity, SQLite integrity and rollback checks passed.
- Rollback: `/var/lib/netlab/rollback/network-path-011-20260812T093618Z-convergence-r25-predeploy`; all files passed `sha256sum -c`.

## Completed Gates

- L2 convergence no longer treats declared but unattached logical switch ports as runtime mismatches.
- The previously pending management switch converged to `active` after R25 deployment.
- Ten consecutive `netlab.service` restarts restored every valid node, object, link and attachment in 4.1–4.7 seconds, below the 30-second requirement.
- BusyBox dual-stack acceptance retained the recorded 100/100 results to the service router and continuous generator in both directions.
- IPv6 route/neighbor-discovery stop diagnostics are available through HTTP, MCP and the SPA.
- The isolated ten-minute privileged workload gate passed for IPv4 and IPv6 ICMP, HTTP and DNS: 120 exchanges per flow, five-second cadence, strict ten-second counter growth and address-family fingerprint validation.
- The workload runtime now bypasses inherited HTTP proxies and can bind DNS requests to an explicit IPv4 or IPv6 resolver address.
- Ruijie Router/Switch configuration used the supported `/ruijie/configure` control plane. A VLAN 110 client path and a routed-side PC were created through authoritative topology APIs. Switching, forward routing and reverse routing each passed 20/20 with 0% loss. The direct-link Traffic Filter recorded 20 packets and 1,680 bytes.

## Remaining Target Findings

### FancyWAN/FortiGate

- FortiGate interface numbering was verified to be one-based inside the guest: NetLab `port0/1/2` maps to guest `port1/2/3`.
- A core-to-DMZ client exchange temporarily passed 20/20 through FancyWAN and FortiGate, proving the forward appliance path.
- The authorized FancyWAN serial credential expected by the operator was rejected by the current `v1-stable` image, so its WAN-side runtime configuration could not be made repeatable or persisted.
- FortiGate performed first-login password initialization. The generated temporary password was memory-only and was not written to source, evidence or logs. The appliance is currently stopped to prevent its temporary `10.20.20.254` address from conflicting with VyOS.
- Because repeatable bidirectional appliance exchange and readiness evidence are not both present, T129 remains incomplete.

### Host-Restart Gate

- The new authoritative gate correctly refuses to reboot without ten successful pre-restart Ubuntu QGA probes and now retries transient HTTP and task-level guest-agent failures.
- It exposed an existing runtime recovery defect: after service/node recovery, the DMZ switch required explicit authoritative reconcile to clear VLAN membership drift, and VyOS `eth1` retained its address but failed IPv4 neighbor resolution to `10.20.20.1`.
- The gate never wrote a marker and the target host was not rebooted when preconditions failed. T133 remains incomplete rather than recording a false pass.

## Validation Commands

- `go test ./... -count=1` — PASS.
- `go vet ./...` — PASS.
- `cd web && npm test` — PASS, 83 files and 394 tests.
- `cd web && npm run lint` — PASS with 1,057 existing warnings and zero errors.
- `cd web && npm run format:check` — PASS.
- `cd web && npm run build` — PASS with the existing chunk-size warning.
- `NETLAB_PRIVILEGED=1 NETLAB_TRAFFIC_OBSERVATION_10M=1 go test ./tests/integration -run '^TestPrivilegedTenMinuteTrafficWorkloadObservation$' -count=1 -v -timeout 12m` — PASS in 596.22 seconds.
