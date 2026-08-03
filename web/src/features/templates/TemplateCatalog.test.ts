import { flushPromises, mount } from "@vue/test-utils";
import { describe, expect, it, vi } from "vitest";
import { api } from "@/api";
import TemplateCatalog from "./TemplateCatalog.vue";
describe("TemplateCatalog", () => {
  it("shows capabilities and image provenance metadata", async () => {
    vi.spyOn(api, "listTemplates").mockResolvedValue([
      {
        id: "t",
        template_key: "fancywan",
        display_name: "FancyWAN",
        runtime_kind: "qemu",
        created_at: "",
        versions: [
          {
            id: "v",
            template_id: "t",
            version: "1",
            manifest_version: 1,
            image_version_id: "img",
            defaults: {
              cpu_count: 2,
              memory_mib: 1024,
              interfaces: 4,
              interface_name_format: "eth%d",
            },
            capabilities: ["cloud-init", "qga"],
            supported_nic_drivers: ["virtio-net-pci"],
            console_modes: ["vnc"],
            runtime_options: {},
            enabled: true,
            created_at: "",
          },
        ],
      },
    ]);
    vi.spyOn(api, "listImages").mockResolvedValue([
      {
        id: "img",
        name: "fancy",
        version: "1",
        runtime_kind: "qemu",
        digest: "sha256:abc",
        source_type: "local",
        source_reference: "redacted",
        format: "qcow2",
        size_bytes: 1,
        availability: "available",
        license_status: "operator_supplied",
        license_notes: "test",
        validation_result: {},
        created_at: "",
      },
    ]);
    const wrapper = mount(TemplateCatalog);
    await flushPromises();
    expect(wrapper.text()).toContain("cloud-init");
    expect(wrapper.text()).toContain("sha256:abc");
    expect(wrapper.text()).toContain("No image bytes");
  });
});
