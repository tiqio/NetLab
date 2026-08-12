# Target Chromium Acceptance — 2026-08-12

## Focused Journeys

- `NETLAB_ACCEPTANCE_SCOPE=topology-unification` ran `network_path_recovery_concurrency.spec.ts` and `network_path_recovery.spec.ts` against `10.72.1.7`.
- Result: PASS, 2/2 focused Playwright journeys.
- The journeys covered recovery diagnostics, structured warnings, VLAN editing, the durable workload panel, API/MCP concurrency and temporary-resource cleanup.

## R22 Real-Traffic Highlight

| Observation | Value |
|-------------|------:|
| Fingerprint count | 2 |
| Matched packets | 2 |
| Matched bytes | 168 |
| Filter observations | 2 |
| Workload attempts | 2 |
| Workload successes | 2 |
| Workload failures | 0 |
| Workload matched bytes | 474 |

- Active overlay attributes: `active=true`, `observations=2`, `recent=2`, `lingering=2`.
- After the recent window expired: `recent=0`, `lingering=2`.
- After the bounded decay window expired: `recent=0`, `lingering=0`.
- This confirms non-zero durable counters and fingerprints remain independent from the visual active → lingering → decayed lifecycle.

## Evidence References

| Evidence | SHA-256 |
|----------|---------|
| `/tmp/netlab-r22-highlight/workload-counters.png` | `4dc4b5a8db879bb0425a1e5b6e53c3d20281459f07b383531b1c851135ea76e2` |
| `/tmp/netlab-r22-highlight/traffic-highlight-active.png` | `63af5e95a0b5b5bfd381553594de2b31a458f288d1685df5ad1473f7aad8601e` |
| `/tmp/netlab-r22-highlight/traffic-highlight-lingering.png` | `bea125651495dc5cd5525c8c58cb7ce4874c3238251a72c87fe53d0410a240b2` |
| `/tmp/netlab-r22-highlight/traffic-highlight-decayed.png` | `dd6a9f4844cc95794eed9a61875811b9bfb71db8231ea4d10fecd7b8afffaf13` |
| `/tmp/netlab-r22-highlight/evidence.json` | `f8d989066fe44016a594ee0a4ecba9c9b81719fbb6d79793e09a5133c2a82af3` |

- R21 focused browser artifacts remain under `/tmp/netlab-r21-browser-focused/`; R22 supersedes R21 for the final highlight behavior.
- The temporary real-traffic workload and filter were removed after screenshots were captured.
