# Quickstart Validation: NetLab Network Simulator Platform

This guide defines the end-to-end acceptance flow to run after implementation. It validates the
contracts in `contracts/`, the entities in `data-model.md`, and the outcomes in `spec.md`.

## 1. Acceptance Host Prerequisites

- x86_64 Ubuntu 24.04 or equivalent Linux with systemd and cgroup v2
- KVM and QEMU >= 8.2
- Docker Engine with API negotiation enabled
- `iproute2`, nftables, tcpdump or dumpcap, `xorriso`, curl, jq, and Wireshark on the operator workstation
- At least 4 host CPU cores, 16 GiB free RAM, and 100 GiB free disk for the reference workload
- Host firewall rules restricting access to trusted management networks; NetLab listens on all
  interfaces by default and provides no login authentication

Do not place commercial images, passwords, bootstrap secrets, or captures in the repository.

## 2. Build and Install

```bash
make bootstrap
make lint
make test
make build
sudo make install
sudo systemctl enable --now netlab
curl --fail http://127.0.0.1:8080/api/v1/capabilities | jq .
```

Expected: the service reports QEMU, Docker, bridge, NAT, namespace, capture, QMP, QGA, console, and MCP
capabilities plus configured quotas. Startup logs include a prominent unauthenticated-listener warning.

## 3. Register Images and Template Versions

Use legally obtained images. Import a QEMU image by upload or server-local path and record its SHA-256,
source, and license notes. Register immutable OCI digests for BusyBox and Ubuntu containers.

```bash
curl -sS -X POST http://127.0.0.1:8080/api/v1/images \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: image-ubuntu-reference-001' \
  -d @acceptance/fixtures/ubuntu-image-import.json | jq .
```

Expected: an operation task validates the image and publishes it only when checksum, format, and
license status pass. A deliberately wrong checksum is rejected before any node can start.

## 4. Shared-State and Idempotency Scenario

1. Open the SPA in two independent browser sessions.
2. Create a lab named `shared-state` with default `auto_restore` policy.
3. Use the API to repeat the same node-create request twice with one idempotency key.
4. In the second browser, change the lab name using its current revision.
5. Submit a stale revision from the first browser.

Expected: exactly one node exists, both browsers converge within 3 seconds, neither session is
invalidated, and the stale write receives a precondition/conflict response without overwriting state.

## 5. Reference Topology

Create one 10-node lab with at most 4 running QEMU nodes:

- QEMU: Ubuntu, VyOS, FancyWAN, and FortiGate or a license-safe placeholder when commercial validation
  is not authorized
- Containers: BusyBox and Ubuntu
- Lightweight nodes: dual-stack PC, layer-2 switch, layer-3 switch
- One NAT bridge

Validate static IPv4/IPv6, DHCPv4, DHCPv6, SLAAC, layer-2 forwarding, layer-3 forwarding, and NAT.
Expected: the complete topology starts without resource-limit failures and is visible identically via
SPA, REST snapshot, event stream, and MCP lab query.

## 6. Cloud-Init, Console, and Guest Commands

1. Attach node-scoped seed data to compatible Ubuntu, VyOS, and FancyWAN template versions.
2. Start each node and verify the expected initial configuration.
3. Open Telnet and VNC sessions where declared by the template.
4. Execute a bounded guest-agent command on a QGA-capable node.

Expected: seed data is isolated, reconnecting consoles does not change node state, and guest execution
returns exit status plus bounded output. An unsupported guest agent returns `capability_unsupported`.

## 7. Live Rewiring and NIC Hot-Plug

1. Generate continuous traffic between two running nodes.
2. Disconnect and reconnect their existing interfaces without stopping either node.
3. Hot-add a QEMU NIC with a supported driver, connect it, verify guest visibility, then hot-remove it.
4. Inject a failure between `device_add` and bridge attachment.

Expected: ordinary rewiring completes within 10 seconds in at least 95% of valid attempts. Hot-plug
reports each stage, confirms QMP events, and reconciles or compensates partial failure without an
unowned tap, netdev, or guest device.

## 8. Port Mapping and CPU Quota

1. Map an unused host TCP port to an SSH-capable guest and verify external connectivity.
2. Attempt the same host binding for a second node.
3. Delete the first mapping and verify the nftables rule disappears.
4. Configure a QEMU node with 2 vCPUs and a one-host-core CPU-time quota, then run sustained load.

Expected: the first mapping works, the collision is rejected with the conflicting resource, deletion
removes all rules, the guest sees 2 vCPUs, and measured CPU time remains within the configured tolerance.

## 9. Wireshark Capture and Traffic Filter

Start a retained capture and stream it to Wireshark:

```bash
curl --fail --no-buffer \
  http://127.0.0.1:8080/api/v1/captures/$CAPTURE_ID/stream \
  | wireshark -k -i -
```

Run a traffic filter for a known three-hop TCP/UDP flow. Expected: first matching traffic appears
within 5 seconds; all traversed test links are identified for 100 consecutive flows; capture metadata
contains counts, limits, retention, completion/truncation state, and an artifact handle. MCP returns
metadata and a bounded summary, never packet bytes.

## 10. Control-Service Restart Adoption

With all reference nodes running:

```bash
sudo systemctl restart netlab
```

Expected: QEMU, Docker, and namespace nodes continue running. Within 60 seconds the service adopts
owned resources, restores task/event visibility, and reports the same state through all clients.

## 11. Full Host Restart Recovery

1. Set one lab to `auto_restore` and another to `remain_stopped`.
2. Record each running set and reboot the acceptance host.
3. Observe recovery through the event stream and task API.

Expected: within 5 minutes every previously running node in the automatic lab is recreated or has a
specific terminal failure; the disabled lab starts no node automatically.

## 12. Export, Import, and Redaction

Export a lab containing image references, node configuration, bootstrap secrets, and a retained capture.
Validate the artifact against `contracts/lab-export.schema.json`, inspect it for sensitive material, and
import it under a new name.

Expected: topology, pinned template/image references, and non-secret configuration round-trip exactly.
No image bytes, credentials, bootstrap secrets, or capture payloads appear. Missing image versions are
reported and never silently substituted.

## 13. Failure and Leak Validation

Run the privileged recovery suite and 100-cycle resource test:

```bash
sudo -E make test-integration TARGET=local
sudo -E make test-recovery TARGET=local
sudo -E make test-leaks CYCLES=100 TARGET=local
```

Expected after the cleanup window: no owned QEMU/container/helper process, cgroup, tap/veth, bridge
attachment, namespace, nftables rule, port mapping, capture process, runtime socket, seed ISO, or expired
artifact remains. Unowned host resources are never deleted.

## 14. Contract and UI Validation

```bash
make test-contract
make test-web
make test-e2e
```

Expected: OpenAPI, event, MCP, and export schemas validate; every SPA mutation maps to a documented API
operation; Playwright confirms shared state, task progress, loading/error states, consoles, capture, and
topology editing. At least 90% of evaluation users complete topology creation and live capture without
assistance.

