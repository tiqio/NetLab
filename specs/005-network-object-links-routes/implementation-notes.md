# Implementation Notes: Network Object Links and Docker Routes

## Baseline

- Baseline commit inspected: `74dadf3` (`chore: establish recovered NetLab repository baseline`).
- Feature planning commits: `09ace49` and `326d9ef`.
- The recovered repository has no earlier file-level history, so compatibility decisions are derived
  from the baseline code, public contracts, current tests, and feature design artifacts.

## Compatibility Constraints

- Preserve the public flat object-link endpoint fields: `object_a_id`, `port_a_name`, `object_b_id`,
  and `port_b_name`.
- Preserve existing node-interface links, network attachments, NAT, host-interface capture, standard
  link capture, and existing Traffic Filter scopes.
- Replace only the runtime shape of network-object links: one veth pair moved directly between the two
  namespaces instead of a host bridge plus two veth pairs.
- Keep endpoint A/B ordering stable because capture and Traffic Filter direction use endpoint A as the
  orientation reference.
- Never derive deletion ownership from a generic interface shape; use deterministic NetLab names and
  durable resource identity.
- Docker endpoint route execution already uses argument vectors through `CommandExecutor`; retain that
  boundary and carry typed route declarations through the control and persistence layers.
- Browser, HTTP, and MCP mutations must converge through shared application services and durable tasks.
- All implementation and tests run locally and receive focused commits before deployment to
  `10.72.1.7`; source files are never edited directly on the target.

## Milestone Evidence

| Milestone | Local gates | Commit SHA | Notes |
|---|---|---|---|
| Setup | Ignore rules and fixture compilation | pending | No ignore changes required |
| Foundation | `go test ./internal/domain ./internal/store/sqlite ./internal/runtime/linuxnet ./internal/app/reconcile`; all passed | `3813579` | Added endpoint reservations, typed routes, direct veth links, and namespace-aware object-link capture |
| US1 | pending | pending | |
| US2 | pending | pending | |
| US3 | pending | pending | |
| US4 | `go test ./internal/app/command ./internal/store/sqlite ./internal/runtime/linuxnet`; all passed | `bb07b51` | Docker route declarations are validated on create/settings and persisted for runtime reconciliation |
| Final candidate | pending | pending | |
