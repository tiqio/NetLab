import { expect, test } from "./fixtures/acceptanceFixture";
import { result } from "./journeys/completeRealJourney";
import { createOwnedLaboratory } from "./journeys/completeRealJourney";
import { TopologyPage } from "./pages/TopologyPage";

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

test("topology add drawer is keyboard operable and restores trigger focus", async ({
  page,
  automation,
  ledger,
  runId,
}) => {
  await createOwnedLaboratory(page, automation, ledger, runId);
  const topology = new TopologyPage(page, automation);
  const trigger = page.getByRole("button", { name: "添加资源" });
  await trigger.focus();
  await page.keyboard.press("Enter");
  await expect(
    page.getByRole("dialog", { name: "Add resource" }),
  ).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(page.getByRole("dialog", { name: "Add resource" })).toBeHidden();
  await expect(trigger).toBeFocused();

  await topology.openResourceDrawer();
  const form = await topology.chooseDrawerResource("PC");
  await form.getByLabel("Name", { exact: true }).fill("keyboard draft");
  await page.keyboard.press("Escape");
  const confirmation = page.getByRole("alertdialog", {
    name: "放弃未保存的更改",
  });
  await expect(confirmation).toBeVisible();
  await confirmation.getByRole("button", { name: "继续编辑" }).press("Enter");
  await expect(form.getByLabel("Name", { exact: true })).toHaveValue(
    "keyboard draft",
  );
});
