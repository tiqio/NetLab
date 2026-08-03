import { mount } from "@vue/test-utils";
import { createPinia } from "pinia";
import { afterEach, describe, expect, it, vi } from "vitest";
import { api } from "@/api";
import { nodeFactory, taskFactory } from "@/test/factories";
import NodeOperationsPanel from "./NodeOperationsPanel.vue";
describe("NodeOperationsPanel", () => {
  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it("submits lifecycle once and reports durable task identity", async () => {
    vi.useFakeTimers();
    vi.spyOn(api, "setNodeState").mockResolvedValue({ task: taskFactory() });
    vi.spyOn(api, "getTask").mockResolvedValue(
      taskFactory({
        state: "succeeded",
        progress_current: 2,
        progress_total: 2,
      }),
    );
    const wrapper = mount(NodeOperationsPanel, {
      props: { node: nodeFactory(), interfaces: [] },
      global: {
        plugins: [createPinia()],
        stubs: {
          InterfaceOperations: true,
          NodeResourcesEditor: true,
          NodeConfigurationPanel: true,
          GuestCommandPanel: true,
          PortMappingsPanel: true,
          NodeCapabilityPanel: true,
        },
      },
    });
    await wrapper.findAll("button")[0].trigger("click");
    await Promise.resolve();
    expect(api.setNodeState).toHaveBeenCalledTimes(1);
    expect(wrapper.text()).toContain("task-1");
    expect(wrapper.text()).toContain("正在启动");

    await vi.advanceTimersByTimeAsync(500);

    expect(api.getTask).toHaveBeenCalledWith("task-1");
    expect(wrapper.text()).toContain("运行中");
    expect(wrapper.emitted("changed")?.length).toBeGreaterThanOrEqual(2);
  });
});
