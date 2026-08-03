import { mount } from "@vue/test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { nextTick, watchEffect } from "vue";
import { defaultWorkspacePreferences } from "@/composables/useWorkspacePreferences";
import { nodeFactory } from "@/test/factories";
let captured: unknown;
vi.mock("@/components/charts/EChart.vue", () => ({
  default: {
    props: ["option", "ariaLabel"],
    emits: [
      "chartClick",
      "canvasWheel",
      "nodeDragStart",
      "nodeDrag",
      "graphRoam",
    ],
    setup(props: { option: unknown }, { expose }: { expose: (value: object) => void }) {
      watchEffect(() => {
        captured = props.option;
      });
      expose({
        graphItemPixel: () => ({ x: 100, y: 80 }),
        dataPointAtCanvasCenter: () => ({ x: 0, y: 0 }),
      });
    },
    template: `<div>
      <button data-chart @click="$emit('chartClick',{data:{id:'node-1',resourceType:'node'},event:{event:{offsetX:10,offsetY:10}}})">chart</button>
      <button data-connector @click="$emit('chartClick',{data:{id:'connector:node-1',resourceType:'connector',ownerId:'node-1'}})">connector</button>
      <button data-roam @click="$emit('graphRoam',{zoom:2,centerX:5,centerY:4})">roam</button>
      <button data-wheel @click="$emit('graphRoam',{zoom:2.2,centerX:-5,centerY:2})">wheel</button>
      <button data-drag @click="$emit('nodeDragStart',{data:{id:'node-1',resourceType:'node'},event:{offsetX:10,offsetY:10}});$emit('nodeDrag',{data:{id:'node-1'},event:{offsetX:30,offsetY:20},graphPoint:{x:12,y:34}})">drag</button>
    </div>`,
  },
}));
import TopologyCanvas from "./TopologyCanvas.vue";
describe("TopologyCanvas", () => {
  beforeEach(() => {
    captured = undefined;
  });

  it("builds stable graph IDs and emits selection", async () => {
    const wrapper = mount(TopologyCanvas, {
      props: {
        nodes: [nodeFactory()],
        interfaces: [],
        links: [],
        networkObjects: [],
        preferences: defaultWorkspacePreferences("lab"),
      },
    });
    const option = captured as {
      series: Array<{ data: Array<{ id: string }> }>;
    };
    expect(option.series[0].data[0].id).toBe("node-1");
    await wrapper.get("[data-chart]").trigger("click");
    expect(wrapper.emitted("select")?.[0]?.slice(0, 2)).toEqual([
      "node-1",
      "node",
    ]);
  });
  it("normalizes roam and drag completion", async () => {
    const wrapper = mount(TopologyCanvas, {
      props: {
        nodes: [nodeFactory()],
        interfaces: [],
        links: [],
        networkObjects: [],
        preferences: defaultWorkspacePreferences("lab"),
      },
    });
    await wrapper.get("[data-roam]").trigger("click");
    expect(wrapper.emitted("viewport")?.[0]?.[0]).toEqual({
      centerX: 5,
      centerY: 4,
      zoom: 2,
    });
    await wrapper.get("[data-drag]").trigger("click");
    expect(wrapper.emitted("move")?.[0]).toEqual(["node-1", 12, 34]);
    await wrapper.get("[data-wheel]").trigger("click");
    const wheelViewport = wrapper.emitted("viewport")?.[1]?.[0] as {
      centerX: number;
      centerY: number;
      zoom: number;
    };
    expect(wheelViewport.zoom).toBeGreaterThan(1);
    expect(wheelViewport.centerX).toBeLessThan(0);
  });
  it("keeps the canvas reachable at minimum height", () => {
    const wrapper = mount(TopologyCanvas, {
      props: {
        nodes: [nodeFactory()],
        interfaces: [],
        links: [],
        networkObjects: [],
        preferences: defaultWorkspacePreferences("lab"),
      },
    });
    expect(wrapper.classes()).toContain("min-h-[320px]");
  });
  it("renders and activates a connector for the selected node", async () => {
    const wrapper = mount(TopologyCanvas, {
      props: {
        nodes: [nodeFactory()],
        interfaces: [
          {
            id: "if-1",
            node_id: "node-1",
            slot: 0,
            name: "eth0",
            driver: "virtio-net-pci",
            mac_address: "02:00:00:00:00:01",
            operational_state: "up",
            revision: 1,
          },
        ],
        links: [],
        networkObjects: [],
        preferences: defaultWorkspacePreferences("lab"),
        selectedIds: ["node-1"],
      },
    });
    await nextTick();
    expect(wrapper.find("[data-topology-connector]").exists()).toBe(true);
    await wrapper.get("[data-topology-connector]").trigger("click");
    expect(wrapper.emitted("connector")?.[0]).toEqual(["node-1"]);
  });

  it("renders parallel object links with distinct readable routes", () => {
    mount(TopologyCanvas, {
      props: {
        nodes: [],
        interfaces: [],
        links: [],
        networkObjects: [
          {
            id: "a",
            laboratory_id: "lab",
            name: "Switch A",
            kind: "switch_l2",
            revision: 1,
            desired_state: "active",
            observed_state: "active",
            config: {},
          },
          {
            id: "b",
            laboratory_id: "lab",
            name: "Switch B",
            kind: "switch_l2",
            revision: 1,
            desired_state: "active",
            observed_state: "active",
            config: {},
          },
        ],
        networkObjectLinks: [
          {
            id: "ol-1",
            laboratory_id: "lab",
            object_a_id: "a",
            port_a_name: "swp1",
            object_b_id: "b",
            port_b_name: "swp1",
            revision: 1,
            desired_state: "connected",
            observed_state: "connected",
          },
          {
            id: "ol-2",
            laboratory_id: "lab",
            object_a_id: "a",
            port_a_name: "swp2",
            object_b_id: "b",
            port_b_name: "swp2",
            revision: 1,
            desired_state: "connected",
            observed_state: "connected",
          },
        ],
        preferences: defaultWorkspacePreferences("lab"),
      },
    });
    const option = captured as {
      series: Array<{
        links: Array<{
          id: string;
          label: string;
          lineStyle: { curveness: number };
        }>;
      }>;
    };
    const objectLinks = option.series[0].links.filter((item) =>
      item.id.startsWith("ol-"),
    );
    expect(objectLinks.map((item) => item.label)).toEqual([
      "Switch A:swp1 ↔ Switch B:swp1",
      "Switch A:swp2 ↔ Switch B:swp2",
    ]);
    expect(objectLinks[0].lineStyle.curveness).not.toBe(
      objectLinks[1].lineStyle.curveness,
    );
  });
});
