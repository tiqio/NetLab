# Acceptance Harness

Run frontend interaction acceptance locally against the disposable real service:

```bash
NETLAB_ACCEPTANCE_PROFILE=local ./acceptance/frontend-acceptance.sh
```

Run the clean-baseline target-host suite (the URL must not contain credentials):

```bash
NETLAB_ACCEPTANCE_PROFILE=target-host \
NETLAB_ACCEPTANCE_BASE_URL=http://10.72.1.7:8088 \
  ./acceptance/frontend-acceptance.sh
```

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
