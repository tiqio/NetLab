import { test, expect } from "../fixtures/acceptanceFixture";
import { activateByKeyboard } from "../fixtures/inputAccessibility";

test("primary shell controls are keyboard reachable at the configured viewport", async ({
  page,
}) => {
  await page.goto("/");
  await activateByKeyboard(page, page.getByTestId("laboratory-switcher"));
  const newButton = page.getByTestId("new-laboratory");
  await activateByKeyboard(page, newButton);
  await expect(
    page.getByRole("dialog", { name: "Create laboratory" }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Close dialog" }).click();
  const commands = page.getByRole("button", { name: "⌘K" });
  await activateByKeyboard(page, commands);
  await expect(
    page.getByRole("dialog", { name: "Command palette" }),
  ).toBeVisible();
  await page.keyboard.press("Escape");
});
