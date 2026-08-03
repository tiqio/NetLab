import { flushPromises, mount } from "@vue/test-utils";
import { describe, expect, it, vi } from "vitest";
import { api } from "@/api";
import AutomationView from "./AutomationView.vue";
describe("AutomationView", () => {
  it("explains API and MCP shared-state parity", async () => {
    vi.spyOn(api, "listTasks").mockResolvedValue([]);
    vi.spyOn(api, "listAuditEvents").mockResolvedValue([]);
    const wrapper = mount(AutomationView, {
      global: {
        stubs: {
          RouterLink: { template: "<a><slot /></a>" },
          TaskCenter: true,
        },
      },
    });
    await flushPromises();
    expect(wrapper.text()).toContain("REST and MCP parity");
    expect(wrapper.text()).toContain("No UI-only mutations");
  });
});
