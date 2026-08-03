import { flushPromises, mount } from "@vue/test-utils";
import { createPinia } from "pinia";
import { beforeEach, describe, expect, it, vi } from "vitest";
import NodeCapabilityPanel from "./NodeCapabilityPanel.vue";

const { getNodeCapabilities } = vi.hoisted(() => ({
  getNodeCapabilities: vi.fn(),
}));
vi.mock("@/api", async () => {
  const actual = await vi.importActual<typeof import("@/api")>("@/api");
  return { ...actual, api: { ...actual.api, getNodeCapabilities } };
});

describe("NodeCapabilityPanel", () => {
  beforeEach(() => getNodeCapabilities.mockReset());

  it("shows unavailable guest-agent readiness without claiming guest exec works", async () => {
    getNodeCapabilities.mockResolvedValue({
      node_id: "node-1",
      observations: [
        {
          node_id: "node-1",
          capability: "qga",
          revision: 1,
          state: "unavailable",
          required: true,
          observed_at: new Date().toISOString(),
          problem: {
            code: "qga_unavailable",
            message: "guest agent is not ready",
            retryable: true,
            operator_hint: "install and enable qemu-guest-agent",
          },
        },
      ],
    });
    const wrapper = mount(NodeCapabilityPanel, {
      props: { nodeId: "node-1" },
      global: { plugins: [createPinia()] },
    });
    await flushPromises();
    expect(wrapper.text()).toContain("qga");
    expect(wrapper.text()).toContain("unavailable");
    expect(wrapper.text()).toContain("install and enable qemu-guest-agent");
  });
});
