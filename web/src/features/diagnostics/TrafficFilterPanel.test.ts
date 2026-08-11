import { flushPromises, mount } from "@vue/test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  api,
  type Link,
  type NetworkAttachment,
  type NetworkObject,
  type Node,
  type NodeInterface,
} from "@/api";

vi.mock("@/components/charts/EChart.vue", () => ({
  default: {
    props: ["option"],
    template: '<div data-testid="traffic-chart" />',
  },
}));

import TrafficFilterPanel from "./TrafficFilterPanel.vue";

const nodes = [
  { id: "node-a", name: "BusyBox" },
  { id: "node-b", name: "Ubuntu" },
] as Node[];
const interfaces = [
  {
    id: "if-a",
    node_id: "node-a",
    name: "eth0",
    slot: 0,
    operational_state: "up",
    desired_link_id: "link-a",
  },
  {
    id: "if-b",
    node_id: "node-b",
    name: "ens0",
    slot: 0,
    operational_state: "up",
    desired_link_id: "link-a",
  },
] as NodeInterface[];
const links = [
  {
    id: "link-a",
    laboratory_id: "lab",
    endpoint_a_id: "if-a",
    endpoint_b_id: "if-b",
    observed_state: "connected",
  },
] as Link[];
const attachments = [
  {
    id: "attachment-a",
    network_object_id: "switch-a",
    interface_id: "if-a",
    port_name: "lan0",
    revision: 1,
    observed_state: "active",
  },
] as NetworkAttachment[];
const networkObjects = [
  {
    id: "switch-a",
    laboratory_id: "lab",
    name: "Lightweight L2",
    kind: "switch_l2",
    revision: 1,
    desired_state: "active",
    observed_state: "active",
    config: {},
  },
] as NetworkObject[];
const networkObjectLinks = [
  {
    id: "object-link-a",
    laboratory_id: "lab",
    object_a_id: "switch-a",
    port_a_name: "swp1",
    object_b_id: "switch-b",
    port_b_name: "swp1",
    revision: 1,
    desired_state: "connected",
    observed_state: "connected",
  },
] as const;

describe("TrafficFilterPanel interactions", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    vi.spyOn(api, "listTrafficFilters").mockResolvedValue([]);
  });

  it("requires an observation scope before starting", async () => {
    const wrapper = mount(TrafficFilterPanel, {
      props: { laboratoryId: "lab", nodes, interfaces, links },
    });
    await flushPromises();
    const start = wrapper
      .findAll("button")
      .find((button) => button.text().trim() === "启动")!;
    expect(start.attributes("disabled")).toBeDefined();
  });

  it("scopes a filter to the selected network object link", async () => {
    const startTrafficFilter = vi
      .spyOn(api, "startTrafficFilter")
      .mockResolvedValue({
        task: {
          id: "task-object-link",
          kind: "traffic_filter.start",
          resource_type: "traffic_filter",
          resource_id: "filter-object-link",
          state: "queued",
          progress_current: 0,
          progress_total: 2,
          created_at: "2026-08-03T00:00:00Z",
        },
        traffic_filter: {
          id: "filter-object-link",
          laboratory_id: "lab",
          expression: "icmp",
          color: "#f59e0b",
          state: "starting",
          max_observations: 1000,
          network_object_link_ids: ["object-link-a"],
          observations: [],
          created_at: "2026-08-03T00:00:00Z",
        },
      });
    vi.spyOn(api, "getTask").mockResolvedValue({
      id: "task-object-link",
      kind: "traffic_filter.start",
      resource_type: "traffic_filter",
      resource_id: "filter-object-link",
      state: "succeeded",
      progress_current: 2,
      progress_total: 2,
      created_at: "2026-08-03T00:00:00Z",
    });
    vi.spyOn(api, "getTrafficFilter").mockResolvedValue({
      ambiguous: false,
      traffic_filter: {
        id: "filter-object-link",
        laboratory_id: "lab",
        expression: "icmp",
        color: "#f59e0b",
        state: "running",
        max_observations: 1000,
        network_object_link_ids: ["object-link-a"],
        observations: [],
        created_at: "2026-08-03T00:00:00Z",
      },
    });
    const wrapper = mount(TrafficFilterPanel, {
      props: {
        laboratoryId: "lab",
        objectLinkId: "object-link-a",
        nodes,
        interfaces,
        links,
        networkObjects: [
          ...networkObjects,
          { ...networkObjects[0], id: "switch-b", name: "Lightweight B" },
        ],
        networkObjectLinks: [...networkObjectLinks],
      },
    });
    await flushPromises();
    await wrapper
      .findAll("button")
      .find((button) => button.text().trim() === "启动")!
      .trigger("click");
    await flushPromises();
    expect(startTrafficFilter).toHaveBeenCalledWith(
      expect.objectContaining({
        interface_ids: [],
        link_ids: [],
        network_object_link_ids: ["object-link-a"],
      }),
    );
  });

  it("treats a Lightweight attachment as an observable link segment", async () => {
    const startTrafficFilter = vi
      .spyOn(api, "startTrafficFilter")
      .mockResolvedValue({
        task: {
          id: "task-attachment",
          kind: "traffic_filter.start",
          resource_type: "traffic_filter",
          resource_id: "filter-attachment",
          state: "queued",
          progress_current: 0,
          progress_total: 2,
          created_at: "2026-07-31T00:00:00Z",
        },
        traffic_filter: {
          id: "filter-attachment",
          laboratory_id: "lab",
          expression: "icmp",
          color: "#f59e0b",
          state: "starting",
          max_observations: 1000,
          interface_ids: ["if-a"],
          link_ids: [],
          observations: [],
          created_at: "2026-07-31T00:00:00Z",
        },
      });
    vi.spyOn(api, "getTask").mockResolvedValue({
      id: "task-attachment",
      kind: "traffic_filter.start",
      resource_type: "traffic_filter",
      resource_id: "filter-attachment",
      state: "succeeded",
      progress_current: 2,
      progress_total: 2,
      created_at: "2026-07-31T00:00:00Z",
    });
    vi.spyOn(api, "getTrafficFilter").mockResolvedValue({
      ambiguous: false,
      traffic_filter: {
        id: "filter-attachment",
        laboratory_id: "lab",
        expression: "icmp",
        color: "#f59e0b",
        state: "running",
        max_observations: 1000,
        interface_ids: ["if-a"],
        link_ids: [],
        observations: [],
        created_at: "2026-07-31T00:00:00Z",
      },
    });
    const wrapper = mount(TrafficFilterPanel, {
      props: {
        laboratoryId: "lab",
        interfaceId: "if-a",
        nodes,
        interfaces,
        links: [],
        attachments,
        networkObjects,
      },
    });
    await flushPromises();

    expect(wrapper.text()).toContain("BusyBox:eth0 ↔ Lightweight L2:lan0");
    await wrapper
      .findAll("button")
      .find((button) => button.text().trim() === "启动")!
      .trigger("click");
    await flushPromises();

    expect(startTrafficFilter).toHaveBeenCalledWith(
      expect.objectContaining({ interface_ids: ["if-a"], link_ids: [] }),
    );
  });

  it("reports two capture slots for each directional object link", async () => {
    const wrapper = mount(TrafficFilterPanel, {
      props: {
        laboratoryId: "lab",
        nodes,
        interfaces,
        links: [],
        attachments,
        networkObjects,
        networkObjectLinks: [...networkObjectLinks],
      },
    });
    await flushPromises();

    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("已连接链路"))!
      .trigger("click");

    expect(wrapper.text()).toContain("2 个监听源，预计占用 3 个抓包槽位");
    expect(wrapper.text()).toContain("对象链路为区分方向占 2 个");
  });

  it("stops the active filter before applying a changed expression", async () => {
    vi.mocked(api.listTrafficFilters).mockResolvedValue([
      {
        ambiguous: false,
        traffic_filter: {
          id: "filter-running",
          laboratory_id: "lab",
          expression: "icmp",
          color: "#22c55e",
          state: "running",
          max_observations: 1000,
          interface_ids: ["if-a", "if-b"],
          fingerprint_count: 3,
          matched_packets: 12,
          matched_bytes: 2048,
          link_ids: [],
          network_object_link_ids: [],
          observations: [],
          created_at: "2026-08-04T00:00:00Z",
        },
      },
    ]);
    const stopTrafficFilter = vi
      .spyOn(api, "stopTrafficFilter")
      .mockResolvedValue({
        task: {
          id: "task-stop",
          kind: "traffic_filter.stop",
          resource_type: "traffic_filter",
          resource_id: "filter-running",
          state: "queued",
          progress_current: 0,
          progress_total: 2,
          created_at: "2026-08-04T00:00:01Z",
        },
      });
    const startTrafficFilter = vi
      .spyOn(api, "startTrafficFilter")
      .mockResolvedValue({
        task: {
          id: "task-start-replacement",
          kind: "traffic_filter.start",
          resource_type: "traffic_filter",
          resource_id: "filter-replacement",
          state: "queued",
          progress_current: 0,
          progress_total: 2,
          created_at: "2026-08-04T00:00:02Z",
        },
        traffic_filter: {
          id: "filter-replacement",
          laboratory_id: "lab",
          expression: "udp and dst port 19002",
          color: "#22c55e",
          state: "starting",
          max_observations: 1000,
          interface_ids: ["if-a", "if-b"],
          link_ids: [],
          network_object_link_ids: [],
          observations: [],
          created_at: "2026-08-04T00:00:02Z",
        },
      });
    vi.spyOn(api, "getTask").mockImplementation(async (id) => ({
      id,
      kind: id === "task-stop" ? "traffic_filter.stop" : "traffic_filter.start",
      resource_type: "traffic_filter",
      resource_id: id === "task-stop" ? "filter-running" : "filter-replacement",
      state: "succeeded",
      progress_current: 2,
      progress_total: 2,
      created_at: "2026-08-04T00:00:03Z",
    }));
    vi.spyOn(api, "getTrafficFilter").mockResolvedValue({
      ambiguous: false,
      traffic_filter: {
        id: "filter-replacement",
        laboratory_id: "lab",
        expression: "udp and dst port 19002",
        color: "#22c55e",
        state: "running",
        max_observations: 1000,
        interface_ids: ["if-a", "if-b"],
        link_ids: [],
        network_object_link_ids: [],
        observations: [],
        created_at: "2026-08-04T00:00:02Z",
      },
    });
    const wrapper = mount(TrafficFilterPanel, {
      props: { laboratoryId: "lab", nodes, interfaces, links },
    });
    await flushPromises();

    expect(wrapper.text()).toContain("应用并重启");
    await wrapper
      .find('input[aria-label="pcap 过滤表达式"]')
      .setValue("udp dst port 19002");
    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("应用并重启"))!
      .trigger("click");
    await flushPromises();

    expect(stopTrafficFilter).toHaveBeenCalledWith("filter-running");
    expect(startTrafficFilter).toHaveBeenCalledWith(
      expect.objectContaining({
        match: { protocol: "udp", destination_port: 19002 },
        interface_ids: ["if-a", "if-b"],
      }),
    );
    expect(stopTrafficFilter.mock.invocationCallOrder[0]).toBeLessThan(
      startTrafficFilter.mock.invocationCallOrder[0],
    );
    expect(wrapper.text()).toContain("旧会话已自动停止");
  });

  it("searches sessions and stops an active session before deleting it", async () => {
    vi.mocked(api.listTrafficFilters).mockResolvedValue([
      {
        ambiguous: false,
        traffic_filter: {
          id: "filter-running",
          laboratory_id: "lab",
          expression: "udp and dst port 19002",
          color: "#a855f7",
          state: "running",
          max_observations: 1000,
          interface_ids: ["if-a"],
          link_ids: [],
          network_object_link_ids: [],
          observations: [],
          created_at: "2026-08-04T00:00:02Z",
        },
      },
      {
        ambiguous: false,
        traffic_filter: {
          id: "filter-stopped",
          laboratory_id: "lab",
          expression: "icmp",
          color: "#22c55e",
          state: "stopped",
          max_observations: 1000,
          interface_ids: ["if-b"],
          link_ids: [],
          network_object_link_ids: [],
          observations: [],
          created_at: "2026-08-04T00:00:01Z",
        },
      },
    ]);
    const stopTrafficFilter = vi
      .spyOn(api, "stopTrafficFilter")
      .mockResolvedValue({
        task: {
          id: "task-delete-stop",
          kind: "traffic_filter.stop",
          resource_type: "traffic_filter",
          resource_id: "filter-running",
          state: "queued",
          progress_current: 0,
          progress_total: 2,
          created_at: "2026-08-04T00:00:03Z",
        },
      });
    vi.spyOn(api, "getTask").mockResolvedValue({
      id: "task-delete-stop",
      kind: "traffic_filter.stop",
      resource_type: "traffic_filter",
      resource_id: "filter-running",
      state: "succeeded",
      progress_current: 2,
      progress_total: 2,
      created_at: "2026-08-04T00:00:03Z",
    });
    const deleteTrafficFilterHistory = vi
      .spyOn(api, "deleteTrafficFilterHistory")
      .mockResolvedValue({
        traffic_filter: {
          id: "filter-running",
          laboratory_id: "lab",
          expression: "udp and dst port 19002",
          color: "#a855f7",
          state: "stopped",
          max_observations: 1000,
          interface_ids: ["if-a"],
          link_ids: [],
          network_object_link_ids: [],
          observations: [],
          created_at: "2026-08-04T00:00:02Z",
        },
      });
    const wrapper = mount(TrafficFilterPanel, {
      props: { laboratoryId: "lab", nodes, interfaces, links },
    });
    await flushPromises();

    await wrapper
      .find('input[aria-label="搜索流量过滤会话"]')
      .setValue("udp 19002");
    expect(
      wrapper.findAll('button[aria-label^="选择流量过滤会话"]'),
    ).toHaveLength(1);
    await wrapper
      .find('button[aria-label="删除流量过滤会话 filter-running"]')
      .trigger("click");
    await flushPromises();

    expect(document.body.textContent).toContain("删除流量过滤会话");
    expect(document.body.textContent).toContain("filter-running");
    expect(stopTrafficFilter).not.toHaveBeenCalled();
    Array.from(document.body.querySelectorAll("button"))
      .find((button) => button.textContent?.trim() === "确认删除")!
      .click();
    await flushPromises();

    expect(stopTrafficFilter).toHaveBeenCalledWith("filter-running");
    expect(deleteTrafficFilterHistory).toHaveBeenCalledWith("filter-running");
    expect(stopTrafficFilter.mock.invocationCallOrder[0]).toBeLessThan(
      deleteTrafficFilterHistory.mock.invocationCallOrder[0],
    );
    expect(wrapper.text()).toContain("已删除 1 条流量过滤会话");
    expect(wrapper.text()).toContain("没有匹配的流量过滤会话");
    wrapper.unmount();
  });

  it("confirms and deletes all sessions while stopping active sessions", async () => {
    const trafficFilters = [
      {
        ambiguous: false,
        traffic_filter: {
          id: "filter-running",
          laboratory_id: "lab",
          expression: "tcp dst port 443",
          color: "#a855f7",
          state: "running",
          max_observations: 1000,
          interface_ids: ["if-a"],
          link_ids: [],
          network_object_link_ids: [],
          observations: [],
          created_at: "2026-08-04T00:00:02Z",
        },
      },
      {
        ambiguous: false,
        traffic_filter: {
          id: "filter-stopped",
          laboratory_id: "lab",
          expression: "icmp",
          color: "#22c55e",
          state: "stopped",
          max_observations: 1000,
          interface_ids: ["if-b"],
          link_ids: [],
          network_object_link_ids: [],
          observations: [],
          created_at: "2026-08-04T00:00:01Z",
        },
      },
    ];
    vi.mocked(api.listTrafficFilters).mockResolvedValue(trafficFilters);
    const stopTrafficFilter = vi
      .spyOn(api, "stopTrafficFilter")
      .mockResolvedValue({
        task: {
          id: "task-delete-all-stop",
          kind: "traffic_filter.stop",
          resource_type: "traffic_filter",
          resource_id: "filter-running",
          state: "queued",
          progress_current: 0,
          progress_total: 2,
          created_at: "2026-08-04T00:00:03Z",
        },
      });
    vi.spyOn(api, "getTask").mockResolvedValue({
      id: "task-delete-all-stop",
      kind: "traffic_filter.stop",
      resource_type: "traffic_filter",
      resource_id: "filter-running",
      state: "succeeded",
      progress_current: 2,
      progress_total: 2,
      created_at: "2026-08-04T00:00:03Z",
    });
    const deleteTrafficFilterHistory = vi
      .spyOn(api, "deleteTrafficFilterHistory")
      .mockImplementation(async (filterId) => ({
        traffic_filter: {
          ...trafficFilters.find(
            (entry) => entry.traffic_filter.id === filterId,
          )!.traffic_filter,
          state: "stopped",
        },
      }));
    const wrapper = mount(TrafficFilterPanel, {
      props: { laboratoryId: "lab", nodes, interfaces, links },
    });
    await flushPromises();

    await wrapper
      .find('button[aria-label="删除全部流量过滤会话"]')
      .trigger("click");
    await flushPromises();
    expect(document.body.textContent).toContain(
      "当前实验室的 2 条流量过滤会话",
    );
    expect(document.body.textContent).toContain("1 条运行中会话会先停止");
    Array.from(document.body.querySelectorAll("button"))
      .find((button) => button.textContent?.trim() === "取消")!
      .click();
    await flushPromises();
    expect(deleteTrafficFilterHistory).not.toHaveBeenCalled();

    await wrapper
      .find('button[aria-label="删除全部流量过滤会话"]')
      .trigger("click");
    Array.from(document.body.querySelectorAll("button"))
      .find((button) => button.textContent?.trim() === "确认全部删除")!
      .click();
    await flushPromises();

    expect(stopTrafficFilter).toHaveBeenCalledTimes(1);
    expect(deleteTrafficFilterHistory).toHaveBeenCalledTimes(2);
    expect(deleteTrafficFilterHistory).toHaveBeenNthCalledWith(
      1,
      "filter-running",
    );
    expect(deleteTrafficFilterHistory).toHaveBeenNthCalledWith(
      2,
      "filter-stopped",
    );
    expect(wrapper.text()).toContain("已删除 2 条流量过滤会话");
    expect(
      wrapper
        .find('button[aria-label="删除全部流量过滤会话"]')
        .attributes("disabled"),
    ).toBeDefined();
    wrapper.unmount();
  });

  it("starts multiple connected interfaces and follows the durable task", async () => {
    const startTrafficFilter = vi
      .spyOn(api, "startTrafficFilter")
      .mockResolvedValue({
        task: {
          id: "task-start",
          kind: "traffic_filter.start",
          resource_type: "traffic_filter",
          resource_id: "filter-a",
          state: "queued",
          progress_current: 0,
          progress_total: 2,
          created_at: "2026-07-30T00:00:00Z",
        },
        traffic_filter: {
          id: "filter-a",
          laboratory_id: "lab",
          expression: "icmp",
          color: "#22c55e",
          state: "starting",
          max_observations: 1000,
          interface_ids: ["if-a", "if-b"],
          observations: [],
          created_at: "2026-07-30T00:00:00Z",
        },
      });
    vi.spyOn(api, "getTask").mockResolvedValue({
      id: "task-start",
      kind: "traffic_filter.start",
      resource_type: "traffic_filter",
      resource_id: "filter-a",
      state: "succeeded",
      progress_current: 2,
      progress_total: 2,
      created_at: "2026-07-30T00:00:00Z",
    });
    vi.spyOn(api, "getTrafficFilter").mockResolvedValue({
      ambiguous: false,
      traffic_filter: {
        id: "filter-a",
        laboratory_id: "lab",
        expression: "icmp",
        color: "#22c55e",
        state: "running",
        max_observations: 1000,
        interface_ids: ["if-a", "if-b"],
        fingerprint_count: 3,
        matched_packets: 12,
        matched_bytes: 2048,
        observations: [
          {
            fingerprint: "icmp-a",
            interface_id: "if-a",
            link_id: "link-a",
            direction: "egress",
            first_seen: "2026-07-30T00:00:01Z",
            last_seen: "2026-07-30T00:00:02Z",
            count: 4,
            bytes: 392,
          },
        ],
        created_at: "2026-07-30T00:00:00Z",
      },
    });
    const wrapper = mount(TrafficFilterPanel, {
      props: { laboratoryId: "lab", nodes, interfaces, links },
    });
    await flushPromises();
    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("已连接接口"))!
      .trigger("click");
    await wrapper
      .findAll("button")
      .find((button) => button.text().trim() === "启动")!
      .trigger("click");
    await flushPromises();

    expect(startTrafficFilter).toHaveBeenCalledWith(
      expect.objectContaining({
        interface_ids: ["if-a", "if-b"],
        link_ids: [],
        match: { protocol: "icmp" },
        color: "#f59e0b",
      }),
    );
    expect(wrapper.text()).toContain("running");
    expect(wrapper.text()).toContain("自动刷新匹配结果");
    expect(wrapper.text()).toContain("拓扑流量高亮");
    expect(wrapper.text()).toContain("匹配的数据包直接在主拓扑链路上流动显示");
    expect(wrapper.findAll("dd").map((value) => value.text())).toEqual(
      expect.arrayContaining(["3", "12", "2048"]),
    );
    const overlayEvents = wrapper.emitted("overlay") || [];
    expect(overlayEvents.at(-1)).toEqual([
      [expect.objectContaining({ link_id: "link-a", count: 4 })],
      true,
      "#22c55e",
    ]);
  });

  it("restores the observation scope when selecting a historical session", async () => {
    vi.mocked(api.listTrafficFilters).mockResolvedValue([
      {
        ambiguous: false,
        traffic_filter: {
          id: "filter-old",
          laboratory_id: "lab",
          expression: "tcp port 443",
          color: "#38bdf8",
          state: "stopped",
          max_observations: 500,
          interface_ids: [],
          link_ids: ["link-a"],
          observations: [],
          created_at: "2026-07-29T00:00:00Z",
        },
      },
    ]);
    const wrapper = mount(TrafficFilterPanel, {
      props: { laboratoryId: "lab", nodes, interfaces, links },
    });
    await flushPromises();

    const linkCheckbox = wrapper
      .findAll('input[type="checkbox"]')
      .find((item) => item.element.parentElement?.textContent?.includes("↔"));
    expect((linkCheckbox?.element as HTMLInputElement).checked).toBe(true);
    expect(
      (wrapper.find('input[placeholder^="icmp"]').element as HTMLInputElement)
        .value,
    ).toBe("tcp port 443");
    expect(
      (
        wrapper.find('input[aria-label="过滤器颜色值"]')
          .element as HTMLInputElement
      ).value,
    ).toBe("#38bdf8");
  });

  it("applies a pcap example and switches to custom after editing", async () => {
    const wrapper = mount(TrafficFilterPanel, {
      props: { laboratoryId: "lab", nodes, interfaces, links },
    });
    await flushPromises();

    const examples = wrapper.find('select[aria-label="过滤模板"]');
    const expressionInput = wrapper.find('input[aria-label="pcap 过滤表达式"]');
    await examples.setValue("udp port 53");
    expect((expressionInput.element as HTMLInputElement).value).toBe(
      "udp port 53",
    );

    await expressionInput.setValue("tcp port 8080");
    expect((examples.element as HTMLSelectElement).value).toBe("custom");
  });
});
