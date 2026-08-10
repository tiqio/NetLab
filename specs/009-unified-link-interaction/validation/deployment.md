# Deployment Candidate — 2026-08-10

## Candidate Identity

```text
Feature candidate: unified-link-009-20260810T074325Z
Source commit SHA: e20c127aaf758989873372604b10e3eb6f40d3e1
Source branch: main
Source remote: git@github.com:tiqio/NetLab.git
Worktree clean before build: yes
Artifact path: /tmp/unified-link-009-20260810T074325Z-netlabd
Artifact SHA-256: 606ea4be706e3e90d504acb4cc7b159cac09aa6627dfdb8569153576513324ff
Version: 009-unified-link-interaction
Contract digest: sha256:2e1a4279f16d7e6c282b741f06c52e75f1db210e91dbf4b99180bea5c9da808b
Build time: 2026-08-10T07:43:25Z
Latest repository migration: 0014_network_attachment_revision.sql
Deployment target: 10.72.1.7
```

The artifact was built with `make build` from a clean identified commit, copied to the path above,
verified with `sha256sum -c`, and passed `validate-config` against
`deploy/config/netlab.test.yaml`. The prebuilt installer computes and records the actual binary digest
during installation; the binary release payload intentionally leaves `binary_digest` empty before
that installation step.

## Build Command

```bash
make build \
  VERSION=009-unified-link-interaction \
  CANDIDATE_ID=unified-link-009-20260810T074325Z \
  CONTRACT_DIGEST=sha256:2e1a4279f16d7e6c282b741f06c52e75f1db210e91dbf4b99180bea5c9da808b \
  BUILT_AT=2026-08-10T07:43:25Z
```

## Migration State

- Required repository migration after deployment: through `0014_network_attachment_revision.sql`.
- Target migration state before deployment: not observed because the service is not listening and the
  available SSH key is not authorized.
- Target migration state after deployment: pending T087.

## Previous Artifact and Rollback

The latest repository-recorded target artifact is candidate `ui-simplify-theme-20260806T101629Z`
with SHA-256 `343f11cea4b5d30117db2f357db4880ee05200fb6b9efbd32ffada0d7a99ccab`.
This is the last known rollback identity, not a live verification of the current target filesystem.
Before replacement, the operator must copy the actual `/usr/local/bin/netlabd`,
`/etc/netlab/netlab.yaml`, and `/etc/netlab/template-readiness.json` to candidate-specific rollback
paths and record their digests.

Rollback restores those recorded files, runs `systemctl daemon-reload`, restarts `netlab.service`,
waits for `/healthz`, and verifies the restored release identity. No source file may be edited on the
target.

## Deployment Readiness

```text
ICMP reachability: PASS
TCP 22 reachability: PASS
TCP 8088 reachability: FAIL (connection refused)
SSH root authentication: FAIL (public key rejected)
Candidate deployment: PENDING
```

T087 requires an authorized target-host operator/session. Once available, upload only the recorded
artifact plus deployment scripts/configuration from this commit, use
`NETLAB_PREBUILT_BINARY=<uploaded-path> deploy/scripts/install.sh install`, and append the installed
digest, migration result, deployment time, service health, and previous-artifact paths here.
