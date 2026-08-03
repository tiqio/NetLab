# MCP Additions

## `netlab.topology.set_positions`

Input: laboratory ID, expected laboratory revision, 1–100 placement items, idempotency key.

Output: updated laboratory revision and placements. Semantics match `updateTopologyPlacements`; no browser-local
viewport or route preferences are exposed.

## `netlab.links.reconnect`

Input: link ID, expected revision, retained endpoint ID, replacement endpoint ID, idempotency key.

Output: durable operation task. The original link remains authoritative until the task succeeds; terminal failure
includes bounded structured rollback information.
