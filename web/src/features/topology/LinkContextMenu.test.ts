import { mount } from "@vue/test-utils";
import { describe, expect, it } from "vitest";
import LinkContextMenu from "./LinkContextMenu.vue";

describe("LinkContextMenu", () => {
  it("exposes inspect, reconnect, route, and disconnect actions", async () => {
    const wrapper = mount(LinkContextMenu);
    const labels = [
      ["检查", "inspect"],
      ["重新连接端点", "reconnect"],
      ["编辑本地路由", "route"],
      ["断开连接", "disconnect"],
    ] as const;
    for (const [label, event] of labels) {
      await wrapper.get("button").trigger("click");
      await wrapper
        .get(`[aria-label="链路操作"]`)
        .get(`button`)
        .trigger("click");
      if (label === "检查") expect(wrapper.emitted(event)).toHaveLength(1);
    }
  });

  it("offers inspect and delete only for an object link", async () => {
    const wrapper = mount(LinkContextMenu, {
      props: { kind: "network_object_link" },
    });
    await wrapper.get("button").trigger("click");
    const menu = wrapper.get('[aria-label="链路操作"]');
    expect(menu.text()).toContain("检查");
    expect(menu.text()).toContain("删除链路");
    expect(menu.text()).not.toContain("重新连接端点");
    expect(menu.text()).not.toContain("编辑本地路由");
    const deleteButton = menu
      .findAll("button")
      .find((button) => button.text().includes("删除链路"));
    expect(deleteButton).toBeDefined();
    await deleteButton!.trigger("click");
    expect(wrapper.emitted("delete")).toHaveLength(1);
  });

  it("keeps destructive and disabled states readable without icon overlap", async () => {
    const wrapper = mount(LinkContextMenu, {
      props: { kind: "network_object_link", pending: true },
    });
    await wrapper.get("button").trigger("click");
    const deleteButton = wrapper
      .get('[aria-label="链路操作"]')
      .findAll("button")
      .find((button) => button.text().includes("正在删除"));
    expect(deleteButton).toBeDefined();
    expect(deleteButton!.attributes("disabled")).toBeDefined();
    expect(deleteButton!.text()).toContain("正在删除");
    expect(deleteButton!.find("svg").classes()).toContain("shrink-0");
  });

  it("allows a network attachment to use the unified delete action", async () => {
    const wrapper = mount(LinkContextMenu, {
      props: { kind: "network_attachment" },
    });
    await wrapper.get("button").trigger("click");
    const deleteButton = wrapper
      .get('[aria-label="链路操作"]')
      .findAll("button")
      .find((button) => button.text().includes("解除附件"));
    expect(deleteButton?.attributes("disabled")).toBeUndefined();
    await deleteButton!.trigger("click");
    expect(wrapper.emitted("delete")).toHaveLength(1);
  });
});
