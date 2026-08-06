import { test, expect } from "../fixtures/acceptanceFixture";
import { activateByKeyboard } from "../fixtures/inputAccessibility";

test("primary shell controls are keyboard reachable at the configured viewport", async ({
  page,
}) => {
  await page.goto("/");
  await page.evaluate(() => {
    document.documentElement.style.zoom = "1.25";
  });
  const addResource = page.getByRole("button", { name: "添加资源" });
  await activateByKeyboard(page, addResource);
  await expect(page.getByRole("dialog", { name: "添加资源" })).toBeVisible();
  await page.getByRole("button", { name: "关闭抽屉" }).click();
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
