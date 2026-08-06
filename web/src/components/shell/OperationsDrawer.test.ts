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

  it("keeps localized tabs separate from the scrollable content region", () => {
    const wrapper = mount(OperationsDrawer, {
      props: { modelValue: "tasks", tasks: [], nodes: [] },
    });
    expect(wrapper.text()).toContain("任务 (0)");
    expect(wrapper.text()).toContain("终端");
    expect(wrapper.text()).toContain("抓包");
    expect(
      wrapper.get('[data-layout-region="operations-content"]').classes(),
    ).toContain("overflow-auto");
  });
});
