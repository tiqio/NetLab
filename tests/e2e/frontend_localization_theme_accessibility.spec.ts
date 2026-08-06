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
      await page.goto("/");
      const selector = page.getByRole("combobox", { name: "外观主题" });
      await selector.selectOption(theme);
      await selector.focus();
      await expect(selector).toBeFocused();
      await expect(page.locator("html")).toHaveAttribute("data-theme", theme);
      await expect(page.locator("body")).not.toHaveCSS("overflow-x", "scroll");
      const rootOverflow = await page.evaluate(
        () =>
          document.documentElement.scrollWidth -
          document.documentElement.clientWidth,
      );
      expect(rootOverflow).toBeLessThanOrEqual(1);
      await selector.press("Shift+Tab");
      const activeElement = await page.evaluate(() => ({
        tag: document.activeElement?.tagName,
        label: document.activeElement?.getAttribute("aria-label"),
      }));
      expect(activeElement.tag).not.toBe("BODY");
      expect(activeElement.label).not.toBe("外观主题");
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
