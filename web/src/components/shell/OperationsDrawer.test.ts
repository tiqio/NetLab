import { mount } from "@vue/test-utils";
import { describe, expect, it } from "vitest";
import OperationsDrawer from "./OperationsDrawer.vue";

describe("OperationsDrawer", () => {
  it("keeps the console mounted while Tasks is selected", async () => {
    const wrapper = mount(OperationsDrawer, {
      props: {
        modelValue: "console",
        tasks: [],
        nodes: [],
      },
    });

    const consoleWorkspace = wrapper.get("[data-global-console-workspace]");

    await wrapper.setProps({ modelValue: "tasks" });
    expect(wrapper.find("[data-global-console-workspace]").exists()).toBe(true);
    expect(consoleWorkspace.element.parentElement?.style.display).toBe("none");

    await wrapper.setProps({ modelValue: "console" });
    expect(
      wrapper.get("[data-global-console-workspace]").element.parentElement
        ?.style.display || "",
    ).not.toBe("none");
  });
});
