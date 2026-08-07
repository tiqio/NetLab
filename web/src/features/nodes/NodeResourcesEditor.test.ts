import { flushPromises, mount } from "@vue/test-utils";
import { createPinia } from "pinia";
import { afterEach, describe, expect, it, vi } from "vitest";
import { api } from "@/api";
import { nodeFactory } from "@/test/factories";
import NodeResourcesEditor from "./NodeResourcesEditor.vue";

describe("NodeResourcesEditor", () => {
  afterEach(() => vi.restoreAllMocks());

  it("shows quota in cores and converts it back to microseconds", async () => {
    const update = vi
      .spyOn(api, "updateNodeResources")
      .mockResolvedValue(
        nodeFactory({ cpu_count: 2, cpu_quota_micros: 75_000 }),
      );
    const node = nodeFactory({
      cpu_count: 2,
      cpu_quota_micros: 50_000,
      memory_mib: 2048,
    });
    const wrapper = mount(NodeResourcesEditor, {
      props: { node },
      global: { plugins: [createPinia()] },
    });
    const inputs = wrapper.findAll("input");

    expect((inputs[1].element as HTMLInputElement).value).toBe("0.5");
    await inputs[1].setValue("0.75");
    await wrapper.find("form").trigger("submit");
    await flushPromises();

    expect(update).toHaveBeenCalledWith(node, {
      cpu_count: 2,
      cpu_quota_micros: 75_000,
      memory_mib: 2048,
    });
    expect(wrapper.text()).toContain("CPU 配额为 0.75 核");
  });
});
