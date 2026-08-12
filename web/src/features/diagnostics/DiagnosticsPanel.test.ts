import { flushPromises, mount } from "@vue/test-utils";
import { afterEach, describe, expect, it, vi } from "vitest";
import { api, type Node } from "@/api";
import DiagnosticsPanel from "./DiagnosticsPanel.vue";

describe("DiagnosticsPanel", () => {
  afterEach(() => vi.restoreAllMocks());

  it("prompts for a node terminal before a console is requested", () => {
    const wrapper = mount(DiagnosticsPanel, { props: { nodeId: "node-1" } });
    expect(wrapper.text()).toContain("请右键节点并选择“终端”");
  });

  it("guides capture source selection when no interface is selected", () => {
    const wrapper = mount(DiagnosticsPanel, {
      props: { initialSection: "captures" },
    });
    expect(wrapper.text()).toContain(
      "请在上方选择节点和接口，或右键单击拓扑节点并选择抓包。",
    );
  });

  it("keeps the global console mounted while capture is selected", async () => {
    const wrapper = mount(DiagnosticsPanel, {
      props: { initialSection: "console" },
    });
    const consoleWorkspace = wrapper.get("[data-global-console-workspace]");

    await wrapper.setProps({ initialSection: "captures" });

    expect(wrapper.find("[data-global-console-workspace]").exists()).toBe(true);
    expect(consoleWorkspace.attributes("style")).toContain("display: none");

    await wrapper.setProps({ initialSection: "console" });
    expect(
      wrapper.get("[data-global-console-workspace]").attributes("style") || "",
    ).not.toContain("display: none");
  });

  it("passes Lightweight attachments into Traffic Filter scope", () => {
    const wrapper = mount(DiagnosticsPanel, {
      props: {
        initialSection: "traffic-filter",
        attachments: [
          {
            id: "attachment-1",
            network_object_id: "switch-1",
            interface_id: "interface-1",
            port_name: "eth0",
            revision: 1,
            observed_state: "active",
          },
        ],
        networkObjects: [
          {
            id: "switch-1",
            laboratory_id: "lab-1",
            name: "Lightweight L2 Switch",
            kind: "switch_l2",
            revision: 1,
            desired_state: "active",
            observed_state: "active",
            config: {},
          },
        ],
      },
      global: {
        stubs: {
          TrafficFilterPanel: {
            props: ["attachments", "networkObjects"],
            template:
              '<div data-testid="traffic-filter-props">{{ attachments.length }} / {{ networkObjects[0].name }}</div>',
          },
        },
      },
    });

    expect(wrapper.get('[data-testid="traffic-filter-props"]').text()).toBe(
      "1 / Lightweight L2 Switch",
    );
  });

  it("shows honest backing failure and emits retry for a network object", async () => {
    vi.spyOn(api, "getNetworkObjectDiagnostics").mockResolvedValue({
      object_id: "switch-1",
      desired_state: "active",
      observed_state: "failed",
      backing: {
        backing_kind: "namespace",
        runtime_name: "n2s-deadbeef",
        owned: true,
        usable: false,
        adoptable: true,
        recreatable: true,
        observed_at: "2026-08-11T08:00:00Z",
        problem: {
          code: "runtime_backing_unusable",
          message: "network namespace backing is not usable",
          retryable: true,
          phase: "runtime_inspection",
          cleanup: "owned namespace will be recreated",
          operator_hint: "retry reconciliation",
        },
      },
      runtime: {},
    });
    const object = {
      id: "switch-1",
      laboratory_id: "lab-1",
      name: "业务三层交换机",
      kind: "switch_l3" as const,
      revision: 3,
      desired_state: "active",
      observed_state: "failed",
      config: {},
    };
    const wrapper = mount(DiagnosticsPanel, {
      props: { networkObjectId: object.id, networkObjects: [object] },
    });
    await flushPromises();
    expect(
      wrapper.get('[data-testid="recovery-diagnostics"]').text(),
    ).toContain("实际：failed");
    expect(wrapper.text()).toContain("运行承载：namespace");
    expect(wrapper.text()).toContain("阶段：runtime_inspection");
    expect(wrapper.text()).toContain("建议：retry reconciliation");
    await wrapper.get("button").trigger("click");
    expect(wrapper.emitted("reconcileNetworkObject")).toEqual([[object]]);
  });

  it("shows desired and observed VLAN membership with mismatch guidance", async () => {
    vi.spyOn(api, "getNetworkObjectDiagnostics").mockResolvedValue({
      object_id: "switch-1",
      desired_state: "active",
      observed_state: "pending",
      runtime: {
        desired: {
          vlan_filtering: true,
          ports: [{ name: "trunk0", pvid: 10, tagged: [20] }],
        },
        observed: {
          ports: [{ name: "trunk0", pvid: 1, tagged: [] }],
        },
        mismatches: ["trunk0: VLAN membership differs"],
      },
    });
    const object = {
      id: "switch-1",
      laboratory_id: "lab-1",
      name: "核心二层交换机",
      kind: "switch_l2" as const,
      revision: 4,
      desired_state: "active",
      observed_state: "pending",
      config: {},
    };
    const wrapper = mount(DiagnosticsPanel, {
      props: { networkObjectId: object.id, networkObjects: [object] },
    });
    await flushPromises();
    expect(wrapper.get('[data-testid="vlan-diagnostics"]').text()).toContain(
      "trunk0",
    );
    expect(wrapper.get('[data-testid="vlan-diagnostics"]').text()).toContain(
      "10 / 20",
    );
    expect(wrapper.text()).toContain("trunk0: VLAN membership differs");
  });

  it("shows IPv6 neighbor discovery as the failed path stop", async () => {
    vi.spyOn(api, "getNetworkObjectDiagnostics").mockResolvedValue({
      object_id: "router-1",
      desired_state: "active",
      observed_state: "active",
      runtime: {
        observed: {
          ipv6_neighbor_status: "available",
          ipv6_neighbors: [
            { address: "fd10::fe", device: "eth0", state: ["FAILED"] },
          ],
        },
        path_checks: [
          {
            destination: "fd20::/64",
            gateway: "fd10::fe",
            status: "unverified",
            stop_at: "neighbor_discovery",
          },
        ],
        mismatches: [],
      },
    });
    const wrapper = mount(DiagnosticsPanel, {
      props: {
        networkObjectId: "router-1",
        networkObjects: [
          {
            id: "router-1",
            laboratory_id: "lab-1",
            name: "三层交换机",
            kind: "switch_l3" as const,
            revision: 1,
            desired_state: "active",
            observed_state: "active",
            config: {},
          },
        ],
      },
    });
    await flushPromises();
    const diagnostics = wrapper.get('[data-testid="ipv6-path-diagnostics"]');
    expect(diagnostics.text()).toContain("fd20::/64");
    expect(diagnostics.text()).toContain("fd10::fe");
    expect(diagnostics.text()).toContain("neighbor_discovery");
  });

  it("offers retry and delete for a failed object link", async () => {
    const link = {
      id: "link-1",
      laboratory_id: "lab-1",
      object_a_id: "a",
      port_a_name: "eth0",
      object_b_id: "b",
      port_b_name: "eth0",
      revision: 2,
      desired_state: "connected",
      observed_state: "failed",
      last_error: {
        code: "connection_recovery_exhausted",
        message: "interrupted topology connection has no active recovery task",
        retryable: true,
        cleanup: "endpoint reservations are retained for retry or delete",
      },
    };
    const wrapper = mount(DiagnosticsPanel, {
      props: { objectLinkId: link.id, networkObjectLinks: [link] },
    });
    const buttons = wrapper.findAll("button");
    expect(wrapper.text()).toContain("实际：failed");
    expect(wrapper.text()).toContain(
      "清理：endpoint reservations are retained",
    );
    await buttons[0].trigger("click");
    await buttons[1].trigger("click");
    expect(wrapper.emitted("reconcileNetworkObjectLink")).toEqual([[link]]);
    expect(wrapper.emitted("deleteNetworkObjectLink")).toEqual([[link]]);
  });

  it("shows Docker forwarding mismatch and emits a revisioned control intent", async () => {
    vi.spyOn(api, "getNodeNetworkDiagnostics").mockResolvedValue({
      desired: { forward_ipv4: true, forward_ipv6: true },
      observed: {
        available: true,
        forward_ipv4: false,
        forward_ipv6: true,
        pid: 42,
      },
      mismatches: ["forward_ipv4 desired=true observed=false"],
    });
    const node: Node = {
      id: "router",
      laboratory_id: "lab-1",
      name: "服务路由器",
      kind: "docker" as const,
      revision: 4,
      desired_state: "stopped",
      observed_state: "stopped",
      cpu_count: 1,
      cpu_quota_micros: 0,
      memory_mib: 128,
      storage_gib: 0,
      interface_limit: 4,
      process_limit: 32,
      config: {},
    };
    const wrapper = mount(DiagnosticsPanel, {
      props: { nodeId: node.id, nodes: [node] },
    });
    await flushPromises();
    expect(wrapper.text()).toContain("IPv4 转发：true / false");
    expect(wrapper.text()).toContain(
      "forward_ipv4 desired=true observed=false",
    );
    await wrapper.findAll("button")[0].trigger("click");
    expect(wrapper.emitted("setNodeForwarding")).toEqual([[node, true, true]]);
  });
});
