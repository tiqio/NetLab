import { mount } from "@vue/test-utils";
import { describe, expect, it } from "vitest";
import type { Node, NodeInterface } from "@/api";
import GlobalCaptureWorkspace from "./GlobalCaptureWorkspace.vue";

const nodes = [
  { id: "node-1", name: "Ubuntu" },
  { id: "node-2", name: "BusyBox" },
] as Node[];

const interfaces = [
  { id: "if-1", node_id: "node-1", name: "ens0", slot: 0 },
  { id: "if-2", node_id: "node-1", name: "ens1", slot: 1 },
  { id: "if-3", node_id: "node-2", name: "eth0", slot: 0 },
] as NodeInterface[];

function mountWorkspace(requestInterfaceId = "if-1") {
  return mount(GlobalCaptureWorkspace, {
    props: {
      laboratoryId: "lab-1",
      nodes,
      interfaces,
      links: [],
      requestInterfaceId,
    },
    global: {
      stubs: {
        CapturePanel: {
          name: "CapturePanel",
          props: ["interfaceId", "linkId", "sourceLabel"],
          emits: ["captureChange"],
          template:
            '<div data-testid="capture-panel">{{ sourceLabel }} / {{ interfaceId || linkId }}</div>',
        },
      },
    },
  });
}

describe("GlobalCaptureWorkspace", () => {
  it("groups capture sources by node and interface", () => {
    const wrapper = mountWorkspace();
    expect(wrapper.text()).toContain("Ubuntu");
    expect(wrapper.text()).toContain("BusyBox");
    expect(wrapper.text()).toContain("ens0");
    expect(wrapper.text()).toContain("ens1");
    expect(wrapper.text()).toContain("Ubuntu · ens0");
  });

  it("keeps multiple interface capture panels mounted while switching", async () => {
    const wrapper = mountWorkspace();
    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("ens1"))!
      .trigger("click");

    const panels = wrapper.findAll('[data-testid="capture-panel"]');
    expect(panels).toHaveLength(2);
    expect(panels[0].attributes("style")).toContain("display: none");
    expect(panels[1].text()).toContain("Ubuntu · ens1");

    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("BusyBox"))!
      .trigger("click");
    expect(wrapper.findAll('[data-testid="capture-panel"]')).toHaveLength(3);
    expect(wrapper.text()).toContain("BusyBox · eth0");
  });

  it("reports active capture sources without changing stream or artifact semantics", async () => {
    const wrapper = mountWorkspace();
    const panel = wrapper.findComponent({ name: "CapturePanel" });
    panel.vm.$emit("captureChange", {
      id: "capture-1",
      source_type: "interface",
      source_id: "if-1",
      state: "streaming",
      format: "pcap",
      retain: true,
      max_bytes: 1024,
      bytes_written: 128,
      packets: 2,
      truncated: false,
      artifact_url: "/api/v1/artifacts/capture-1",
      created_at: "2026-08-07T00:00:00Z",
    });
    await wrapper.vm.$nextTick();
    expect(wrapper.emitted("captureOverlay")?.at(-1)?.[0]).toEqual({
      connectionIds: [],
      interfaceIds: ["if-1"],
    });
    panel.vm.$emit("captureChange", {
      id: "capture-1",
      state: "completed",
    });
    await wrapper.vm.$nextTick();
    expect(wrapper.emitted("captureOverlay")?.at(-1)?.[0]).toEqual({
      connectionIds: [],
      interfaceIds: [],
    });
  });
});
