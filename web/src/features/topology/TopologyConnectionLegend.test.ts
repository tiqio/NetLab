import { mount } from "@vue/test-utils";
import { describe, expect, it } from "vitest";
import TopologyConnectionLegend from "./TopologyConnectionLegend.vue";

describe("TopologyConnectionLegend", () => {
  it("renders Chinese reasons and emits non-destructive highlighting", async () => {
    const wrapper = mount(TopologyConnectionLegend, {
      props: { items: [{ key: "managed-nat-uplink", label: "NAT 管理上联", description: "该连接接入 NetLab 管理的地址转换和互联网出口。", count: 2, connectionIds: ["a", "b"] }] },
    });
    const item = wrapper.get('[data-semantic-marker="managed-nat-uplink"]');
    expect(item.text()).toContain("地址转换");
    await item.trigger("focus");
    expect(wrapper.emitted("highlight")?.[0]).toEqual([["a", "b"]]);
    await item.trigger("blur");
    expect(wrapper.emitted("clear")).toHaveLength(1);
  });
});
