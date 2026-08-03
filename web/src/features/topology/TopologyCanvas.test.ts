import { mount } from "@vue/test-utils";
import { describe, expect, it, vi } from "vitest";
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
    setup(props: { option: unknown }) {
      captured = props.option;
    },
    template: `<div>
      <button data-chart @click="$emit('chartClick',{data:{id:'node-1',resourceType:'node'},event:{event:{offsetX:10,offsetY:10}}})">chart</button>
      <button data-connector @click="$emit('chartClick',{data:{id:'connector:node-1',resourceType:'connector',ownerId:'node-1'}})">connector</button>
      <button data-roam @click="$emit('graphRoam',{zoom:2,dx:5,dy:4})">roam</button>
      <button data-wheel @click="$emit('canvasWheel',{offsetX:100,offsetY:50,deltaY:-100,ctrlKey:false})">wheel</button>
      <button data-drag @click="$emit('nodeDragStart',{data:{id:'node-1',resourceType:'node'},event:{offsetX:10,offsetY:10}});$emit('nodeDrag',{data:{id:'node-1'},event:{offsetX:30,offsetY:20},graphPoint:{x:12,y:34}})">drag</button>
    </div>`,
  },
}));
import TopologyCanvas from "./TopologyCanvas.vue";
describe("TopologyCanvas", () => {
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
    const option = captured as {
      series: Array<{ data: Array<{ resourceType: string }> }>;
    };
    expect(
      option.series[0].data.some((item) => item.resourceType === "connector"),
    ).toBe(true);
    await wrapper.get("[data-connector]").trigger("click");
    expect(wrapper.emitted("connector")?.[0]).toEqual(["node-1"]);
  });
});
