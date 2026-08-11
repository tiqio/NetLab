# Target Acceptance — 2026-08-11

## Candidate and Environment

```text
Target: root@10.72.1.7 (eve-ng)
Base URL: http://10.72.1.7:18082
Candidate ID: topology-visual-010-20260811T013126Z
Product source SHA: 2a8a7961e1e3341d5c778f325f5a570272068506
Final acceptance harness SHA: ce1eca0
Artifact SHA-256: ac275806eb8bbb9f06aa1cf0f9949c3ad5b908ec2ab996506a13ed338bd8c100
Contract digest: sha256:2e1a4279f16d7e6c282b741f06c52e75f1db210e91dbf4b99180bea5c9da808b
Migration: 14
Deployment time: 2026-08-11T01:42:02Z
Service state after acceptance: active
Health after acceptance: ok
```

The target retained the two pre-existing user laboratories, `ddtest` and `aaaaaaaaa`. Acceptance ran
with `NETLAB_ACCEPTANCE_BASELINE_MODE=preserve`; no source file was edited on the target.

## Topology Acceptance

Final command profile:

```text
NETLAB_ACCEPTANCE_PROFILE=target-host
NETLAB_ACCEPTANCE_SCOPE=topology-unification
NETLAB_ACCEPTANCE_BASELINE_MODE=preserve
NETLAB_ACCEPTANCE_RESTART_COMMAND=ssh root@10.72.1.7 systemctl restart netlab
```

Final result:

```text
Run ID: topology-visual-010-20260811T013126Z-target-final2
Acceptance-unit: 30 passed
Playwright: 27 passed
Aggregate status: passed
Interaction results: 27
Cleanup attempted: true
Cleanup resources: 30
Cleanup baseline_restored: true
Cleanup remaining_count: 0
Cleanup remediation: []
```

Evidence and logs:

```text
Evidence: web/test-results/acceptance/010-target/topology-visual-010-20260811T013126Z-target-final2/evidence.json
Run summary: web/test-results/acceptance/010-target/topology-visual-010-20260811T013126Z-target-final2/run-summary.json
Summary: web/test-results/acceptance/010-target/topology-visual-010-20260811T013126Z-target-final2/summary.txt
Log: /tmp/netlab-topology-visual-010-20260811T013126Z-target-acceptance-final2.log
```

The run passed on desktop, standard and minimum projects. It covered ten concurrent browser/API/MCP
creation groups, twenty mixed resources at one preferred center, authoritative refresh persistence,
mixed connection states, semantic legend changes, three parallel links at 50%, 100% and 200% zoom,
Traffic Filter particle and direction-guide decay, service restart recovery, and terminal laboratory
cleanup.

## Preservation and Recovery

The candidate was compared with the pre-deployment SQLite summaries both immediately after install
and after all Playwright-triggered service restarts:

```text
Existing laboratories preserved: 2
Existing placements preserved: 10/10
Existing node links preserved: 2/2
Existing network attachments preserved: 1/1
Existing network-object links preserved: 1/1
Coordinate changes: 0
Connection endpoint or state changes: 0
Acceptance-prefixed laboratories after cleanup: 0
Captures after cleanup: 0
Traffic filters after cleanup: 0
```

Temporary animation state was not expected to survive restart. Authoritative placements, connection
identity, observed state, dynamic presentation inputs and user laboratory revisions remained intact.

## Failure and Leak Validation

The target also ran the precompiled integration validation binary from the exact product source SHA:

```text
Command: NETLAB_PRIVILEGED=1 NETLAB_OWNERSHIP_DOMAIN=topology-010-leak-20 CYCLES=20 ./netlab-integration.test -test.run Leak -test.count=1 -test.v
Result: PASS
Log: /tmp/netlab-topology-visual-010-20260811T013126Z-leak-20.log
```

All leak tests passed. The final domain resource counts returned to the user baseline: two
laboratories, six nodes, four network objects, ten placements, two node links, one network
attachment and one network-object link.

## Acceptance Harness Correction

The first topology-only run executed all 27 Playwright tests successfully and restored the baseline,
but its aggregate was marked failed because the global teardown recognized only the historical
`focused` scope and therefore incorrectly required full-suite interaction and image-version coverage.
The relative evidence path also resolved below `web/web/`. No candidate behavior failed and no target
resource remained.

The issue was fixed locally in commit `ce1eca0` by making complete target coverage conditional on the
`full` scope and resolving acceptance output paths before changing directories. Thirty acceptance-unit
tests and the artifact hygiene gate passed, the fix was pushed, and the final run above passed. The
deployed product artifact was not rebuilt because the correction changes only the local Playwright
acceptance harness; the target remained healthy and no rollback was necessary.

## Rollback Readiness

The previous 009 r8 binary and its database/configuration state remain at:

```text
/var/lib/netlab/rollback/topology-visual-010-20260811T013126Z-predeploy
```

The 1.6G SQLite online backup passes `pragma integrity_check`, and all recorded rollback file hashes
pass `sha256sum -c`. Rollback instructions are recorded in `validation/deployment.md`.

## Pending Connection Investigation and r2 Verification

On 2026-08-11, the two non-terminal connections in user laboratory `ddtest` were inspected without
editing the database or changing user coordinates:

```text
Node link 019fea54196f-2f873dbbf65e064032d7:
  Nginx:eth0 running ↔ Ubuntu:ens0 stopped
  desired_state=connected, observed_state=pending
  conclusion=expected while the user endpoint remains intentionally stopped

Object link 019fdb166900-c8df00aa592a31491d7d:
  Lightweight L3:eth0 ↔ Lightweight L2:eth3
  L3 error=layer-3 interface "eth0" requires an address
  conclusion=runtime/default-configuration mismatch
```

The L3 mismatch was fixed in commit `f509048`. Focused Go domain, application, HTTP, MCP and Linux
runtime tests passed, together with 23 focused Vitest tests, ESLint, Prettier and the production build.
Candidate `topology-visual-010-20260811T021524Z-r2` was deployed at `2026-08-11T02:16:14Z` with binary
SHA-256 `e776eed9b0ef5193137c7f6baec29ac74121d6ff36c43e93a3332286006584f3` and migration `14`.

After startup reconciliation:

```text
Lightweight L3 object: active
L3 ↔ L2 object link: connected
Stopped Ubuntu node link: pending, as expected
L3 last_error: cleared
```

A temporary target-host API scenario then proved that malformed CIDR input returns HTTP 400 without
persisting a network object, a default four-port unaddressed L3 reaches `active`, and its L3 ↔ L2 link
reaches `connected`; laboratory cascade cleanup completed successfully.

The three-backing runtime test was repeated on r2 for node link, network attachment and network-object
link at 1920×1080, 1366×768 and 1024×768. All three projects passed service restart recovery and
ownership cleanup:

```text
Run ID: topology-visual-010-20260811T021524Z-r2-three-backing
Result: 3 passed
Cleanup baseline_restored: true
Cleanup remaining_count: 0
Evidence: web/test-results/acceptance/010-r2-three-backing/topology-visual-010-20260811T021524Z-r2-three-backing/evidence.json
Log: /tmp/netlab-topology-visual-010-20260811T021524Z-r2-three-backing.log
```

The target finished healthy with only the two original user laboratories, zero captures and zero
Traffic Filters. The Ubuntu node was not started because doing so would alter user-owned desired state;
its connection will reconcile when the user starts that endpoint.
