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
      "grid-cols-3",
    );
  });
});
