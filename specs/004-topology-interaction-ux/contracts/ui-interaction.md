# UI Interaction Contract

## Gesture precedence

1. Active modal or port chooser receives input first.
2. Active connection gesture receives port/node targets and Escape cancellation.
3. Pointer press on a resource body may become selection or resource drag after threshold.
4. Pointer press on canvas background may become pan or box selection after threshold.
5. Wheel over focused/hovered canvas zooms around pointer; controls outside canvas retain normal scrolling.

## Familiar connector journey

- Hovering or keyboard-selecting a connectable node reveals one high-contrast connector handle, following the
  familiar EVE-NG discovery pattern without permanently displaying every port.
- Dragging starts only from the connector handle or a visible port, never from the node body used for movement.
- During drag, the source node remains highlighted, a temporary line follows the pointer, valid target nodes gain
  a positive affordance, and invalid targets show a non-color-only rejection cue.
- Dropping on a target node resolves its available interfaces. One free interface is selected automatically;
  multiple free interfaces open a compact chooser listing stable interface names, drivers, and connection state.
- The chooser has explicit `Connect` and `Cancel` actions. Closing, Escape, outside click, focus loss, or target
  invalidation cancels without creating or deleting a link.
- Direct port-to-port drag bypasses the chooser when both endpoints are unambiguous.

## Drag commit

Pointer movement updates only local preview. Pointer release submits one placement batch. Focus loss, Escape,
pointer cancellation, laboratory switch, or revision conflict discards preview and reloads authoritative positions.

## Connection target

Dropping on a specific free port selects it. Dropping on a node with one free port selects it automatically.
Dropping on a node with multiple free ports opens a keyboard-accessible chooser. No link mutation occurs until a
valid target is confirmed.

## Existing-link actions

Selecting a link or opening its context menu exposes Inspect, Reconnect endpoint, Disconnect, and Edit local route.
Endpoint labels remain visible on hover/selection. Reconnect reuses the connector journey with one endpoint fixed;
the original link remains displayed until the reconnect task succeeds. Link suspend/resume and link-quality
controls are not introduced by this feature.

## Accessibility

Every pointer workflow has a keyboard path. Focus is visible. Resource kind and lifecycle are conveyed by symbol,
text, and status semantics rather than color alone. Reduced-motion preference disables nonessential animation.
