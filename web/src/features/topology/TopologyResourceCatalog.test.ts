import { flushPromises, mount } from "@vue/test-utils";
import { afterEach, describe, expect, it, vi } from "vitest";
import { api, type DeviceTemplate } from "@/api";
import TopologyResourceCatalog from "./TopologyResourceCatalog.vue";

const template = (
  id: string,
  name: string,
  runtime: "qemu" | "docker",
  enabled = true,
): DeviceTemplate => ({
  id,
  template_key: id,
  display_name: name,
  runtime_kind: runtime,
  created_at: "2026-08-05T00:00:00Z",
  versions: [
    {
      id: `${id}-version`,
      template_id: id,
      version: "latest",
      manifest_version: 1,
      defaults: {
        cpu_count: 1,
        memory_mib: 128,
        interfaces: 1,
        interface_name_format: "eth%d",
      },
      capabilities: [],
      supported_nic_drivers: [],
      console_modes: [],
      runtime_options: {},
      enabled,
      created_at: "2026-08-05T00:00:00Z",
    },
  ],
});

afterEach(() => {
  vi.restoreAllMocks();
  document.body.innerHTML = "";
});

describe("TopologyResourceCatalog", () => {
  it("searches categorized templates and emits the enabled version", async () => {
    vi.spyOn(api, "listTemplates").mockResolvedValue([
      template("ubuntu-qemu", "Ubuntu QEMU", "qemu"),
      template("busybox", "BusyBox", "docker"),
    ]);
    const wrapper = mount(TopologyResourceCatalog);
    await flushPromises();
    await wrapper
      .get('[aria-label="Search device templates"]')
      .setValue("ubuntu");
    expect(wrapper.text()).toContain("Ubuntu QEMU");
    expect(wrapper.text()).not.toContain("BusyBox");
    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("Ubuntu QEMU"))!
      .trigger("click");
    expect(wrapper.emitted("choose")?.[0]?.[0]).toMatchObject({
      kind: "qemu",
      template: { id: "ubuntu-qemu" },
      version: { id: "ubuntu-qemu-version" },
    });
  });

  it("explains and blocks templates without an enabled version", async () => {
    vi.spyOn(api, "listTemplates").mockResolvedValue([
      template("disabled", "Unavailable appliance", "qemu", false),
    ]);
    const wrapper = mount(TopologyResourceCatalog);
    await flushPromises();
    const button = wrapper
      .findAll("button")
      .find((item) => item.text().includes("Unavailable appliance"))!;
    expect(button.attributes("disabled")).toBeDefined();
    expect(button.text()).toContain("没有已启用版本");
  });

  it("emits lightweight network objects from the same catalog", async () => {
    vi.spyOn(api, "listTemplates").mockResolvedValue([]);
    const wrapper = mount(TopologyResourceCatalog);
    await flushPromises();
    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("NAT bridge"))!
      .trigger("click");
    expect(wrapper.emitted("choose")?.[0]?.[0]).toMatchObject({
      networkObjectKind: "nat_bridge",
    });
  });
});
