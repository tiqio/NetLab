import { mount } from "@vue/test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";
import ThemeSwitcher from "./ThemeSwitcher.vue";
import { mockColorScheme } from "@/test/themeTestUtils";

describe("ThemeSwitcher", () => {
  beforeEach(() => {
    vi.resetModules();
    mockColorScheme(false);
  });

  it("提供中文三态主题选择", () => {
    const wrapper = mount(ThemeSwitcher);
    const select = wrapper.get('select[aria-label="外观主题"]');
    expect(select.findAll("option").map((option) => option.text())).toEqual([
      "跟随系统（当前深色）",
      "浅色",
      "深色",
    ]);
    expect(wrapper.attributes("data-resolved-theme")).toBe("dark");
  });

  it("选择浅色后更新根主题", async () => {
    const wrapper = mount(ThemeSwitcher);
    await wrapper.get("select").setValue("light");
    expect(document.documentElement.dataset.theme).toBe("light");
    expect(wrapper.attributes("data-resolved-theme")).toBe("light");
    expect(wrapper.attributes("title")).toBe("当前生效：浅色主题");
    expect(wrapper.get('span[aria-hidden="true"]').classes()).toContain(
      "bg-primary",
    );
  });
});
