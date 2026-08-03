# Quickstart: Constitution Gap Closure Validation

This guide validates one exact NetLab production candidate against the constitution gap-closure
specification. It intentionally separates local quality gates, target-host runtime validation, and
operator-only security or licensing attestations.

## 1. Prerequisites

- Go, Node.js, npm, QEMU/KVM, Docker, systemd, nftables, iproute2, `dnsmasq`, packet-capture tools,
  SQLite tooling, and `xorriso` are available where required.
- The target host has `/dev/kvm`, cgroup v2, sufficient CPU/memory/disk, and an approved management
  network.
- Operator-approved images are supplied outside the repository. Commercial images have documented
  entitlement review.
- SSH uses an agent or another out-of-band mechanism. Do not put passwords in shell history,
  environment examples, evidence, or command-line arguments.
- Set non-secret validation variables:

```bash
export TARGET_HOST=10.72.1.7
export BASE_URL=http://10.72.1.7:18082
export CANDIDATE_ID=constitution-gap-closure-rc1
export APPROVED_MANAGEMENT_CIDR=192.0.2.0/24
```

## 2. Validate Planning Contracts

```bash
jq empty specs/002-constitution-gap-closure/contracts/*.json
ruby -e 'require "yaml"; YAML.load_file("specs/002-constitution-gap-closure/contracts/openapi-delta.yaml")'
```

Expected:

- Every JSON schema parses.
- The API delta parses as YAML.
- No specification or contract contains credentials, image bytes, bootstrap secrets, private keys,
  packet payloads, or captures.

## 3. Build and Identify the Candidate

The worktree MUST be clean and the milestone MUST already be committed. Record the exact commit
before building; do not deploy a working-tree-only candidate.

```bash
test -z "$(git status --porcelain)"
export SOURCE_COMMIT="$(git rev-parse HEAD)"
make build
sha256sum bin/netlabd
go run ./cmd/netlab-compliance capture-candidate \
  --candidate-id "$CANDIDATE_ID" \
  --version "${VERSION:-dev}" \
  --binary bin/netlabd \
  --contracts specs/002-constitution-gap-closure/contracts \
  --output compliance/evidence/candidate.json
```

Expected:

- `SOURCE_COMMIT` identifies the focused milestone commit and `git status --porcelain` is empty.
- The evidence record identifies `CANDIDATE_ID`, release version, binary digest, contract digest, and
  build time when available.
- `GET /api/v1/capabilities` on the deployed candidate reports the same release identity.

## 4. Validate the Compliance Ledger

```bash
go run ./cmd/netlab-compliance validate \
  --ledger compliance/constitution-ledger.json \
  --deployment compliance/deployment-authority.json \
  --templates compliance/template-readiness.json \
  --evidence-dir compliance/evidence

go run ./cmd/netlab-compliance report \
  --ledger compliance/constitution-ledger.json \
  --acceptance compliance/evidence/current-candidate.json > compliance/report.txt
```

Expected:

- Every constitution obligation has one status, owner, evidence/exception reference, and next action.
- No verified finding depends on stale, partial, rejected, substitute-workload, or different-candidate
  evidence.
- The report has one conclusion and no contradictory top-level status.

## 5. Run Local Quality Gates

```bash
make fmt
make lint
make test
make test-contract
make test-web
make test-security
make test-frontend-artifacts
make build
```

Expected:

- Formatting and static analysis have no errors.
- Unit, contract, frontend, and security tests pass.
- Contract tests prove REST/MCP/event parity for node capabilities and existing mutations.
- Frontend tests map network-object `active` to a healthy active state and reconcile ordered state
  events without refresh or stale resurrection.

## 6. Verify One Authoritative Deployment

Deploy only the locally built artifact associated with `SOURCE_COMMIT`. Do not edit source files on
`10.72.1.7`. If validation fails, fix and test locally, create a new milestone commit, rebuild, and
redeploy.

Inspect the target before changing services:

```bash
ssh "$TARGET_HOST" 'systemctl list-units --type=service --all | grep -E "netlab" || true'
ssh "$TARGET_HOST" 'ss -ltnp | grep -E "netlab|:8088|:18080" || true'
ssh "$TARGET_HOST" 'find /etc/netlab /opt/netlab -maxdepth 3 -name "*.db" -o -name "*.yaml" 2>/dev/null'
```

Deploy the candidate using the project deployment procedure, then either retire the old preview
instance or bind it to an isolated validation boundary. Do not point two active services at one SQLite
database or runtime directory.

Validate:

```bash
scripts/verify-production-authority.sh \
  --host "$TARGET_HOST" \
  --base-url "$BASE_URL" \
  --candidate "$CANDIDATE_ID" \
  --inventory compliance/deployment-authority.json
```

Expected:

- Exactly one externally reachable instance is `authoritative`.
- Validation instances are loopback-only or otherwise isolated and have separate state directories.
- The production contract digest equals the candidate contract digest.
- Current documented routes, including runtime ownership and node capability queries, respond as
  specified.

## 7. Enforce and Test the Trusted Network Boundary

Apply the reviewed deployment policy with the approved CIDR; retain an out-of-band rollback session.
The implementation must provide the exact operator procedure in `deploy/nftables/`.

From an approved client:

```bash
curl --fail "$BASE_URL/healthz"
curl --fail "$BASE_URL/api/v1/capabilities"
```

From an unapproved network vantage point, attempt the application, MCP, console, and artifact ports.

Expected:

- Approved clients retain access.
- Unapproved clients cannot establish a connection to any management surface.
- The deployment inventory records policy verification without recording credentials.
- The project owner records a non-secret credential-rotation attestation.

## 8. Verify Network-Object State Parity

Create bridge, NAT, PC, L2, and L3 objects through the UI while observing REST, MCP, and the event
stream from separate clients.

Expected for each object:

1. Durable task transitions through queued/running to a terminal state.
2. Actual state transitions `provisioning -> active` or reaches a structured `failed` state.
3. `network_object.observed_state_changed` is ordered with the stored change.
4. SPA, REST snapshot, MCP query, diagnostics, and host state agree within 10 seconds.
5. The canvas labels `active` as healthy, not stopped/unknown.
6. Successful deletion removes the object and every owned helper, attachment, rule, address, and
   ownership record without refresh.

## 9. Verify Automatic NAT Networking

Create a NAT object with DHCPv4, DHCPv6, router advertisement, DNS, and an approved uplink. Attach an
Ubuntu QEMU node and one namespace PC without manually setting guest addresses.

Expected:

- The owned `dnsmasq` helper is active, adopted after service restart, and visible in diagnostics.
- Clients acquire expected IPv4/IPv6 addresses, routes, DNS, DHCP lease state, and SLAAC state.
- IPv4 and IPv6 connectivity works for 100 consecutive flows according to the configured mode.
- Repeated NAT reconciliation creates one owned masquerade rule, not duplicates.
- Cancellation/deletion removes the helper, lease/config files, bridge state, rules, and ownership.

## 10. Verify QEMU Capability Readiness

Start an Ubuntu node whose image includes and enables QGA, and another compatible image without QGA.

```bash
curl --fail "$BASE_URL/api/v1/nodes/$NODE_ID/capabilities" | jq .
```

Expected:

- QMP, serial, VNC, image, bootstrap, and QGA observations have independent state and revisions.
- The QGA-ready node reports `qga=ready` and bounded guest commands succeed.
- The image without QGA remains a healthy running node when QGA is optional, but reports
  `qga=unavailable` and actionable guidance.
- A template that requires QGA cannot be described as fully ready until QGA is observed ready.
- UI, REST, MCP, and events show the same capability state.

## 11. Verify Genuine Template Readiness

For each operator-approved image, record provenance and run create/start/bootstrap/console/health/
stop/restart/delete scenarios.

Required template records:

- QEMU: Ubuntu, VyOS, FancyWAN, FortiGate.
- Docker: BusyBox and Ubuntu.

Expected:

- Ubuntu, VyOS, and FancyWAN use genuine workloads for device-family acceptance.
- A substitute image may prove mechanics only and is labeled `mechanics_validated`.
- FortiGate is either genuinely validated with license-reviewed media or remains `blocked`/
  `accepted_exception` with owner, risk, expiration condition, and removal task.
- Running Docker image digests exactly match selected immutable versions.
- No image bytes or proprietary metadata requiring secrecy enters source or retained evidence.

## 12. Verify Failure, Cancellation, and Live Operations

Run the target-host integration suites and explicit failure injection:

```bash
sudo -E make test-integration TARGET=local
sudo -E make test-recovery TARGET=local
```

Exercise:

- concurrent revisions and idempotency replay;
- task cancellation during NAT helper provisioning, QEMU startup, capture, filter, and deletion;
- link endpoint removal and reconnect failures;
- QMP NIC add/remove and injected partial failure;
- capture/Wireshark stream cancellation and retention;
- Traffic Filter multi-hop correlation;
- port collision and resource-admission failures.

Expected:

- Every terminal failure includes resource/task identity, phase, retryability, cleanup, and operator
  action.
- No unexplained generic `not found` or unknown cleanup result remains.
- Partial failures compensate or leave a clearly owned retryable state.
- UI, REST, MCP, task, event, and audit outcomes are equivalent.

## 13. Verify Service and Host Recovery

With a mixed QEMU, Docker, namespace, link, NAT-helper, console, and capture topology running:

```bash
sudo systemctl restart netlab
```

Expected within 60 seconds:

- The service adopts owned resources and preserves ordered state visibility.
- NAT DHCP/RA helper ownership is adopted without duplicate helpers.
- Optional capabilities are re-probed and publish updated observations.

Then run the documented supervised host-restart scenario with one `auto_restore` and one
`remain_stopped` laboratory.

Expected within five minutes:

- Every previously running automatic resource is restored or has a specific terminal failure.
- The stopped-policy laboratory starts no node automatically.
- Recovery tasks and events are durable and visible to all clients.

## 14. Run the 100-Cycle Cleanup Gate

```bash
sudo -E make test-leaks CYCLES=100 TARGET=local
```

Expected after the cleanup window:

- Process, container, namespace, cgroup, TAP/veth, bridge, helper, rule, mapping, socket, seed, capture,
  artifact, and ownership counts match the recorded baseline.
- Remaining owned resource count is zero.
- No unowned host resource is modified or deleted.

## 15. Run Browser and Multi-Client Acceptance

```bash
make test-e2e-local
make test-e2e-target
```

Expected:

- Two browsers, REST, MCP, and event clients observe one shared topology and task history.
- Laboratory/node/link deletion disappears without refresh and cannot be restored by a stale response.
- Users complete template selection, topology creation, live rewiring, console access, capture, and
  Wireshark handoff across supported viewport and keyboard/pointer modes.
- At least 90% of evaluation users complete the primary workflow without assistance.

## 16. Generate Final Evidence and Release Conclusion

```bash
CANDIDATE_ID="$CANDIDATE_ID" \
NETLAB_BASE_URL="$BASE_URL" \
NETLAB_PRIVILEGED=1 \
NETLAB_RUN_BROWSER_ACCEPTANCE=1 \
scripts/run-constitution-acceptance.sh

go run ./cmd/netlab-compliance validate \
  --ledger compliance/constitution-ledger.json \
  --deployment compliance/deployment-authority.json \
  --templates compliance/template-readiness.json \
  --evidence-dir compliance/evidence
```

Expected:

- All mandatory gates are tied to the exact candidate.
- Cleanup baseline is restored and redaction reports zero prohibited content.
- Every unresolved item is `open`, `partial`, `blocked`, or an approved unexpired exception.
- The final report contains exactly one release conclusion and no stale contradictory summary.
