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
    expect(resolveInitialTheme()).toEqual({ preference: "system", resolved: "light" });
  });
});

