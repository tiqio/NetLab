# US5 Stable Traffic Validation — 2026-08-12

## Milestones

- `2dad58b`: durable workload domain and migration 0016.
- `40c1720`: revisioned workload repository and aggregate transactions.
- `0e200db`: idempotent create/start/stop/delete durable commands.
- `04655b1`: argv-safe namespace, Docker and QGA runtime adapters.
- `2c73d1f`: scheduler, restart recovery and real-capture correlation.
- `7337be7`: HTTP, MCP, stream constants and service wiring.
- `e21695b`: Pinia convergence, localized panel, aggregates and topology overlay.

## Local Gates

| Gate | Result |
|------|--------|
| Migration/domain/repository tests | PASS |
| Durable command idempotency, cancellation and recovery | PASS |
| Namespace/Docker/QGA allowlist, timeout and output bounds | PASS |
| HTTP workload contract and structured failures | PASS |
| MCP lifecycle and aggregate parity | PASS |
| Scheduler success/failure aggregates and restart recovery | PASS |
| Traffic Filter correlation without duplicate packet counters | PASS |
| Frontend workload panel and topology workspace | PASS, 15/15 focused |
| Frontend production build | PASS; existing chunk-size warning only |
| Privileged 10-minute ICMP/HTTP/DNS observation | PASS, 120 exchanges per protocol over 595.98 seconds; packet, byte and fingerprint counters were non-zero |
| Workload restart recovery | PASS, durable running workload resumed after service restart |
| Browser workload journey | PASS, workload panel loaded the durable task and temporary-lab cleanup completed |

## Behavior

- Workload creation admits a durable task before inserting the workload, so idempotent replay cannot create orphan rows.
- Start, stop and delete enforce the requested revision and recover committed mutations without repeating them.
- Only fixed protocol allowlists are executed: `ping`, `curl` with HTTP-only validation, and `getent`.
- Namespace, Docker and QGA execution use argv boundaries, bounded timeouts and 64 KiB output limits; no workload command uses a shell string.
- Scheduler startup follows topology recovery and resumes durable `running` workloads.
- Attempts, successes, failures and matched bytes are durable; failures retain a structured problem.
- Workload-to-filter correlation reads real capture timestamps and never synthesizes or increments packet counters.
- The UI exposes honest degraded state when desired `running` has not converged to observed `running`.
- The privileged gate ran with local NetLab reconcilers paused so their orphan-resource scanners could not reclaim the isolated test namespaces; the system service was restored healthy after the gate and no test namespaces, servers or captures remained.
