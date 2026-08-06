import { mount } from "@vue/test-utils";
import { describe, expect, it, vi } from "vitest";
let option: unknown;
vi.mock("@/components/charts/EChart.vue", () => ({
  default: {
    props: ["option"],
    setup(props: { option: unknown }) {
      option = props.option;
    },
    template: "<div />",
  },
}));
import TrafficFilterChart from "./TrafficFilterChart.vue";
describe("TrafficFilterChart", () => {
  it("keeps the selected scope visible before traffic matches", () => {
    const wrapper = mount(TrafficFilterChart, {
      props: {
        observations: [],
        listening: true,
        expression: "icmp",
        scopeNodes: [
          { id: "node-a", label: "BusyBox", x: 10, y: 20 },
          { id: "node-b", label: "Ubuntu", x: 100, y: 20 },
        ],
        scopeLinks: [
          {
            id: "link-a",
            source: "node-a",
            target: "node-b",
            label: "BusyBox eth0 ↔ Ubuntu ens0",
          },
        ],
      },
    });
    const value = option as {
      series: Array<{
        data: Array<{ id: string }>;
        links: Array<{
          source: string;
          target: string;
          lineStyle: { type: string };
        }>;
      }>;
    };
    expect(value.series[0].data).toHaveLength(2);
    expect(value.series[0].links).toEqual([
      expect.objectContaining({
        source: "node-a",
        target: "node-b",
        lineStyle: expect.objectContaining({ type: "dashed" }),
      }),
    ]);
    expect(wrapper.text()).toContain("监听 icmp");
    expect(wrapper.text()).toContain("灰色虚线表示监听范围");
  });

  it("replaces a scope preview edge with an observed traffic edge", () => {
    const wrapper = mount(TrafficFilterChart, {
      props: {
        sessionStartedAt: "2026-07-30T00:00:00.000Z",
        sessionFinishedAt: "2026-07-30T00:05:00.000Z",
        observations: [
          {
            fingerprint: "request",
            interface_id: "if-a",
            direction: "egress",
            first_seen: "2026-07-30T00:00:00.000Z",
            last_seen: "2026-07-30T00:00:00.000Z",
            count: 1,
            bytes: 64,
          },
          {
            fingerprint: "request",
            interface_id: "if-b",
            direction: "ingress",
            first_seen: "2026-07-30T00:00:00.001Z",
            last_seen: "2026-07-30T00:00:00.001Z",
            count: 1,
            bytes: 64,
          },
        ],
        interfaceOwners: { "if-a": "node-a", "if-b": "node-b" },
        scopeNodes: [
          { id: "node-a", label: "BusyBox" },
          { id: "node-b", label: "Ubuntu" },
        ],
        scopeLinks: [
          {
            id: "link-a",
            source: "node-a",
            target: "node-b",
            label: "BusyBox eth0 ↔ Ubuntu ens0",
          },
        ],
      },
    });
    const value = option as {
      series: Array<{
        links: Array<{
          source: string;
          target: string;
          lineStyle: { type: string };
        }>;
      }>;
    };
    expect(value.series[0].links).toHaveLength(1);
    expect(value.series[0].links[0]).toEqual(
      expect.objectContaining({
        source: "node-a",
        target: "node-b",
        lineStyle: expect.objectContaining({ type: "solid" }),
      }),
    );
    expect(wrapper.text()).toContain("会话：");
    expect(wrapper.text()).toContain("匹配流量：");
    expect(wrapper.text()).toContain("2026-07-30");
    expect(wrapper.findAll("time")).toHaveLength(4);
  });

  it("renders packet direction, count, and ambiguity", () => {
    const wrapper = mount(TrafficFilterChart, {
      props: {
        ambiguous: true,
        observations: [
          {
            fingerprint: "a",
            interface_id: "if-1",
            direction: "ingress",
            first_seen: "",
            last_seen: "",
            count: 2,
            bytes: 20,
          },
          {
            fingerprint: "a",
            interface_id: "if-2",
            link_id: "link",
            direction: "egress",
            first_seen: "",
            last_seen: "",
            count: 3,
            bytes: 30,
          },
        ],
      },
    });
    expect(wrapper.text()).toContain("观测到多条路径");
    const value = option as {
      series: Array<{ links: Array<{ label: string }> }>;
    };
    expect(value.series[0].links[0].label).toContain("3 observations");
  });

  it("does not connect observations from different fingerprints", () => {
    mount(TrafficFilterChart, {
      props: {
        observations: [
          {
            fingerprint: "request",
            interface_id: "if-a",
            direction: "observed",
            first_seen: "2026-07-30T00:00:00.000Z",
            last_seen: "2026-07-30T00:00:00.000Z",
            count: 1,
            bytes: 64,
          },
          {
            fingerprint: "request",
            interface_id: "if-b",
            direction: "observed",
            first_seen: "2026-07-30T00:00:00.001Z",
            last_seen: "2026-07-30T00:00:00.001Z",
            count: 1,
            bytes: 64,
          },
          {
            fingerprint: "response",
            interface_id: "if-b",
            direction: "observed",
            first_seen: "2026-07-30T00:00:00.002Z",
            last_seen: "2026-07-30T00:00:00.002Z",
            count: 1,
            bytes: 64,
          },
          {
            fingerprint: "response",
            interface_id: "if-a",
            direction: "observed",
            first_seen: "2026-07-30T00:00:00.003Z",
            last_seen: "2026-07-30T00:00:00.003Z",
            count: 1,
            bytes: 64,
          },
        ],
      },
    });
    const value = option as {
      series: Array<{ links: Array<{ source: string; target: string }> }>;
    };
    expect(value.series[0].links).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ source: "if-a", target: "if-b" }),
        expect.objectContaining({ source: "if-b", target: "if-a" }),
      ]),
    );
    expect(value.series[0].links).toHaveLength(2);
  });

  it("uses packet MAC addresses instead of capture arrival order", () => {
    mount(TrafficFilterChart, {
      props: {
        interfaceOwners: { "if-a": "node-a", "if-b": "node-b" },
        macOwners: {
          "02:00:00:00:00:0a": "node-a",
          "02:00:00:00:00:0b": "node-b",
        },
        observations: [
          {
            fingerprint: "request",
            interface_id: "if-b",
            direction: "observed",
            source_mac: "02:00:00:00:00:0a",
            destination_mac: "02:00:00:00:00:0b",
            first_seen: "2026-07-30T00:00:00.000Z",
            last_seen: "2026-07-30T00:00:00.000Z",
            count: 1,
            bytes: 64,
          },
          {
            fingerprint: "request",
            interface_id: "if-a",
            direction: "observed",
            source_mac: "02:00:00:00:00:0a",
            destination_mac: "02:00:00:00:00:0b",
            first_seen: "2026-07-30T00:00:00.010Z",
            last_seen: "2026-07-30T00:00:00.010Z",
            count: 1,
            bytes: 64,
          },
        ],
      },
    });
    const value = option as {
      series: Array<{ links: Array<{ source: string; target: string }> }>;
    };
    expect(value.series[0].links).toEqual([
      expect.objectContaining({ source: "node-a", target: "node-b" }),
    ]);
  });

  it("directs link observations from the packet sender to receiver", () => {
    mount(TrafficFilterChart, {
      props: {
        observations: [
          {
            fingerprint: "reply",
            interface_id: "",
            link_id: "link-a",
            direction: "observed",
            source_mac: "02:00:00:00:00:0b",
            destination_mac: "02:00:00:00:00:0a",
            first_seen: "2026-07-30T00:00:00.000Z",
            last_seen: "2026-07-30T00:00:00.000Z",
            count: 1,
            bytes: 64,
          },
        ],
        macOwners: {
          "02:00:00:00:00:0a": "node-a",
          "02:00:00:00:00:0b": "node-b",
        },
        scopeNodes: [
          { id: "node-a", label: "BusyBox" },
          { id: "node-b", label: "Ubuntu" },
        ],
        scopeLinks: [
          {
            id: "link-a",
            source: "node-a",
            target: "node-b",
            label: "BusyBox eth0 ↔ Ubuntu ens0",
          },
        ],
      },
    });
    const value = option as {
      series: Array<{
        links: Array<{
          source: string;
          target: string;
          symbol: string[];
        }>;
      }>;
    };
    expect(value.series[0].links).toEqual([
      expect.objectContaining({
        source: "node-b",
        target: "node-a",
        symbol: ["none", "arrow"],
      }),
    ]);
  });
});
