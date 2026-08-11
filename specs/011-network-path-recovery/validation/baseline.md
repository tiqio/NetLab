# Baseline — 2026-08-11

## Deployment

- Target: `10.72.1.7:18082`
- Candidate: `qemu-unlimited-20260811T071624Z-r2`
- Service: active
- Health: `ok`
- Laboratory: `019feee704ee-c5b2bd57159bebe86bed` (`组件矩阵-20260811T033914Z`)
- Laboratory revision: 25

## Resource Counts

| Resource | Count |
|----------|------:|
| Nodes | 11 |
| QEMU processes carrying `netlab:` ownership | 7 |
| Laboratory Docker containers | 5 |
| Network objects | 9 |
| Direct node links | 3 |
| Network attachments | 8 |
| Network-object links | 7 |
| Named namespace files | 7 |

## Failed or Non-Converged State

- L3 object `019feee71261-536b824c99d9083627f8` is desired `active` but observed `failed`.
- Object links `019feee73cee-5e8ed657387bbfaf7f23`, `019feee73daa-5fe336e01b5692b8f1f6`, and `019fef8acc6f-9e7d2092af71ca45a14b` are `pending` against the failed L3 object.
- Bridge-to-L3 object link `019fef7b284d-280a689d6233f257dd08` is stuck `disconnecting`.
- Files under `/run/netns` are listed, but entering each namespace returns `Invalid argument`.
- Host VLAN readback shows VLAN 1 membership while desired L2 PVIDs are 10, 20 and 30; no tagged trunk exists.

## Connectivity Baseline

- Ubuntu QEMU ↔ VyOS transit IPv4 and IPv6: pass.
- Ubuntu QEMU → VyOS downstream interface addresses: pass.
- Ubuntu QEMU → DMZ and management endpoints: fail.
- BusyBox → service router and traffic generator: fail.
- Traffic generator → service router: pass for IPv4 and IPv6.
- Traffic generator → core endpoints and HTTP target: fail.
- Docker service router observed forwarding: IPv4 enabled, IPv6 disabled.

## Evidence Rules

Only resource identifiers, aggregate counts, state, redacted diagnostics and command outcomes are retained. Credentials, proprietary image paths and packet payloads are excluded.
