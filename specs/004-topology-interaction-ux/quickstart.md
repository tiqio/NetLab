# Quickstart: Validate Topology Interaction UX

## Prerequisites

- Local Go/Node toolchain and browser dependencies installed.
- For live link/reconnect validation: designated single-host deployment with legal operator images and clean lab
  baseline. Credentials remain out of band.

## Local validation

1. Run formatting, type checking, frontend unit/component tests, and Go unit/contract tests:

   ```bash
   make test
   npm --prefix web run format:check
   npm --prefix web run test
   npm --prefix web run test:acceptance-unit
   npm --prefix web run build
   ```

2. Run interaction-controller tests for click/drag thresholds, pan, box selection, connection cancellation, port
   chooser, focus loss, and keyboard equivalence.
3. Run repository/API tests for placement batches, revision conflicts, idempotency, event ordering, deleted-resource
   cleanup, and snapshot inclusion.
4. Run browser journeys at 1920×1080 and 1024×768 for wheel zoom, blank-canvas pan, group drag, refresh persistence,
   link creation, node-body target selection, disconnect, and reconnect failure recovery.

   ```bash
   NETLAB_ACCEPTANCE_PROFILE=local bash acceptance/frontend-acceptance.sh
   ```

Expected: all commands pass; one completed drag creates one durable placement mutation; browser-local viewport and
manual link route never appear in server responses or events.

## Concurrent-client validation

Open two browser contexts on the same laboratory and use HTTP or MCP automation as a third client. Move nodes,
start a node, and reconnect a link. Verify all clients retain their sessions and converge within five seconds on
the same placements, endpoints, task states, and observed node state. Verify viewport and manual route preferences
remain local.

## EVE-NG familiarity validation

Using the designated EVE-NG host only as a behavioral reference, compare the primary connection journey without
retaining credentials, screenshots, minified source, or proprietary assets. Verify that a user familiar with
EVE-NG can discover the connector on node hover, drag to a target, select an interface, confirm, inspect endpoint
labels, and open link actions without documentation. Also verify NetLab-specific improvements: one-port targets do
not require a dialog, task state is visible, API/MCP mutations do not invalidate browser sessions, and failed
reconnect preserves the original link.

## Dense-topology validation

Load a legal synthetic 100-node/200-link topology. Record wheel, pan, drag, selection, port-hover, and route-edit
feedback timing. Expected: at least 95% visible feedback within 100 ms and no freeze over 250 ms in at least 95%
of measured interaction windows.

## Target-host live reconnect

Create two supported running nodes with a live link. Reconnect one endpoint successfully, then repeat with injected
validation failure, runtime failure, cancellation, timeout, and service restart. Expected: old endpoints remain
until success; every unsuccessful task restores the original link; snapshots/events converge; cleanup reports zero
owned leaks.

```bash
NETLAB_ACCEPTANCE_PROFILE=target-host \
NETLAB_ACCEPTANCE_BASE_URL=http://10.72.1.7:8088 \
bash acceptance/frontend-acceptance.sh
```

## Artifact hygiene

Run the frontend artifact scanner after browser tests. Retain only schema-valid sanitized metadata. Never retain
screenshots or traces containing credentials, console output, packet payloads, guest output, proprietary image
content, or deployment secrets.
