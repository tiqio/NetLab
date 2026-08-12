import { mount } from "@vue/test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";
import DeviceReadinessPanel from "./DeviceReadinessPanel.vue";

const { getDeviceReadiness } = vi.hoisted(() => ({ getDeviceReadiness: vi.fn() }));
vi.mock("@/api", () => ({ api: { getDeviceReadiness } }));

describe("DeviceReadinessPanel", () => {
  beforeEach(() => getDeviceReadiness.mockReset());

  it("separates cable, guest, management and data path states", async () => {
    getDeviceReadiness.mockResolvedValue({ node_id: "node-1", roles: [], cable: { state: "ready" }, guest: { state: "ready" }, management: { state: "prerequisite" }, data_path: { state: "unverified" } });
    const wrapper = mount(DeviceReadinessPanel, { props: { node: { id: "node-1", name: "vendor", kind: "qemu", revision: 1, desired_state: "stopped", observed_state: "stopped", cpu_count: 1, cpu_quota_micros: 0, memory_mib: 512, storage_gib: 1, interface_limit: 4, process_limit: 32, laboratory_id: "lab" }, interfaces: [] } });
    await vi.waitFor(() => expect(wrapper.text()).toContain("prerequisite"));
    expect(wrapper.text()).toContain("线缆");
    expect(wrapper.text()).toContain("设备系统");
    expect(wrapper.text()).toContain("管理可达");
    expect(wrapper.text()).toContain("数据路径");
  });
});
