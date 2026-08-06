import { flushPromises, mount } from "@vue/test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { api, type DeviceTemplate } from "@/api";
import TemplatePicker from "./TemplatePicker.vue";

const catalog: DeviceTemplate[] = [
  {
    id: "template-1",
    template_key: "ubuntu",
    display_name: "Ubuntu",
    runtime_kind: "qemu",
    created_at: "2026-07-27T00:00:00Z",
    versions: [
      {
        id: "version-1",
        template_id: "template-1",
        version: "24.04",
        manifest_version: 1,
        defaults: {
          cpu_count: 1,
          memory_mib: 1024,
          interfaces: 1,
          interface_name_format: "eth%d",
        },
        capabilities: ["cloud-init"],
        supported_nic_drivers: ["virtio-net-pci"],
        console_modes: ["telnet"],
        runtime_options: {},
        enabled: true,
        created_at: "2026-07-27T00:00:00Z",
      },
    ],
  },
];

describe("TemplatePicker", () => {
  beforeEach(() => vi.restoreAllMocks());

  it("renders selectors from the generated API", async () => {
    vi.spyOn(api, "listTemplates").mockResolvedValue(catalog);
    const wrapper = mount(TemplatePicker);
    await flushPromises();
    expect(wrapper.findAll("select")).toHaveLength(2);
    expect(wrapper.text()).toContain("Ubuntu");
  });

  it("revalidates a selection before emitting", async () => {
    const list = vi
      .spyOn(api, "listTemplates")
      .mockResolvedValueOnce(catalog)
      .mockResolvedValueOnce(catalog);
    const wrapper = mount(TemplatePicker);
    await flushPromises();
    await wrapper.findAll("select")[0].setValue("template-1");
    await wrapper.findAll("select")[1].setValue("version-1");
    await wrapper.get("button:not([aria-label])").trigger("click");
    await flushPromises();
    expect(list).toHaveBeenCalledTimes(2);
    expect(wrapper.emitted("select")?.[0]).toEqual(["template-1", "version-1"]);
  });

  it("blocks a version removed during revalidation", async () => {
    vi.spyOn(api, "listTemplates")
      .mockResolvedValueOnce(catalog)
      .mockResolvedValueOnce([
        {
          ...catalog[0],
          versions: [{ ...catalog[0].versions[0], enabled: false }],
        },
      ]);
    const wrapper = mount(TemplatePicker);
    await flushPromises();
    await wrapper.findAll("select")[0].setValue("template-1");
    await wrapper.findAll("select")[1].setValue("version-1");
    await wrapper.get("button:not([aria-label])").trigger("click");
    await flushPromises();
    expect(wrapper.emitted("select")).toBeUndefined();
    expect(wrapper.text()).toContain("已不可用");
  });
});
