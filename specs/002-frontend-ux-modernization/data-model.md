# Data Model: Frontend UX Modernization

This feature does not add a new server-side persistence model. It defines client projections of existing
authoritative API resources and a strictly non-authoritative browser-local preference document.

## State Ownership

| Model | Owner | Persistence | Shared Across Clients |
|---|---|---|---|
| Laboratory resource projection | NetLab server | SQLite/event stream | Yes |
| Task and structured problem projection | NetLab server | SQLite/event stream | Yes |
| Console, capture, and traffic-filter metadata | NetLab server | Runtime/durable artifact stores | Yes |
| Workspace preferences | Current browser | Versioned `localStorage` adapter | No |
| Selection, open menus, drag state | Current Vue view | Memory only | No |
| ECharts/xterm/noVNC instances | Current Vue component | Memory only | No |

## WorkspacePreferences

The sole browser-persisted aggregate, stored under a versioned key containing the laboratory ID.

### Fields

- `schemaVersion`: positive integer identifying the preference schema.
- `laboratoryId`: authoritative laboratory identifier used only for namespacing.
- `updatedAt`: ISO-8601 timestamp of the last local write.
- `panels`: sizes and collapsed state for palette, inspector, and bottom drawer.
- `viewport`: topology center and zoom.
- `placements`: map from authoritative resource ID to local node/network-object placement.
- `groups`: local-only visual groups and their member resource IDs.
- `linkRoutes`: optional local bend points keyed by authoritative link ID.
- `activeBottomTab`: last selected task/console/diagnostics tab.

### Validation

- The document MUST satisfy `contracts/workspace-preferences.schema.json` before use.
- Panel sizes and zoom MUST be clamped to supported limits after validation.
- Resource IDs not present in the current authoritative snapshot MAY remain stored but MUST be ignored.
- Unknown fields MUST NOT be copied into application state.
- Corrupt, unsupported, or unavailable storage MUST fall back to deterministic defaults.
- No credential, cloud-init data, guest-command output, packet payload, image content, or console content
  may be represented in this aggregate.

### Lifecycle

`missing → defaults → modified → debounced persisted`

`older supported version → migrated → validated → active`

`invalid or unsupported version → discarded → defaults`

## PanelLayout

Browser-local shell configuration embedded in `WorkspacePreferences`.

### Fields

- `devicePalette`: `{ collapsed, size }`
- `inspector`: `{ collapsed, size }`
- `bottomDrawer`: `{ collapsed, size }`

### Rules

- Sizes are CSS pixel values at desktop breakpoints.
- At responsive drawer breakpoints, stored desktop sizes remain unchanged.
- Collapsing a panel MUST NOT close an active task, console, or diagnostic session.
- The canvas receives all space not assigned to visible panels.

## TopologyPlacement

Browser-local visual position for an authoritative resource.

### Fields

- `x`, `y`: finite ECharts graph coordinates.
- `pinned`: whether automatic local placement may move the resource.
- `updatedAt`: local ISO-8601 timestamp.

### Rules

- Resource identity always comes from the authoritative snapshot.
- A resource without placement receives a deterministic collision-aware position based on its stable ID.
- Moving a resource updates only `WorkspacePreferences`; it MUST NOT call a laboratory mutation endpoint.

## LocalVisualGroup

A browser-only grouping aid that does not create a laboratory resource.

### Fields

- `id`: browser-generated stable identifier.
- `label`: display label.
- `memberResourceIds`: authoritative node or network-object IDs.
- `bounds`: optional local rectangle.
- `collapsed`: local display state.

### Rules

- Deleting a group never deletes or mutates its members.
- Missing members are ignored during rendering and may be pruned on the next preference write.

## LocalLinkRoute

Browser-only route hints for an authoritative link.

### Fields

- `linkId`: authoritative link identifier.
- `controlPoints`: zero or more finite `{ x, y }` points.

### Rules

- Link endpoints and connectivity always come from the server.
- Removing or replacing an authoritative link invalidates the local route without mutating the lab.

## AuthoritativeWorkspaceProjection

A normalized Pinia projection assembled from the laboratory snapshot and ordered events.

### Fields

- `laboratory`: ID, name, revision, policy, and timestamps.
- `nodes`: normalized node map including desired state, observed state, runtime identity, capabilities,
  resources, interfaces, mappings, and latest problem.
- `links`: normalized link map including endpoints, state, revision, and latest problem.
- `networkObjects`: normalized bridge, NAT, and namespace-backed resources.
- `tasks`: normalized durable tasks related to the active laboratory.
- `captures`: capture metadata and artifact handles, never persisted packet bytes.
- `trafficFilters`: filter definitions, status, and observations.
- `eventCursor`: latest safely applied ordered-event position.
- `syncState`: `initializing | live | reconnecting | refreshing | degraded`.

### State Transitions

- `initializing → live` after snapshot and event subscription establish a consistent cursor.
- `live → reconnecting` on transport loss.
- `reconnecting → live` when replay is complete and contiguous.
- `reconnecting → refreshing` on a replay gap or unsafe revision.
- `refreshing → live` after authoritative snapshot replacement.
- Any state may enter `degraded` on an unrecoverable contract or transport failure.

## TopologyViewModel

Derived, non-persisted ECharts input joining authoritative resources and local preferences.

### Fields

- `nodes`: graph data items with stable resource IDs, local coordinates, label, icon/category, lifecycle
  text, accessibility descriptor, and pending/problem flags.
- `edges`: graph links with stable link IDs, endpoint IDs, interface labels, direction/status styling, and
  optional local route hints.
- `categories`: device/network-object categories and legends.
- `trafficObservations`: observed path overlays with count, direction, timing, confidence, loop, and ambiguity.
- `selection`: current resource IDs and primary selected item.

### Rules

- Color MUST NOT be the sole carrier of lifecycle, error, selection, or traffic state.
- Pending affordances are removed only after authoritative completion/failure or snapshot reconciliation.
- ECharts instance objects and event objects never enter Pinia or persistence.

## PendingSubmission

Transient protection against accidental duplicate operations.

### Fields

- `operationKey`: operation kind plus target identity and normalized input identity.
- `idempotencyKey`: request key when supported by the operation.
- `submittedAt`: client timestamp.
- `taskId`: durable server task ID once accepted.
- `state`: `submitting | accepted | reconciling | terminal | unknown`.

### Rules

- A matching `submitting`, `accepted`, or `reconciling` operation disables accidental resubmission.
- Refresh/reconnect resolves accepted operations through the durable task endpoint.
- `unknown` permits an explicit safe replay that reuses the idempotency key when available.

## TaskPresentation

A UI projection of an authoritative durable task.

### Fields

- Task identity, operation kind, resource reference, state, progress, timestamps, result, cancellation
  availability, and structured problem.
- Derived navigation target and concise status label.

### Rules

- The UI MUST NOT synthesize a second task state machine.
- Filters are local presentation state; task content and terminal state remain authoritative.
- Task-to-resource navigation tolerates resources not yet visible or already deleted.

## StructuredProblemPresentation

Normalized display model for backend problem details.

### Fields

- `code`, `title`, `detail`, `resource`, `lifecyclePhase`, `retryable`, `retryAfter`, `cleanupState`,
  `operatorAction`, and optional field-level violations.

### Rules

- Unknown problem fields remain available in a details disclosure but do not change semantics.
- Revision conflicts preserve unsaved form values and offer refresh/compare/retry actions.
- Errors shown in toasts MUST also remain accessible in the relevant inspector, form, or task entry.

## DiagnosticSessionPresentation

Transient UI state for a console, capture, or traffic-filter session.

### Fields

- `sessionKey`, `kind`, authoritative resource/task ID, target resource, connection state, reconnect count,
  selected tab, and renderer-local status.

### Rules

- Closing a console tab disconnects the browser renderer but does not change node lifecycle state.
- Capture bytes are streamed or downloaded and are never written to workspace preferences.
- Traffic observations may be rendered on the topology but remain authoritative diagnostic data.
