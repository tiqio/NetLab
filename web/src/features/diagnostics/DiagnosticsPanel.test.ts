import { mount } from "@vue/test-utils";
import { describe, expect, it } from "vitest";
import DiagnosticsPanel from "./DiagnosticsPanel.vue";

describe("DiagnosticsPanel", () => {
  it("prompts for a node terminal before a console is requested", () => {
    const wrapper = mount(DiagnosticsPanel, { props: { nodeId: "node-1" } });
    expect(wrapper.text()).toContain("Right-click a node and choose Terminal.");
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
});
