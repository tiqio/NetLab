# NetLab Operations Guide

## Security Boundary

NetLab intentionally has no application login. The default listener is `0.0.0.0:8080`, so the host
firewall or an isolated management VLAN must restrict access before the service starts. Never expose
the API, WebSocket streams, artifact downloads, or `/mcp` directly to the Internet.

Example nftables policy (replace the trusted prefix):

```bash
nft add table inet netlab_host
nft add chain inet netlab_host input '{ type filter hook input priority 0; policy accept; }'
nft add rule inet netlab_host input tcp dport 8080 ip saddr 10.72.0.0/16 accept
nft add rule inet netlab_host input tcp dport 8080 drop
```

The service logs a prominent warning whenever it binds an unspecified address. Run it as the supplied
systemd unit so writable paths, capabilities, devices, and address families remain bounded.

## Installation

Required host tools are QEMU/KVM, Docker, `iproute2`, `bridge`, nftables, tcpdump or dumpcap, and
`xorriso`. Validate and install with:

```bash
sudo deploy/scripts/install.sh check
make bootstrap
make test-all
sudo deploy/scripts/install.sh install
systemctl status netlab
curl --fail http://127.0.0.1:8080/api/v1/capabilities
```

Configuration is read from `/etc/netlab/netlab.yaml`. Persistent state defaults to `/var/lib/netlab`,
runtime locks to `/run/netlab`, and built-in manifests to `/usr/local/share/netlab/templates`.

## Images and Licensing

Operators must supply every VM image they are legally entitled to use. NetLab never downloads
commercial appliance images from unofficial collections. Image records include source, immutable
digest, format, validation outcome, license status, and notes. An image cannot start when its checksum,
format, availability, or license review is invalid.

FortiGate and other commercial products require an explicit entitlement review. Keep image files,
activation material, cloud-init secrets, packet captures, and deployment credentials outside the
repository. Rotate any credential that has been disclosed through an interactive troubleshooting
session.

## Template Authoring

QEMU and Docker manifests live below the configured `template_dir`. A template family declares a
stable key, display name, runtime kind, immutable versions, defaults, capabilities, supported NIC
drivers, console modes, and runtime options. Add versions as data; do not add device-specific branches
to scheduling or topology code.

Before publishing a manifest:

1. Validate it against `templates/schema.json`.
2. Pin an OCI digest or a reviewed image digest.
3. Declare only capabilities verified for that version, such as cloud-init, QGA, VNC, or NIC hot-plug.
4. Run `go test ./internal/domain ./tests/contract`.
5. Restart NetLab and verify `/api/v1/templates`.

## REST and MCP Automation

The SPA, REST clients, and MCP tools use the same application services and shared SQLite state. Use an
`Idempotency-Key` for retried mutations and `If-Match` for revision-controlled changes. Long operations
return durable task IDs; poll `/api/v1/tasks/{id}` or use `netlab.tasks.get`.

MCP uses Streamable HTTP at `/mcp`. It validates `Origin`, returns typed tool errors, and never embeds
image bytes, packet bytes, secrets, or unbounded guest output. Capture tools return an opaque HTTP
stream or artifact handle. Restrict MCP with the same firewall policy as the REST API.

## Wireshark and Traffic Filters

Start a capture on an owned host interface and pipe its stream to Wireshark:

```bash
curl --fail --no-buffer \
  http://NETLAB_HOST:8080/api/v1/captures/CAPTURE_ID/stream \
  | wireshark -k -i -
```

Capture limits include concurrency, duration, per-session bytes, retention, and a global artifact
ceiling. Slow consumers are dropped instead of blocking reconciliation. Retained captures continue
after a viewer disconnects; non-retained sessions may stop when no consumer remains.

Traffic Filters compile structured address/protocol/port matches to BPF, fingerprint matching packets,
and aggregate observations by interface, link, direction, first/last time, packets, and bytes. A loop or
bidirectional correlation is reported as ambiguous rather than shown as a false linear path.

## Recovery

A control-service restart adopts owned QEMU processes, Docker containers, namespaces, sockets, and
network objects. A full host restart honors each laboratory policy:

- `auto_restore`: reconcile the previous desired running set.
- `remain_stopped`: convert running desired state to stopped before runtime restoration.

Startup concurrency is bounded separately for QEMU and other nodes. Unknown objects in a NetLab-owned
runtime directory are quarantined; unowned host resources are never deleted.

Back up and maintain state with:

```bash
sudo deploy/scripts/maintenance.sh backup
sudo deploy/scripts/maintenance.sh verify
sudo deploy/scripts/maintenance.sh cleanup
sudo systemctl stop netlab
sudo deploy/scripts/maintenance.sh restore /var/lib/netlab/backups/BACKUP.db
sudo systemctl start netlab
```

Exports are topology bundles, not backups. They exclude image bytes, credentials, bootstrap secrets,
and captures, and include a mandatory redaction report.

## Troubleshooting

- **API unavailable**: check `systemctl status netlab`, `journalctl -u netlab`, bind address, and firewall.
- **Template load failure**: verify `template_dir`, manifest permissions, YAML schema, and pinned images.
- **QEMU fails**: check `/dev/kvm`, QEMU version, image format, runtime directory, QMP socket, and cgroup.
- **Docker fails**: check Docker socket access, daemon API negotiation, image digest, and ownership labels.
- **Console unavailable**: verify the node is running and its serial/VNC Unix socket exists.
- **QGA command fails**: verify the guest agent is installed, running, and declared by the template.
- **Port conflict**: inspect the reported binding and `nft -a list table inet netlab`; never delete
  unrelated host rules.
- **Capture fails**: verify dumpcap/tcpdump capabilities, interface ownership, filter syntax, and quota.
- **Namespace/NAT fails**: check `ip netns list`, owned bridge names, forwarding sysctls, nftables, and uplink.
- **Recovery stalls**: inspect failed task details and quarantine; do not manually adopt unknown resources.

## Routine Validation

Run `make test-all` after upgrades. Privileged suites are enabled with `NETLAB_PRIVILEGED=1` only on an
isolated acceptance host with operator-supplied images and test uplinks.
