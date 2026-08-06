import { test, expect } from "../fixtures/acceptanceFixture";
import { activateByKeyboard } from "../fixtures/inputAccessibility";

test("primary shell controls are keyboard reachable at the configured viewport", async ({
  page,
}) => {
  await page.goto("/");
  await page.evaluate(() => {
    document.documentElement.style.zoom = "1.25";
  });
  const switcher = page.getByTestId("laboratory-switcher");
  if ((await switcher.getAttribute("aria-expanded")) !== "true")
    await activateByKeyboard(page, switcher);
  const newButton = page.getByTestId("new-laboratory");
  await activateByKeyboard(page, newButton);
  await expect(page.getByRole("dialog", { name: "创建实验室" })).toBeVisible();
  await page.getByRole("button", { name: "关闭对话框" }).click();
  const commands = page.getByRole("button", { name: "⌘K" });
  await activateByKeyboard(page, commands);
  await expect(page.getByRole("dialog", { name: "命令面板" })).toBeVisible();
  await page.keyboard.press("Escape");
  const overflow = await page.evaluate(
    () =>
      document.documentElement.scrollWidth -
      document.documentElement.clientWidth,
  );
  expect(overflow).toBeLessThanOrEqual(1);
});
