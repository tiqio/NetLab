import { flushPromises, mount } from "@vue/test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { api, type OperationTask, type PortMapping } from "@/api";
import PortMappingsPanel from "./PortMappingsPanel.vue";

const mapping: PortMapping = {
  id: "mapping-1",
  node_id: "node-1",
  protocol: "tcp",
  host_address: "10.72.1.159",
  host_port: 18090,
  guest_address: "10.77.30.10",
  guest_port: 80,
  revision: 1,
  observed_state: "active",
};

function task(overrides: Partial<OperationTask> = {}): OperationTask {
  return {
    id: "task-1",
    kind: "port_mapping.create",
    resource_type: "port_mapping",
    resource_id: "mapping-1",
    state: "succeeded",
    progress_current: 1,
    progress_total: 1,
    created_at: "2026-07-30T00:00:00Z",
    ...overrides,
  };
}

describe("PortMappingsPanel", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    vi.spyOn(api, "listNodePortMappings").mockResolvedValue([mapping]);
  });

  it("loads durable mappings and copies a usable access URL", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    vi.stubGlobal("navigator", { ...navigator, clipboard: { writeText } });
    const wrapper = mount(PortMappingsPanel, { props: { nodeId: "node-1" } });
    await flushPromises();

    expect(api.listNodePortMappings).toHaveBeenCalledWith("node-1");
    expect(wrapper.text()).toContain("10.72.1.159:18090");
    expect(wrapper.text()).toContain("10.77.30.10:80");
    expect(wrapper.text()).toContain("http://10.72.1.159:18090");

    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("复制"))!
      .trigger("click");
    await flushPromises();
    expect(writeText).toHaveBeenCalledWith("http://10.72.1.159:18090");
    expect(wrapper.text()).toContain("访问地址已复制");
  });

  it("detects the guest IPv4 through QGA and publishes a preset mapping", async () => {
    vi.spyOn(api, "executeGuestCommand").mockResolvedValue(
      task({
        id: "task-detect",
        kind: "node.guest_exec",
        resource_type: "node",
        resource_id: "node-1",
        result: { stdout_base64: btoa("10.77.30.10\n") },
      }),
    );
    const create = vi.spyOn(api, "createPortMapping").mockResolvedValue({
      port_mapping: mapping,
      task: task(),
    });
    const wrapper = mount(PortMappingsPanel, { props: { nodeId: "node-1" } });
    await flushPromises();

    await wrapper.find("select").setValue("http");
    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("高级设置"))!
      .trigger("click");
    await wrapper
      .find('button[title="通过 QEMU Guest Agent 自动探测 IPv4"]')
      .trigger("click");
    await flushPromises();

    expect(
      wrapper.find<HTMLInputElement>('input[placeholder="自动识别"]').element
        .value,
    ).toBe("10.77.30.10");
    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("保存并生效"))!
      .trigger("click");
    await flushPromises();

    expect(create).toHaveBeenCalledWith(
      "node-1",
      expect.objectContaining({
        protocol: "tcp",
        host_port: 0,
        guest_address: "10.77.30.10",
        guest_port: 80,
      }),
    );
    expect(wrapper.text()).toContain("端口映射已生效");
  });

  it("shows immediate feedback while the create request is pending", async () => {
    let resolveCreate!: (value: {
      port_mapping: PortMapping;
      task: OperationTask;
    }) => void;
    vi.spyOn(api, "createPortMapping").mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveCreate = resolve;
        }),
    );
    const wrapper = mount(PortMappingsPanel, { props: { nodeId: "node-1" } });
    await flushPromises();

    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("保存并生效"))!
      .trigger("click");

    expect(wrapper.text()).toContain("正在创建端口映射");
    resolveCreate({ port_mapping: mapping, task: task() });
    await flushPromises();
  });
});
