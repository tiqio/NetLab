# Deployment Candidate — 2026-08-11

## Candidate Identity

```text
Feature candidate: topology-visual-010-20260811T013126Z
Source commit SHA: 2a8a7961e1e3341d5c778f325f5a570272068506
Source branch: main
Source remote: git@github.com:tiqio/NetLab.git
Worktree clean before build: yes
Artifact path: /tmp/topology-visual-010-20260811T013126Z-netlabd
Artifact SHA-256: ac275806eb8bbb9f06aa1cf0f9949c3ad5b908ec2ab996506a13ed338bd8c100
Version: 010-topology-visual-unification
Contract digest: sha256:2e1a4279f16d7e6c282b741f06c52e75f1db210e91dbf4b99180bea5c9da808b
Build time: 2026-08-11T01:31:26Z
Latest repository migration: 0014_network_attachment_revision.sql
Deployment target: 10.72.1.7:18082
```

The candidate was built with `make build` from the clean source commit above. The production SPA was
embedded into the Go binary, the copied artifact passed `sha256sum -c`, its `release` output matched
the recorded version, candidate, contract digest and build time, and it passed `validate-config`
against `deploy/config/netlab.test.yaml` in the validation role. Build output is retained locally at
`/tmp/netlab-topology-visual-010-20260811T013126Z-build.log`.

## Build Command

```bash
VERSION=010-topology-visual-unification \
  CANDIDATE_ID=topology-visual-010-20260811T013126Z \
  CONTRACT_DIGEST=sha256:2e1a4279f16d7e6c282b741f06c52e75f1db210e91dbf4b99180bea5c9da808b \
  BUILT_AT=2026-08-11T01:31:26Z \
  make build
```

## Migration State

- Required repository migration after deployment: through `0014_network_attachment_revision.sql`.
- Target migration before deployment: `14`.
- Candidate introduces no migration after `0014_network_attachment_revision.sql`.

## Previous Artifact and Rollback Baseline

Before deployment, the target reported:

```text
Candidate ID: unified-link-009-20260810T162640Z-r8
Version: 009-unified-link-interaction
Binary SHA-256: 3d3511d1ee4c77baffa0386ab6a136b04654c8fc40c43543447faccda5f6efae
Contract digest: sha256:2e1a4279f16d7e6c282b741f06c52e75f1db210e91dbf4b99180bea5c9da808b
Build time: 2026-08-10T16:26:40Z
Service state: active
Health endpoint: ok
Migration: 14
```

Immediately before replacement, the live binary, SQLite online backup, service configuration and
template readiness document are copied to a candidate-specific directory under
`/var/lib/netlab/rollback/`. Rollback restores those recorded files, restarts `netlab.service`, waits
for `/healthz`, verifies the restored release identity and confirms migration `14`. No source file is
edited on the target.

## Local Gate Reference

The feature's focused Go, Vitest, static, Playwright and real local restart gates are recorded in
`validation/final-local.md`. Deployment proceeds only with the exact artifact recorded above; target
results and cleanup evidence are recorded separately in `validation/target-acceptance.md`.

## Installed Candidate

```text
Deployment time: 2026-08-11T01:42:02Z
Installed binary SHA-256: ac275806eb8bbb9f06aa1cf0f9949c3ad5b908ec2ab996506a13ed338bd8c100
Installed migration: 14
Service state: active
Health endpoint: ok
Release identity: matched candidate
Configuration release identity: matched candidate and installed digest
Template readiness candidate: matched candidate
Rollback directory: /var/lib/netlab/rollback/topology-visual-010-20260811T013126Z-predeploy
Rollback database integrity: ok
Rollback package size: 1.6G
```

The rollback directory contains the previous binary, SQLite online backup, service configuration,
template readiness document, service unit, release identity, upgrade-before placement and connection
summaries, and verified SHA-256 records. The database backup completed before any installed file was
replaced. An initial summary query used historical link column names and stopped safely before
installation; the query was corrected against the live schema, the backup passed `integrity_check`,
and the recorded candidate was then installed without changing target source files.

Rollback command:

```bash
rollback=/var/lib/netlab/rollback/topology-visual-010-20260811T013126Z-predeploy
install -m0755 "$rollback/netlabd" /usr/local/bin/netlabd
install -m0600 "$rollback/netlab.yaml" /etc/netlab/netlab.yaml
install -m0644 "$rollback/template-readiness.json" /etc/netlab/template-readiness.json
install -m0644 "$rollback/netlab.service" /etc/systemd/system/netlab.service
systemctl daemon-reload
systemctl restart netlab
curl -fsS http://127.0.0.1:18082/healthz
```

## Post-Acceptance Correction — r2

Investigation of two user-laboratory connections found that the node link was correctly waiting on an
Ubuntu endpoint whose desired and observed state were both `stopped`. The network-object link was
waiting because its lightweight L3 endpoint had been created with the feature-defined default four
unaddressed interfaces, while the Linux runtime incorrectly required every L3 interface to already
have an address. The same create path also persisted explicitly malformed CIDR configuration before
the runtime rejected it.

Commit `f509048c832f8644534077de0c64e8321c1bc0a7` aligns creation and runtime behavior: unaddressed L3
ports remain active and connectable until configured, while malformed explicit addresses and routes
are rejected before persistence across SPA, HTTP and MCP command paths.

```text
Candidate ID: topology-visual-010-20260811T021524Z-r2
Source commit SHA: f509048c832f8644534077de0c64e8321c1bc0a7
Artifact SHA-256: e776eed9b0ef5193137c7f6baec29ac74121d6ff36c43e93a3332286006584f3
Contract digest: sha256:2e1a4279f16d7e6c282b741f06c52e75f1db210e91dbf4b99180bea5c9da808b
Build time: 2026-08-11T02:15:24Z
Deployment time: 2026-08-11T02:16:14Z
Migration: 14 (unchanged)
Previous candidate: topology-visual-010-20260811T013126Z
Rollback directory: /var/lib/netlab/rollback/topology-visual-010-20260811T021524Z-r2-predeploy
```

The r2 rollback directory contains the previous binary, release configuration, template readiness,
service unit and affected-resource snapshots. No database migration or direct database edit was
performed. Restoring those files with the rollback command above returns to the first 010 candidate.

## Post-Acceptance Correction — r3

Commits `90fbea8` and `0e3c528` separate durable Traffic Filter session statistics from the short
topology-animation window and add a hold-to-pan Ctrl shortcut. The candidate keeps the existing
click-to-toggle hand tool while treating Ctrl as a temporary hand-tool press; releasing Ctrl,
changing page visibility or losing window focus releases the temporary mode.

```text
Candidate ID: topology-visual-010-20260811T024306Z-r3
Source commit SHA: 0e3c528951186bee8a3cb59c38ffd864cb29499a
Artifact SHA-256: 03c4cf394b57a59098afa29b5a1c9f281127d73c4001e150556c85f8d031120f
Contract digest: sha256:2e1a4279f16d7e6c282b741f06c52e75f1db210e91dbf4b99180bea5c9da808b
Build time: 2026-08-11T02:43:06Z
Deployment time: 2026-08-11T02:53:19Z
Migration: 15
Previous candidate: topology-visual-010-20260811T021524Z-r2
Rollback directory: /var/lib/netlab/rollback/topology-visual-010-20260811T024306Z-r3-predeploy
Rollback package size: 1.6G
Rollback database integrity: ok
```

The candidate was built after `main` was pushed to `origin`. The production build, 37 focused
topology tests, ESLint and Prettier passed locally. Before replacement, the live r2 binary,
configuration, template readiness document, service unit and a SQLite online backup were stored in
the rollback directory. Every rollback file passed `sha256sum -c`, and the database backup passed
`PRAGMA integrity_check` before installation.

Startup installed `0015_traffic_filter_statistics.sql`. The service, health endpoint, release
identity, configuration identity and template-readiness identity all matched r3. A real Chromium
session against `http://10.72.1.7:18082` observed the hand-tool button changing from
`aria-pressed=false` to `true` while Ctrl was held and returning to `false` after release. Deployment
did not change the two user laboratories, six nodes, two links or zero existing Traffic Filters.

Rollback for r3 stops the service, restores `netlabd`, `netlab.yaml`, `template-readiness.json` and
`netlab.service` from the recorded directory, replaces `/var/lib/netlab/netlab.db` with the online
backup after removing its WAL/SHM files, reloads systemd and restarts the service.

## Post-Acceptance Correction — r4

Commit `e6bac12` corrects QEMU console workspace behavior. QEMU serial remains one exclusive physical
channel and the disabled add-terminal tooltip now explains that multiple independent terminals require
a reachable SSH service. Multiple VNC tabs remain available, but only the active VNC renderer stays
mounted and connected; noVNC scales locally and no longer requests remote desktop resizing from QEMU.

```text
Candidate ID: topology-visual-010-20260811T031446Z-r4
Source commit SHA: e6bac120b24774851ccef7c5d036200283dd6e62
Artifact SHA-256: 39ccf99e66f71df82bee5b215432f71f56310d239615d4ce73a72e0e28ffcbb7
Contract digest: sha256:2e1a4279f16d7e6c282b741f06c52e75f1db210e91dbf4b99180bea5c9da808b
Build time: 2026-08-11T03:14:46Z
Deployment time: 2026-08-11T03:16:28Z
Migration: 15 (unchanged)
Previous candidate: topology-visual-010-20260811T024306Z-r3
Rollback directory: /var/lib/netlab/rollback/topology-visual-010-20260811T031446Z-r4-predeploy
```

The production build, 15 focused console/diagnostics tests, ESLint and Prettier passed before
deployment. Because r4 introduces no migration, the rollback package contains the r3 binary,
configuration, template-readiness document, service unit, release identity, service status and
resource summary; all recorded hashes passed `sha256sum -c`.

Target Chromium restored one serial tab and two VNC tabs for the running Ubuntu node. Each VNC tab
connected successfully when selected, switching back also reconnected successfully, exactly one VNC
canvas existed while a VNC tab was active, and zero VNC canvases existed while the serial tab was
active. The browser reported no page errors. The authoritative service remained healthy with no
error-level journal entries and unchanged user resource counts.

## Post-Acceptance Correction — Unlimited QEMU Capacity

Commits `efe2fbf` and `3b7f412` replace the original four-running-QEMU admission ceiling with the
`resources.max_running_qemu` operator setting. A value of `0` disables the count ceiling; negative
values are rejected and positive values retain finite admission control. Host-memory, disk,
per-node resource, interface-capacity and cgroup protections remain active independently of this
setting. Resource creation failures and asynchronous node-operation failures now open structured
warning dialogs in the topology workspace.

```text
Candidate ID: qemu-unlimited-20260811T071624Z-r2
Source commit SHA: 3b7f4121f1cf3faf270f957b2fdd7834837f4fe5
Artifact SHA-256: d2bfd22b133dde62677f94b9013f282ff0907a43aafa66e25eac6eeb75575b1d
Contract digest: sha256:2e1a4279f16d7e6c282b741f06c52e75f1db210e91dbf4b99180bea5c9da808b
Build time: 2026-08-11T07:16:24Z
Deployment time: 2026-08-11T07:25:14Z
Migration: 15 (unchanged)
Configured max_running_qemu: 0 (unlimited)
Rollback directory: /var/lib/netlab/rollback/qemu-unlimited-20260811T071624Z-r2-predeploy
```

The clean candidate passed `go test ./...`, `go vet ./...`, the focused topology Vitest suite,
production frontend build, ESLint and Prettier before deployment. The rollback directory contains
the previous binary and configuration, template readiness data, an online SQLite backup and verified
SHA-256 manifests. Both the backup and live database passed `PRAGMA integrity_check`.

On the target, laboratory `019feee704ee-c5b2bd57159bebe86bed` ran six QEMU nodes concurrently:
Ubuntu QEMU/QGA, VyOS, FancyWAN, FortiGate, Ruijie Router and Ruijie Switch. All six reported
`running` without `last_error`, and all three tested direct QEMU links reported `connected`. The
service remained active, `/healthz` returned `ok`, migration 15 remained installed, and the host
reported approximately 115 GiB of 125 GiB RAM available during acceptance.
