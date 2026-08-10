import { expect, test } from "./fixtures/acceptanceFixture";
import { expectNoSeriousAxeViolations } from "./fixtures/axe";

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
      const selector = page.getByRole("combobox", { name: "外观主题" });
      for (let attempt = 0; attempt < 3; attempt += 1) {
        await page.goto("/", { waitUntil: "domcontentloaded" });
        if (await selector.isVisible({ timeout: 5_000 }).catch(() => false))
          break;
      }
      await expect(selector).toBeVisible({ timeout: 10_000 });
      await selector.selectOption(theme);
      await expect(page.locator("html")).toHaveAttribute("data-theme", theme);
      await expect(page.locator("body")).not.toHaveCSS("overflow-x", "scroll");
      const rootOverflow = await page.evaluate(
        () =>
          document.documentElement.scrollWidth -
          document.documentElement.clientWidth,
      );
      expect(rootOverflow).toBeLessThanOrEqual(1);
      await selector.press("Shift+Tab");
      if (
        (await page.evaluate(() => document.activeElement?.tagName)) === "BODY"
      )
        await page.keyboard.press("Tab");
      await expect(page.locator(":focus")).toBeVisible();
      await expect(selector).not.toBeFocused();
      await expectNoSeriousAxeViolations(page);

      for (const route of ["/templates", "/automation"]) {
        await page.goto(route);
        await expect(page.locator("html")).toHaveAttribute("data-theme", theme);
        await expectNoSeriousAxeViolations(page);
        const overflow = await page.evaluate(
          () =>
            document.documentElement.scrollWidth -
            document.documentElement.clientWidth,
        );
        expect(overflow).toBeLessThanOrEqual(1);
      }
    });
  }
}
