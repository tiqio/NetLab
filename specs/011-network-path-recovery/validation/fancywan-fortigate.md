# FancyWAN and FortiGate Roles — 2026-08-12

- Candidate: `network-path-011-20260812T035729Z-us4-r20` from `44b0e1d007e1fa67fc69273f630763588d78fef7`.
- FancyWAN roles: management on `eth1`, WAN on `eth0`, LAN on `eth2`.
- FortiGate roles: management on `port1`, WAN on `port0`, LAN on `port2`.
- FancyWAN `eth2` is attached to core VLAN 10 access port `eth4`; FortiGate `port2` is attached to DMZ VLAN 20 access port `eth4`.
- Both nodes retain NetLab-owned connected attachments and report cable state `ready`.
- Neither image exposes an authorized guest control channel at validation time. Management therefore reports `prerequisite`, and data forwarding reports `unverified` rather than a false success.
- Role metadata contains no credentials, secrets, tokens or proprietary configuration.
