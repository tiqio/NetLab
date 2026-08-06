import { beforeEach, describe, expect, it, vi } from "vitest";
import { mockColorScheme } from "@/test/themeTestUtils";

describe("theme bootstrap", () => {
  beforeEach(() => vi.resetModules());

  it("在挂载前应用持久化主题", async () => {
    mockColorScheme(false);
    localStorage.setItem("netlab.appearance.v1", "light");
    const { bootstrapTheme } = await import("./themeBootstrap");
    bootstrapTheme();
    expect(document.documentElement.dataset.theme).toBe("light");
  });

  it("无偏好时跟随系统", async () => {
    mockColorScheme(true);
    const { resolveInitialTheme } = await import("./themeBootstrap");
    expect(resolveInitialTheme()).toEqual({
      preference: "system",
      resolved: "light",
    });
  });

  it("matchMedia 不可用时回退深色", async () => {
    vi.stubGlobal("matchMedia", undefined);
    const { resolveInitialTheme } = await import("./themeBootstrap");
    expect(resolveInitialTheme()).toEqual({
      preference: "system",
      resolved: "dark",
    });
  });

  it("本地存储不可用时不阻塞首屏并回退系统偏好", async () => {
    mockColorScheme(false);
    vi.spyOn(Storage.prototype, "getItem").mockImplementation(() => {
      throw new Error("storage unavailable");
    });
    const { bootstrapTheme } = await import("./themeBootstrap");
    expect(() => bootstrapTheme()).not.toThrow();
    expect(document.documentElement.dataset.theme).toBe("dark");
  });
});
