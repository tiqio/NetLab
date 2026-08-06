import { mount } from "@vue/test-utils";
import { describe, expect, it, vi } from "vitest";
import { nodeFactory, taskFactory } from "@/test/factories";
vi.mock("@/components/charts/EChart.vue", () => ({
  default: {
    props: ["option", "ariaLabel"],
    template: '<div :aria-label="ariaLabel" />',
  },
}));
import ResourceCharts from "./ResourceCharts.vue";
describe("ResourceCharts", () => {
  it("renders ECharts resource and task statistics", () => {
    const wrapper = mount(ResourceCharts, {
      props: { node: nodeFactory(), tasks: [taskFactory()] },
    });
    expect(wrapper.findAll('[aria-label$="图"]')).toHaveLength(2);
    expect(wrapper.text()).toContain("CPU 配额（核心）");
  });
  it("gives the resource chart the full inspector width without task data", () => {
    const wrapper = mount(ResourceCharts, {
      props: { node: nodeFactory() },
    });
    expect(wrapper.findAll('[aria-label$="图"]')).toHaveLength(1);
    expect(wrapper.get('[aria-label="节点资源"] dl').classes()).toContain(
      "resource-metrics",
    );
    expect(
      wrapper.get('[data-layout-region="resource-chart"]').classes(),
    ).toContain("min-w-0");
  });

  it("keeps metrics outside the chart drawing region for long labels", () => {
    const wrapper = mount(ResourceCharts, {
      props: {
        node: nodeFactory({ name: "超长节点名称".repeat(12) }),
        tasks: [taskFactory()],
      },
    });
    const region = wrapper.get('[data-layout-region="resource-chart"]');
    expect(
      region.get("dl").element.nextElementSibling?.getAttribute("aria-label"),
    ).toBe("所选节点资源分配图");
    expect(wrapper.classes()).toContain("resource-charts");
  });
});
