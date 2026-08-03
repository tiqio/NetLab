# NetLab

NetLab is a single-host network simulator for QEMU virtual machines, Docker containers, Linux
network namespaces, bridges, NAT bridges, live links, consoles, packet capture, and traffic-path
observation. The Vue SPA, REST API, and MCP server share the same durable application commands and
SQLite state.

## Build and Run

```bash
make bootstrap
CANDIDATE_ID=dev-$(date -u +%Y%m%dT%H%M%SZ) VERSION=dev make build
./bin/netlabd -config deploy/config/netlab.example.yaml
```

Production requires KVM/QEMU, Docker, cgroup v2, iproute2, nftables, `dnsmasq`, systemd, capture
tools, and `xorriso`. Use `deploy/config/netlab-validation.yaml` only for loopback-isolated validation.

## Deployment Authority

The application intentionally has no account/password authentication. Bind the authoritative
instance only to a reviewed management network and install `deploy/nftables/netlab-management.nft`.
Exactly one externally reachable instance may be authoritative. Preview and validation instances
must use loopback or an isolated namespace, database, state directory, sockets, and ports.

Verify the deployed candidate and inventory with:

```bash
NETLAB_BASE_URL=http://127.0.0.1:8088 \
  CANDIDATE_ID=<candidate-id> \
  ./scripts/verify-production-authority.sh
```

See `docs/production-readiness.md` and `docs/release-governance.md` before exposing the service.

## Template Media

Built-in families are Ubuntu QEMU, VyOS, FancyWAN, FortiGate, BusyBox container, and Ubuntu
container. Images are operator-supplied and registered by immutable SHA-256 digest. Do not add image
contents, cloud-init payloads, licenses, credentials, or packet captures to the repository.

- Ubuntu, VyOS, and FancyWAN acceptance requires genuine family workloads and bootstrap material.
- FortiGate remains blocked until legally supplied media has an explicit license-review attestation.
- Substitute images may validate runtime mechanics only and must never be reported as genuine device
  support.
- QEMU acceptance reads `NETLAB_UBUNTU_QCOW2`, `NETLAB_VYOS_QCOW2`, and
  `NETLAB_FANCYWAN_QCOW2` or the documented operator image IDs without retaining image bytes.

Run `acceptance/operator-image-acceptance.sh` and `acceptance/qemu-acceptance.sh` on the target host.

## Validation

```bash
make lint
make test
make test-contract
make test-web
make test-security
make build

sudo -E NETLAB_PRIVILEGED=1 make test-integration
sudo -E NETLAB_PRIVILEGED=1 make test-recovery
sudo -E NETLAB_PRIVILEGED=1 CYCLES=100 make test-leaks
make test-e2e-local
NETLAB_ACCEPTANCE_PROFILE=target-host make test-e2e-target
```

Candidate-wide aggregation:

```bash
CANDIDATE_ID=<candidate-id> \
NETLAB_PRIVILEGED=1 \
NETLAB_RUN_BROWSER_ACCEPTANCE=1 \
./scripts/run-constitution-acceptance.sh

./scripts/validate-compliance.sh
```

Skipped privileged, browser, host-restart, or genuine-image gates retain a `blocked` release
conclusion with a repeatable rerun command. Evidence under `compliance/evidence/` is metadata-only.
The full procedure is in `specs/002-constitution-gap-closure/quickstart.md`.

## Interfaces

- SPA and REST: `/` and `/api/v1`
- MCP JSON-RPC: `/mcp`
- Ordered events: `/api/v1/events`
- Telnet/VNC console streams: `/api/v1/nodes/{nodeId}/consoles/...`
- Capture stream/Wireshark handoff: `/api/v1/captures/{captureId}/stream`

## Network Object Links

Lightweight PC, L2 switch, and L3 switch objects expose named namespace ports. Connect two free object
ports from the topology canvas to create a first-class direct veth link. Parallel links are independent
resources: select, inspect, capture, filter, or delete the exact line identified by its link ID and
human-readable `object:port ↔ object:port` label.

Deleting a connected object link is live and revision checked. The durable task stops dependent capture
and Traffic Filter workers, removes the owned veth pair, releases both port reservations, publishes the
ordered delete event, and lets every browser remove the line without a refresh. The released ports can
be reconnected immediately; deleting a network object cascades through its owned links.

## Capture and Traffic Filter

Capture sources support node interfaces, standard links, and network-object links. For an object link,
select the exact line and start Capture or use its context menu; the server resolves the namespace and
port, while clients receive only the durable link ID, bounded metadata, stream URL, counters, retention,
truncation, and completion reason. Deleting the source completes an active capture with `link_deleted`.

Traffic Filter accepts pcap-style expressions and can scope observations to exact object-link IDs.
Matching traffic highlights only those lines, reports `a_to_b`, `b_to_a`, or explicit `ambiguous`
direction, and decays after traffic stops. Parallel idle links must remain unmarked. Observations never
contain unbounded packet payloads.

## Docker Static Routes

Stopped Docker nodes can configure ordered IPv4 and IPv6 routes per interface in node Settings or the
REST/MCP settings contract. Each route declares a canonical destination CIDR, optional same-family
gateway, and optional nonnegative metric. The gateway must be reachable through an address configured on
that interface. Invalid destinations, family mismatches, duplicate/conflicting prefixes, unreachable
gateways, and negative metrics fail before node start with structured problems.

On start and recovery, NetLab creates the managed Docker veth endpoints, waits for a usable container
namespace and IPv6 DAD, applies the exact owned route set before readiness, and removes stale routes that
NetLab previously owned. Stop/start and service recovery reapply the declaration; unrelated kernel,
Docker, connected, or operator routes are not removed. Route edits require the node to be fully stopped.

## Recovery Limits

Service restart adopts or recreates owned object links and Docker endpoints from durable intent. Host
restart follows each laboratory recovery policy. Capture processes cannot survive a host restart, but
retained metadata remains authoritative. The deployment remains single-host and does not provide Cisco
images, clustered scheduling, or application-level account/password authentication.

External contract deltas are maintained in `specs/002-constitution-gap-closure/contracts/` and
`specs/005-network-object-links-routes/contracts/`.
