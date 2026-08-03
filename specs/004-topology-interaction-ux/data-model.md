# Data Model: Topology Interaction UX

## TopologyPlacement

Shared, durable position of a topology resource.

| Field | Type | Rules |
|---|---|---|
| `laboratory_id` | ID | Required; owning laboratory |
| `resource_id` | ID | Required; existing node or network object |
| `resource_type` | enum | `node` or `network_object` |
| `x` | decimal | Finite; bounded to configured canvas extent |
| `y` | decimal | Finite; bounded to configured canvas extent |
| `revision` | integer | Starts at 1; increments on successful mutation |
| `updated_at` | timestamp | Server generated |

**Identity**: `(laboratory_id, resource_id)` is unique.

**Relationships**: Belongs to one laboratory and one topology resource. Deleting either cascades placement
deletion. A topology snapshot includes resolved placement for every node/network object; missing placement uses a
deterministic initial position until first commit.

## PlacementBatchMutation

One completed drag or keyboard move.

| Field | Type | Rules |
|---|---|---|
| `laboratory_id` | ID | Required |
| `expected_laboratory_revision` | integer | Required concurrency precondition |
| `placements` | list | 1–100 unique resource updates |
| `idempotency_key` | string | Required for retry safety |

Each item contains `resource_id`, `resource_type`, `x`, `y`, and expected placement revision when one exists.
Validation is all-or-nothing. The transaction updates placements, advances the laboratory revision, writes audit
metadata, and publishes one ordered `topology.placements_changed` event containing bounded placement summaries.

## BrowserTopologyPreference

Local, sanitized presentation state keyed by laboratory ID.

| Field | Type | Rules |
|---|---|---|
| `viewport` | object | Center and zoom within supported bounds |
| `label_density` | enum | `comfortable`, `compact`, `minimal` |
| `reduced_motion` | boolean | Defaults from browser preference |
| `link_routes` | map | Link ID to bounded list of finite control points |

Node placements are removed from this entity during migration. Unknown/deleted links and invalid coordinates are
discarded on load.

## TopologyInteractionSession

Ephemeral browser state, never persisted server-side.

States: `idle`, `pressing`, `panning`, `box_selecting`, `dragging_resources`, `connecting`,
`choosing_target_port`, `cancelling`.

Legal transitions begin from `idle`; pointer/keyboard cancellation or focus loss returns to `idle`. Only release
from `dragging_resources` may create a placement mutation. Only confirmed valid target selection from
`connecting`/`choosing_target_port` may create a link task.

## LinkReconnectRequest

| Field | Type | Rules |
|---|---|---|
| `link_id` | ID | Existing link |
| `expected_revision` | integer | Required |
| `retained_endpoint_id` | ID | Must be one current endpoint |
| `replacement_endpoint_id` | ID | Different, available, same laboratory |
| `idempotency_key` | string | Required |

Transitions: `queued → running → succeeded`; cancellation uses `queued/running → cancelling → cancelled`; failure
uses `running → failed`. Until success, the persisted link endpoints remain unchanged. Runtime compensation
restores the original link before a failed/cancelled task reaches terminal state.

## VisualSymbol

Static asset metadata: resource kind, SVG path reference, accessible label, source package/version, license, and
fallback symbol. It contains no appliance image or remote URL.
