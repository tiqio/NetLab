# Authoritative Deployment — 2026-08-12

## Target

- Host: `root@10.72.1.7`.
- Service: `netlab.service`, listening on `10.72.1.7:18082`.
- Deployment role: `authoritative`.
- Source files were not edited on the target; identified local artifacts were installed through the deployment workflow.

## R21 Initial US5 Candidate

| Field           | Value                                                                          |
| --------------- | ------------------------------------------------------------------------------ |
| Candidate       | `network-path-011-20260812T072942Z-us5-r21`                                    |
| Source commit   | `d05bdfea564264eba77419a16aa0e21fcc507853`                                     |
| Artifact digest | `sha256:6ed930d9a652024f5b2669839f8eb033a129d4e76a96293a642790035fd04927`      |
| Contract digest | `sha256:a61b993c5a9a96443705cb912115eb8ee357824d3c4e39273e30524ae59a5901`      |
| Built at        | `2026-08-12T07:29:42Z`                                                         |
| Migration       | `0016_traffic_workloads.sql`, schema version 16                                |
| Rollback        | `/var/lib/netlab/rollback/network-path-011-20260812T072942Z-us5-r21-predeploy` |

- R21 passed the backend, recovery, data-plane and ten-minute workload gates.
- Target Chromium then exposed a millisecond-level ordering race: a real capture observation could precede the corresponding workload success timestamp by a few milliseconds, causing the topology overlay to reject an otherwise valid highlight.

## R22 Authoritative Candidate

| Field           | Value                                                                          |
| --------------- | ------------------------------------------------------------------------------ |
| Candidate       | `network-path-011-20260812T075902Z-us5-r22`                                    |
| Source commit   | `f30ad2e2d54817971f24d34c78fdb2b06821df39`                                     |
| Artifact digest | `sha256:f5a8d61b6dc2b3b01c7802eee625def601866a6353c61acdd2d426ee87d55281`      |
| Contract digest | `sha256:a61b993c5a9a96443705cb912115eb8ee357824d3c4e39273e30524ae59a5901`      |
| Built at        | `2026-08-12T07:59:02Z`                                                         |
| Installed at    | `2026-08-12T08:06:28Z`                                                         |
| Migration       | `0016_traffic_workloads.sql`, schema version 16                                |
| Rollback        | `/var/lib/netlab/rollback/network-path-011-20260812T075902Z-us5-r22-predeploy` |

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

| Field           | Value                                                                                  |
| --------------- | -------------------------------------------------------------------------------------- |
| Candidate       | `network-path-011-20260812T093618Z-convergence-r25`                                    |
| Source commit   | `0d5016ca5972ab5860c4a7cbe9b3c83b4ed1530a`                                             |
| Artifact digest | `sha256:dbb286ce4ec2d22e1eab25adde46c02bafaf8189685b29d26eeca9aeb6cd8842`              |
| Contract digest | `sha256:a61b993c5a9a96443705cb912115eb8ee357824d3c4e39273e30524ae59a5901`              |
| Built at        | `2026-08-12T09:36:18Z`                                                                 |
| Migration       | no new migration; SQLite user version `0`                                              |
| Rollback        | `/var/lib/netlab/rollback/network-path-011-20260812T093618Z-convergence-r25-predeploy` |

R25 passed authority, readiness, release identity, database integrity and ten service-restart recovery checks. Phase 9 residual findings are recorded in `validation/convergence.md`.

## R30 QEMU and Data-Plane Recovery Candidate

| Field                    | Value                                                                                    |
| ------------------------ | ---------------------------------------------------------------------------------------- |
| Candidate                | `network-path-011-20260813T025824Z-qemu-recovery-r30`                                    |
| Binary source commit     | `d5cb0dbc70c7b47668c224c9041a86f8e043afdb`                                               |
| Acceptance source commit | `e6706a704186b6ecfd5fdf4e218bbeb3d84ec84f`                                               |
| Artifact digest          | `sha256:b49b58e45799c80b937e522538074a4cf414afa0277c79d463d73c3e98c9cadc`                |
| Contract digest          | `sha256:71d5721ac41dd2d55ccdd7934d4ab0a77c7c77ecd602df43f24af89e1699668a`                |
| Acceptance script digest | `sha256:164e0217905b2544ab8b4a24408c2c2a92bcb0fb9eeaae850700d27c77f6eb96`                |
| Built at                 | `2026-08-13T02:58:24Z`                                                                   |
| Migration                | no new migration; SQLite user version `0`                                                |
| Rollback                 | `/var/lib/netlab/rollback/network-path-011-20260813T025824Z-qemu-recovery-r30-predeploy` |

- R30 was installed through `deploy/scripts/install.sh` with `NETLAB_DEPLOYMENT_ROLE=authoritative`; no target source file or database row was edited directly.
- The rollback directory contains the prior binary, configuration, template readiness, release identity, authoritative lab snapshot and a previously verified no-migration online database backup. Its `SHA256SUMS` verification passed before deployment.
- The deployment corrected a stale QEMU manifest whose PID had been reused after host restart. Ubuntu QEMU restarted automatically, both owned TAP interfaces reappeared, the NAT attachment became `active`, and the authoritative Ubuntu-to-VyOS link converged to `connected`.
- Final host-restart acceptance used boot IDs `6d459476-cda1-47b3-97ec-1bc4ece341b6` and `08262fae-edf5-4563-b691-01f45a8dd8bd`. One `verify` invocation handled the cold QGA delay through a bounded retry and passed all ten IPv4/IPv6 path probes.
- After verification, `netlab.service` was `active`, `/readyz` returned `{"status":"ok"}`, the candidate remained R30, `PRAGMA integrity_check` returned `ok`, and SQLite user version was `0`.

## R31 FortiGate Credential Candidate

| Field           | Value                                                                                            |
| --------------- | ------------------------------------------------------------------------------------------------ |
| Candidate       | `network-path-011-20260813T035016Z-fortigate-credentials-r31`                                    |
| Source commit   | `83693581de6d77f7a92819dd2237253623423c15`                                                       |
| Artifact digest | `sha256:6680f0c194f89199d3287b7fc64c1741beba61c2a8fa0c4c0d0b35930fe5dd6d`                        |
| Contract digest | `sha256:8d22f88384ad333d1d16452e85be3537fb6f6011947181116487024c0e8e03d9`                        |
| Built at        | `2026-08-13T03:50:16Z`                                                                           |
| Installed at    | `2026-08-13T03:55:41Z`                                                                           |
| Migration       | no main-database migration; schema migration version remains `16`                                |
| Rollback        | `/var/lib/netlab/rollback/network-path-011-20260813T035016Z-fortigate-credentials-r31-predeploy` |

- R31 adds a node-scoped AES-256-GCM credential vault for FortiGate console credentials, with root-only master-key and vault paths, active/staged password rotation, management-scope-restricted APIs and an inspector workflow that never reads stored plaintext back into the browser.
- The deployment installer generated `/etc/netlab/credential-master.key` only because it did not already exist. The key is `0600`, `/var/lib/netlab/secrets` is `0700`, and `credentials.db` is `0600`; future installs do not overwrite the key.
- A temporary generated credential was written through the target API, confirmed absent from both SQLite files, task and audit responses, and the service journal, then deleted. The failed verification task contained only `node:<id>:console_admin` and returned the expected `console_unreachable` error because the authorized FortiGate node remained stopped.
- The embedded UI chunk contains the FortiGate credential inspector. The legacy bootstrap-credential endpoint no longer returns username or password fields.
- Final target state: `netlab.service` active, `/readyz` healthy, release identity and binary digest matched R31, `PRAGMA integrity_check` returned `ok`, schema migration version remained `16`, the FortiGate node remained stopped, and the credential vault contained zero rows after cleanup.

## Interface-Only Link Label Candidate

| Field           | Value                                                                     |
| --------------- | ------------------------------------------------------------------------- |
| Candidate       | `topology-labels-20260813T0646Z-r1`                                       |
| Source commit   | `5995cdb1d143a278df99377998afe5f015a76014`                                |
| Artifact digest | `sha256:041c85c2dca28fe6924824349d48a091be614086c459769b180716ebfacc18d8` |
| Contract digest | `sha256:71d5721ac41dd2d55ccdd7934d4ab0a77c7c77ecd602df43f24af89e1699668a` |
| Built at        | `2026-08-13T06:47:49Z`                                                    |
| Installed at    | `2026-08-13T06:51:25Z`                                                    |
| Migration       | no database migration                                                     |
| Rollback        | `/var/lib/netlab/rollback/topology-labels-20260813T0646Z-r1-predeploy`    |

- Visible link labels now use only endpoint interface or port names, such as `eth0 ↔ ens0`; resource names remain in the accessibility label and endpoint details.
- Focused Vitest coverage passed `10/10`, and the production frontend plus embedded Go binary built successfully.
- The deployed topology asset contains the interface-only formatter, while its accessibility formatter retains `resourceName:portName` for both endpoints.
- A read-only Chromium target smoke test loaded the integrated laboratory without console or page errors.
- After authoritative restart, `/readyz` returned `{"status":"ok"}` and the integrated laboratory retained 11 running nodes, 10 active network objects, 21 placements and revision 22.

## Topology Organize and Selection Candidate

| Field           | Value                                                                      |
| --------------- | -------------------------------------------------------------------------- |
| Candidate       | `topology-organize-20260813T072731Z-r2`                                    |
| Source commit   | `4025c790148d9da27ffdb8057b0ee6ae9004ee95`                                 |
| Artifact digest | `sha256:17a271944b2803934b4e1bbf7ce71993d00cf50849ee30bb648b75af0a07c3f5`  |
| Contract digest | `sha256:71d5721ac41dd2d55ccdd7934d4ab0a77c7c77ecd602df43f24af89e1699668a`  |
| Built at        | `2026-08-13T07:27:31Z`                                                     |
| Installed at    | `2026-08-13T07:42:36Z`                                                     |
| Migration       | no database migration; schema migration version remains `16`               |
| Rollback        | `/var/lib/netlab/rollback/topology-organize-20260813T072731Z-r2-predeploy` |

- The candidate upgrades the former Fit All control to an explicit layered organize-and-fit mutation. NAT objects are placed highest, L3 and L2 objects occupy intermediate layers, and PC/QEMU/Docker endpoints occupy the lowest layer. Connectivity-weighted sweeps reduce adjacent-layer crossings before placements are persisted in batches.
- Selection interaction now enables Fit Selection after a resource click or drag. Dragging an unselected resource selects it first, blank-canvas dragging performs box selection without a modifier, Shift adds to selection, and Ctrl remains reserved for temporary canvas panning.
- Local validation passed 31 focused topology tests, the production Vue build, embedded Go build, `git diff --check`, and the two-test desktop Playwright topology-navigation journey. The source commit was pushed to `origin/main` before deployment.
- The first online SQLite backup attempt became stuck in uninterruptible target-disk I/O while the 2.7 GiB database was actively written. The service was stopped to end writes, then restored with the previous candidate before deployment; it returned active with all running nodes recovered. Because this UI candidate has no migration or direct data mutation, the incomplete backup was removed and the verified rollback package intentionally contains the prior binary, configuration, template readiness, unit, release identity, health state, and resource summary rather than a database replacement. `database-rollback-scope.txt` records that rollback leaves the authoritative database in place.
- Deployment preserved the only integrated laboratory at revision `34`: 11 of 11 nodes remained `running`, 10 of 10 network objects remained `active`, all 21 placements remained present, and all 25 persisted connections remained present. SQLite `PRAGMA integrity_check` returned `ok`; `/readyz` and `/healthz` returned `ok`; no error-priority service journal entries appeared after installation.
- A target Chromium session verified that Fit Selection starts disabled, becomes enabled after keyboard selection, executes successfully, and remains enabled after a real blank-canvas mouse box selection that selected two resources. The deployed button text is `整理并适应`, and the browser reported no page or console errors.

### Selection Highlight Correction — r3

| Field           | Value                                                                      |
| --------------- | -------------------------------------------------------------------------- |
| Candidate       | `topology-organize-20260813T075645Z-r3`                                    |
| Source commit   | `cc0d9cd3c62e67174db3d0d0d20dc571a0b55ee6`                                 |
| Artifact digest | `sha256:1933a0cb44bd93515f7957f71812e895ad05a19d015cf2d71809928eeabe6149`  |
| Contract digest | `sha256:71d5721ac41dd2d55ccdd7934d4ab0a77c7c77ecd602df43f24af89e1699668a`  |
| Built at        | `2026-08-13T07:56:45Z`                                                     |
| Migration       | no database migration; schema migration version remains `16`               |
| Rollback        | `/var/lib/netlab/rollback/topology-organize-20260813T075645Z-r3-predeploy` |

- R3 adds persistent double-ring selection halos, a bottom-right `已选 N 项` indicator and a stronger box-selection rectangle. Multi-selection no longer opens every selected resource's port overlay, reducing visual noise while preserving single-resource port access.
- Focused topology validation passed 32 tests and the production frontend plus embedded Go build. The target artifact, configuration identity and rollback hashes matched the recorded candidate.
- Real Chromium mouse box selection selected 11 resources, rendered visible halos for the selected resources currently inside the viewport, displayed `已选 11 项`, and produced no page or console errors.
- After deployment, the integrated laboratory retained 11 running nodes, 10 active network objects, 21 placements and 25 connections. Database integrity remained `ok`, and no error-priority service journal entries appeared after the R3 restart.

### NAT Bottom-Layer Correction — r4

| Field           | Value                                                                      |
| --------------- | -------------------------------------------------------------------------- |
| Candidate       | `topology-organize-20260813T080427Z-r4`                                    |
| Source commit   | `275b6899037ce5d55a6756f243eae4cb5319341e`                                 |
| Artifact digest | `sha256:b03bc57bbea1e5a8620ecf359618e949622f6dc60f8d879ea995f90e4072abb3`  |
| Contract digest | `sha256:71d5721ac41dd2d55ccdd7934d4ab0a77c7c77ecd602df43f24af89e1699668a`  |
| Built at        | `2026-08-13T08:04:27Z`                                                     |
| Migration       | no database migration; schema migration version remains `16`               |
| Rollback        | `/var/lib/netlab/rollback/topology-organize-20260813T080427Z-r4-predeploy` |

- R4 changes the organizer to five layers and gives NAT bridges an exclusive bottom layer below PC, QEMU and Docker endpoints. The target layout places the integrated laboratory NAT at `y=820`, while the lowest non-NAT endpoint layer remains at `y=500`.
- Focused topology validation passed 32 tests, followed by the production frontend and embedded Go build. The candidate and rollback package hashes were verified before installation.
- Target Chromium invoked `整理并适应`, persisted laboratory revision `45`, and confirmed every NAT placement was below every non-NAT placement without page or console errors.
- The integrated laboratory remained healthy with 11 running nodes, 10 active network objects, 21 placements and 25 connections; database integrity remained `ok`.

### Box-Selection Coordinate Correction — r5

| Field           | Value                                                                    |
| --------------- | ------------------------------------------------------------------------ |
| Candidate       | topology-organize-20260813T081329Z-r5                                    |
| Source commit   | cde2c78a174d5e688ede7c063adcc5b3b34e3a99                                 |
| Artifact digest | sha256:1237d4fa95bd8b6819ba08ecd2b73c30e01b31755dba42ab6846ca9f64bb9cd0  |
| Contract digest | sha256:71d5721ac41dd2d55ccdd7934d4ab0a77c7c77ecd602df43f24af89e1699668a  |
| Built at        | 2026-08-13T08:13:29Z                                                     |
| Migration       | no database migration; schema migration version remains 16               |
| Rollback        | /var/lib/netlab/rollback/topology-organize-20260813T081329Z-r5-predeploy |

- R5 replaces the simplified viewport arithmetic used by box selection with the authoritative ECharts convertFromPixel transformation. The visible rectangle and resource hit testing now share the same chart-local coordinate system, including canvas offsets, zoom and graph center.
- Focused topology validation passed 32 tests, including a non-zero chart pixel-offset case. The production frontend and embedded Go build also passed.
- Target Chromium reset the viewport, selected a visible resource at canvas pixel approximately (1054.8, 429.5), cleared selection, and drew a 130-pixel box centered on that visible resource. The resulting selected-resource halo matched the same resource ID, with no page or console errors.
- The integrated laboratory remains healthy with 11 running nodes, 10 active network objects, 21 placements and 25 connections. Database integrity remained ok.

### Attachment Label and Selection Correction — r8

| Field           | Value                                                                    |
| --------------- | ------------------------------------------------------------------------ |
| Candidate       | topology-organize-20260813T083157Z-r8                                    |
| Source commit   | d462a33a9d82e51e14e096460b6d50edd2852abe                                 |
| Artifact digest | sha256:c0d8e3cd933e529d0caa9eadda3e0a85d66ba0f8f16939193812006929b81555  |
| Contract digest | sha256:71d5721ac41dd2d55ccdd7934d4ab0a77c7c77ecd602df43f24af89e1699668a  |
| Built at        | 2026-08-13T08:31:57Z                                                     |
| Migration       | no database migration; schema migration version remains 16               |
| Rollback        | /var/lib/netlab/rollback/topology-organize-20260813T083157Z-r8-predeploy |

- The BusyBox connection is a network attachment from BusyBox业务客户端 interface eth0 to 服务区交换机 object port busybox. R8 renders attachment labels as eth0 ↔ 端口 busybox so the second token is not mistaken for the remote node interface name; the full accessible identity remains BusyBox业务客户端:eth0 ↔ 服务区交换机:busybox.
- Every normalized connection now has an 18-pixel transparent selectable path, trimmed away from endpoint node bodies so resource clicks remain available. Connection targets take priority over blank-canvas box selection, and keyboard Enter/Space also select the connection.
- Focused connection, canvas and interaction coverage passed 36 tests. Additional chart stability coverage passed after graph-coordinate reads were made tolerant of transient ECharts pipeline rebuilds, and the production frontend plus embedded Go build passed.
- Target Chromium clicked the real midpoint of attachment 019ff9d3ed51-cb1baa6c1cebea640c17. The line entered selected state, displayed 已选 1 项, opened the 网络附件 inspector with 删除附件 available, and emitted no page or console errors.
- The integrated laboratory remains healthy with 11 running nodes, 10 active network objects, 21 placements and 25 connections. Database integrity remained ok and no error-priority service journal entries appeared after deployment.

### Grouped Drag and Endpoint Label Correction — r9

| Field           | Value                                                                      |
| --------------- | -------------------------------------------------------------------------- |
| Candidate       | `topology-organize-20260813T090555Z-r9`                                    |
| Source commit   | `b8dc88f418c3877a473a5e3fe8d9698f25791548`                                 |
| Artifact digest | `sha256:f75b9ca72c34f7b3ac8cc563b61576bf7ef71c9b65389df7f4738b56ef3a9083`  |
| Contract digest | `sha256:71d5721ac41dd2d55ccdd7934d4ab0a77c7c77ecd602df43f24af89e1699668a`  |
| Built at        | `2026-08-13T09:05:55Z`                                                     |
| Migration       | no database migration; schema migration version remains `16`               |
| Rollback        | `/var/lib/netlab/rollback/topology-organize-20260813T090555Z-r9-predeploy` |

- R9 keeps every selected topology resource moving in real time while one member is dragged. The ECharts graph adapter updates the complete selected group and refreshes connected edge layouts without changing the existing batch placement commit contract.
- Every connection adjacent to any resource in the active drag group receives a dedicated focus-colored glow during the drag. This adjacency is calculated from normalized connection endpoints and does not depend on the box-selection rectangle.
- The previous centered combined edge label is replaced by source and target endpoint labels positioned along their respective ends of each connection. Target Chromium rendered 50 endpoint labels for 25 connections, preserving the correct resource ID and port name at each end.
- Focused canvas and chart coverage passed 31 tests, including grouped graph movement, drag adjacency, and endpoint-label ownership. The production frontend and embedded Go build passed, along with Prettier checks for changed files and `git diff --check`.
- Target Chromium box-selected two existing network objects and dragged one by 90 horizontal and 45 vertical pixels. Both resources moved by exactly the same delta during the interaction, eight adjacent connections were highlighted, and no page or console errors occurred. The two placement coordinates were restored through the authoritative batch placement API after validation.
- The integrated laboratory remains healthy with 11 running nodes, 10 active network objects, 21 placements and 25 connections. `/readyz`, `/healthz`, database integrity, migration version 16, error-priority service journal checks, and rollback package checksum verification all passed.
