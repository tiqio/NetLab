import { beforeEach, describe, expect, it } from "vitest";
import { createPinia, setActivePinia } from "pinia";
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
});
