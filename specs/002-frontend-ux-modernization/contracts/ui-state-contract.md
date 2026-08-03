# UI State Contract

## Purpose

Define the boundary between server-authoritative NetLab state, browser-local visual preferences, and
ephemeral renderer state. This contract prevents the modernized SPA from recreating the account/session
state isolation that the platform was designed to eliminate.

## Authoritative State

The following always comes from the REST API, ordered event stream, durable tasks, or diagnostic streams:

- Laboratories, revisions, recovery policy, nodes, interfaces, links, and network objects.
- Desired/observed lifecycle state, runtime identity, capabilities, resource limits, and latest problems.
- Port mappings, consoles, captures, traffic filters, templates, images, provenance, and validation state.
- Task acceptance, progress, cancellation, results, timestamps, errors, cleanup state, and audit data.

The SPA MUST NOT persist, override, or independently advance these states.

## Browser-Local State

Only the following may be persisted in the current browser:

- Node/network-object coordinates and pinned placement.
- Local visual groups and link route control points.
- Topology pan, zoom, and center.
- Panel sizes, collapsed state, and active bottom tab.

These values MUST conform to `workspace-preferences.schema.json`, be keyed by laboratory ID, and never be
sent as a laboratory mutation. A second browser or API/MCP client is not expected to share them.

## Ephemeral State

Selection, hover, open menus, drag targets, unsaved forms, active renderer instances, socket handles, and
in-flight submit guards remain in memory only. A refresh may discard them except that accepted mutations
must be rediscovered through their durable task IDs or authoritative resource state.

## Mutation Rules

1. Validate form input and capability/revision preconditions.
2. Create or reuse the operation's idempotency key where the API supports it.
3. Submit through the existing typed API client.
4. Display `submitting`, then the returned durable task identity and progress.
5. Apply ordered events to the authoritative projection.
6. Mark completion only from terminal task/resource state; never from optimistic appearance alone.
7. On revision conflict, preserve user input, refresh authoritative state, and offer compare/retry.
8. On an event gap, suspend unsafe incremental application and refresh the laboratory snapshot.

## ECharts Rules

- Use authoritative IDs as graph data IDs and browser-local coordinates as visual inputs.
- Drag completion writes placement preferences only.
- Link creation/disconnection and node operations call the mapped backend operation.
- Pending styling is an affordance and cannot replace authoritative desired/observed state.
- Lifecycle, failure, selection, and traffic state require non-color indicators.
- Chart instances, DOM nodes, observers, and ECharts event objects never enter Pinia persistence.

## Renderer Lifecycle

- Create one ECharts instance per mounted chart container and dispose it on unmount.
- Observe actual container dimensions and call resize after panel layout changes.
- Register and unregister chart events explicitly.
- Dispose xterm/noVNC sessions and transport handles when their panel closes, without mutating node state.
- Stop or cancel a capture/traffic filter only through its backend operation, not by merely closing a tab.

## Storage Hygiene

The storage adapter MUST reject keys or values representing credentials, tokens, bootstrap/cloud-init data,
guest command output, console content, packet bytes, image contents, or retained artifacts. Tests, screenshots,
traces, videos, and logs use synthetic metadata and redacted payloads only.
