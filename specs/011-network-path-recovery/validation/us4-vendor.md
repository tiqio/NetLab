# US4 Vendor Role Validation — 2026-08-12

## Candidate

- Candidate: `network-path-011-20260812T035729Z-us4-r20`.
- Source commit: `44b0e1d007e1fa67fc69273f630763588d78fef7`.
- Artifact digest: `sha256:c73e59e5a35457800a2bb786ecaed7deedea81b08766e2a44fc0193620bc627b`.
- SQLite integrity: `ok`; no migration added for role metadata.
- Rollback: `/var/lib/netlab/rollback/network-path-011-20260812T035729Z-us4-r20-predeploy`.

## Gates

| Gate | Result |
|------|--------|
| `go test -p 1 ./...` | PASS |
| `go vet ./...` | PASS |
| Device-role domain, HTTP and MCP tests | PASS |
| Vendor fixture and restart metadata tests | PASS |
| `DeviceReadinessPanel` and `DiagnosticsPanel` tests | PASS, 9/9 |
| Frontend production build | PASS; existing chunk-size warning only |

## Target Result

- All four vendor nodes persist explicit management and data-side roles without credentials.
- FancyWAN LAN is attached to core VLAN 10 and FortiGate LAN is attached to DMZ VLAN 20; both attachments are active and VLAN diagnostics have zero mismatches.
- The Ruijie Router/Switch direct link is connected and operationally up.
- Readiness independently reports cable, authorized guest channel, management prerequisite/reachability and data-path proof.
- Missing guest configuration is reported as `prerequisite` or `unverified`, never as a successful management or forwarding path.
