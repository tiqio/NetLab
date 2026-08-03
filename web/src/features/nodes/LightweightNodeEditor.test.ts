import { mount } from "@vue/test-utils";
import { describe, expect, it, vi } from "vitest";
import { api } from "../../api";
import LightweightNodeEditor from "./LightweightNodeEditor.vue";

describe("LightweightNodeEditor", () => {
  it("offers every lightweight kind and dual-stack mode", () => {
    const wrapper = mount(LightweightNodeEditor, {
      props: { laboratoryId: "lab-1" },
    });
    expect(wrapper.text()).toContain("Layer-2 switch");
    expect(wrapper.text()).toContain("Layer-3 switch");
    expect(wrapper.text()).toContain("NAT bridge");
    expect(wrapper.text()).toContain("DHCPv4");
    expect(wrapper.text()).toContain("DHCPv6");
    expect(wrapper.text()).toContain("IPv6 SLAAC");
  });

  it("submits usable PC and switch configurations", async () => {
    const create = vi.spyOn(api, "createNetworkObject").mockResolvedValue({
      task: {
        id: "task-1",
        kind: "network_object.create",
        resource_type: "network_object",
        resource_id: "object-1",
        state: "queued",
        progress_current: 0,
        progress_total: 2,
        created_at: new Date().toISOString(),
      },
      network_object: {
        id: "object-1",
        laboratory_id: "lab-1",
        name: "pc1",
        kind: "pc",
        revision: 1,
        desired_state: "active",
        observed_state: "provisioning",
        config: {},
      },
    });
    const wrapper = mount(LightweightNodeEditor, {
      props: { laboratoryId: "lab-1" },
    });
    await wrapper.get("button").trigger("click");
    expect(create).toHaveBeenCalledWith(
      "lab-1",
      expect.objectContaining({
        kind: "pc",
        config: expect.objectContaining({
          interfaces: [
            expect.objectContaining({ dns: ["192.0.2.53", "2001:db8::53"] }),
          ],
        }),
      }),
    );
    create.mockRestore();
  });
});
