# NetLab Frontend Workspace

The NetLab SPA uses an EVE-NG-familiar information architecture without copying EVE-NG branding,
source code, icons, or proprietary assets.

## Workspace Regions

- **Top toolbar**: switch, create, duplicate, import, export, refresh, and delete laboratories; open
  templates, automation, and the command palette.
- **Device palette**: search QEMU and Docker templates or add PC, bridge, NAT bridge, layer-2 switch,
  and layer-3 switch resources.
- **Topology canvas**: ECharts graph with pan, zoom, selection, local placement, lifecycle semantics,
  links, network objects, and Traffic Filter overlays.
- **Inspector**: authoritative desired/observed state, revision, runtime identity, interfaces, resource
  limits, guest commands, port mappings, live connections, structured problems, and diagnostics.
- **Operations drawer**: durable tasks, Telnet/VNC consoles, packet capture, and Traffic Filter paths.

## State Model

Laboratories, nodes, interfaces, links, network objects, tasks, captures, and Traffic Filters are shared
server state. Coordinates, viewport, visual groups, link routing, and panel sizes are browser-local and
stored under a versioned laboratory key. Moving a node never changes another browser's layout.

## Console and Capture

Telnet uses xterm.js and VNC uses noVNC. Closing a console renderer never stops its node. Capture panels
show task identity, quota, packet/byte counts, truncation, retention, and artifact links. Packet payloads,
console output, guest-command output, credentials, and cloud-init data are never stored in workspace
preferences.

## Keyboard

- `Ctrl/⌘+K`: open the command palette.
- `Escape`: clear topology selection or close the current transient interaction.
- Standard `Tab`, arrow, dialog, menu, and form behavior is retained through the shared UI primitives.

At narrow widths, secondary panels become sheets while the active topology and operation remain usable.
