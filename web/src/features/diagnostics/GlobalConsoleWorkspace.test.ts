import { flushPromises, mount } from "@vue/test-utils";
import { describe, expect, it, vi } from "vitest";
import { api } from "@/api";
import { nodeFactory } from "@/test/factories";

vi.mock("./ConsoleWorkspace.vue", () => ({
  default: {
    props: ["nodeId", "resourceType", "autoOpen"],
    template:
      '<div data-console-workspace :data-node-id="nodeId" :data-resource-type="resourceType">{{ nodeId }}</div>',
  },
}));

import GlobalConsoleWorkspace from "./GlobalConsoleWorkspace.vue";

describe("GlobalConsoleWorkspace", () => {
  it("restores node and terminal tabs after a browser refresh", async () => {
    vi.spyOn(api, "listNodeConsoles").mockResolvedValue([
      {
        mode: "telnet",
        stream_url: "/console",
        reconnectable: true,
        idle_seconds: 1800,
      },
    ]);
    const nodes = [
      nodeFactory({ id: "ubuntu", name: "Ubuntu", kind: "docker" }),
    ];
    const first = mount(GlobalConsoleWorkspace, {
      props: {
        laboratoryId: "lab-restore",
        nodes,
        requestNodeId: "ubuntu",
        requestKey: 1,
      },
    });
    await flushPromises();
    await first.get('[aria-label="Add terminal session"]').trigger("click");
    expect(first.get('[aria-label="Console sessions"]').text()).toContain(
      "SERIAL 2",
    );
    first.unmount();

    const restored = mount(GlobalConsoleWorkspace, {
      props: { laboratoryId: "lab-restore", nodes },
    });
    expect(
      restored.get('[aria-label="Node console workspaces"]').text(),
    ).toContain("Ubuntu");
    expect(restored.get('[aria-label="Console sessions"]').text()).toContain(
      "SERIAL 1",
    );
    expect(restored.get('[aria-label="Console sessions"]').text()).toContain(
      "SERIAL 2",
    );
    expect(restored.findAll("[data-console-workspace]")).toHaveLength(2);
  });

  it("defaults QEMU to serial and only enables SSH when advertised", async () => {
    vi.spyOn(api, "listNodeConsoles").mockResolvedValue([
      {
        mode: "telnet",
        stream_url: "/serial",
        reconnectable: true,
        idle_seconds: 1800,
      },
      {
        mode: "vnc",
        stream_url: "/vnc",
        reconnectable: true,
        idle_seconds: 1800,
      },
    ]);
    const wrapper = mount(GlobalConsoleWorkspace, {
      props: {
        laboratoryId: "lab-qemu-console",
        nodes: [nodeFactory({ id: "ubuntu", name: "Ubuntu", kind: "qemu" })],
        requestNodeId: "ubuntu",
        requestKey: 1,
      },
    });
    await flushPromises();

    expect(wrapper.get('[aria-label="Console sessions"]').text()).toContain(
      "SERIAL 1",
    );
    expect(
      wrapper.get('[aria-label="Add terminal session"]').attributes("disabled"),
    ).toBeDefined();
    await wrapper.get('[aria-label="Add serial console"]').trigger("click");
    expect(
      wrapper.get('[aria-label="Add serial console"]').attributes("disabled"),
    ).toBeDefined();
    expect(wrapper.findAll("[data-console-workspace]")).toHaveLength(1);
  });

  it("keeps each opened node console mounted while switching tabs", async () => {
    vi.spyOn(api, "listNodeConsoles").mockResolvedValue([
      {
        mode: "telnet",
        stream_url: "/console",
        reconnectable: true,
        idle_seconds: 1800,
      },
    ]);
    const wrapper = mount(GlobalConsoleWorkspace, {
      props: {
        nodes: [
          nodeFactory({ id: "ubuntu", name: "Ubuntu", kind: "docker" }),
          nodeFactory({ id: "busybox", name: "BusyBox", kind: "docker" }),
        ],
        requestNodeId: "ubuntu",
        requestKey: 1,
      },
    });
    await flushPromises();

    await wrapper.setProps({ requestNodeId: "busybox", requestKey: 2 });
    await flushPromises();

    const consoles = wrapper.findAll("[data-console-workspace]");
    expect(consoles).toHaveLength(2);
    expect(consoles.map((item) => item.attributes("data-node-id"))).toEqual([
      "ubuntu",
      "busybox",
    ]);
    expect(consoles[0].isVisible()).toBe(false);
    expect(consoles[1].isVisible()).toBe(true);

    await wrapper.setProps({ requestNodeId: "ubuntu", requestKey: 3 });
    await flushPromises();

    const switchedConsoles = wrapper.findAll("[data-console-workspace]");
    expect(switchedConsoles).toHaveLength(2);
    expect(switchedConsoles[0].attributes("style") || "").not.toContain(
      "display: none",
    );
    expect(switchedConsoles[1].attributes("style")).toContain("display: none");
    expect(
      wrapper.get('[aria-label="Node console workspaces"]').text(),
    ).toContain("Ubuntu");
    expect(
      wrapper.get('[aria-label="Node console workspaces"]').text(),
    ).toContain("BusyBox");
    expect(wrapper.get('[aria-label="Console sessions"]').text()).toContain(
      "SERIAL 1",
    );

    await wrapper.get('[aria-label="Add terminal session"]').trigger("click");
    expect(wrapper.findAll("[data-console-workspace]")).toHaveLength(3);
    expect(wrapper.get('[aria-label="Console sessions"]').text()).toContain(
      "SERIAL 2",
    );

    await wrapper.setProps({
      nodes: [nodeFactory({ id: "busybox", name: "BusyBox", kind: "docker" })],
    });

    expect(wrapper.findAll("[data-console-workspace]")).toHaveLength(1);
    expect(wrapper.text()).not.toContain("Ubuntu");
  });

  it("opens persistent PC shell sessions through the network object console API", async () => {
    const listConsoles = vi
      .spyOn(api, "listNetworkObjectConsoles")
      .mockResolvedValue([
        {
          mode: "telnet",
          stream_url: "/api/v1/network-objects/pc-1/consoles/telnet/stream",
          reconnectable: true,
          idle_seconds: 1800,
        },
      ]);
    const wrapper = mount(GlobalConsoleWorkspace, {
      props: {
        laboratoryId: "lab-pc-console",
        nodes: [],
        networkObjects: [
          {
            id: "pc-1",
            laboratory_id: "lab-pc-console",
            name: "PC Client",
            kind: "pc",
            config: {},
            desired_state: "active",
            observed_state: "active",
            revision: 1,
          },
        ],
        requestNetworkObjectId: "pc-1",
        requestKey: 1,
      },
    });
    await flushPromises();

    expect(listConsoles).toHaveBeenCalledWith("pc-1");
    expect(
      wrapper.get('[aria-label="Node console workspaces"]').text(),
    ).toContain("PC Client");
    const consoleWorkspace = wrapper.get("[data-console-workspace]");
    expect(consoleWorkspace.attributes("data-node-id")).toBe("pc-1");
    expect(consoleWorkspace.attributes("data-resource-type")).toBe(
      "network_object",
    );

    await wrapper.get('[aria-label="Add terminal session"]').trigger("click");
    expect(wrapper.findAll("[data-console-workspace]")).toHaveLength(2);
  });
});
