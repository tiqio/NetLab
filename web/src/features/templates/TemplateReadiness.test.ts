import { flushPromises, mount } from "@vue/test-utils";
import { describe, expect, it, vi } from "vitest";
import { api } from "@/api";
import TemplateCatalog from "./TemplateCatalog.vue";

describe("template readiness", () => {
  it("distinguishes genuine workloads from blocked operator assets", async () => {
    vi.spyOn(api, "listImages").mockResolvedValue([]);
    vi.spyOn(api, "listTemplates").mockResolvedValue([
      {
        id: "template",
        template_key: "vyos",
        display_name: "VyOS",
        runtime_kind: "qemu",
        created_at: "",
        versions: [
          {
            id: "version",
            template_id: "template",
            version: "rolling",
            manifest_version: 1,
            defaults: {
              cpu_count: 1,
              memory_mib: 1024,
              interfaces: 4,
              interface_name_format: "eth%d",
            },
            capabilities: ["cloud_init"],
            supported_nic_drivers: ["virtio-net-pci"],
            console_modes: ["telnet"],
            runtime_options: {},
            enabled: true,
            readiness: {
              status: "blocked",
              genuine_workload: false,
              checks: {},
            },
            created_at: "",
          },
        ],
      },
    ]);
    const wrapper = mount(TemplateCatalog);
    await flushPromises();
    expect(wrapper.text()).toContain("blocked");
    expect(wrapper.text()).toContain("Mechanics or operator asset only");
  });
});
