import { beforeEach, describe, expect, it, vi } from "vitest";
import { flushPromises, mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import TrafficWorkloadPanel from "./TrafficWorkloadPanel.vue";
import { api, type TrafficWorkload } from "@/api";

const workload: TrafficWorkload = {
  id: "workload",
  laboratory_id: "lab",
  name: "steady ping",
  revision: 2,
  source: { kind: "node", resource_id: "node" },
  protocol: "icmp",
  address_family: "ipv4",
  destination: { address: "192.0.2.1" },
  interval_seconds: 5,
  timeout_seconds: 2,
  desired_state: "running",
  observed_state: "queued",
  attempts: 4,
  successes: 3,
  failures: 1,
  matched_bytes: 192,
  last_success_at: "2026-08-12T00:00:00Z",
  created_at: "2026-08-12T00:00:00Z",
  updated_at: "2026-08-12T00:00:05Z",
};

describe("TrafficWorkloadPanel", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.restoreAllMocks();
    vi.spyOn(api, "listTrafficWorkloads").mockResolvedValue([workload]);
    vi.spyOn(api, "listTrafficFilters").mockResolvedValue([]);
  });

  it("renders durable aggregates and degraded running state", async () => {
    const wrapper = mount(TrafficWorkloadPanel, {
      props: {
        laboratoryId: "lab",
        nodes: [
          {
            id: "node",
            laboratory_id: "lab",
            name: "Ubuntu",
            kind: "qemu",
            revision: 1,
            desired_state: "running",
            observed_state: "running",
            cpu_count: 1,
            cpu_quota_micros: 0,
            memory_mib: 512,
            storage_gib: 8,
            interface_limit: 2,
            process_limit: 128,
          },
        ],
      },
      global: { plugins: [createPinia()] },
    });
    await flushPromises();
    expect(wrapper.text()).toContain("尝试 4");
    expect(wrapper.text()).toContain("成功 3");
    expect(wrapper.text()).toContain("失败 1");
    expect(wrapper.text()).toContain("匹配字节 192");
    expect(wrapper.text()).toContain("运行降级");
  });

  it("highlights only filters correlated with the last success window", async () => {
    vi.mocked(api.listTrafficFilters).mockResolvedValue([
      {
        ambiguous: false,
        traffic_filter: {
          id: "filter",
          laboratory_id: "lab",
          expression: "icmp and ip",
          color: "#22c55e",
          state: "running",
          max_observations: 100,
          observations: [
            {
              fingerprint: "icmp:ipv4",
              interface_id: "if-1",
              link_id: "link-1",
              direction: "egress",
              first_seen: "2026-08-12T00:00:01Z",
              last_seen: "2026-08-12T00:00:01Z",
              count: 1,
              bytes: 64,
            },
          ],
          matched_packets: 1,
          matched_bytes: 64,
          last_match_at: "2026-08-12T00:00:01Z",
          created_at: "2026-08-12T00:00:00Z",
        },
      },
    ]);
    const wrapper = mount(TrafficWorkloadPanel, {
      props: { laboratoryId: "lab" },
      global: { plugins: [createPinia()] },
    });
    await flushPromises();
    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("高亮匹配路径"))!
      .trigger("click");
    await flushPromises();
    expect(wrapper.emitted("overlay")?.[0]?.[1]).toBe(true);
    expect(wrapper.emitted("overlay")?.[0]?.[2]).toBe("#22c55e");
  });
});
