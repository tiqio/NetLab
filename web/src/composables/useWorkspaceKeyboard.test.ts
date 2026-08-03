import { mount } from "@vue/test-utils";
import { defineComponent, ref } from "vue";
import { describe, expect, it, vi } from "vitest";
import { useWorkspaceKeyboard } from "./useWorkspaceKeyboard";
describe("workspace keyboard", () => {
  it("opens commands and navigates without hijacking inputs", async () => {
    const openCommands = vi.fn();
    const selectNext = vi.fn();
    const component = defineComponent({
      setup() {
        useWorkspaceKeyboard(ref(true), {
          openCommands,
          clearSelection: vi.fn(),
          selectNext,
          openInspector: vi.fn(),
          openTasks: vi.fn(),
        });
        return {};
      },
      template: '<input aria-label="editor" />',
    });
    const wrapper = mount(component);
    window.dispatchEvent(
      new KeyboardEvent("keydown", { key: "k", ctrlKey: true }),
    );
    window.dispatchEvent(new KeyboardEvent("keydown", { key: "ArrowRight" }));
    expect(openCommands).toHaveBeenCalled();
    expect(selectNext).toHaveBeenCalledWith(1);
    await wrapper.get("input").trigger("keydown", { key: "ArrowRight" });
    wrapper.unmount();
  });
});
