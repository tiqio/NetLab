import { flushPromises, mount } from "@vue/test-utils";
import { describe, expect, it, vi } from "vitest";
import { api, type DeviceTemplate } from "@/api";
import DevicePalette from "./DevicePalette.vue";

function qemuTemplate(
  templateKey: string,
  displayName: string,
): DeviceTemplate {
  return {
    id: `${templateKey}-template`,
    template_key: templateKey,
    display_name: displayName,
    runtime_kind: "qemu" as const,
    created_at: "",
    versions: [
      {
        id: `${templateKey}-version`,
        template_id: `${templateKey}-template`,
        version: "V1.06",
        manifest_version: 1,
        defaults: {
          cpu_count: 1,
          cpu_quota_micros: 100000,
          memory_mib: 1024,
          disk_gib: 8,
          interfaces: 10,
          interface_name_format: "G0/%d",
        },
        capabilities: ["nic_hotplug"],
        supported_nic_drivers: ["e1000"],
        console_modes: ["telnet", "vnc"],
        runtime_options: {},
        enabled: true,
        created_at: "",
      },
    ],
  };
}

describe("DevicePalette", () => {
  it("shows categorized templates and lightweight nodes", async () => {
    vi.spyOn(api, "listTemplates").mockResolvedValue([
      {
        id: "t",
        template_key: "ubuntu",
        display_name: "Ubuntu",
        runtime_kind: "qemu",
        created_at: "",
        versions: [
          {
            id: "v",
            template_id: "t",
            version: "24.04",
            manifest_version: 1,
            defaults: {
              cpu_count: 1,
              memory_mib: 512,
              interfaces: 1,
              interface_name_format: "eth%d",
            },
            capabilities: [],
            supported_nic_drivers: [],
            console_modes: ["vnc"],
            runtime_options: {},
            enabled: true,
            created_at: "",
          },
        ],
      },
      qemuTemplate("ruijie-switch", "Ruijie Switch"),
      qemuTemplate("ruijie-router", "Ruijie Router"),
    ]);
    const wrapper = mount(DevicePalette);
    await flushPromises();
    expect(wrapper.text()).toContain("Ubuntu");
    expect(wrapper.text()).toContain("锐捷二层交换机");
    expect(wrapper.text()).toContain("轻量级二层交换机");
    expect(wrapper.text()).toContain("轻量级三层交换机");
    expect(wrapper.text()).toContain("无需 KVM");
    const layer2Button = wrapper
      .findAll("button")
      .find((button) => button.text().includes("锐捷二层交换机"));
    await layer2Button!.trigger("click");
    expect(wrapper.emitted("choose")?.[0]?.[0]).toMatchObject({
      kind: "qemu",
      name: "锐捷二层交换机",
      template: { template_key: "ruijie-switch" },
      version: { version: "V1.06" },
    });

    const lightweightL2 = wrapper
      .findAll("button")
      .find((button) => button.text().includes("轻量级二层交换机"));
    await lightweightL2!.trigger("click");
    expect(wrapper.emitted("choose")?.[1]?.[0]).toMatchObject({
      kind: "switch_l2",
      name: "轻量级二层交换机",
      networkObjectKind: "switch_l2",
    });

    const lightweightL3 = wrapper
      .findAll("button")
      .find((button) => button.text().includes("轻量级三层交换机"));
    await lightweightL3!.trigger("click");
    expect(wrapper.emitted("choose")?.[2]?.[0]).toMatchObject({
      kind: "switch_l3",
      name: "轻量级三层交换机",
      networkObjectKind: "switch_l3",
    });
  });
});
