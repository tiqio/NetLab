# Target Acceptance — 2026-08-12

## Recovery and Capacity

| Gate | Result |
|------|--------|
| Ten `netlab.service` restarts | PASS; every round retained 6 running NetLab QEMU nodes, zero bad connections and unchanged candidate identity |
| Twenty bridge-to-L3 cleanup cycles | PASS; `TestBridgeToL3CleanupCompletesTwentyCyclesWithoutNamespaceLeak` completed without namespace leakage |
| Six-QEMU runtime | PASS; all six declared NetLab QEMU node IDs have one matching `guest=netlab:<node-id>` process |
| QEMU admission configuration | PASS; `resources.max_running_qemu: 0` retains the requested unlimited protection value |

- One additional host QEMU belongs to EVE-NG under `/opt/unetlab`; it is not NetLab-owned and was neither adopted nor modified.

## Dual-Stack Paths

- BusyBox `10.40.40.11` reached service-router IPv4 `10.40.40.1` and IPv6 `fd40::1` with 5/5 probes each.
- The service-router diagnostic reports desired and observed IPv4/IPv6 forwarding enabled with an empty mismatch list.
- Ubuntu QEMU retains `172.16.0.2/30` and `fd16::2/64` and the downstream static routes through VyOS.
- Ubuntu-to-VyOS/downstream acceptance passed 100/100 for each of five IPv4 destinations (`172.16.0.1`, `10.20.20.1`, `10.20.20.10`, `10.30.30.1`, `10.30.30.10`) and five IPv6 destinations (`fd16::1`, `fd20::1`, `fd20::10`, `fd30::1`, `fd30::10`).
- A final non-mutating R22 spot check again passed 5/5 to `172.16.0.1`, `10.20.20.1`, `10.20.20.10` and `10.30.30.1`. QGA then produced a transient `invalid guest-exec response`; this was recorded as guest-channel variability rather than a false path failure, and no target configuration was changed.

## VLAN and Vendor Paths

- `TestPrivilegedVLANAccessTrunkAndIsolation` passed approved VLAN forwarding and cross-VLAN isolation.
- `TestVLANMembershipPersistsAcrossTenRuntimeRestarts` passed with exact membership retained.
- Core and DMZ L2 objects report empty mismatch lists; both `eth2` ports carry tagged VLANs 10 and 20, while access ports retain PVID 10 or 20.
- FancyWAN ↔ FortiGate, Ruijie Router ↔ Ruijie Switch and Ubuntu QEMU ↔ VyOS direct links all report `connected`.
- FancyWAN, FortiGate, Ruijie Router and Ruijie Switch retain explicit interface roles. Cable readiness is reported independently from management prerequisites and unverified guest data paths.

## Stable Traffic and Recovery

| Gate | Result |
|------|--------|
| Ten-minute ICMP/HTTP/DNS observation | PASS in 596.25 seconds |
| Protocol exchanges | 120 ICMP, 120 HTTP and 120 DNS exchanges |
| Traffic Filter evidence | Non-zero packet, byte and fingerprint counters from real captures |
| Workload restart recovery | PASS; a durable running workload resumed after service restart |
| Dual-stack route recovery | PASS after service restart |
| Vendor role recovery | PASS without storing credentials |

- Temporary acceptance workloads and filters were deleted after observation; final target counts are zero.
