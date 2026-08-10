# Acceptance Harness

Run frontend interaction acceptance locally against the disposable real service:

```bash
NETLAB_ACCEPTANCE_PROFILE=local ./acceptance/frontend-acceptance.sh
```

Run only the topology-visual-unification milestone in a dedicated acceptance laboratory namespace:

```bash
NETLAB_ACCEPTANCE_PROFILE=local \
NETLAB_ACCEPTANCE_SCOPE=topology-unification \
NETLAB_ACCEPTANCE_REUSE_SERVER=1 \
NETLAB_ACCEPTANCE_RESTART_COMMAND='<command that restarts the local candidate and waits for /healthz>' \
  ./acceptance/frontend-acceptance.sh
```

This scope runs mixed connection visuals, 50%/100%/200% parallel-link keyboard selection,
Traffic Filter particle/guide decay, 20-resource authoritative placement, ten concurrent
browser/API/MCP creation groups, service restart recovery, and terminal laboratory deletion. Every
test creates a uniquely prefixed acceptance laboratory, and child resources are recorded only
against that laboratory for cascade cleanup.

Run the clean-baseline target-host suite (the URL must not contain credentials):

```bash
NETLAB_ACCEPTANCE_PROFILE=target-host \
NETLAB_ACCEPTANCE_BASE_URL=http://10.72.1.7:8088 \
  ./acceptance/frontend-acceptance.sh
```

For feature 009 final acceptance, preserve evidence in a candidate-specific directory and bypass any
management-network proxy for both the remote address and loopback:

```bash
NETLAB_ACCEPTANCE_PROFILE=target-host \
NETLAB_ACCEPTANCE_BASE_URL=http://10.72.1.7:8088 \
NETLAB_ACCEPTANCE_OUTPUT_DIR=test-results/acceptance/009-<candidate-id> \
NO_PROXY=10.72.1.7,127.0.0.1,localhost \
no_proxy=10.72.1.7,127.0.0.1,localhost \
  ./acceptance/frontend-acceptance.sh
```

The full target profile includes the unified direct-port, plus/keyboard, four-port, contention, live
runtime, capture/Traffic Filter, restart/recovery, zoom, and cleanup journeys. Do not replace it with a
focused run for final sign-off. Record the output path with the candidate SHA and artifact digest.

For an explicitly focused diagnostic rerun, set `NETLAB_ACCEPTANCE_SCOPE=focused`. This preserves
schema, cleanup, release-identity, and unknown-interaction validation while deferring complete
interaction and template-version coverage to the subsequent full target-host run.

Use `NETLAB_ACCEPTANCE_FAILURE_INJECTION=after-runtime-create` for the controlled cleanup proof. Evidence is
written below ignored `web/test-results/acceptance/`; the artifact check rejects captures, images, credentials,
console payloads, guest output, and other prohibited runtime data.

Run `qemu-acceptance.sh` on the single-host installation after an operator has legally registered at
least four available QEMU image/template versions. The harness never downloads or registers images.

```bash
NETLAB_BASE_URL=http://127.0.0.1:8088 ./acceptance/qemu-acceptance.sh
```

Run the controlled hot-add failure acceptance directly on the host as root. It temporarily fills an
unused QEMU Root Port, verifies full interface/TAP/QMP/ownership compensation, restarts only the test
node to clear the injected devices, and proves that a retry reuses the same lowest free interface slot.

```bash
NETLAB_BASE_URL=http://127.0.0.1:8088 ./acceptance/qemu-hotadd-rollback.sh
```

Run the isolated lifecycle failure matrix without modifying the active service, database, images, or
laboratories:

```bash
./acceptance/lifecycle-failures.sh
```

Run the mixed-runtime service-restart adoption scenario directly on the installed single host as root.
It requires an enabled `ubuntu-qemu` image binding and the local `busybox:latest` image, restarts
`netlab.service`, proves stable QEMU, Docker, and namespace runtime identities, and deletes its lab.

```bash
NETLAB_BASE_URL=http://127.0.0.1:8088 ./acceptance/t225-service-restart.sh
```

The restart scenario records placement and connection summaries before restart, requires exact
coordinate/state equality after restart, rejects orphan placements, and verifies that laboratory
deletion removes placements, links, network-object links, runtime ownership, namespaces, TAP/veth
devices, capture state, and console ownership.

For feature 009, the same restart scenario also verifies Link, NetworkAttachment, and NetworkObjectLink
identity/state/reservation equality, orphan-reservation cleanup, missing-reservation reconstruction,
failed-operation cleanup, and zero connection/task/runtime residue after laboratory deletion.

Run the complete operator-image scenario after registering one legal QEMU image plus the official
`busybox:1.36.1` and `ubuntu:24.04` OCI images. It creates one ten-node laboratory with four QEMU,
two digest-pinned Docker, and four namespace nodes; verifies cloud-init/QGA, consoles, CPU quota,
live linking/capture, shared state, service-restart adoption, and terminal cleanup. Set
`NETLAB_OPERATOR_REBOOT_PREPARE=1` to preserve the lab and write a marker for a supervised host reboot.

```bash
NETLAB_BASE_URL=http://127.0.0.1:8088 ./acceptance/operator-image-acceptance.sh
```

Exit code `77` means suitable operator images are missing. Set
`NETLAB_ACCEPTANCE_ALLOW_REBOOT=1` only during a supervised maintenance window. In that mode the
harness writes a marker containing resource IDs and exits `75`; the operator reboots, verifies the
resources and recovery task through the API, then reruns without the flag for cleanup and validation.
