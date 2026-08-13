# Phase 9 Convergence Validation — 2026-08-13

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

## R30 Host-Restart Closure

- Candidate `network-path-011-20260813T025824Z-qemu-recovery-r30` was deployed to the authoritative target from binary source commit `d5cb0dbc70c7b47668c224c9041a86f8e043afdb`; the final acceptance script revision was `e6706a704186b6ecfd5fdf4e218bbeb3d84ec84f`.
- Namespace attachment recovery now performs exact bridge/VLAN convergence even when the host-side path appears healthy. An empty attachment VLAN configuration inherits the matching switch port policy; explicit attachment fields remain overrides.
- Data-plane reconciliation retains authoritative links and attachments as retryable `pending` when a running interface has not appeared yet. Attachments owned by intentionally stopped nodes remain `pending` with their interfaces down instead of becoming false failures.
- QEMU inspection now validates that the PID in `launch.json` belongs to the expected `qemu-system` process and contains `guest=netlab:<node-id>`. A host-restart PID reuse can no longer leave a missing QEMU reported as `running`.
- The first R30 deployment corrected the previously false-running Ubuntu node automatically: the QEMU process, both TAP interfaces, NAT attachment and Ubuntu-to-VyOS point-to-point link were recreated without direct database edits.
- The final controlled restart changed boot ID from `6d459476-cda1-47b3-97ec-1bc4ece341b6` to `08262fae-edf5-4563-b691-01f45a8dd8bd`.
- Before restart, all ten Ubuntu probes passed: IPv4 `172.16.0.1`, `10.20.20.1`, `10.20.20.10`, `10.30.30.1`, `10.30.30.10`; IPv6 `fd16::1`, `fd20::1`, `fd20::10`, `fd30::1`, `fd30::10`.
- The single post-restart `verify` invocation observed one expected cold-guest/QGA retry, then passed all ten probes. The before/after topology states matched: two connected links plus one intentional pending vendor link, eleven active attachments plus two intentional pending FortiGate attachments.
- The latest `system.recovery` task completed with `state=succeeded` and `mode=host_restart`; candidate identity remained unchanged, `netlab.service` remained active, `/readyz` returned `{"status":"ok"}`, `PRAGMA integrity_check` returned `ok`, and SQLite user version remained `0`.
- T133 is complete. T129 remains open because repeatable FancyWAN/FortiGate credentials, persistent appliance configuration and bidirectional vendor-path evidence are still unavailable.

## Validation Commands

- `go test ./... -count=1` — PASS.
- `go vet ./...` — PASS.
- `cd web && npm test` — PASS, 83 files and 394 tests.
- `cd web && npm run lint` — PASS with 1,057 existing warnings and zero errors.
- `cd web && npm run format:check` — PASS.
- `cd web && npm run build` — PASS with the existing chunk-size warning.
- `NETLAB_PRIVILEGED=1 NETLAB_TRAFFIC_OBSERVATION_10M=1 go test ./tests/integration -run '^TestPrivilegedTenMinuteTrafficWorkloadObservation$' -count=1 -v -timeout 12m` — PASS in 596.22 seconds.
- `NETLAB_PRIVILEGED_TESTS=1 go test ./tests/recovery -run 'Test(VLANMembershipPersists|NamespaceAttachmentRestoresForwarding)AcrossTenRuntimeRestarts' -count=1 -v` — PASS; exact VLAN recovery and real IPv4 forwarding passed across ten corruption/recovery cycles.
- `go test ./...` after commits `3d37179`, `81438e2`, `69ce19c`, `ea229b5`, `d5cb0db` and `e6706a7` — PASS.
- `acceptance/network-path-host-restart.sh prepare`, controlled host reboot, then one `verify` invocation — PASS; the first cold-guest snapshot retry was absorbed by the bounded gate and the second snapshot passed 10/10.
