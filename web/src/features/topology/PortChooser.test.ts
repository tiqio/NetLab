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
      '[aria-label="Use eth0"]',
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
      (button) => button.textContent?.trim() === "Cancel",
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
        title: "Choose interface to capture",
        description: "Select the interface for live Wireshark capture.",
        interfaces: [interfaceFactory({ name: "ens0" })],
      },
    });
    expect(document.body.textContent).toContain("Choose interface to capture");
    expect(document.body.textContent).toContain(
      "Select the interface for live Wireshark capture.",
    );
    wrapper.unmount();
  });
});
