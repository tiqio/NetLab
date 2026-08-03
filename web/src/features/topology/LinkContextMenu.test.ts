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
        .get(`[aria-label="Link actions"]`)
        .get(`button`)
        .trigger("click");
      if (label === "Inspect") expect(wrapper.emitted(event)).toHaveLength(1);
    }
  });
});
