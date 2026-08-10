import { beforeEach, describe, expect, it } from "vitest";
import { createPinia, setActivePinia } from "pinia";
import { laboratoryFactory } from "@/test/factories";
import { useLaboratoryStore } from "./laboratory";

function activeSnapshot() {
  return {
    laboratory: laboratoryFactory({ id: "lab", revision: 3 }),
    nodes: [
      {
        id: "node-a",
        laboratory_id: "lab",
        name: "A",
        kind: "docker",
        revision: 1,
        desired_state: "running",
        observed_state: "running",
      },
      {
        id: "node-b",
        laboratory_id: "lab",
        name: "B",
        kind: "docker",
        revision: 1,
        desired_state: "running",
        observed_state: "running",
      },
    ],
    interfaces: [
      { id: "if-a", node_id: "node-a", name: "eth0", slot: 0, revision: 1 },
      { id: "if-b", node_id: "node-b", name: "eth0", slot: 0, revision: 1 },
    ],
    links: [
      {
        id: "link-1",
        laboratory_id: "lab",
        endpoint_a_id: "if-a",
        endpoint_b_id: "if-b",
        revision: 1,
        desired_state: "connected",
        observed_state: "pending",
      },
    ],
    network_objects: [],
    network_attachments: [],
    network_object_links: [],
    placements: [],
    event_sequence: 10,
  } as ReturnType<typeof useLaboratoryStore>["active"];
}

describe("unified connection event convergence", () => {
  beforeEach(() => setActivePinia(createPinia()));

  it("ignores stale connection events after a newer state arrives", () => {
    const store = useLaboratoryStore();
    store.active = activeSnapshot();
    store.sequence = 10;
    store.applyEvent({
      sequence: 12,
      type: "link.state_changed",
      laboratory_id: "lab",
      resource_type: "link",
      resource_id: "link-1",
      revision: 2,
      data: { observed_state: "connected" },
    });
    store.applyEvent({
      sequence: 11,
      type: "link.state_changed",
      laboratory_id: "lab",
      resource_type: "link",
      resource_id: "link-1",
      revision: 1,
      data: { observed_state: "failed" },
    });
    expect(store.active?.links[0].observed_state).toBe("connected");
    expect(store.sequence).toBe(12);
  });

  it("converges a submitted task to its authoritative terminal state", () => {
    const store = useLaboratoryStore();
    store.active = activeSnapshot();
    store.sequence = 10;
    store.tasks = [
      {
        id: "task-1",
        kind: "link.connect",
        resource_type: "link",
        resource_id: "link-1",
        state: "running",
        progress_current: 1,
        progress_total: 2,
      },
    ];
    store.applyEvent({
      sequence: 11,
      type: "operation_task.succeeded",
      resource_type: "operation_task",
      resource_id: "task-1",
      revision: 1,
      task_id: "task-1",
      data: {
        id: "task-1",
        kind: "link.connect",
        resource_type: "link",
        resource_id: "link-1",
        state: "succeeded",
        progress_current: 2,
        progress_total: 2,
      },
    });
    expect(store.tasks).toHaveLength(1);
    expect(store.tasks[0]).toMatchObject({
      id: "task-1",
      state: "succeeded",
      progress_current: 2,
    });
  });

  it("removes incident connections when an endpoint node is deleted", () => {
    const store = useLaboratoryStore();
    store.active = activeSnapshot();
    store.sequence = 10;
    store.applyEvent({
      sequence: 11,
      type: "node.deleted",
      laboratory_id: "lab",
      resource_type: "node",
      resource_id: "node-a",
      revision: 2,
      data: {},
    });
    expect(store.active?.nodes.map((node) => node.id)).toEqual(["node-b"]);
    expect(store.active?.interfaces.map((item) => item.id)).toEqual(["if-b"]);
    expect(store.active?.links).toHaveLength(0);
  });
});
