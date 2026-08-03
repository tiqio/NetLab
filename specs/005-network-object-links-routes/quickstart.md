# Quickstart: Validate Object Links and Docker Routes

## Preconditions

- Work locally in `/home/dd/netlab` with a clean Git worktree.
- Linux privileged tests require network namespace, veth, Docker, and packet capture capabilities.
- Use only approved public test images pinned by digest. Do not add images, credentials, captures, or
  target-host secrets to Git.
- Follow the broader validation sequence in `specs/001-network-simulator-platform/quickstart.md`.

## 1. Focused Local Validation

Run the narrowest affected tests first:

```bash
go test ./internal/domain/... ./internal/store/sqlite/...
go test ./internal/app/reconcile/... ./internal/api/http/... ./internal/api/mcp/...
go test ./tests/contract/...
```

Run frontend validation:

```bash
cd web
npm test
npm run build
npm run lint
npm run format:check
cd ..
```

## 2. Privileged Link Lifecycle

The integration test creates three network-object namespaces and two durable links:

```text
endpoint-a -- sw1:swp1 <-> sw2:swp1 -- sw2:swp2 <-> sw3:swp1 -- endpoint-b
```

Verify:

1. Each durable link owns exactly one veth pair and no per-link host bridge.
2. ICMP, TCP, and UDP pass bidirectionally across the complete path.
3. A second link on different ports remains separately selectable and does not merge with the first.
4. Creating another resource on an occupied port fails atomically.
5. Live deletion stops only the selected path, removes both veth ends, releases both reservations, and
   allows immediate reconnection without restarting objects or endpoint nodes.

Run the repository's privileged integration suite using its documented privilege wrapper, followed by:

```bash
go test ./tests/integration/... ./tests/recovery/...
```

## 3. Capture and Traffic Filter

1. Start a capture with `source_type=network_object_link` on the first link.
2. Generate traffic in both directions and confirm one capture records nonzero packets/bytes without
   duplicate accounting.
3. Start a Traffic Filter scoped only to that link; confirm its exact line highlights within 500 ms and
   reports `a_to_b` and `b_to_a` correctly, or explicit `ambiguous` when direction cannot be proven.
4. Send traffic over a parallel link and confirm the idle link remains unmarked.
5. Delete the captured link and confirm capture completion reason `link_deleted`, retained metadata,
   no ghost topology line, and no continuing observation updates.

## 4. Docker IPv4/IPv6 Routes

Create two Docker endpoints behind the multi-object path. Configure addresses and routes entirely through
the normal node creation/settings contract, for example:

```json
{
  "name": "eth0",
  "addresses": ["10.10.1.2/24", "2001:db8:1::2/64"],
  "routes": [
    { "destination": "10.30.0.0/24", "gateway": "10.10.1.1", "metric": 100 },
    { "destination": "2001:db8:3::/64", "gateway": "2001:db8:1::1" }
  ]
}
```

Verify before readiness:

- the exact declared IPv4 and IPv6 routes exist;
- routed ICMP, TCP, and UDP succeed without manual `nsenter` or `ip route` configuration;
- stop/start and service recovery restore the same routes;
- removing a declared route while stopped removes only that managed route on next start;
- invalid family, prefix, gateway, duplicate, conflict, and ambiguous egress produce actionable errors.

## 5. Recovery and Leak Checks

Exercise service termination at each boundary: after durable create, after one veth end moves, during
capture, during deletion, and during route replacement. Restart and verify desired state converges.

After each scenario, assert there are no unowned/stale NetLab veths, reservations, capture processes,
namespace handles, tasks, or managed routes. Unrelated host interfaces and routes must remain unchanged.

## 6. Full Local Gate and Milestone Commit

```bash
go test ./...
cd web && npm test && npm run build && npm run lint && npm run format:check && cd ..
git status --short
```

Commit each independently testable milestone only after its focused gate passes. Do not deploy a dirty
worktree.

## 7. Candidate Build and Target Validation

From a clean commit:

```bash
SOURCE_SHA="$(git rev-parse HEAD)"
# Build using the repository's established release command.
sha256sum <candidate-artifact>
```

Record source SHA, artifact digest, SQLite migration state, UTC deployment time, and local results. Deploy
that immutable artifact to `10.72.1.7` using the established deployment scripts; never edit source there.

On the target, repeat the multi-object path, parallel-link, live-delete, capture, Traffic Filter direction,
Docker IPv4/IPv6 route, service restart, host recovery, and leak checks. Save only redacted test evidence.

If validation fails, restore the previously recorded artifact and migration-compatible state, verify
service health/topology recovery, then fix and recommit locally before redeployment.
