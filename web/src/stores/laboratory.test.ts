import { beforeEach, describe, expect, it, vi } from "vitest";
import { createPinia, setActivePinia } from "pinia";
import { api } from "@/api";
import { laboratoryFactory } from "@/test/factories";
import { shouldReloadEventStream, useLaboratoryStore } from "./laboratory";

describe("laboratory store", () => {
  beforeEach(() => setActivePinia(createPinia()));
  it("applies ordered events once", () => {
    const store = useLaboratoryStore();
    store.active = {
      laboratory: {
        id: "lab",
        name: "old",
        description: "",
        revision: 1,
        recovery_policy: "auto_restore",
        lifecycle_state: "active",
      },
      nodes: [],
      interfaces: [],
      links: [],
      network_objects: [],
      placements: [],
      event_sequence: 1,
    };
    store.sequence = 1;
    store.applyEvent({
      sequence: 2,
      type: "laboratory.updated",
      laboratory_id: "lab",
      resource_type: "laboratory",
      resource_id: "lab",
      revision: 2,
      data: { name: "new" },
    });
    store.applyEvent({
      sequence: 1,
      type: "laboratory.updated",
      laboratory_id: "lab",
      resource_type: "laboratory",
      resource_id: "lab",
      revision: 1,
      data: { name: "stale" },
    });
    expect(store.active?.laboratory.name).toBe("new");
    expect(store.sequence).toBe(2);
  });

  it("applies observed node state changes at the current revision", () => {
    const store = useLaboratoryStore();
    store.active = {
      laboratory: laboratoryFactory({ id: "lab" }),
      nodes: [
        {
          id: "node-1",
          laboratory_id: "lab",
          name: "busybox",
          kind: "docker",
          revision: 4,
          desired_state: "running",
          observed_state: "stopped",
          cpu_count: 1,
          cpu_quota_micros: 0,
          memory_mib: 128,
          storage_gib: 0,
          interface_limit: 4,
          process_limit: 64,
          config: {},
        },
      ],
      interfaces: [],
      links: [],
      network_objects: [],
      placements: [],
      event_sequence: 10,
    };
    store.sequence = 10;

    store.applyEvent({
      sequence: 11,
      type: "node.observed_state_changed",
      laboratory_id: "lab",
      resource_type: "node",
      resource_id: "node-1",
      revision: 4,
      data: { observed_state: "running" },
    });

    expect(store.active!.nodes[0].observed_state).toBe("running");
  });

  it("applies observed link state changes at the current revision", () => {
    const store = useLaboratoryStore();
    store.active = {
      laboratory: laboratoryFactory({ id: "lab" }),
      nodes: [],
      interfaces: [],
      links: [
        {
          id: "link-1",
          laboratory_id: "lab",
          endpoint_a_id: "if-a",
          endpoint_b_id: "if-b",
          revision: 3,
          desired_state: "connected",
          observed_state: "pending",
        },
      ],
      network_objects: [],
      placements: [],
      event_sequence: 20,
    };
    store.sequence = 20;

    store.applyEvent({
      sequence: 21,
      type: "link.observed_state_changed",
      laboratory_id: "lab",
      resource_type: "link",
      resource_id: "link-1",
      revision: 3,
      data: { observed_state: "connected" },
    });

    expect(store.active!.links[0].observed_state).toBe("connected");
  });

  it("reconciles shared network-object links without a refresh", () => {
    const store = useLaboratoryStore();
    store.active = {
      laboratory: laboratoryFactory({ id: "lab" }),
      nodes: [],
      interfaces: [],
      links: [],
      network_objects: [],
      network_object_links: [],
      placements: [],
      event_sequence: 30,
    };
    store.sequence = 30;

    store.applyEvent({
      sequence: 31,
      type: "network_object_link.created",
      laboratory_id: "lab",
      resource_type: "network_object_link",
      resource_id: "object-link-1",
      revision: 1,
      data: {
        id: "object-link-1",
        laboratory_id: "lab",
        object_a_id: "switch-a",
        port_a_name: "swp1",
        object_b_id: "switch-b",
        port_b_name: "swp2",
        desired_state: "connected",
        observed_state: "pending",
      },
    });
    expect(store.active.network_object_links?.[0]).toMatchObject({
      id: "object-link-1",
      observed_state: "pending",
    });

    store.applyEvent({
      sequence: 32,
      type: "network_object_link.state_changed",
      laboratory_id: "lab",
      resource_type: "network_object_link",
      resource_id: "object-link-1",
      revision: 1,
      data: { observed_state: "connected" },
    });
    expect(store.active.network_object_links?.[0].observed_state).toBe(
      "connected",
    );

    store.applyEvent({
      sequence: 33,
      type: "network_object_link.deleted",
      laboratory_id: "lab",
      resource_type: "network_object_link",
      resource_id: "object-link-1",
      revision: 2,
      data: {},
    });
    expect(store.active.network_object_links).toHaveLength(0);
  });

  it("normalizes object links when reopening a laboratory", async () => {
    const store = useLaboratoryStore();
    vi.spyOn(api, "getLab").mockResolvedValueOnce({
      laboratory: laboratoryFactory({ id: "lab" }),
      nodes: [],
      interfaces: [],
      links: [],
      network_objects: [],
      network_object_links: [
        {
          id: "object-link-1",
          laboratory_id: "lab",
          object_a_id: "switch-a",
          port_a_name: "swp1",
          object_b_id: "switch-b",
          port_b_name: "swp1",
          revision: 4,
          desired_state: "connected",
          observed_state: "connected",
        },
      ],
      placements: [],
      event_sequence: 40,
    });

    await store.open("lab");

    expect(store.active?.network_object_links?.[0]).toMatchObject({
      id: "object-link-1",
      revision: 4,
      observed_state: "connected",
    });
  });

  it("incrementally reconciles topology resources and tasks", () => {
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
      network_objects: [],
      placements: [],
      event_sequence: 0,
    };
    store.applyEvent({
      sequence: 1,
      type: "node.created",
      laboratory_id: "lab",
      resource_type: "node",
      resource_id: "node-1",
      revision: 1,
      data: {
        id: "node-1",
        laboratory_id: "lab",
        name: "router",
        kind: "qemu",
        desired_state: "stopped",
        observed_state: "stopped",
        interfaces: [
          {
            id: "if-1",
            node_id: "node-1",
            name: "eth0",
            driver: "virtio-net-pci",
            mac_address: "02:00:00:00:00:01",
            operational_state: "down",
            revision: 1,
          },
        ],
      },
    });
    store.applyEvent({
      sequence: 2,
      type: "link.created",
      laboratory_id: "lab",
      resource_type: "link",
      resource_id: "link-1",
      revision: 1,
      data: {
        id: "link-1",
        laboratory_id: "lab",
        endpoint_a_id: "if-1",
        endpoint_b_id: "if-2",
        desired_state: "connected",
        observed_state: "connected",
      },
    });
    store.applyEvent({
      sequence: 3,
      type: "network_object.created",
      laboratory_id: "lab",
      resource_type: "network_object",
      resource_id: "net-1",
      revision: 1,
      data: {
        id: "net-1",
        laboratory_id: "lab",
        name: "lan",
        kind: "bridge",
        desired_state: "present",
        observed_state: "present",
        config: {},
      },
    });
    store.applyEvent({
      sequence: 4,
      type: "task.updated",
      resource_type: "operation_task",
      resource_id: "task-1",
      revision: 0,
      data: {
        id: "task-1",
        kind: "node.start",
        resource_type: "node",
        resource_id: "node-1",
        state: "running",
        progress_current: 1,
        progress_total: 2,
        created_at: "2026-07-24T00:00:00Z",
      },
    });

    expect(store.active!.nodes.map((item) => item.id)).toEqual(["node-1"]);
    expect(store.active!.interfaces[0].desired_link_id).toBe("link-1");
    expect(store.active!.network_objects?.[0].id).toBe("net-1");
    expect(store.tasks[0].state).toBe("running");

    store.applyEvent({
      sequence: 5,
      type: "link.deleted",
      laboratory_id: "lab",
      resource_type: "link",
      resource_id: "link-1",
      revision: 2,
      data: {},
    });
    expect(store.active!.links).toHaveLength(0);
    expect(store.active!.interfaces[0].desired_link_id).toBeUndefined();
  });

  it("tracks explicit synchronization state", () => {
    const store = useLaboratoryStore();
    expect(store.syncState).toBe("initializing");
    store.syncState = "reconnecting";
    expect(store.syncState).toBe("reconnecting");
  });

  it("hides a deleting laboratory immediately and clears it when active", () => {
    const store = useLaboratoryStore();
    const deleting = laboratoryFactory({ id: "lab-deleting" });
    const fallback = laboratoryFactory({ id: "lab-fallback" });
    store.labs = [deleting, fallback];
    store.active = {
      laboratory: deleting,
      nodes: [],
      interfaces: [],
      links: [],
      network_objects: [],
      placements: [],
      event_sequence: 1,
    };

    store.hideLaboratory(deleting.id);

    expect(store.labs.map((laboratory) => laboratory.id)).toEqual([
      "lab-fallback",
    ]);
    expect(store.active).toBeNull();
  });

  it("filters deleting laboratories from server list refreshes", async () => {
    const store = useLaboratoryStore();
    vi.spyOn(api, "listLabs").mockResolvedValueOnce([
      laboratoryFactory({ id: "active" }),
      laboratoryFactory({ id: "deleting", lifecycle_state: "deleting" }),
    ]);

    await store.loadLabs();

    expect(store.labs.map((laboratory) => laboratory.id)).toEqual(["active"]);
  });

  it("does not restore a locally hidden laboratory from a stale list response", async () => {
    const store = useLaboratoryStore();
    const active = laboratoryFactory({ id: "active" });
    const deleting = laboratoryFactory({ id: "deleting" });
    store.labs = [active, deleting];
    store.hideLaboratory(deleting.id);
    vi.spyOn(api, "listLabs").mockResolvedValueOnce([active, deleting]);

    await store.loadLabs();

    expect(store.labs.map((laboratory) => laboratory.id)).toEqual(["active"]);
    expect(store.hiddenLaboratoryIds).toEqual(["deleting"]);
  });

  it("reconciles deleting and failed deletion events across clients", async () => {
    const store = useLaboratoryStore();
    const deleting = laboratoryFactory({ id: "lab-deleting" });
    store.labs = [deleting];
    store.active = {
      laboratory: deleting,
      nodes: [],
      interfaces: [],
      links: [],
      network_objects: [],
      placements: [],
      event_sequence: 1,
    };
    const restored = laboratoryFactory({
      id: "lab-deleting",
      lifecycle_state: "delete_failed",
    });
    const listLabs = vi
      .spyOn(api, "listLabs")
      .mockResolvedValueOnce([restored]);

    store.applyEvent({
      sequence: 2,
      type: "laboratory.deleting",
      laboratory_id: deleting.id,
      resource_type: "laboratory",
      resource_id: deleting.id,
      revision: 2,
      data: {},
    });
    expect(store.labs).toHaveLength(0);
    expect(store.active).toBeNull();

    store.applyEvent({
      sequence: 3,
      type: "laboratory.delete_failed",
      laboratory_id: deleting.id,
      resource_type: "laboratory",
      resource_id: deleting.id,
      revision: 2,
      data: {},
    });
    await vi.waitFor(() => expect(listLabs).toHaveBeenCalledOnce());
    await vi.waitFor(() =>
      expect(store.labs[0]?.lifecycle_state).toBe("delete_failed"),
    );
    expect(store.hiddenLaboratoryIds).toHaveLength(0);
  });

  it("advances the global sequence without mutating another laboratory", () => {
    const store = useLaboratoryStore();
    store.active = {
      laboratory: {
        id: "lab-a",
        name: "current",
        description: "",
        revision: 4,
        recovery_policy: "auto_restore",
        lifecycle_state: "active",
      },
      nodes: [],
      interfaces: [],
      links: [],
      network_objects: [],
      placements: [],
      event_sequence: 8,
    };
    store.sequence = 8;

    store.applyEvent({
      sequence: 9,
      type: "laboratory.updated",
      laboratory_id: "lab-b",
      resource_type: "laboratory",
      resource_id: "lab-b",
      revision: 10,
      data: { name: "other" },
    });

    expect(store.sequence).toBe(9);
    expect(store.active.laboratory.name).toBe("current");
    expect(store.active.laboratory.revision).toBe(4);
  });

  it("ignores stale updates but applies terminal deletion events", () => {
    const store = useLaboratoryStore();
    store.active = {
      laboratory: {
        id: "lab",
        name: "current",
        description: "",
        revision: 8,
        recovery_policy: "auto_restore",
        lifecycle_state: "active",
      },
      nodes: [
        {
          id: "node-1",
          laboratory_id: "lab",
          name: "current-node",
          kind: "qemu",
          revision: 6,
          desired_state: "running",
          observed_state: "running",
          cpu_count: 1,
          cpu_quota_micros: 100000,
          memory_mib: 512,
          storage_gib: 4,
          interface_limit: 2,
          process_limit: 128,
        },
      ],
      interfaces: [],
      links: [
        {
          id: "link-1",
          laboratory_id: "lab",
          endpoint_a_id: "if-a",
          endpoint_b_id: "if-b",
          revision: 7,
          desired_state: "connected",
          observed_state: "connected",
        },
      ],
      network_objects: [],
      placements: [
        {
          laboratory_id: "lab",
          resource_id: "node-1",
          resource_type: "node",
          x: 100,
          y: 200,
          revision: 5,
        },
      ],
      event_sequence: 20,
    };
    store.sequence = 20;

    store.applyEvent({
      sequence: 21,
      type: "node.updated",
      laboratory_id: "lab",
      resource_type: "node",
      resource_id: "node-1",
      revision: 5,
      data: { name: "stale-node" },
    });
    store.applyEvent({
      sequence: 22,
      type: "link.deleted",
      laboratory_id: "lab",
      resource_type: "link",
      resource_id: "link-1",
      revision: 6,
      data: {},
    });
    store.applyEvent({
      sequence: 23,
      type: "topology.placements_changed",
      laboratory_id: "lab",
      resource_type: "laboratory",
      resource_id: "lab",
      revision: 7,
      data: {
        placements: [
          {
            laboratory_id: "lab",
            resource_id: "node-1",
            resource_type: "node",
            x: 1,
            y: 2,
            revision: 4,
          },
        ],
      },
    });

    expect(store.sequence).toBe(23);
    expect(store.active.nodes[0].name).toBe("current-node");
    expect(store.active.links).toHaveLength(0);
    expect(store.active.placements).toHaveLength(1);
    expect(store.active.placements[0]).toMatchObject({
      x: 100,
      y: 200,
      revision: 5,
    });
    expect(store.active.laboratory.revision).toBe(8);
  });

  it("removes a deleted node and its topology dependencies immediately", () => {
    const store = useLaboratoryStore();
    store.active = {
      laboratory: laboratoryFactory({ id: "lab", revision: 8 }),
      nodes: [
        {
          id: "node-1",
          laboratory_id: "lab",
          name: "ubuntu",
          kind: "docker",
          revision: 9,
          desired_state: "stopped",
          observed_state: "stopped",
          cpu_count: 1,
          cpu_quota_micros: 100000,
          memory_mib: 512,
          storage_gib: 4,
          interface_limit: 2,
          process_limit: 128,
        },
      ],
      interfaces: [
        {
          id: "if-1",
          node_id: "node-1",
          slot: 0,
          name: "eth0",
          driver: "virtio-net-pci",
          mac_address: "02:00:00:00:00:01",
          operational_state: "down",
          desired_link_id: "link-1",
          revision: 4,
        },
      ],
      links: [
        {
          id: "link-1",
          laboratory_id: "lab",
          endpoint_a_id: "if-1",
          endpoint_b_id: "if-2",
          revision: 7,
          desired_state: "connected",
          observed_state: "connected",
        },
      ],
      network_objects: [],
      placements: [
        {
          laboratory_id: "lab",
          resource_id: "node-1",
          resource_type: "node",
          x: 100,
          y: 200,
          revision: 5,
        },
      ],
      event_sequence: 30,
    };
    store.sequence = 30;

    store.applyEvent({
      sequence: 31,
      type: "node.deleted",
      laboratory_id: "lab",
      resource_type: "node",
      resource_id: "node-1",
      revision: 2,
      data: {},
    });

    expect(store.active!.nodes).toHaveLength(0);
    expect(store.active!.interfaces).toHaveLength(0);
    expect(store.active!.links).toHaveLength(0);
    expect(store.active!.placements).toHaveLength(0);
  });

  it("detects reset requests and gaps before applying an event", () => {
    expect(
      shouldReloadEventStream(10, {
        sequence: 11,
        type: "stream.reset_required",
        resource_type: "stream",
        resource_id: "events",
        revision: 0,
        data: {},
      }),
    ).toBe(true);
    expect(
      shouldReloadEventStream(10, {
        sequence: 12,
        type: "node.updated",
        resource_type: "node",
        resource_id: "node-1",
        revision: 2,
        data: {},
      }),
    ).toBe(true);
    expect(
      shouldReloadEventStream(10, {
        sequence: 11,
        type: "node.updated",
        resource_type: "node",
        resource_id: "node-1",
        revision: 2,
        data: {},
      }),
    ).toBe(false);
  });
});
