# Authoritative Deployment — 2026-08-12

## Target

- Host: `root@10.72.1.7`.
- Service: `netlab.service`, listening on `10.72.1.7:18082`.
- Deployment role: `authoritative`.
- Source files were not edited on the target; identified local artifacts were installed through the deployment workflow.

## R21 Initial US5 Candidate

| Field | Value |
|-------|-------|
| Candidate | `network-path-011-20260812T072942Z-us5-r21` |
| Source commit | `d05bdfea564264eba77419a16aa0e21fcc507853` |
| Artifact digest | `sha256:6ed930d9a652024f5b2669839f8eb033a129d4e76a96293a642790035fd04927` |
| Contract digest | `sha256:a61b993c5a9a96443705cb912115eb8ee357824d3c4e39273e30524ae59a5901` |
| Built at | `2026-08-12T07:29:42Z` |
| Migration | `0016_traffic_workloads.sql`, schema version 16 |
| Rollback | `/var/lib/netlab/rollback/network-path-011-20260812T072942Z-us5-r21-predeploy` |

- R21 passed the backend, recovery, data-plane and ten-minute workload gates.
- Target Chromium then exposed a millisecond-level ordering race: a real capture observation could precede the corresponding workload success timestamp by a few milliseconds, causing the topology overlay to reject an otherwise valid highlight.

## R22 Authoritative Candidate

| Field | Value |
|-------|-------|
| Candidate | `network-path-011-20260812T075902Z-us5-r22` |
| Source commit | `f30ad2e2d54817971f24d34c78fdb2b06821df39` |
| Artifact digest | `sha256:f5a8d61b6dc2b3b01c7802eee625def601866a6353c61acdd2d426ee87d55281` |
| Contract digest | `sha256:a61b993c5a9a96443705cb912115eb8ee357824d3c4e39273e30524ae59a5901` |
| Built at | `2026-08-12T07:59:02Z` |
| Installed at | `2026-08-12T08:06:28Z` |
| Migration | `0016_traffic_workloads.sql`, schema version 16 |
| Rollback | `/var/lib/netlab/rollback/network-path-011-20260812T075902Z-us5-r22-predeploy` |

- Commit `f30ad2e` adds a bounded correlation tolerance of at least two seconds, or the workload timeout when larger, without synthesizing Traffic Filter counters.
- The target binary SHA-256 is `f5a8d61b6dc2b3b01c7802eee625def601866a6353c61acdd2d426ee87d55281`.
- `/api/v1/capabilities` reports the R22 candidate, source contract digest, artifact digest and build time shown above.
- `netlab.service` is active, `/readyz` returns `{"status":"ok"}`, schema migration is 16 and `PRAGMA integrity_check` returns `ok`.
- R21 and R22 share the accepted US5 backend. R22 therefore retained the R21 backend and ten-minute results and focused target revalidation on the Chromium correlation fix, release identity, service health and database integrity.

## Rollback Integrity

- Each rollback directory contains the previous `netlabd`, `netlab.yaml`, `template-readiness.json`, `release.json`, an online `netlab.db.gz` backup and `SHA256SUMS`.
- `sha256sum -c SHA256SUMS` passed for every recorded file in both rollback directories.
- The R21 rollback preserves the preceding R20 release and pre-migration database state; the R22 rollback preserves R21 and schema version 16.

## R25 Convergence Candidate

| Field | Value |
|-------|-------|
| Candidate | `network-path-011-20260812T093618Z-convergence-r25` |
| Source commit | `0d5016ca5972ab5860c4a7cbe9b3c83b4ed1530a` |
| Artifact digest | `sha256:dbb286ce4ec2d22e1eab25adde46c02bafaf8189685b29d26eeca9aeb6cd8842` |
| Contract digest | `sha256:a61b993c5a9a96443705cb912115eb8ee357824d3c4e39273e30524ae59a5901` |
| Built at | `2026-08-12T09:36:18Z` |
| Migration | no new migration; SQLite user version `0` |
| Rollback | `/var/lib/netlab/rollback/network-path-011-20260812T093618Z-convergence-r25-predeploy` |

R25 passed authority, readiness, release identity, database integrity and ten service-restart recovery checks. Phase 9 residual findings are recorded in `validation/convergence.md`.

## R30 QEMU and Data-Plane Recovery Candidate

| Field | Value |
|-------|-------|
| Candidate | `network-path-011-20260813T025824Z-qemu-recovery-r30` |
| Binary source commit | `d5cb0dbc70c7b47668c224c9041a86f8e043afdb` |
| Acceptance source commit | `e6706a704186b6ecfd5fdf4e218bbeb3d84ec84f` |
| Artifact digest | `sha256:b49b58e45799c80b937e522538074a4cf414afa0277c79d463d73c3e98c9cadc` |
| Contract digest | `sha256:71d5721ac41dd2d55ccdd7934d4ab0a77c7c77ecd602df43f24af89e1699668a` |
| Acceptance script digest | `sha256:164e0217905b2544ab8b4a24408c2c2a92bcb0fb9eeaae850700d27c77f6eb96` |
| Built at | `2026-08-13T02:58:24Z` |
| Migration | no new migration; SQLite user version `0` |
| Rollback | `/var/lib/netlab/rollback/network-path-011-20260813T025824Z-qemu-recovery-r30-predeploy` |

- R30 was installed through `deploy/scripts/install.sh` with `NETLAB_DEPLOYMENT_ROLE=authoritative`; no target source file or database row was edited directly.
- The rollback directory contains the prior binary, configuration, template readiness, release identity, authoritative lab snapshot and a previously verified no-migration online database backup. Its `SHA256SUMS` verification passed before deployment.
- The deployment corrected a stale QEMU manifest whose PID had been reused after host restart. Ubuntu QEMU restarted automatically, both owned TAP interfaces reappeared, the NAT attachment became `active`, and the authoritative Ubuntu-to-VyOS link converged to `connected`.
- Final host-restart acceptance used boot IDs `6d459476-cda1-47b3-97ec-1bc4ece341b6` and `08262fae-edf5-4563-b691-01f45a8dd8bd`. One `verify` invocation handled the cold QGA delay through a bounded retry and passed all ten IPv4/IPv6 path probes.
- After verification, `netlab.service` was `active`, `/readyz` returned `{"status":"ok"}`, the candidate remained R30, `PRAGMA integrity_check` returned `ok`, and SQLite user version was `0`.
