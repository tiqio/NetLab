import { flushPromises, mount } from "@vue/test-utils";
import { describe, expect, it, vi } from "vitest";
import { api } from "@/api";
const dispose = vi.fn();
const disconnect = vi.fn();
let terminalOptions: Record<string, unknown> | undefined;
let terminalInstanceOptions: { theme?: Record<string, string> } | undefined;
vi.mock("@xterm/xterm", () => ({
  Terminal: class {
    options: { theme?: Record<string, string> };
    constructor(options: { theme?: Record<string, string> }) {
      this.options = options;
      terminalOptions = options;
      terminalInstanceOptions = this.options;
    }
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
    expect(wrapper.text()).toContain("打开 串口");
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

  it("applies the active page theme to an open terminal without reconnecting", async () => {
    class WebSocketStub {
      static OPEN = 1;
      readyState = WebSocketStub.OPEN;
      binaryType = "";
      onopen?: () => void;
      onmessage?: (event: MessageEvent) => void;
      onclose?: () => void;
      close() {}
      send() {}
    }
    vi.stubGlobal("WebSocket", WebSocketStub);
    document.documentElement.style.setProperty(
      "--terminal-background",
      "#f8fafc",
    );
    document.documentElement.style.setProperty(
      "--terminal-foreground",
      "#17212b",
    );
    vi.spyOn(api, "listNodeConsoles").mockResolvedValue([
      { mode: "telnet", stream_url: "/console", idle_seconds: 60 },
    ]);

    const wrapper = mount(ConsoleWorkspace, {
      props: { nodeId: "node-1", autoOpen: true },
    });
    await flushPromises();
    expect((terminalOptions?.theme as Record<string, string>).background).toBe(
      "#f8fafc",
    );

    document.documentElement.style.setProperty(
      "--terminal-background",
      "#050a0f",
    );
    document.documentElement.dataset.theme = "dark";
    await new Promise((resolve) => setTimeout(resolve, 0));
    expect(terminalInstanceOptions?.theme?.background).toBe("#050a0f");

    wrapper.unmount();
    vi.unstubAllGlobals();
    document.documentElement.style.removeProperty("--terminal-background");
    document.documentElement.style.removeProperty("--terminal-foreground");
  });
});
