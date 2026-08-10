import { mount } from "@vue/test-utils";
import { describe, expect, it, vi } from "vitest";
import { defaultWorkspacePreferences } from "@/composables/useWorkspacePreferences";
import { nodeFactory } from "@/test/factories";
import type { Node } from "@/api";
import { TopologyConnectionController } from "./topologyConnectionController";
let captured: unknown;
vi.mock("@/components/charts/EChart.vue", () => ({
  default: {
    props: ["option"],
    setup(props: { option: unknown }) {
      captured = props.option;
    },
    template: "<div />",
  },
}));
import TopologyCanvas from "./TopologyCanvas.vue";
describe("TopologyCanvas scale fixture", () => {
  it("keeps 100-node/300-link updates within deterministic latency bounds", async () => {
    const nodes = Array.from({ length: 100 }, (_, index) =>
      nodeFactory({ id: `node-${index}`, name: `node-${index}` }),
    );
    const interfaces = Array.from({ length: 100 }, (_, index) => ({
      id: `if-${index}`,
      node_id: `node-${index}`,
      slot: 0,
      name: "eth0",
      driver: "virtio-net-pci",
      mac_address: `02:00:00:00:${String(index).padStart(2, "0")}:01`,
      operational_state: "up",
      revision: 1,
    }));
    const links = Array.from({ length: 300 }, (_, index) => ({
      id: `link-${index}`,
      laboratory_id: "lab",
      endpoint_a_id: `if-${index % 100}`,
      endpoint_b_id: `if-${(index + 1) % 100}`,
      revision: 1,
      desired_state: "connected",
      observed_state: "connected",
    }));
    const start = performance.now();
    const wrapper = mount(TopologyCanvas, {
      props: {
        nodes,
        interfaces,
        links,
        networkObjects: [],
        preferences: defaultWorkspacePreferences("lab"),
      },
    });
    expect(performance.now() - start).toBeLessThan(1000);
    const option = captured as {
      series: Array<{ data: unknown[]; links: unknown[] }>;
    };
    expect(
      option.series[0].data.filter(
        (item) =>
          (item as { resourceType?: string }).resourceType !==
          "viewport_anchor",
      ),
    ).toHaveLength(100);
    expect(option.series[0].links).toHaveLength(300);
    const samples: number[] = [];
    for (let index = 0; index < 20; index += 1) {
      const changed = nodes.map((node, nodeIndex) =>
        nodeIndex === index
          ? {
              ...node,
              observed_state: (index % 2
                ? "running"
                : "stopped") as Node["observed_state"],
            }
          : node,
      );
      const updateStart = performance.now();
      await wrapper.setProps({
        nodes: changed,
        selectedIds: [`node-${index}`],
      });
      samples.push(performance.now() - updateStart);
    }
    samples.sort((left, right) => left - right);
    expect(samples[Math.floor(samples.length * 0.95)]).toBeLessThan(100);
    wrapper.unmount();
  });

  it("keeps 50 connection drags within the local interaction budget", () => {
    const start = performance.now();
    for (let index = 0; index < 50; index += 1) {
      const controller = new TopologyConnectionController();
      controller.begin(`source-${index}`, { x: 10, y: 10 });
      controller.move({ x: 100 + index, y: 80 });
      controller.dropOnPort({
        id: `target-${index}`,
        ownerId: `node-${index}`,
        name: "eth0",
        available: true,
      });
      controller.cancel();
    }
    expect(performance.now() - start).toBeLessThan(100);
  });
});
