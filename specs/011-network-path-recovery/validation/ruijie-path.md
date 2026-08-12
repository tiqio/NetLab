# Ruijie Roles and Path — 2026-08-12

- Ruijie Router roles: LAN on `G0/0`, management on `G0/1`, WAN on `G0/2`.
- Ruijie Switch roles: trunk on `G0/0`, management on `G0/1`, client-facing on `G0/2`.
- The existing router `G0/0` to switch `G0/0` link is connected and both interfaces are operationally up.
- Router diagnostics report cable `ready` and an authorized guest control channel `ready`.
- Switch diagnostics report cable `ready`; its guest control channel is unavailable.
- Management addressing remains an explicit guest prerequisite and the client data path remains `unverified`; the UI/API do not collapse these states into cable success.
