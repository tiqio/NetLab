# Frontend UX Modernization Host Validation

**Date**: 2026-07-27
**Host**: `10.72.1.7`

## Deployment

- Backed up the previous binary, systemd unit, and template manifests under the service state directory.
- Installed the current embedded-SPA server build, current QEMU/Docker manifests, and hardened systemd unit.
- `netlab.service` restarted successfully and remained active on the configured management port.
- Capability discovery reported single-host mode, no application authentication, QEMU/Docker/namespace
  runtimes, Telnet/VNC, MCP, capture, Traffic Filter, QMP hot-plug, QGA execution, port mapping, and CPU quota.

## Host Capabilities

- `/dev/kvm` is available; QEMU 8.2.x starts with KVM acceleration.
- Docker Engine, cgroup v2, netlink/bridge/nftables, `tcpdump`, and required namespace tools are available.
- The host has sufficient free memory and storage for the acceptance workload.
- Operator-registered official Ubuntu QGA, BusyBox, and Ubuntu container images remained metadata-authoritative;
  no image content was copied into the repository or report.

## Acceptance Results

- Runtime availability: PASS for QEMU adapter and Docker Engine.
- Isolated dual-stack/NAT suite: PASS for namespace creation, bridge attachment, DHCPv4, DHCPv6, IPv6 SLAAC,
  IPv4/IPv6 forwarding, nftables NAT, observed translation, capture, and cleanup.
- Standard operator-image acceptance: PASS with 10 nodes: four QEMU, two digest-pinned Docker, and four
  namespace PC nodes.
- Cloud-init seed mounting: PASS for Ubuntu, FancyWAN, and VyOS template paths using an operator-supplied
  Ubuntu QGA image.
- QGA and QMP: PASS for bounded guest commands and live virtio NIC hot-add/remove.
- Console metadata: PASS for reconnectable Telnet and VNC descriptors backed by owned Unix sockets.
- Live rewiring and diagnostics: PASS for QEMU-to-Docker connection, capture streaming/retention metadata,
  and Traffic Filter path observations.
- Port mapping and ZTP: PASS for an owned dynamic host TCP mapping reaching a guest service.
- CPU quota: PASS for a two-vCPU guest constrained to approximately one host core over the bounded measurement.
- Service restart adoption: PASS; mixed running resources returned to authoritative running state.

## Cleanup

- The temporary acceptance laboratory was deleted through the durable cleanup path.
- No acceptance QEMU process, NetLab Docker container, owned TAP/bridge interface, node cgroup, or temporary
  laboratory remained after cleanup.
- Temporary deployment and acceptance files were removed from `/tmp`.
- `netlab.service` remained active and capability discovery succeeded after cleanup.

## Post-Validation Hygiene

- A later audit found one imported acceptance laboratory in `delete_failed` because its already-absent bridge
  was treated as a deletion error, plus stale ownership rows, orphan capture files, and namespace-holding
  DHCP client processes from earlier privileged runs.
- A timestamped rollback bundle containing the database, service binary, configuration, and sanitized resource
  inventories was created under the protected service backup directory before remediation.
- The orphan namespace processes and host link were removed, all mutable laboratory/task/event/audit/
  idempotency/diagnostic/ownership history was cleared offline, and SQLite was checkpointed and vacuumed.
- Device templates, template versions, image metadata, and operator image files were preserved. The database
  reduced from approximately 161 MiB to less than 1 MiB before normal WAL activity resumed.
- Obsolete executable backups and packet-capture remnants were removed, leaving one intentional rollback
  bundle. The service restarted cleanly with no warning-level journal entries.
- Real-browser checks at desktop and minimum supported viewport confirmed the empty-workspace state renders
  without JavaScript errors and is ready for creating a new laboratory.

This report contains metadata only. It intentionally excludes credentials, bootstrap contents, guest-command
output, console transcripts, packet payloads, capture files, proprietary images, and sensitive traces.
