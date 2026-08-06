import { flushPromises, mount } from "@vue/test-utils";
import { afterEach, describe, expect, it, vi } from "vitest";
import { api } from "@/api";
import CapturePanel from "./CapturePanel.vue";

afterEach(() => {
  document.body.innerHTML = "";
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

describe("CapturePanel", () => {
  it("disables capture without an interface and shows retained metadata", async () => {
    const wrapper = mount(CapturePanel, { props: { laboratoryId: "lab" } });
    expect(wrapper.findAll("button")[0].attributes("disabled")).toBeDefined();
    await wrapper.setProps({ interfaceId: "if-1" });
    const startCapture = vi.spyOn(api, "startCapture").mockResolvedValue({
      task: {
        id: "task",
        kind: "capture.start",
        resource_type: "capture",
        resource_id: "cap",
        state: "queued",
        progress_current: 0,
        progress_total: 2,
        created_at: "2026-07-27T00:00:00Z",
      },
      capture: {
        id: "cap",
        source_type: "interface",
        source_id: "if-1",
        format: "pcap",
        state: "queued",
        retain: true,
        max_bytes: 100,
        bytes_written: 0,
        packets: 0,
        truncated: false,
        created_at: "2026-07-27T00:00:00Z",
      },
      stream_url: "/stream",
      wireshark: { mode: "pipe", media_type: "application/vnd.tcpdump.pcap" },
    });
    await wrapper.findAll("button")[0].trigger("click");
    await Promise.resolve();
    expect(startCapture).toHaveBeenCalledWith(
      expect.objectContaining({
        source_type: "interface",
        source_id: "if-1",
      }),
    );
    expect(startCapture.mock.calls[0][0]).not.toHaveProperty("interface");
    expect(wrapper.text()).toContain("保留方式");
    expect(wrapper.text()).toContain("cap");
  });

  it("opens an active capture through the local Wireshark helper", async () => {
    vi.spyOn(api, "listCaptures").mockResolvedValue([
      {
        id: "cap-live",
        laboratory_id: "lab",
        source_type: "interface",
        source_id: "if-1",
        format: "pcap",
        state: "running",
        retain: true,
        max_bytes: 1024,
        bytes_written: 24,
        packets: 0,
        truncated: false,
        created_at: "2026-07-30T00:00:00Z",
      },
    ]);
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            status: "ok",
            allowed_origin: window.location.origin,
            wireshark_available: true,
          }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        ),
      )
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ status: "launched" }), {
          status: 202,
          headers: { "Content-Type": "application/json" },
        }),
      );
    vi.stubGlobal("fetch", fetchMock);
    const wrapper = mount(CapturePanel, {
      props: { laboratoryId: "lab", interfaceId: "if-1" },
    });
    await flushPromises();
    await wrapper
      .get('button[title="在本机 Wireshark 中打开实时流"]')
      .trigger("click");
    await flushPromises();
    expect(fetchMock).toHaveBeenCalledTimes(2);
    expect(String(fetchMock.mock.calls[1][1]?.body)).toContain(
      "/api/v1/captures/cap-live/stream",
    );
    expect(wrapper.text()).toContain("已使用 Wireshark 打开实时抓包流。");
  });

  it("selects the newest capture for an interface", async () => {
    vi.spyOn(api, "listCaptures").mockResolvedValue([
      {
        id: "cap-new",
        laboratory_id: "lab",
        source_type: "interface",
        source_id: "if-1",
        format: "pcap",
        state: "running",
        retain: false,
        max_bytes: 1024,
        bytes_written: 24,
        packets: 0,
        truncated: false,
        created_at: "2026-07-30T02:00:00Z",
      },
      {
        id: "cap-old",
        laboratory_id: "lab",
        source_type: "interface",
        source_id: "if-1",
        format: "pcap",
        state: "completed",
        retain: false,
        max_bytes: 1024,
        bytes_written: 24,
        packets: 0,
        truncated: false,
        created_at: "2026-07-30T01:00:00Z",
      },
    ]);
    const wrapper = mount(CapturePanel, {
      props: { laboratoryId: "lab", interfaceId: "if-1" },
    });
    await flushPromises();
    expect(wrapper.text()).toContain("cap-new");
    expect(
      wrapper
        .get('button[title="在本机 Wireshark 中打开实时流"]')
        .attributes("disabled"),
    ).toBeUndefined();
  });

  it("prompts for helper installation when localhost detection fails", async () => {
    vi.spyOn(api, "listCaptures").mockResolvedValue([
      {
        id: "cap-live",
        laboratory_id: "lab",
        source_type: "interface",
        source_id: "if-1",
        format: "pcap",
        state: "running",
        retain: true,
        max_bytes: 1024,
        bytes_written: 24,
        packets: 0,
        truncated: false,
        created_at: "2026-07-30T00:00:00Z",
      },
    ]);
    vi.stubGlobal(
      "fetch",
      vi.fn().mockRejectedValue(new Error("connection refused")),
    );
    const wrapper = mount(CapturePanel, {
      attachTo: document.body,
      props: { laboratoryId: "lab", interfaceId: "if-1" },
    });
    await flushPromises();
    await wrapper
      .get('button[title="在本机 Wireshark 中打开实时流"]')
      .trigger("click");
    await flushPromises();
    expect(document.body.textContent).toContain("需要配置 Wireshark 集成");
    expect(document.body.textContent).toContain("仅安装 Wireshark 还不够");
    expect(document.body.textContent).toContain("Windows 辅助程序");
    expect(document.body.textContent).toContain("安装 Wireshark");
  });
});

it("starts capture for a selected network object link", async () => {
  vi.spyOn(api, "listCaptures").mockResolvedValue([]);
  const startCapture = vi.spyOn(api, "startCapture").mockResolvedValue({
    task: {
      id: "task-object-link",
      kind: "capture.start",
      resource_type: "capture",
      resource_id: "capture-object-link",
      state: "queued",
      progress_current: 0,
      progress_total: 2,
      created_at: "2026-08-03T00:00:00Z",
    },
    capture: {
      id: "capture-object-link",
      laboratory_id: "lab",
      source_type: "network_object_link",
      source_id: "object-link",
      format: "pcap",
      state: "starting",
      retain: true,
      max_bytes: 1024,
      bytes_written: 0,
      packets: 0,
      truncated: false,
      created_at: "2026-08-03T00:00:00Z",
    },
    stream_url: "/api/v1/captures/capture-object-link/stream",
    wireshark: {
      mode: "http_stream",
      media_type: "application/vnd.tcpdump.pcap",
    },
  });
  const wrapper = mount(CapturePanel, {
    props: {
      laboratoryId: "lab",
      objectLinkId: "object-link",
      sourceLabel: "A:swp1 ↔ B:swp1",
    },
  });
  await flushPromises();
  await wrapper.findAll("button")[0].trigger("click");
  await flushPromises();
  expect(startCapture).toHaveBeenCalledWith(
    expect.objectContaining({
      source_type: "network_object_link",
      source_id: "object-link",
    }),
  );
  expect(wrapper.text()).toContain("A:swp1 ↔ B:swp1");
});
