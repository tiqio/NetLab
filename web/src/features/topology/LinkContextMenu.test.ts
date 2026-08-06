import { mount } from "@vue/test-utils";
import { describe, expect, it } from "vitest";
import LinkContextMenu from "./LinkContextMenu.vue";

describe("LinkContextMenu", () => {
  it("exposes inspect, reconnect, route, and disconnect actions", async () => {
    const wrapper = mount(LinkContextMenu);
    const labels = [
      ["Inspect", "inspect"],
      ["Reconnect endpoint", "reconnect"],
      ["Edit local route", "route"],
      ["Disconnect", "disconnect"],
    ] as const;
    for (const [label, event] of labels) {
      await wrapper.get("button").trigger("click");
      await wrapper
        .get(`[aria-label="链路操作"]`)
        .get(`button`)
        .trigger("click");
      if (label === "Inspect") expect(wrapper.emitted(event)).toHaveLength(1);
    }
  });

  it("offers inspect and delete only for an object link", async () => {
    const wrapper = mount(LinkContextMenu, {
      props: { objectLink: true },
    });
    await wrapper.get("button").trigger("click");
    const menu = wrapper.get('[aria-label="链路操作"]');
    expect(menu.text()).toContain("Inspect");
    expect(menu.text()).toContain("Delete link");
    expect(menu.text()).not.toContain("Reconnect endpoint");
    expect(menu.text()).not.toContain("Edit local route");
    await menu.findAll("button")[1].trigger("click");
    expect(wrapper.emitted("delete")).toHaveLength(1);
  });
});
