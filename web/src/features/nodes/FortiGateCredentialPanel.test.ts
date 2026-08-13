import { flushPromises, mount } from "@vue/test-utils";
import { describe, expect, it, vi } from "vitest";
import { api, type Node } from "@/api";
import FortiGateCredentialPanel from "./FortiGateCredentialPanel.vue";

const node = {
  id: "fortigate-node",
  laboratory_id: "lab-a",
  name: "FortiGate",
  kind: "qemu",
  revision: 1,
  desired_state: "running",
  observed_state: "running",
  cpu_count: 1,
  cpu_quota_micros: 0,
  memory_mib: 2048,
  storage_gib: 10,
  interface_limit: 64,
  process_limit: 4096,
  config: { template_key: "fortigate" },
  created_at: "2026-08-13T00:00:00Z",
  updated_at: "2026-08-13T00:00:00Z",
} as Node;

describe("FortiGateCredentialPanel", () => {
  it("clears password inputs after encrypted storage", async () => {
    vi.spyOn(api, "getNodeCredentials").mockResolvedValue({
      node_id: node.id,
      laboratory_id: node.laboratory_id,
      kind: "console_admin",
      configured: false,
      staged: false,
      state: "credential_missing",
      revision: 0,
    });
    const save = vi.spyOn(api, "setFortiGateCredential").mockResolvedValue({
      node_id: node.id,
      laboratory_id: node.laboratory_id,
      kind: "console_admin",
      configured: true,
      staged: true,
      state: "pending_verification",
      revision: 1,
    });
    const wrapper = mount(FortiGateCredentialPanel, { props: { node } });
    await flushPromises();
    await wrapper
      .get('input[aria-label="FortiGate 当前密码"]')
      .setValue("old-secret");
    await wrapper
      .get('input[aria-label="FortiGate 新密码"]')
      .setValue("new-secret");
    await wrapper.get("button").trigger("click");
    await flushPromises();

    expect(save).toHaveBeenCalledWith(node.id, {
      username: "admin",
      current_password: "old-secret",
      new_password: "new-secret",
    });
    expect(
      wrapper.get<HTMLInputElement>('input[aria-label="FortiGate 当前密码"]')
        .element.value,
    ).toBe("");
    expect(
      wrapper.get<HTMLInputElement>('input[aria-label="FortiGate 新密码"]')
        .element.value,
    ).toBe("");
    expect(wrapper.text()).not.toContain("old-secret");
    expect(wrapper.text()).not.toContain("new-secret");
  });
});
