# Ubuntu QGA and VyOS Path — 2026-08-12

## Ubuntu QGA

- QGA and guest execution capabilities reported `ready`.
- QGA readback showed `ens1` with `172.16.0.2/30` and `fd16::2/64`.
- IPv4 routes included `10.20.20.0/24` and `10.30.30.0/24` through `172.16.0.1`.
- IPv6 routes included `fd20::/64` and `fd30::/64` through `fd16::1`.

## Authorized VyOS Repair

- VyOS QMP and serial capabilities were ready; QGA was unavailable and was not treated as guest readiness.
- The managed console automatically used target-local credentials. Credentials were never printed, copied to the worktree or included in evidence.
- Boot console output identified VyOS `2026.08.04-0035-rolling` and a functioning serial getty.
- The repaired configuration assigns:
  - `eth0`: `172.16.0.1/30`, `fd16::1/64`.
  - `eth1`: `10.20.20.254/24`, `fd20::fe/64`.
  - `eth2`: `10.30.30.254/24`, `fd30::fe/64`.
- A stale duplicate transit address on `eth2` was removed before commit/save.

## Path Results

All destinations passed `100/100` from Ubuntu through QGA execution:

| Family | Destinations | Result |
|--------|--------------|--------|
| IPv4 | `172.16.0.1`, `10.20.20.1`, `10.20.20.10`, `10.30.30.1`, `10.30.30.10` | PASS, 0% loss |
| IPv6 | `fd16::1`, `fd20::1`, `fd20::10`, `fd30::1`, `fd30::10` | PASS, 0% loss |

The evidence distinguishes physical connection, QMP/serial availability, QGA availability and verified guest routing instead of treating them as one state.
