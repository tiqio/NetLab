import { expect, test } from "./fixtures/acceptanceFixture";
import { result } from "./journeys/completeRealJourney";

test("command palette has keyboard-equivalent visible outcomes", async ({
  page,
  interactionResults,
}, testInfo) => {
  await page.goto("/");
  const trigger = page.getByRole("button", { name: "⌘K" });
  await trigger.focus();
  await page.keyboard.press("Enter");
  await expect(
    page.getByRole("dialog", { name: "Command palette" }),
  ).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(
    page.getByRole("dialog", { name: "Command palette" }),
  ).toBeHidden();
  interactionResults.push(
    result(
      "workspace.commands",
      testInfo.project.use.viewport!,
      "keyboard activation opened and Escape closed the palette",
      [],
      "keyboard",
    ),
  );
});
