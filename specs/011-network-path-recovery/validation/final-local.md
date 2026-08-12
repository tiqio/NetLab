# Final Local Validation — 2026-08-12

## Commands

- `go test ./... -count=1`: PASS.
- `go vet ./...`: PASS.
- `cd web && npm test`: PASS, 83 files and 392 tests. Initial parallel runs exposed scheduler noise in the wall-clock topology performance gate and an incorrect 20-sample P95 index; the gate now uses nearest-rank P95 and Vitest runs files serially while retaining the original 100 ms bound.
- `cd web && npm run lint`: PASS with 0 errors; 1064 pre-existing style warnings remain.
- `cd web && npm run format:check`: PASS after formatting the two existing US4 readiness files.
- `cd web && npm run build`: PASS; existing large-chunk warning only.
- `NODE_PATH=$PWD/node_modules npx playwright test --list ...`: PASS, six T113/T116 cases discovered across three viewports.
- `NODE_PATH=$PWD/node_modules NETLAB_ACCEPTANCE_REUSE_SERVER=1 ... playwright test ... --project desktop`: PASS, the two-browser HTTP/MCP concurrency journey and complete US1–US5 temporary-lab journey both passed.
- `NETLAB_PRIVILEGED=1 NETLAB_TRAFFIC_OBSERVATION_10M=1 go test ./tests/integration -run TestPrivilegedTenMinuteTrafficWorkloadObservation -count=1 -v -timeout 12m`: PASS in 595.98 seconds with 120 ICMP, HTTP and DNS exchanges and non-zero packet, byte and fingerprint counters.
- Focused export/import, security and restart-recovery tests: PASS.
- Post-gate service restoration: PASS, local `netlab.service` active and `/healthz` returned `{"status":"ok"}` with no traffic-test namespace, server or capture residue.

## Coverage

- Recovery, dual-stack routing, VLAN membership, device roles and workload lifecycle all have unit, contract, recovery or privileged acceptance coverage.
- Security coverage rejects non-HTTP URL schemes, preserves argv boundaries, rejects secret-bearing role metadata and enforces supported source capabilities.
- Export/import and duplicate preserve forwarding flags, VLAN configuration, device roles and remapped workload sources.
- Temporary-lab Playwright journeys cover HTTP/MCP concurrency, workload lifecycle, VLAN configuration, role metadata, export and cleanup ownership.
