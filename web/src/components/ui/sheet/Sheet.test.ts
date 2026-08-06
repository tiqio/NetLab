import { mount } from "@vue/test-utils";
import { nextTick } from "vue";
import { afterEach, describe, expect, it } from "vitest";
import Sheet from "./Sheet.vue";

afterEach(() => {
  document.body.innerHTML = "";
});

describe("Sheet", () => {
  it("keeps the header and footer outside the independently scrolling body", () => {
    const wrapper = mount(Sheet, {
      attachTo: document.body,
      props: { modelValue: true, title: "Add resource" },
      slots: {
        default: '<div style="height: 1200px">long form</div>',
        footer: "submit actions",
      },
    });

    const dialog = document.body.querySelector<HTMLElement>('[role="dialog"]');
    const body = document.body.querySelector<HTMLElement>("[data-sheet-body]");
    expect(dialog?.className).toContain("h-full");
    expect(dialog?.className).toContain("flex-col");
    expect(body?.className).toContain("overflow-y-auto");
    expect(body?.parentElement).toBe(dialog);
    expect(
      document.body.querySelector("[data-sheet-footer]")?.parentElement,
    ).toBe(dialog);
    wrapper.unmount();
  });

  it.each([
    ["left", "left-0"],
    ["right", "right-0"],
    ["bottom", "bottom-0"],
  ] as const)("preserves the %s side contract", (side, expectedClass) => {
    const wrapper = mount(Sheet, {
      attachTo: document.body,
      props: { modelValue: true, title: "Panel", side },
    });
    expect(
      document.body.querySelector<HTMLElement>('[role="dialog"]')?.className,
    ).toContain(expectedClass);
    wrapper.unmount();
  });

  it("routes overlay, close button, and Escape through close requests", async () => {
    const wrapper = mount(Sheet, {
      attachTo: document.body,
      props: { modelValue: true, title: "Panel", preventClose: true },
    });
    document.body
      .querySelector<HTMLButtonElement>('[aria-label="Close sheet"]')!
      .click();
    document.body
      .querySelector<HTMLElement>("[data-sheet-overlay]")!
      .dispatchEvent(new MouseEvent("click", { bubbles: true }));
    window.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape" }));
    await nextTick();

    expect(
      wrapper.emitted("closeRequested")?.map(([reason]) => reason),
    ).toEqual(["button", "overlay"]);
    expect(wrapper.emitted("update:modelValue")).toBeUndefined();
    wrapper.unmount();
  });

  it("moves focus into the sheet and restores the trigger after closing", async () => {
    const trigger = document.createElement("button");
    trigger.textContent = "open";
    document.body.append(trigger);
    trigger.focus();
    const wrapper = mount(Sheet, {
      attachTo: document.body,
      props: { modelValue: true, title: "Panel" },
      slots: { default: "<button autofocus data-first-action>first</button>" },
    });
    await nextTick();
    expect(document.activeElement).toBe(
      document.body.querySelector("[data-first-action]"),
    );

    await wrapper.setProps({ modelValue: false });
    await nextTick();
    expect(document.activeElement).toBe(trigger);
    wrapper.unmount();
  });

  it("contains dirty close confirmation and supports keep editing or discard", async () => {
    const wrapper = mount(Sheet, {
      attachTo: document.body,
      props: { modelValue: true, title: "Panel", preventClose: true },
      slots: { default: "<input aria-label='Draft name' />" },
    });
    document.body
      .querySelector<HTMLButtonElement>('[aria-label="Close sheet"]')!
      .click();
    await nextTick();

    const alert = document.body.querySelector<HTMLElement>(
      '[role="alertdialog"]',
    );
    expect(alert).not.toBeNull();
    expect(document.activeElement).toBe(
      alert?.querySelector("[data-keep-editing]"),
    );

    alert?.querySelector<HTMLButtonElement>("[data-keep-editing]")?.click();
    await nextTick();
    expect(document.body.querySelector('[role="alertdialog"]')).toBeNull();
    expect(wrapper.props("modelValue")).toBe(true);

    window.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape" }));
    await nextTick();
    document.body
      .querySelector<HTMLButtonElement>("[data-discard-changes]")!
      .click();
    await nextTick();
    expect(wrapper.emitted("update:modelValue")?.at(-1)).toEqual([false]);
    wrapper.unmount();
  });

  it("contains keyboard focus inside the discard alertdialog", async () => {
    const wrapper = mount(Sheet, {
      attachTo: document.body,
      props: { modelValue: true, title: "Panel", preventClose: true },
      slots: { default: "<button data-background>background</button>" },
    });
    document.body
      .querySelector<HTMLButtonElement>('[aria-label="Close sheet"]')!
      .click();
    await nextTick();
    const keep = document.body.querySelector<HTMLButtonElement>(
      "[data-keep-editing]",
    )!;
    const discard = document.body.querySelector<HTMLButtonElement>(
      "[data-discard-changes]",
    )!;
    discard.focus();
    window.dispatchEvent(new KeyboardEvent("keydown", { key: "Tab" }));
    expect(document.activeElement).toBe(keep);
    keep.focus();
    window.dispatchEvent(
      new KeyboardEvent("keydown", { key: "Tab", shiftKey: true }),
    );
    expect(document.activeElement).toBe(discard);
    window.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape" }));
    await nextTick();
    expect(document.body.querySelector('[role="alertdialog"]')).toBeNull();
    wrapper.unmount();
  });
});
