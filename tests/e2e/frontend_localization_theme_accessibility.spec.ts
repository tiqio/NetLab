import { expect, test } from "./fixtures/acceptanceFixture";

for (const viewport of [
  { width: 1024, height: 768 },
  { width: 1366, height: 768 },
  { width: 1920, height: 1080 },
]) {
  for (const theme of ["light", "dark"] as const) {
    test(`${theme} 主题在 ${viewport.width}x${viewport.height} 下可键盘操作`, async ({
      page,
    }) => {
      await page.setViewportSize(viewport);
      await page.goto("/");
      const selector = page.getByRole("combobox", { name: "外观主题" });
      await selector.selectOption(theme);
      await selector.focus();
      await expect(selector).toBeFocused();
      await expect(page.locator("html")).toHaveAttribute("data-theme", theme);
      await expect(page.locator("body")).not.toHaveCSS("overflow-x", "scroll");
    });
  }
}
