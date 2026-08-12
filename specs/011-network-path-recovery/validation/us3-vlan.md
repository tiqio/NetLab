# US3 VLAN Validation — 2026-08-12

## Gates

| Gate | Result |
|------|--------|
| Domain/runtime/reconcile/HTTP/MCP tests filtered by `VLAN|Vlan` | PASS |
| `NETLAB_PRIVILEGED_TESTS=1 go test -v ./tests/integration -run TestPrivilegedVLANAccessTrunkAndIsolation -count=1` | PASS |
| `NETLAB_PRIVILEGED_TESTS=1 go test -v ./tests/recovery -run TestVLANMembershipPersistsAcrossTenRuntimeRestarts -count=1` | PASS |
| `npm test -- --run src/features/topology/NetworkObjectEditor.test.ts src/features/diagnostics/DiagnosticsPanel.test.ts` | PASS, 10/10 |
| `npm run build` | PASS; existing chunk-size warning only |

## Target Result

- VLAN 10 and 20 access membership and a two-VLAN trunk were created through revisioned/idempotent API operations.
- Both runtime diagnostics have zero membership mismatches.
- The trunk is connected and kernel bridge readback matches desired membership exactly.
- No credentials, proprietary images or packet payloads are present in this evidence.
