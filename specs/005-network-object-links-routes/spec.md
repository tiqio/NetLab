# Feature Specification: First-Class Network Object Links and Docker Routes

**Feature Branch**: `main`

**Created**: 2026-08-03

**Status**: Draft

**Input**: User description: "Add first-class links between network objects so lightweight switches
can form real multi-switch topologies, and automatically apply custom static routes declared when
Docker nodes are created. Network-object links must support selection, capture, Traffic Filter
highlighting, and live deletion."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Build Multi-Switch Topologies (Priority: P1)

As a laboratory operator, I can connect a named port on one lightweight network object to a named
port on another network object so that traffic can traverse multiple switches and routers instead of
terminating at a single aggregation object.

**Why this priority**: Without network-object-to-network-object connectivity, lightweight switching
and routing objects cannot represent production-like multi-hop topologies.

**Independent Test**: Create three lightweight network objects, connect them through two independent
object links, attach endpoint nodes at opposite sides, and verify bidirectional traffic crosses both
links without any manual host-side networking commands.

**Acceptance Scenarios**:

1. **Given** two active network objects with available named ports in the same laboratory, **When**
   the operator connects those ports, **Then** one durable link appears between the selected ports and
   reaches a connected state without stopping attached nodes.
2. **Given** two network objects already joined by one link, **When** the operator connects a second
   pair of available ports, **Then** both parallel links remain independently identifiable and do not
   visually or operationally collapse into one resource.
3. **Given** a three-object path with endpoint nodes on opposite sides, **When** the endpoint network
   settings are valid, **Then** traffic traverses the complete path and survives a control-service
   restart according to the laboratory recovery policy.

---

### User Story 2 - Observe and Diagnose Object Links (Priority: P1)

As a network troubleshooter, I can select a network-object link, inspect its endpoints and state,
capture packets on it, and see matching Traffic Filter activity directly on the topology line.

**Why this priority**: A link that forwards traffic but cannot be observed or diagnosed is not a
first-class topology resource and is unsuitable for troubleshooting workflows.

**Independent Test**: Select one link in a topology containing multiple object links, start a packet
capture and a protocol-specific Traffic Filter, generate traffic across only that link, and verify
that packet data and visual activity are attributed only to the selected resource and correct
direction.

**Acceptance Scenarios**:

1. **Given** a connected network-object link, **When** the operator selects it, **Then** the inspector
   displays a human-readable `object:port ↔ object:port` identity, desired state, actual state, and any
   actionable failure.
2. **Given** a connected link carrying bidirectional traffic, **When** capture starts on that link,
   **Then** packets from both directions are available through the standard live capture workflow
   with bounded size, packet counts, completion state, and an artifact or stream reference.
3. **Given** several links and a Traffic Filter that matches traffic on only one path, **When** matching
   packets arrive, **Then** only traversed links show directional activity and the activity decays
   after matching traffic stops.
4. **Given** two parallel links between the same objects, **When** traffic uses only one port pair,
   **Then** capture and Traffic Filter observations do not mark the unused parallel link.

---

### User Story 3 - Rewire and Delete Links Live (Priority: P1)

As a laboratory operator, I can remove a network-object link while the laboratory is running, and
the topology, observations, and host resources converge without ghost lines or leaked resources.

**Why this priority**: Live topology mutation is a core product requirement and is necessary for
failure simulation, path switching, and iterative laboratory design.

**Independent Test**: With endpoint traffic crossing an object link, delete the link while all
objects and nodes remain active, verify traffic stops on that path, verify the link disappears for
all clients, and reconnect the same ports successfully.

**Acceptance Scenarios**:

1. **Given** an active link with no capture, **When** it is deleted, **Then** the link disappears from
   every connected client, traffic stops on that path, and both ports become reusable without
   restarting any node or network object.
2. **Given** an active capture on a link, **When** the link is deleted, **Then** capture ends with an
   explicit link-deleted completion reason, retained packet metadata remains accessible, and deletion
   is not blocked indefinitely.
3. **Given** deletion fails after only part of the runtime path is removed, **When** reconciliation
   runs, **Then** the link exposes an actionable failed state and cleanup completes without deleting
   unrelated host networking resources.
4. **Given** a network object is deleted, **When** it owns one or more object links, **Then** those links
   and their observations are removed as part of the same durable lifecycle without stale topology
   entries.

---

### User Story 4 - Reproduce Docker L3 Routes (Priority: P1)

As a topology author, I can declare custom IPv4 or IPv6 static routes for a Docker node and trust that
the node receives those routes automatically whenever it starts or is restored.

**Why this priority**: Requiring an operator to enter the container's network context and add routes
manually makes L3 laboratories non-repeatable and breaks automation.

**Independent Test**: Create a Docker node with declared routes behind a multi-object L3 path, start
it without manual host commands, verify routed ICMP, TCP, and UDP traffic, restart the node and the
control service, and verify the same routes and traffic behavior remain.

**Acceptance Scenarios**:

1. **Given** a Docker node with valid custom static routes, **When** it reaches running state, **Then**
   all declared routes are present before the node is reported ready.
2. **Given** a running Docker node whose routes were applied, **When** the node is stopped and started
   or recreated during recovery, **Then** the same route set is restored without manual intervention.
3. **Given** an invalid, conflicting, or unusable route declaration, **When** the node starts, **Then**
   readiness fails with an actionable route-specific error rather than reporting a healthy node with
   missing routes.
4. **Given** a stopped Docker node whose route configuration is updated, **When** it next starts,
   **Then** the new route set is applied and removed routes do not remain inside the node.

### Edge Cases

- A port is already occupied by another attachment or object link when a new connection is requested.
- The two selected endpoints belong to different laboratories or one object no longer exists.
- Multiple parallel links connect the same two objects through different port names.
- A link is created while one endpoint is still converging or temporarily unavailable.
- A capture or Traffic Filter references a link while that link is being deleted.
- The control service restarts after durable link creation but before runtime connectivity completes.
- Runtime connectivity exists after a crash but the durable link state was not fully updated.
- A route duplicates a connected route, conflicts with another declared route, or defines an invalid
  destination, gateway, address family, or output interface.
- A route gateway is syntactically valid but unreachable when the Docker node starts.
- A Docker node has several interfaces and a route does not identify an unambiguous egress path.
- IPv4 and IPv6 routes coexist, including default routes for one or both address families.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST represent every network-object-to-network-object connection as a durable,
  server-authoritative resource with stable identity, laboratory ownership, two named endpoints,
  revision, desired state, actual state, and actionable error state.
- **FR-002**: The system MUST allow links only between connectable ports belonging to network objects
  in the same active laboratory.
- **FR-003**: The system MUST support multiple simultaneous links between the same pair of network
  objects when each link uses a distinct available port pair.
- **FR-004**: The system MUST reject ambiguous or occupied endpoint selections without altering an
  existing attachment or link.
- **FR-005**: Operators and automation MUST be able to create and delete object links while attached
  nodes and network objects remain active.
- **FR-006**: UI, HTTP automation, and MCP automation MUST use the same link lifecycle and observe the
  same ordered final state without requiring a browser refresh.
- **FR-007**: Every object link MUST be selectable as an independent topology resource and MUST expose
  a human-readable endpoint label using object names and port names.
- **FR-008**: The system MUST expose object-link desired state, actual state, revision, lifecycle task,
  and structured failure information through all applicable control surfaces.
- **FR-009**: A connected object link MUST be a valid packet-capture target using the same bounded,
  cancellable, retained capture workflow available to other topology links.
- **FR-010**: Object-link capture MUST account for traffic in both directions and MUST associate packet
  counts, byte counts, limits, completion reason, and stream or artifact references with the selected
  link identity.
- **FR-011**: Traffic Filter observations MUST identify traversed object links and traffic direction,
  highlight only links that carried matching traffic, and decay after matching traffic stops.
- **FR-012**: Parallel links MUST remain distinct for selection, capture, Traffic Filter attribution,
  task results, events, export, and deletion.
- **FR-013**: Deleting a link with an active capture MUST terminate that capture with an explicit
  link-deleted reason while preserving already retained metadata and packet data according to policy.
- **FR-014**: Deleting a network object MUST durably remove its object links and prevent stale lines,
  observations, captures, or occupied-port state from reappearing after refresh or restart.
- **FR-015**: Link creation and deletion MUST be idempotent or accept idempotency protection, enforce
  revision conflicts, and return durable task progress for retryable runtime work.
- **FR-016**: Startup and periodic reconciliation MUST adopt owned running object-link resources,
  complete pending desired state, and clean partial or orphaned owned resources without modifying
  unrelated host networking.
- **FR-017**: Laboratory export and import MUST preserve object-link identity, endpoint intent, and
  non-secret configuration without exporting packet payloads or host-specific runtime identifiers.
- **FR-018**: Docker node configuration MUST persist an ordered set of custom IPv4 and IPv6 static
  route declarations as part of the node's reproducible topology definition.
- **FR-019**: Each Docker route declaration MUST identify a destination prefix and enough gateway or
  egress information to select an unambiguous path.
- **FR-020**: The system MUST validate Docker routes for address-family consistency, valid prefixes,
  duplicate or conflicting declarations, known interfaces, and unambiguous egress before reporting
  the node ready.
- **FR-021**: All declared Docker routes MUST be applied automatically on initial start, normal
  restart, runtime recreation, service recovery, and host recovery before the node is reported ready.
- **FR-022**: Updating route declarations on a stopped Docker node MUST produce the exact declared
  route set at the next start, including removal of previously managed routes no longer declared.
- **FR-023**: Route application failure MUST place the Docker node in an actionable degraded or failed
  state and MUST NOT silently report healthy operation with only a partial route set.
- **FR-024**: No supported object-link or Docker-route workflow may require an operator or automation
  client to enter a host network context and run an undocumented manual command.
- **FR-025**: Link and route mutations MUST publish ordered shared-state events and audit records that
  identify the resource, actor class, outcome, correlation or task identifier, and structured error.
- **FR-026**: Existing node-interface links, network attachments, NAT behavior, captures, and Traffic
  Filters MUST continue to work without semantic regression when object links are introduced.

### Key Entities *(include if feature involves data)*

- **Network Object Link**: A durable point-to-point connection between two named ports on two network
  objects, including shared lifecycle, revision, observation, task, and error state.
- **Network Object Port**: A named connectable endpoint owned by one network object, with occupancy and
  operational state used to validate attachments and links.
- **Docker Static Route**: A declared destination and path selection associated with a Docker node and
  restored as part of that node's reproducible runtime state.
- **Link Observation**: Capture and Traffic Filter information attributed to a stable object-link
  identity, including direction, time window, counters, completion, and retention state.
- **Lifecycle Task**: Durable progress and outcome information for object-link creation, deletion,
  recovery, capture termination, or Docker route application.

## Delivery and Deployment Constraints *(mandatory)*

- **Local milestone**: Complete the durable model and lifecycle tests first, then runtime connectivity
  and recovery, then control-surface parity, then topology selection/capture/Traffic Filter behavior,
  and finally Docker route application and end-to-end L3 validation. Each slice must pass focused
  local tests before the next slice begins.
- **Commit evidence**: Record each independently testable slice as a focused Git commit and include
  the commit SHA in validation evidence. The specification milestone itself is committed separately.
- **Deployment artifact**: Build the candidate only from a clean committed worktree and record the
  candidate's source commit and artifact digest before deployment.
- **Target validation**: Deploy the committed candidate to `10.72.1.7` and validate multi-switch
  forwarding, parallel links, live deletion, capture, Traffic Filter direction, restart recovery,
  Docker IPv4/IPv6 routes, and resource cleanup.
- **Rollback**: Retain the previously deployed commit and artifact, preserve compatible durable data,
  and verify service health and topology recovery after rollback.
- **Target immutability**: Source fixes MUST NOT be made directly on `10.72.1.7`; failed validation
  returns to the local worktree for tests, a new commit, and redeployment.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: An operator can build and activate a three-network-object, two-link path without manual
  host networking commands, and bidirectional endpoint traffic succeeds in 100% of ten repeated runs.
- **SC-002**: A newly created or deleted object link reaches the same final state in two simultaneous
  browser clients and one automation client within 2 seconds in at least 99% of test operations.
- **SC-003**: For matching traffic, the correct object link and direction become visible within 500
  milliseconds for at least 95% of observations, while unused parallel links remain unmarked.
- **SC-004**: A capture targeted at an object link records both directions, reports accurate non-zero
  packet and byte counts for generated test traffic, and remains retrievable after normal stop or
  link deletion according to retention policy.
- **SC-005**: One hundred repeated object-link create/delete cycles leave no owned link resources,
  occupied ports, capture workers, stale observations, or topology lines after the cleanup window.
- **SC-006**: After control-service and host restart validation, every desired connected object link
  either returns to connected state or exposes an actionable failure without duplicate runtime links.
- **SC-007**: Valid declared Docker route sets are present before readiness after initial start,
  restart, recreation, and recovery in 100% of twenty IPv4/IPv6 test cycles.
- **SC-008**: Invalid Docker route declarations are rejected or fail readiness with a route-specific
  explanation in 100% of validation cases; no case reports healthy with a partial managed route set.
- **SC-009**: A multi-hop L3 scenario using declared Docker routes carries ICMP, TCP, and UDP traffic
  without manual intervention in 100% of ten repeated deployments.
- **SC-010**: At least 90% of evaluation users can connect, select, capture, inspect, and delete a
  network-object link on their first attempt without documentation assistance.

## Assumptions

- First-class object links initially apply to network objects that expose named connectable ports;
  L2 and L3 lightweight switching objects are mandatory participants.
- Self-links on one network object are outside this feature, while multiple links between two
  different objects are supported through distinct port pairs.
- Route declarations are created with the Docker node or edited while it is stopped; live route-list
  editing on a running Docker node is outside this feature.
- The platform continues to operate without account/password authentication, so all clients inside
  the trusted management boundary observe the same durable link and route state.
- Packet capture retention, quotas, artifact handling, and Traffic Filter expression semantics follow
  the platform's existing policies.
- Operator-approved Docker images contain the normal guest networking capabilities needed to install
  declared routes; unsupported images produce an explicit capability error.

## Dependencies

- Existing network objects expose stable names, laboratory ownership, desired/actual lifecycle, and
  named ports that can participate in durable connections.
- Existing capture and Traffic Filter services can address a stable topology resource and return
  bounded observation metadata.
- Existing Docker node configuration supports persistent interface and custom route declarations.
- The target host provides the privileged networking and packet-observation capabilities required by
  the platform, and target validation is performed only from a committed local candidate.
