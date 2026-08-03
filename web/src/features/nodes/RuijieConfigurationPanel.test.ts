import { flushPromises, mount } from "@vue/test-utils";
import { describe, expect, it, vi } from "vitest";
import { api, type Node, type NodeInterface } from "@/api";
import RuijieConfigurationPanel from "./RuijieConfigurationPanel.vue";

const node: Node = {
  id: "switch-1",
  laboratory_id: "lab-1",
  name: "Layer-2 switch",
  kind: "qemu",
  revision: 1,
  desired_state: "running",
  observed_state: "running",
  cpu_count: 1,
  cpu_quota_micros: 100000,
  memory_mib: 1024,
  storage_gib: 8,
  interface_limit: 64,
  process_limit: 4096,
  config: { template_key: "ruijie-switch" },
};

const interfaces: NodeInterface[] = [
  {
    id: "if-1",
    node_id: node.id,
    slot: 0,
    name: "G0/0",
    driver: "e1000",
    mac_address: "02:00:00:00:00:01",
    desired_link_id: "link-1",
    operational_state: "up",
    revision: 1,
  },
];

describe("RuijieConfigurationPanel", () => {
  it("shows interfaces, opens terminal, and applies access VLAN configuration", async () => {
    const configure = vi.spyOn(api, "configureRuijie").mockResolvedValue({
      commands: ["enable", "configure terminal", "write"],
      console_mode: "telnet",
      verified: true,
    });
    const wrapper = mount(RuijieConfigurationPanel, {
      props: { node, interfaces },
    });

    expect(wrapper.text()).toContain("G0/0");
    expect(wrapper.text()).toContain("已接线");
    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("打开终端"))!
      .trigger("click");
    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("应用配置"))!
      .trigger("click");
    await flushPromises();

    expect(wrapper.emitted("terminal")).toHaveLength(1);
    expect(configure).toHaveBeenCalledWith(node.id, {
      operation: "l2_access",
      interface: "G0/0",
      vlan_id: 10,
      vlan_name: "",
      allowed_vlans: "10,20",
      address_cidr: "192.0.2.1/24",
      admin_up: true,
      save: true,
    });
    expect(wrapper.text()).toContain("配置已执行，并通过交换机 CLI 提示符确认");
    configure.mockRestore();
  });
});
