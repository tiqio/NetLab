# Component Matrix VLAN Topology — 2026-08-12

## Revisioned Configuration

- Core L2 `019feee7054b-8381321b7de75f8849b3`:
  - `eth0`, `eth1`, `eth3`: VLAN 10 PVID, untagged.
  - `eth2`: tagged VLANs 10 and 20, no PVID.
- DMZ L2 `019feee70bd8-1f10c9efa71ea3b87936`:
  - `eth0`, `eth1`, `eth3`: VLAN 20 PVID, untagged.
  - `eth2`: tagged VLANs 10 and 20, no PVID.
- Trunk object link `019ff402e28c-cc3674d0ba37ff70c92e` connects core `eth2` to DMZ `eth2` and reports `connected`.

## Runtime Readback

- Both switch diagnostics report `active`, empty mismatch lists and identical desired/observed membership.
- Kernel `bridge -j vlan show` in the service mount namespace confirms access membership and VLAN 10/20 tagged membership on both trunk ports.
- VLAN 1 remains only on each bridge self-port; managed data ports no longer retain VLAN 1 membership.

## Isolation and Forwarding Fixture

- The privileged fixture completed 100 exchanges on each approved VLAN with at least 99% success.
- All unapproved cross-VLAN attempts were blocked while routing was disabled.
- The recovery fixture preserved exact VLAN membership through ten runtime restarts.
