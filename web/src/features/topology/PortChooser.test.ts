import { flushPromises, mount } from "@vue/test-utils";
import { describe, expect, it } from "vitest";
import { interfaceFactory } from "@/test/factories";
import PortChooser from "./PortChooser.vue";
import type { UnifiedConnectionEndpoint } from "./topologyEndpointCompatibility";

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

  it("chooses named object ports and logical access endpoints", async () => {
    const endpoints: UnifiedConnectionEndpoint[] = [
      {
        kind: "network_object_port",
        laboratoryId: "lab",
        resourceId: "switch-a",
        portName: "eth2",
        displayName: "Switch A:eth2",
        capabilities: [],
        availability: "free",
      },
      {
        kind: "network_object_access",
        laboratoryId: "lab",
        resourceId: "bridge-a",
        displayName: "Bridge A",
        capabilities: ["multi_access"],
        availability: "free",
      },
    ];
    const wrapper = mount(PortChooser, {
      attachTo: document.body,
      props: { modelValue: true, endpoints },
    });
    const option = document.body.querySelector<HTMLButtonElement>(
      '[aria-label="使用接口 Bridge A"]',
    );
    expect(option?.textContent).toContain("逻辑接入");
    option?.click();
    await flushPromises();
    expect(wrapper.emitted("choose")?.[0]?.[0]).toEqual(endpoints[1]);
    wrapper.unmount();
  });

  it("labels source and target modes and restores focus after cancellation", async () => {
    const trigger = document.createElement("button");
    document.body.appendChild(trigger);
    trigger.focus();
    const wrapper = mount(PortChooser, {
      attachTo: document.body,
      props: {
        modelValue: true,
        mode: "source" as const,
        endpoints: [],
        "onUpdate:modelValue": (value: boolean) =>
          wrapper.setProps({ modelValue: value }),
      },
    });
    expect(document.body.textContent).toContain("选择源端点");
    expect(document.body.textContent).toContain("与拖拽连接完全一致");
    const cancel = Array.from(document.body.querySelectorAll("button")).find(
      (button) => button.textContent?.trim() === "取消",
    );
    cancel?.click();
    await flushPromises();
    expect(document.activeElement).toBe(trigger);
    wrapper.unmount();
    trigger.remove();
  });
});
