import { mount } from "@vue/test-utils";
import { afterEach, describe, expect, it } from "vitest";
import TopologyConnectionLegend from "./TopologyConnectionLegend.vue";

describe("TopologyConnectionLegend", () => {
  afterEach(() => document.documentElement.removeAttribute("data-theme"));

  it("renders Chinese reasons and emits non-destructive highlighting", async () => {
    const wrapper = mount(TopologyConnectionLegend, {
      props: {
        items: [
          {
            key: "managed-nat-uplink",
            label: "NAT 管理上联",
            description: "该连接接入 NetLab 管理的地址转换和互联网出口。",
            count: 2,
            connectionIds: ["a", "b"],
          },
        ],
      },
    });
    const item = wrapper.get('[data-semantic-marker="managed-nat-uplink"]');
    expect(item.text()).toContain("地址转换");
    await item.trigger("focus");
    expect(wrapper.emitted("highlight")?.[0]).toEqual([["a", "b"]]);
    await item.trigger("blur");
    expect(wrapper.emitted("clear")).toHaveLength(1);
  });

  it.each(["light", "dark"])(
    "supports collapse, scrolling, and accessible names in %s theme",
    async (theme) => {
      document.documentElement.setAttribute("data-theme", theme);
      const wrapper = mount(TopologyConnectionLegend, {
        props: {
          items: [
            {
              key: "managed-nat-uplink",
              label: "NAT 管理上联",
              description: "NAT 描述",
              count: 1,
              connectionIds: ["nat"],
            },
            {
              key: "shared-broadcast-domain",
              label: "共享广播域",
              description: "广播域描述",
              count: 3,
              connectionIds: ["a", "b", "c"],
            },
          ],
        },
      });
      const legend = wrapper.get('[data-testid="topology-connection-legend"]');
      expect(legend.attributes("aria-label")).toBe("连接语义图例");
      expect(legend.find(".overflow-y-auto").exists()).toBe(true);
      const toggle = legend.get('button[aria-expanded="true"]');
      expect(toggle.attributes("aria-label")).toContain("收起");
      await toggle.trigger("click");
      expect(legend.attributes("data-collapsed")).toBe("true");
      expect(
        legend.get('button[aria-expanded="false"]').attributes("aria-label"),
      ).toContain("展开");
    },
  );
});
