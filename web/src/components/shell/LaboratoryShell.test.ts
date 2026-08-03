import { mount } from "@vue/test-utils";
import { describe, expect, it } from "vitest";
import LaboratoryShell from "./LaboratoryShell.vue";
import { defaultWorkspacePreferences } from "@/composables/useWorkspacePreferences";
import {
  CANVAS_MIN_HEIGHT,
  RESIZE_HANDLE_HEIGHT,
} from "./laboratoryShellSizing";
describe("LaboratoryShell", () => {
  it("renders five regions and emits collapse changes", async () => {
    const wrapper = mount(LaboratoryShell, {
      props: { preferences: defaultWorkspacePreferences("lab") },
      slots: {
        toolbar: "toolbar",
        palette: "palette",
        canvas: "canvas",
        inspector: "inspector",
        bottom: "bottom",
      },
    });
    expect(wrapper.text()).toContain("toolbar");
    expect(wrapper.text()).toContain("palette");
    expect(wrapper.text()).toContain("inspector");
    await wrapper.get('[aria-label="Toggle inspector"]').trigger("click");
    expect(wrapper.emitted("panel")?.[0]).toEqual([
      "inspector",
      { collapsed: true },
    ]);
  });

  it("clamps upward drawer resizing so the canvas and handle stay visible", async () => {
    const wrapper = mount(LaboratoryShell, {
      props: { preferences: defaultWorkspacePreferences("lab") },
      slots: { canvas: "canvas", bottom: "bottom" },
    });
    const workspaceColumn = wrapper
      .findAll("section")
      .find((section) => section.classes().includes("flex-col"));
    Object.defineProperty(workspaceColumn!.element, "clientHeight", {
      configurable: true,
      value: 600,
    });
    await wrapper
      .get('[role="separator"][aria-orientation="horizontal"]')
      .trigger("pointerdown", { clientY: 500 });
    window.dispatchEvent(new MouseEvent("pointermove", { clientY: 0 }));
    window.dispatchEvent(new MouseEvent("pointerup"));

    expect(wrapper.emitted("panel")?.at(-1)).toEqual([
      "bottomDrawer",
      { size: 600 - CANVAS_MIN_HEIGHT - RESIZE_HANDLE_HEIGHT },
    ]);
    expect(
      wrapper
        .get('[role="separator"][aria-orientation="horizontal"]')
        .classes(),
    ).toEqual(expect.arrayContaining(["relative", "z-30"]));
    expect(workspaceColumn!.element.firstElementChild?.classList).toContain(
      "overflow-hidden",
    );
  });
});
