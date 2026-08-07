import { flushPromises, mount } from "@vue/test-utils";
import { createPinia } from "pinia";
import { afterEach, describe, expect, it, vi } from "vitest";
import { api } from "@/api";
import { taskFactory } from "@/test/factories";
import GuestCommandPanel from "./GuestCommandPanel.vue";

describe("GuestCommandPanel", () => {
  afterEach(() => vi.restoreAllMocks());

  it("executes a shell expression and renders decoded output", async () => {
    const task = taskFactory({
      kind: "node.guest_exec",
      state: "succeeded",
      progress_current: 2,
      progress_total: 2,
      result: {
        exit_code: 0,
        stdout_base64: btoa("Linux guest\n"),
        stderr_base64: btoa("warning\n"),
        truncated: false,
      },
    });
    const execute = vi.spyOn(api, "executeGuestCommand").mockResolvedValue(task);
    const wrapper = mount(GuestCommandPanel, {
      props: { nodeId: "node-1" },
      global: { plugins: [createPinia()] },
    });

    await wrapper.find("textarea").setValue("uname -a | head -c 20");
    await wrapper.find("form").trigger("submit");
    await flushPromises();

    expect(execute).toHaveBeenCalledWith("node-1", {
      argv: ["/bin/sh", "-lc", "uname -a | head -c 20"],
      timeout_seconds: 30,
      output_limit: 1 << 20,
    });
    expect(wrapper.get('[data-testid="guest-command-result"]').text()).toContain(
      "Linux guest",
    );
    expect(wrapper.text()).toContain("warning");
    expect(wrapper.text()).toContain("退出码 0");
  });
});
