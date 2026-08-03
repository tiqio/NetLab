import { flushPromises, mount } from "@vue/test-utils";
import { describe, expect, it, vi } from "vitest";
import { api } from "@/api";
const dispose = vi.fn();
const disconnect = vi.fn();
vi.mock("@xterm/xterm", () => ({
  Terminal: class {
    loadAddon() {}
    open() {}
    onData() {}
    dispose = dispose;
  },
}));
vi.mock("@xterm/addon-fit", () => ({
  FitAddon: class {
    fit() {}
  },
}));
vi.mock("@novnc/novnc", () => ({
  default: class {
    scaleViewport = false;
    resizeSession = false;
    disconnect = disconnect;
    addEventListener() {}
  },
}));
import ConsoleWorkspace from "./ConsoleWorkspace.vue";
describe("ConsoleWorkspace", () => {
  it("discovers console modes and disposes renderers", async () => {
    vi.spyOn(api, "listNodeConsoles").mockResolvedValue([
      { mode: "telnet", stream_url: "/console", idle_seconds: 60 },
    ]);
    const wrapper = mount(ConsoleWorkspace, { props: { nodeId: "node-1" } });
    await flushPromises();
    expect(wrapper.text()).toContain("Open Serial");
    expect(
      wrapper
        .findAll("button")
        .find((button) => button.text().includes("SSH"))
        ?.attributes("disabled"),
    ).toBeDefined();
    expect(
      wrapper
        .findAll("button")
        .find((button) => button.text().includes("VNC"))
        ?.attributes("disabled"),
    ).toBeDefined();
    wrapper.unmount();
  });

  it("reconnects with the stable browser console session ID", async () => {
    const urls: string[] = [];
    class WebSocketStub {
      static OPEN = 1;
      readyState = WebSocketStub.OPEN;
      binaryType = "";
      onopen?: () => void;
      onmessage?: (event: MessageEvent) => void;
      onclose?: () => void;
      constructor(url: string | URL) {
        urls.push(String(url));
      }
      close() {}
      send() {}
    }
    vi.stubGlobal("WebSocket", WebSocketStub);
    vi.spyOn(api, "listNodeConsoles").mockResolvedValue([
      { mode: "telnet", stream_url: "/console", idle_seconds: 60 },
    ]);

    const wrapper = mount(ConsoleWorkspace, {
      props: {
        nodeId: "node-1",
        sessionId: "stable-session-1",
        autoOpen: true,
      },
    });
    await flushPromises();

    expect(urls).toHaveLength(1);
    expect(new URL(urls[0]).searchParams.get("session_id")).toBe(
      "stable-session-1",
    );
    wrapper.unmount();
    vi.unstubAllGlobals();
  });
});
