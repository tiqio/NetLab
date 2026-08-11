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
