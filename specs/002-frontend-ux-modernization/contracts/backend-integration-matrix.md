# Backend Integration Matrix

This matrix references the existing contracts in
`specs/001-network-simulator-platform/contracts/`. It does not duplicate request or response schemas.

| UI capability | REST operation / stream | Completion and synchronization | Primary story |
|---|---|---|---|
| List/create laboratories | `listLabs`, `createLab` | Response plus shared lab events | US1 |
| Rename laboratory | `updateLab` | Revision-sensitive response/task and lab event | US1 |
| Duplicate laboratory | `duplicateLab` | Durable task, new lab resource | US1 |
| Export/import laboratory | `exportLab`, `importLab`, `downloadArtifact` | Durable task and sanitized artifact handle | US1/US4 |
| Delete laboratory | `deleteLab` | Durable task, deletion event, cleanup problem | US1 |
| Browse capabilities/templates/images | `getCapabilities`, `listTemplates`, `listImages` | Authoritative query response | US1/US4 |
| Import image | `importImage` | Durable task and image validation state | US4 |
| Create node | `createNode` | Durable task, node event, desired/observed state | US1 |
| Start/stop node | `setNodeState` | Durable task and node lifecycle events | US1/US2 |
| Delete node | `deleteNode` | Durable task, deletion/cleanup events | US1/US2 |
| Create/delete network object | `createNetworkObject`, `deleteNetworkObject` | Durable task and resource events | US1 |
| Attach network object | `attachNetworkObject` | Durable task and link/resource events | US1 |
| Connect/disconnect link | `connectLink`, `disconnectLink` | Durable task and link events | US1 |
| Atomically reconnect link | `reconnectLink` | Durable task with original-endpoint rollback on failure or cancellation | 004-US2 |
| Move topology resources | `updateTopologyPlacements` | Revision-sensitive shared placement batch and ordered event | 004-US1 |
| Hot-add/remove interface | `addInterface`, `removeInterface` | Durable staged task, QMP/resource events | US2 |
| Configure resources | `getNodeResources`, `updateNodeResources` | Revision-sensitive durable task | US2 |
| Execute guest command | `executeGuestCommand` | Durable task with bounded result/problem | US2 |
| Create/delete port mapping | `createPortMapping`, `deletePortMapping` | Durable task and mapping events | US2 |
| List/open console | `listNodeConsoles`, `streamNodeConsole` | Metadata query plus reconnectable stream | US3 |
| Start/inspect/stop capture | `startCapture`, `getCapture`, `stopCapture` | Durable task/resource events | US3 |
| Stream/download capture | `streamCapture`, `downloadArtifact` | Bounded stream or artifact handle | US3 |
| Start/inspect/stop Traffic Filter | `startTrafficFilter`, `getTrafficFilter`, `stopTrafficFilter` | Durable task/resource observations | US3 |
| Inspect task center | `listTasks`, `getTask` | Queries plus ordered task events | US2 |
| Cancel task | `cancelTask` | Durable cancellation transition | US2 |
| Inspect audit history | `listAuditEvents` | Authoritative query response | US2/US4 |
| Network diagnostics | `getNetworkObjectDiagnostics` | Authoritative diagnostic response | US3 |

## Event Contract

- The SPA consumes the ordered shared event contract in
  `specs/001-network-simulator-platform/contracts/events.md`.
- Resource identity and revision determine whether an incremental update is safe.
- A cursor gap, unknown event version, or incompatible revision triggers a fresh authoritative snapshot.
- Browser-local placements survive snapshot replacement and are rejoined by stable resource ID.

## Contract Defect Policy

If a required field or action is absent from the existing OpenAPI/event contract, implementation MUST:

1. Record the concrete missing behavior and affected user story.
2. Update the authoritative contract and generated types first.
3. Add contract tests against the real test service.
4. Route UI, REST, and MCP mutations through the same application command handler.

The frontend MUST NOT infer hidden lifecycle state, fabricate task completion, or create a private endpoint.
