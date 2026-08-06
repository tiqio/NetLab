import { mount } from "@vue/test-utils";
import { defineComponent, nextTick } from "vue";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { mockColorScheme } from "@/test/themeTestUtils";

async function loadComposable() {
  vi.resetModules();
  return import("./useThemePreference");
}

describe("useThemePreference", () => {
  beforeEach(() => {
    document.documentElement.removeAttribute("data-theme");
  });

  it("首次访问跟随系统浅色偏好", async () => {
    mockColorScheme(true);
    const { useThemePreference } = await loadComposable();
    const wrapper = mount(
      defineComponent({
        setup: () => useThemePreference(),
        template: "<div />",
      }),
    );
    expect(document.documentElement.dataset.theme).toBe("light");
    wrapper.unmount();
  });

  it("无明确系统浅色偏好时回退深色", async () => {
    mockColorScheme(false);
    const { useThemePreference } = await loadComposable();
    const wrapper = mount(
      defineComponent({
        setup: () => useThemePreference(),
        template: "<div />",
      }),
    );
    expect(document.documentElement.dataset.theme).toBe("dark");
    wrapper.unmount();
  });

  it("保存用户选择并立即应用", async () => {
    mockColorScheme(false);
    const { useThemePreference, THEME_STORAGE_KEY } = await loadComposable();
    let theme: ReturnType<typeof useThemePreference> | undefined;
    const wrapper = mount(
      defineComponent({
        setup() {
          theme = useThemePreference();
          return {};
        },
        template: "<div />",
      }),
    );
    theme?.setPreference("light");
    await nextTick();
    expect(localStorage.getItem(THEME_STORAGE_KEY)).toBe("light");
    expect(document.documentElement.dataset.theme).toBe("light");
    wrapper.unmount();
  });
});
