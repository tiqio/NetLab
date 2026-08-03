import { test, expect } from "../fixtures/acceptanceFixture";
import { LaboratoryPage } from "../pages/LaboratoryPage";

test("laboratory forms protect required and recoverable input", async ({
  page,
  automation,
  ledger,
  runId,
}) => {
  const laboratories = new LaboratoryPage(page, automation);
  await laboratories.open();
  await laboratories.openCreateDialog();
  const createDialog = page.getByRole("dialog", {
    name: "Create laboratory",
  });
  const name = createDialog.getByLabel("Name");
  await createDialog.getByRole("button", { name: "Create laboratory" }).click();
  await expect(name).toHaveAttribute("required", "");
  await expect(createDialog).toBeVisible();
  await page.getByRole("button", { name: "Close dialog" }).click();

  const laboratory = await laboratories.create(`form-${runId.slice(0, 6)}`);
  await ledger.add({
    resource_type: "laboratory",
    resource_id: laboratory.id,
    revision: laboratory.revision,
    cleanup_method: "frontend-delete-with-api-fallback",
  });
  await laboratories.openActiveActions();
  await page.getByRole("menuitem", { name: "Rename" }).click();
  const rename = page.getByRole("dialog", { name: "Rename laboratory" });
  await rename.getByLabel("Name").fill(`${laboratory.name}-changed`);
  await rename.getByRole("button", { name: "Close dialog" }).click();
  const discard = page.getByRole("alertdialog", {
    name: "Discard unsaved changes",
  });
  await expect(discard).toBeVisible();
  await discard.getByRole("button", { name: "Keep editing" }).click();
  await expect(rename.getByLabel("Name")).toHaveValue(
    `${laboratory.name}-changed`,
  );
  await rename.getByRole("button", { name: "Cancel" }).click();
  await laboratories.openActiveActions();
  await page.getByRole("menuitem", { name: "Delete" }).click();
  const deletion = page.getByRole("dialog", { name: "Delete laboratory" });
  await deletion.getByRole("button", { name: "Cancel" }).click();
  await expect(deletion).toBeHidden();
});
