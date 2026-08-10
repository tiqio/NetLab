import { mount } from "@vue/test-utils";
import { describe, expect, it, vi } from "vitest";
import { defaultWorkspacePreferences } from "@/composables/useWorkspacePreferences";
import { nodeFactory, networkObjectFactory } from "@/test/factories";
vi.mock("@/components/charts/EChart.vue", () => ({
  default: {
    name: "EChart",
    props: ["option", "ariaLabel"],
    template: '<div role="img" :aria-label="ariaLabel" />',
  },
}));
import EChart from "@/components/charts/EChart.vue";
import TopologyCanvas from "./TopologyCanvas.vue";

describe("TopologyCanvas accessibility", () => {
  it("exposes keyboard instructions and non-color resource status text", () => {
    const preferences = defaultWorkspacePreferences("lab");
    preferences.reducedMotion = true;
    const wrapper = mount(TopologyCanvas, {
      props: {
        nodes: [nodeFactory({ observed_state: "failed" })],
        interfaces: [],
        links: [],
        networkObjects: [],
        preferences,
        selectedIds: ["node-1"],
      },
    });
    const region = wrapper.get('[aria-label^="拓扑画布键盘操作区"]');
    expect(region.attributes("tabindex")).toBe("0");
    expect(region.classes()).toContain("focus-visible:ring-2");
    const summary = wrapper.get('[data-testid="topology-a11y-summary"]');
    expect(summary.attributes("aria-live")).toBe("polite");
    expect(summary.text()).toContain(
      "Ubuntu: QEMU 虚拟机 · 期望 已停止 · 实际 失败，已选择",
    );
    const chart = wrapper.findComponent(EChart);
    const option = chart.props("option") as {
      series: Array<{ animationDurationUpdate: number }>;
    };
    expect(option.series[0].animationDurationUpdate).toBe(0);
  });

  it("exposes non-color endpoint availability and logical access resources", () => {
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
        networkObjects: [
          networkObjectFactory({ id: "bridge-a", kind: "bridge" }),
        ],
        preferences: defaultWorkspacePreferences("lab"),
        selectedIds: ["node-1"],
      },
    });
    const summary = wrapper.get('[data-testid="topology-a11y-summary"]');
    expect(summary.attributes("aria-live")).toBe("polite");
    expect(summary.text()).toContain("Bridge: 二层网桥");
    expect(summary.text()).toContain("Ubuntu: QEMU 虚拟机");
  });
});
