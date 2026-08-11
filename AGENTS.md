# NetLab Agent Guidance

## Active Feature

- Specification: `specs/011-network-path-recovery/spec.md`
- Plan: `specs/011-network-path-recovery/plan.md`
- Constitution: `.specify/memory/constitution.md`
- Contracts: `specs/011-network-path-recovery/contracts/`
- Topology visual compatibility specification: `specs/010-topology-visual-unification/spec.md`
- Topology visual compatibility plan: `specs/010-topology-visual-unification/plan.md`
- Unified link interaction dependency: `specs/009-unified-link-interaction/spec.md`
- Previous visual overlap specification: `specs/008-ui-overlap-remediation/spec.md`
- Previous visual overlap plan: `specs/008-ui-overlap-remediation/plan.md`
- Previous UI localization specification: `specs/007-ui-localization-theme/spec.md`
- Previous UI localization plan: `specs/007-ui-localization-theme/plan.md`
- Governance closure specification: `specs/002-constitution-gap-closure/spec.md`
- Governance closure plan: `specs/002-constitution-gap-closure/plan.md`
- Base platform specification: `specs/001-network-simulator-platform/spec.md`
- Base platform plan: `specs/001-network-simulator-platform/plan.md`

## Technology

- Backend: Go 1.26.x, Gin 1.12.x, SQLite WAL, Docker Engine Go SDK, digitalocean/go-qemu QMP
- Frontend: Vue 3, TypeScript, Vite, Pinia, ECharts, xterm.js, noVNC
- Target: single x86_64 Linux host with KVM/QEMU, Docker, cgroup v2, netlink, nftables, and capture tools

## Architecture Rules

- Keep domain and application packages independent of Gin, SQLite, QEMU, Docker, and host commands.
- Route SPA, HTTP API, and MCP mutations through the same application command handlers.
- Represent long operations as durable tasks with idempotency, cancellation, progress, and errors.
- Give every process, interface, namespace, bridge, rule, socket, and artifact an explicit resource owner.
- Adopt owned running resources after service restart; use lab recovery policy after host restart.
- Validate revisions on concurrent mutations and publish ordered outbox events with durable state changes.
- Keep initial topology placement server-authoritative across SPA, HTTP API, and MCP creation paths.
- Preserve existing laboratory coordinates; never use automatic global relayout for new-resource placement.
- Normalize connection presentation without collapsing distinct runtime ownership and reconciliation models.
- Route port drag, resource-plus, keyboard, HTTP, and MCP connection creation through one endpoint model and application command.
- Keep connection drafts, pointer positions, and port choosers local to the initiating browser until submission.
- Serialize node-interface and network-object-port occupancy in durable SQLite state across all connection categories.
- Default only newly created lightweight L2/L3 resources to `eth0` through `eth3`; never auto-expand existing objects.
- Report namespace-backed objects healthy only after their owned runtime backing is usable and matches desired state.
- Dispatch connection create, compensation, inspection, and deletion by explicit endpoint backing kind.
- Reconcile L2 PVID and tagged VLAN membership after ports appear and compare desired membership with runtime observation.
- Keep traffic-workload success/failure aggregates durable and separate from Traffic Filter counters and visual decay.

## Safety and Testing

- Never add proprietary images, credentials, bootstrap secrets, or packet captures to source control.
- Never build host commands by interpolating untrusted values into a shell string.
- Require unit, contract, privileged integration, recovery, and resource-leak tests for runtime changes.
- Use the validation sequence in `specs/001-network-simulator-platform/quickstart.md`.

## Delivery Workflow

- Implement and test every change in the local Git worktree before touching the deployment target.
- Inspect relevant `git log` and `git blame` history before non-trivial compatibility-sensitive work.
- Commit each independently testable milestone with a focused Git commit before deployment.
- Deploy only artifacts built from a clean, identified commit to `10.72.1.7`.
- Record the commit SHA, artifact digest, migration state, deployment time, and target-host test results.
- Never edit source files directly on `10.72.1.7`; failed validation returns to local development,
  testing, a new commit, and redeployment.
