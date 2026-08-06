import axe from "axe-core";
import { flushPromises, mount } from "@vue/test-utils";
import { afterEach, describe, expect, it, vi } from "vitest";
import { api } from "@/api";
import CreateTopologyResourceDrawer from "./CreateTopologyResourceDrawer.vue";

vi.mock("@/api", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/api")>()),
  api: {
    listTemplates: vi.fn().mockResolvedValue([]),
    listImages: vi.fn().mockResolvedValue([]),
    createNode: vi.fn(),
    createNetworkObject: vi.fn(),
  },
}));

afterEach(() => {
  document.body.innerHTML = "";
});

describe("CreateTopologyResourceDrawer accessibility", () => {
  it("has no serious axe violations in the form or discard confirmation", async () => {
    vi.mocked(api.listTemplates).mockResolvedValue([]);
    const wrapper = mount(CreateTopologyResourceDrawer, {
      attachTo: document.body,
      props: {
        modelValue: true,
        laboratoryId: "lab-1",
        selection: { kind: "pc", name: "PC", networkObjectKind: "pc" },
      },
    });
    await flushPromises();
    const dialog = document.body.querySelector<HTMLElement>('[role="dialog"]')!;
    const initial = await axe.run(dialog, {
      rules: { "color-contrast": { enabled: false } },
    });
    expect(
      initial.violations.filter((item) =>
        ["serious", "critical"].includes(item.impact || ""),
      ),
    ).toEqual([]);

    const name = dialog.querySelector<HTMLInputElement>(
      '[data-testid="create-resource-name"]',
    )!;
    name.value = "Changed PC";
    name.dispatchEvent(new Event("input", { bubbles: true }));
    await flushPromises();
    dialog.querySelector<HTMLButtonElement>('[aria-label="关闭抽屉"]')!.click();
    await flushPromises();
    const confirmation = document.body.querySelector<HTMLElement>(
      '[role="alertdialog"]',
    )!;
    const confirmed = await axe.run(confirmation, {
      rules: { "color-contrast": { enabled: false } },
    });
    expect(
      confirmed.violations.filter((item) =>
        ["serious", "critical"].includes(item.impact || ""),
      ),
    ).toEqual([]);
    wrapper.unmount();
  });
});
