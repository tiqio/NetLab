import { flushPromises, mount } from "@vue/test-utils";
import { describe, expect, it } from "vitest";
import { interfaceFactory } from "@/test/factories";
import PortChooser from "./PortChooser.vue";

describe("PortChooser", () => {
  it("shows available ports, emits the choice, and supports cancellation", async () => {
    const wrapper = mount(PortChooser, {
      attachTo: document.body,
      props: {
        modelValue: true,
        interfaces: [interfaceFactory({ id: "if-1", name: "eth0" })],
        "onUpdate:modelValue": (value: boolean) =>
          wrapper.setProps({ modelValue: value }),
      },
    });
    const option = document.body.querySelector<HTMLButtonElement>(
      '[aria-label="使用接口 eth0"]',
    );
    expect(option).not.toBeNull();
    await flushPromises();
    expect(document.activeElement).toBe(option);
    await option!.click();
    expect(wrapper.emitted("choose")?.[0]?.[0]).toMatchObject({ id: "if-1" });
    wrapper.unmount();
  });

  it("cancels without choosing an interface", async () => {
    const wrapper = mount(PortChooser, {
      attachTo: document.body,
      props: {
        modelValue: true,
        interfaces: [interfaceFactory()],
        "onUpdate:modelValue": (value: boolean) =>
          wrapper.setProps({ modelValue: value }),
      },
    });
    const cancel = Array.from(document.body.querySelectorAll("button")).find(
      (button) => button.textContent?.trim() === "取消",
    );
    expect(cancel).not.toBeUndefined();
    cancel!.click();
    await flushPromises();
    expect(wrapper.emitted("cancel")).toHaveLength(1);
    expect(wrapper.emitted("choose")).toBeUndefined();
    wrapper.unmount();
  });

  it("shows capture-specific guidance", () => {
    const wrapper = mount(PortChooser, {
      attachTo: document.body,
      props: {
        modelValue: true,
        title: "选择抓包接口",
        description: "选择用于 Wireshark 实时抓包的接口。",
        interfaces: [interfaceFactory({ name: "ens0" })],
      },
    });
    expect(document.body.textContent).toContain("选择抓包接口");
    expect(document.body.textContent).toContain(
      "选择用于 Wireshark 实时抓包的接口。",
    );
    wrapper.unmount();
  });
});
