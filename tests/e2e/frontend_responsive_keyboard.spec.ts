import { expect, test } from "./fixtures/acceptanceFixture";
import { waitForCondition } from "./fixtures/waiters";
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
  await expect(page.getByRole("dialog", { name: "命令面板" })).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(page.getByRole("dialog", { name: "命令面板" })).toBeHidden();
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
  interactionResults,
}, testInfo) => {
  const { laboratory } = await createOwnedLaboratory(
    page,
    automation,
    ledger,
    runId,
  );
  const topology = new TopologyPage(page, automation);
  const trigger = page.getByRole("button", { name: "添加资源" });
  await trigger.focus();
  await page.keyboard.press("Enter");
  await expect(page.getByRole("dialog", { name: "添加资源" })).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(page.getByRole("dialog", { name: "添加资源" })).toBeHidden();
  await expect(trigger).toBeFocused();
  interactionResults.push(
    result(
      "topology.add-drawer.open",
      testInfo.project.use.viewport!,
      "keyboard activation opened the drawer and Escape restored trigger focus",
      [],
      "keyboard",
    ),
  );

  await topology.openResourceDrawer();
  const form = await topology.chooseDrawerResource("PC");
  await form.getByLabel("名称", { exact: true }).fill("keyboard draft");
  await page.keyboard.press("Escape");
  const confirmation = page.getByRole("alertdialog", {
    name: "放弃未保存的更改",
  });
  await expect(confirmation).toBeVisible();
  await confirmation.getByRole("button", { name: "继续编辑" }).press("Enter");
  await expect(form.getByLabel("名称", { exact: true })).toHaveValue(
    "keyboard draft",
  );
  await page.keyboard.press("Escape");
  await confirmation.getByRole("button", { name: "放弃更改" }).press("Enter");
  await expect(form).toBeHidden();
  interactionResults.push(
    result(
      "topology.add-drawer.discard",
      testInfo.project.use.viewport!,
      "keyboard activation discarded the browser-local dirty draft",
      [],
      "keyboard",
    ),
  );

  await trigger.focus();
  await page.keyboard.press("Enter");
  const submission = await topology.chooseDrawerResource("PC");
  const resourceName = `keyboard-${runId.slice(0, 6)}`;
  await submission.getByLabel("名称", { exact: true }).fill(resourceName);
  await submission.getByRole("button", { name: "添加到拓扑" }).press("Enter");
  const resource = await waitForCondition(
    async () => {
      const response = await automation.get(`/api/v1/labs/${laboratory.id}`);
      const snapshot = await response.json();
      return snapshot.network_objects.find(
        (item: { name: string }) => item.name === resourceName,
      );
    },
    (value): value is { id: string; revision: number } => Boolean(value),
    "keyboard-created PC from add drawer",
    30_000,
  );
  await ledger.add({
    resource_type: "network_object",
    resource_id: resource.id,
    laboratory_id: laboratory.id,
    revision: resource.revision,
    cleanup_method: "laboratory-cascade",
  });
  interactionResults.push(
    result(
      "topology.add-drawer.submit",
      testInfo.project.use.viewport!,
      "keyboard activation submitted an authoritative lightweight resource",
      [resource.id],
      "keyboard",
    ),
  );
});
