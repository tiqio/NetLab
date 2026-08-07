import { beforeEach, describe, expect, it } from "vitest";
import { createPinia, setActivePinia } from "pinia";
import type { Node, TopologyPlacement, TopologySnapshot } from "../api";
import { useLaboratoryStore } from "./laboratory";

describe("runtime truth event reconciliation", () => {
  beforeEach(() => setActivePinia(createPinia()));

  it("does not resurrect a deleted network object from an older event", () => {
    const store = useLaboratoryStore();
    store.active = {
      laboratory: {
        id: "lab",
        name: "lab",
        description: "",
        revision: 1,
        recovery_policy: "auto_restore",
        lifecycle_state: "active",
      },
      nodes: [],
      interfaces: [],
      links: [],
      network_objects: [
        {
          id: "net-1",
          laboratory_id: "lab",
          name: "nat",
          kind: "nat_bridge",
          revision: 1,
          desired_state: "active",
          observed_state: "active",
          config: {},
        },
      ],
      placements: [],
      event_sequence: 0,
    };
    store.applyEvent({
      sequence: 10,
      type: "network_object.deleted",
      laboratory_id: "lab",
      resource_type: "network_object",
      resource_id: "net-1",
      revision: 2,
      data: {},
    });
    store.applyEvent({
      sequence: 9,
      type: "network_object.created",
      laboratory_id: "lab",
      resource_type: "network_object",
      resource_id: "net-1",
      revision: 1,
      data: { id: "net-1" },
    });
    expect(store.active!.network_objects).toHaveLength(0);
  });

  it("keeps capability events separate from node lifecycle state", () => {
    const store = useLaboratoryStore();
    store.active = {
      laboratory: {
        id: "lab",
        name: "lab",
        description: "",
        revision: 1,
        recovery_policy: "auto_restore",
        lifecycle_state: "active",
      },
      nodes: [
        {
          id: "node-1",
          laboratory_id: "lab",
          name: "node",
          kind: "qemu",
          revision: 1,
          desired_state: "running",
          observed_state: "running",
          cpu_count: 1,
          cpu_quota_micros: 0,
          memory_mib: 512,
          storage_gib: 1,
          interface_limit: 1,
          process_limit: 0,
        },
      ],
      interfaces: [],
      links: [],
      network_objects: [],
      placements: [],
      event_sequence: 0,
    };
    store.applyEvent({
      sequence: 1,
      type: "node.capability_changed",
      laboratory_id: "lab",
      resource_type: "node",
      resource_id: "node-1",
      revision: 1,
      data: {
        node_id: "node-1",
        capability: "qga",
        revision: 1,
        state: "unavailable",
        required: true,
        observed_at: new Date().toISOString(),
      },
    });
    expect(store.active!.nodes[0].observed_state).toBe("running");
    expect(store.nodeCapabilities["node-1"][0].state).toBe("unavailable");
  });

  it("accepts an authoritative placement before the matching resource event", () => {
    const store = useLaboratoryStore();
    store.active = snapshot();
    store.applyEvent({
      sequence: 1,
      type: "topology.placements_changed",
      laboratory_id: "lab",
      resource_type: "laboratory",
      resource_id: "lab",
      revision: 2,
      data: {
        placements: [placement("node-2", 240, 160, 1)],
      },
    });
    store.applyEvent({
      sequence: 2,
      type: "node.created",
      laboratory_id: "lab",
      resource_type: "node",
      resource_id: "node-2",
      revision: 1,
      data: { ...node("node-2") },
    });
    expect(store.active!.nodes.map((item) => item.id)).toEqual(["node-2"]);
    expect(store.active!.placements).toEqual([
      placement("node-2", 240, 160, 1),
    ]);
    expect(store.active!.laboratory).not.toHaveProperty("placements");
  });

  it("ignores duplicate and stale placement events", () => {
    const store = useLaboratoryStore();
    store.active = snapshot();
    store.applyEvent({
      sequence: 3,
      type: "topology.placements_changed",
      laboratory_id: "lab",
      resource_type: "laboratory",
      resource_id: "lab",
      revision: 3,
      data: { placements: [placement("node-1", 300, 200, 3)] },
    });
    store.applyEvent({
      sequence: 3,
      type: "topology.placements_changed",
      laboratory_id: "lab",
      resource_type: "laboratory",
      resource_id: "lab",
      revision: 3,
      data: { placements: [placement("node-1", 999, 999, 3)] },
    });
    store.applyEvent({
      sequence: 4,
      type: "topology.placements_changed",
      laboratory_id: "lab",
      resource_type: "laboratory",
      resource_id: "lab",
      revision: 2,
      data: { placements: [placement("node-1", 10, 20, 2)] },
    });
    expect(store.active!.placements).toEqual([
      placement("node-1", 300, 200, 3),
    ]);
    expect(store.active!.laboratory.revision).toBe(3);
  });

  it("converges independent clients on the same authoritative creation", () => {
    const firstPinia = createPinia();
    setActivePinia(firstPinia);
    const first = useLaboratoryStore();
    first.active = snapshot();
    const secondPinia = createPinia();
    setActivePinia(secondPinia);
    const second = useLaboratoryStore();
    second.active = snapshot();
    const assignment = {
      placement: placement("node-shared", 480, 320, 1),
      assigned_center: { x: 480, y: 320 },
      adjusted: false,
      reason: "preferred_available" as const,
      footprint_class: "node-standard" as const,
      algorithm_version: 1,
    };
    for (const store of [first, second])
      store.mergeAuthoritativeCreation({
        node: node("node-shared"),
        interfaces: [],
        placement_assignment: assignment,
        laboratory_revision: 2,
      });
    expect(first.active!.placements).toEqual(second.active!.placements);
    expect(first.active!.nodes).toEqual(second.active!.nodes);
  });

  it("keeps snapshot coordinates when a refreshed store replaces event state", () => {
    const store = useLaboratoryStore();
    store.active = snapshot();
    store.applyEvent({
      sequence: 1,
      type: "topology.placements_changed",
      laboratory_id: "lab",
      resource_type: "laboratory",
      resource_id: "lab",
      revision: 2,
      data: { placements: [placement("node-1", 640, 360, 2)] },
    });
    const refreshed = snapshot();
    refreshed.laboratory.revision = 2;
    refreshed.event_sequence = 1;
    refreshed.placements = [placement("node-1", 640, 360, 2)];
    store.active = refreshed;
    store.sequence = refreshed.event_sequence;
    expect(store.active.placements).toEqual([
      placement("node-1", 640, 360, 2),
    ]);
  });
});

function snapshot(): TopologySnapshot {
  return {
    laboratory: {
      id: "lab",
      name: "lab",
      description: "",
      revision: 1,
      recovery_policy: "auto_restore" as const,
      lifecycle_state: "active",
    },
    nodes: [],
    interfaces: [],
    links: [],
    network_objects: [],
    network_attachments: [],
    network_object_links: [],
    placements: [],
    event_sequence: 0,
  };
}

function node(id: string): Node {
  return {
    id,
    laboratory_id: "lab",
    name: id,
    kind: "docker" as const,
    revision: 1,
    desired_state: "stopped" as const,
    observed_state: "stopped" as const,
    cpu_count: 1,
    cpu_quota_micros: 0,
    memory_mib: 128,
    storage_gib: 1,
    interface_limit: 1,
    process_limit: 0,
  };
}

function placement(
  resourceId: string,
  x: number,
  y: number,
  revision: number,
): TopologyPlacement {
  return {
    laboratory_id: "lab",
    resource_id: resourceId,
    resource_type: "node" as const,
    x,
    y,
    revision,
  };
}
