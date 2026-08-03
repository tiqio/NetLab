import { flushPromises, mount } from "@vue/test-utils";
import { describe, expect, it, vi } from "vitest";
import { api, type NetworkObject, type Node } from "@/api";
import TopologyInspector from "./TopologyInspector.vue";
import LightweightSwitchConfigEditor from "@/features/nodes/LightweightSwitchConfigEditor.vue";

const natObject: NetworkObject = {
  id: "nat-1",
  laboratory_id: "lab-1",
  name: "Internet NAT",
  kind: "nat_bridge",
  revision: 1,
  desired_state: "active",
  observed_state: "active",
  config: {},
};

describe("TopologyInspector", () => {
  it("shows authoritative object-link lifecycle, capture metadata, and failure", async () => {
    vi.spyOn(api, "listCaptures").mockResolvedValue([
      {
        id: "capture-object-link",
        laboratory_id: "lab-1",
        source_type: "network_object_link",
        source_id: "object-link-1",
        format: "pcap",
        state: "running",
        retain: true,
        max_bytes: 1024,
        bytes_written: 256,
        packets: 4,
        truncated: false,
        artifact_url: "/api/v1/artifacts/capture-object-link",
        created_at: "2026-08-03T00:00:00Z",
      },
    ]);
    const wrapper = mount(TopologyInspector, {
      props: {
        laboratoryId: "lab-1",
        interfaces: [],
        networkObjects: [
          {
            id: "a",
            laboratory_id: "lab-1",
            name: "Switch A",
            kind: "switch_l2",
            revision: 1,
            desired_state: "active",
            observed_state: "active",
            config: {},
          },
          {
            id: "b",
            laboratory_id: "lab-1",
            name: "Switch B",
            kind: "switch_l2",
            revision: 1,
            desired_state: "active",
            observed_state: "active",
            config: {},
          },
        ],
        networkObjectLink: {
          id: "object-link-1",
          laboratory_id: "lab-1",
          object_a_id: "a",
          port_a_name: "swp1",
          object_b_id: "b",
          port_b_name: "swp2",
          revision: 4,
          desired_state: "connected",
          observed_state: "failed",
          last_error: {
            code: "runtime_failed",
            message: "veth endpoint missing",
          },
        },
        tasks: [
          {
            id: "task-object-link-1",
            kind: "network_object_link.create",
            resource_type: "network_object_link",
            resource_id: "object-link-1",
            state: "failed",
            progress_current: 1,
            progress_total: 2,
            created_at: "2026-08-03T06:00:00Z",
            error: {
              code: "runtime_failed",
              message: "task could not restore the veth pair",
            },
          },
        ],
      },
    });
    expect(wrapper.text()).toContain("Switch A:swp1");
    expect(wrapper.text()).toContain("Switch B:swp2");
    expect(wrapper.text()).toContain("期望状态");
    expect(wrapper.text()).toContain("实际状态");
    expect(wrapper.text()).toContain("veth endpoint missing");
    expect(wrapper.text()).toContain("task-object-link-1 · failed");
    expect(wrapper.text()).toContain("task could not restore the veth pair");
    expect(wrapper.text()).toContain("直接 veth pair");
    expect(wrapper.text()).not.toContain("独立宿主桥");
    await flushPromises();
    expect(wrapper.text()).toContain("capture-object-link · running");
    expect(wrapper.text()).toContain("4 / 256");
    expect(wrapper.text()).toContain("Live stream");
    expect(wrapper.text()).toContain("Retained artifact");
  });

  it("summarizes Docker route readiness and recovery guidance", () => {
    const node: Node = {
      id: "docker-1",
      laboratory_id: "lab-1",
      name: "Router",
      kind: "docker",
      revision: 1,
      desired_state: "running",
      observed_state: "failed",
      cpu_count: 1,
      cpu_quota_micros: 100000,
      memory_mib: 128,
      storage_gib: 0,
      interface_limit: 8,
      process_limit: 128,
      config: {
        network_interfaces: [
          {
            id: "if-1",
            name: "eth0",
            driver: "veth",
            modes: ["static"],
            addresses: ["192.0.2.2/24"],
            routes: [{ destination: "198.51.100.0/24", gateway: "192.0.2.1" }],
          },
        ],
      },
      last_error: {
        code: "runtime_configuration_timeout",
        message: "route reconciliation timed out",
      },
    };
    const wrapper = mount(TopologyInspector, {
      props: { laboratoryId: "lab-1", node, interfaces: [] },
      global: {
        stubs: {
          ResourceCharts: true,
          NodeOperationsPanel: true,
        },
      },
    });

    const readiness = wrapper.get(
      '[data-testid="inspector-docker-route-readiness"]',
    );
    expect(readiness.text()).toContain("路由应用失败");
    expect(readiness.text()).toContain("eth0:198.51.100.0/24");
    expect(readiness.text()).toContain("请检查接口地址、网关可达性");
  });

  it("updates Lightweight L2 configuration from the Inspector", async () => {
    const networkObject: NetworkObject = {
      id: "l2-1",
      laboratory_id: "lab-1",
      name: "Lightweight L2",
      kind: "switch_l2",
      revision: 3,
      desired_state: "active",
      observed_state: "active",
      config: {
        vlan_filtering: true,
        ports: [{ name: "eth0", pvid: 1, tagged: [] }],
      },
    };
    const update = vi
      .spyOn(api, "updateNetworkObject")
      .mockResolvedValue({ task: { id: "task-1" } } as never);
    const wrapper = mount(TopologyInspector, {
      props: {
        laboratoryId: "lab-1",
        networkObject,
        interfaces: [],
      },
    });
    const editor = wrapper.findComponent(LightweightSwitchConfigEditor);
    const inputs = editor.findAll("input");
    await inputs[1].setValue("lan0");
    await inputs[2].setValue("10");
    await inputs[3].setValue("20");
    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("应用配置"))!
      .trigger("click");
    await flushPromises();

    expect(update).toHaveBeenCalledWith(networkObject, {
      name: "Lightweight L2",
      config: {
        vlan_filtering: true,
        ports: [{ name: "lan0", pvid: 10, tagged: [20] }],
      },
    });
    expect(wrapper.text()).toContain("配置更新任务已提交");
    update.mockRestore();
  });

  it("shows Ruijie interface configuration and forwards terminal requests", async () => {
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
    const wrapper = mount(TopologyInspector, {
      props: {
        laboratoryId: "lab-1",
        node,
        interfaces: [
          {
            id: "if-1",
            node_id: node.id,
            slot: 0,
            name: "G0/0",
            driver: "e1000",
            mac_address: "02:00:00:00:00:01",
            operational_state: "up",
            revision: 1,
          },
        ],
      },
      global: {
        stubs: {
          ResourceCharts: true,
          NodeOperationsPanel: true,
        },
      },
    });

    expect(wrapper.find('[data-testid="ruijie-config-panel"]').exists()).toBe(
      true,
    );
    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("打开终端"))!
      .trigger("click");
    expect(wrapper.emitted("terminal")).toEqual([[node]]);
  });

  it("shows friendly node and interface names for link endpoints", () => {
    const wrapper = mount(TopologyInspector, {
      props: {
        laboratoryId: "lab-1",
        link: {
          id: "link-1",
          laboratory_id: "lab-1",
          endpoint_a_id: "if-a",
          endpoint_b_id: "if-b",
          revision: 1,
          desired_state: "connected",
          observed_state: "connected",
        },
        nodes: [
          {
            id: "node-a",
            laboratory_id: "lab-1",
            name: "BusyBox1",
            kind: "docker",
            revision: 1,
            desired_state: "running",
            observed_state: "running",
            cpu_count: 1,
            cpu_quota_micros: 0,
            memory_mib: 128,
            storage_gib: 0,
            interface_limit: 64,
            process_limit: 4096,
            config: {},
          },
          {
            id: "node-b",
            laboratory_id: "lab-1",
            name: "BusyBox2",
            kind: "docker",
            revision: 1,
            desired_state: "running",
            observed_state: "running",
            cpu_count: 1,
            cpu_quota_micros: 0,
            memory_mib: 128,
            storage_gib: 0,
            interface_limit: 64,
            process_limit: 4096,
            config: {},
          },
        ],
        interfaces: [
          {
            id: "if-a",
            node_id: "node-a",
            slot: 0,
            name: "eth0",
            driver: "veth",
            mac_address: "02:00:00:00:00:01",
            operational_state: "up",
            revision: 1,
          },
          {
            id: "if-b",
            node_id: "node-b",
            slot: 0,
            name: "eth1",
            driver: "veth",
            mac_address: "02:00:00:00:00:02",
            operational_state: "up",
            revision: 1,
          },
        ],
      },
    });

    expect(wrapper.text()).toContain("BusyBox1:eth0");
    expect(wrapper.text()).toContain("BusyBox2:eth1");
    expect(wrapper.text()).not.toContain("if-a");
    expect(wrapper.text()).not.toContain("if-b");
  });

  it("shows NAT runtime configuration and attached node status", () => {
    const wrapper = mount(TopologyInspector, {
      props: {
        laboratoryId: "lab-1",
        networkObject: {
          ...natObject,
          config: {
            ipv4_prefix: "10.250.30.0/24",
            uplink: "auto",
            dhcpv4: {
              start: "10.250.30.100",
              end: "10.250.30.200",
              lease_time: "1h",
            },
            dns_servers: ["1.1.1.1", "8.8.8.8"],
          },
        },
        nodes: [
          {
            id: "node-1",
            laboratory_id: "lab-1",
            name: "NAT-BusyBox",
            kind: "docker",
            revision: 1,
            desired_state: "running",
            observed_state: "running",
            cpu_count: 1,
            cpu_quota_micros: 0,
            memory_mib: 128,
            storage_gib: 0,
            interface_limit: 64,
            process_limit: 4096,
            config: {},
          },
        ],
        interfaces: [
          {
            id: "if-1",
            node_id: "node-1",
            slot: 0,
            name: "eth0",
            driver: "veth",
            mac_address: "02:00:00:00:00:01",
            operational_state: "up",
            revision: 1,
          },
        ],
        attachments: [
          {
            id: "attachment-1",
            network_object_id: "nat-1",
            interface_id: "if-1",
            port_name: "lan0",
            observed_state: "active",
          },
        ],
      },
    });

    expect(wrapper.text()).toContain("10.250.30.0/24");
    expect(wrapper.text()).toContain("10.250.30.1");
    expect(wrapper.text()).toContain("10.250.30.100 – 10.250.30.200 · 1h");
    expect(wrapper.text()).toContain("NAT-BusyBox");
    expect(wrapper.text()).toContain("eth0 · veth");
    expect(wrapper.text()).toContain("active");
  });

  it("loads network diagnostics when requested by the context menu", async () => {
    const diagnostics = vi
      .spyOn(api, "getNetworkObjectDiagnostics")
      .mockResolvedValue({ forwarding_status: { outbound_rule: true } });
    const wrapper = mount(TopologyInspector, {
      props: {
        laboratoryId: "lab-1",
        networkObject: natObject,
        interfaces: [],
        diagnosticsRequestKey: 0,
      },
    });

    await wrapper.setProps({ diagnosticsRequestKey: 1 });
    await flushPromises();

    expect(diagnostics).toHaveBeenCalledWith("nat-1");
    expect(wrapper.text()).toContain("outbound_rule");
    expect(wrapper.emitted("diagnosticsLoaded")).toEqual([["Internet NAT"]]);
    diagnostics.mockRestore();
  });
});
