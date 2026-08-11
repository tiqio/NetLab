# UI Contract: Recovery, VLAN and Traffic Workloads

## Runtime Health

- Network-object cards and connection lines use actual state, never desired state alone.
- Failed backing shows resource, phase, cleanup and operator guidance.
- Retry is available only when the problem is retryable; delete remains available for failed/pending/disconnecting resources.
- Reconciliation progress opens the shared task surface and converges through stream or refresh events.

## Diagnostics

- Diagnostics display desired and observed namespace/bridge backing, interfaces, IPv4/IPv6 forwarding, routes and VLAN membership.
- Mismatches identify the first missing or conflicting item rather than showing a generic failure.
- Vendor nodes separately display cable state, guest readiness, management reachability and proven data-path status.

## VLAN Editing

- Each L2 port exposes PVID and tagged VLANs.
- Invalid IDs, duplicates and contradictory membership are blocked before submission while preserving the draft.
- After save, the UI shows a pending state until observed membership matches; mismatch becomes failed with diagnostics.

## Traffic Workloads

- Users select a source, protocol, address family, destination, interval and timeout.
- The panel shows desired/actual state, attempts, successes, failures, received bytes and last error.
- A running process with zero successful exchanges is visibly degraded and is not labeled healthy.
- Associated Traffic Filters retain independent packet/byte/fingerprint totals and topology highlights decay without clearing totals.

## Accessibility and Localization

- All state, warnings and actions use Chinese product terminology and remain keyboard accessible.
- Color is never the only indicator for recovery, VLAN mismatch, workload failure or highlighted traffic.
