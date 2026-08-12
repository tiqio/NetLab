# Component Matrix Repair — 2026-08-12

## Scope

- Laboratory: `019feee704ee-c5b2bd57159bebe86bed` (`组件矩阵-20260811T033914Z`).
- Candidate: `network-path-011-20260812T032024Z-us23-r19`.
- Source commit: `69ea7041885cef775fb98b8e274e3455d70f6a28`.
- All topology mutations used revision preconditions and unique idempotency keys.

## Desired-State Repair

- BusyBox `eth0` retains `10.40.40.11/24`, `fd40::11/64` and dual-stack defaults through the service router.
- Service router retains two authoritative attachments, dual-stack forwarding, service/core addresses and defaults through `10.10.10.1` and `fd10::1`.
- Ubuntu QEMU retains `172.16.0.2/30` and `fd16::2/64` on `ens1` with downstream routes through VyOS.
- VyOS was repaired through the managed console using target-local credentials; no credential or proprietary path was recorded.
- L3 return routes to `172.16.0.0/30` and `fd16::/64` were added through a revisioned network-object update.

## Recovery Evidence

- Docker router restart changed PID `688279` to `812756`.
- Both NetLab-owned attachments remained present while running, stopped and restarted: `2/2/2`.
- Reconciliation restored `default via 10.10.10.1 dev eth1` and `default via fd10::1 dev eth1 metric 1024`.
- Router-local probes to `10.40.40.1`, `10.10.10.1`, `fd40::1` and `fd10::1` each passed `100/100`.
- L3 diagnostics report `active`, dual-stack forwarding enabled and zero desired/observed mismatches.

## Deployment Record

- Artifact digest: `sha256:19f1753ac546ce6004c3dc96407eedff3f6d32b8f320c9cc79dac64e8389ff47`.
- Contract digest: `sha256:8d22f88384ad333d1d16452e85be3537fb6f6011947181116487024c0e8e03d9`.
- Built at: `2026-08-12T03:20:24Z`.
- Target validation completed at: `2026-08-12T03:43:54Z`.
- Migration state: no new migration; SQLite `PRAGMA integrity_check` returned `ok`.
- Rollback: `/var/lib/netlab/rollback/network-path-011-20260812T032024Z-us23-r19-predeploy` with verified binary, configuration, readiness and compressed SQLite backup.
