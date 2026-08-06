import { test, expect } from "../fixtures/acceptanceFixture";
import { LaboratoryPage } from "../pages/LaboratoryPage";
import { TopologyPage } from "../pages/TopologyPage";

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
    name: "创建实验室",
  });
  const name = createDialog.getByLabel("名称");
  await createDialog.getByRole("button", { name: "创建实验室" }).click();
  await expect(name).toHaveAttribute("required", "");
  await expect(createDialog).toBeVisible();
  await page.getByRole("button", { name: "关闭对话框" }).click();

  const laboratory = await laboratories.create(`form-${runId.slice(0, 6)}`);
  await ledger.add({
    resource_type: "laboratory",
    resource_id: laboratory.id,
    revision: laboratory.revision,
    cleanup_method: "frontend-delete-with-api-fallback",
  });
  await laboratories.openActiveActions();
  await page.getByRole("menuitem", { name: "重命名" }).click();
  const rename = page.getByRole("dialog", { name: "重命名实验室" });
  await rename.getByLabel("名称").fill(`${laboratory.name}-changed`);
  await rename.getByRole("button", { name: "关闭对话框" }).click();
  const discard = page.getByRole("alertdialog", {
    name: "放弃未保存的更改",
  });
  await expect(discard).toBeVisible();
  await discard.getByRole("button", { name: "继续编辑" }).click();
  await expect(rename.getByLabel("名称")).toHaveValue(
    `${laboratory.name}-changed`,
  );
  await rename.getByRole("button", { name: "取消" }).click();
  await laboratories.openActiveActions();
  await page.getByRole("menuitem", { name: "删除" }).click();
  const deletion = page.getByRole("dialog", { name: "删除实验室" });
  await deletion.getByRole("button", { name: "取消" }).click();
  await expect(deletion).toBeHidden();

  const topology = new TopologyPage(page, automation);
  await topology.openResourceDrawer();
  const resourceForm = await topology.chooseDrawerResource("PC");
  await resourceForm.getByLabel("名称", { exact: true }).fill("drawer draft");
  await resourceForm.getByRole("button", { name: "取消" }).click();
  const resourceDiscard = page.getByRole("alertdialog", {
    name: "放弃未保存的更改",
  });
  await expect(resourceDiscard).toBeVisible();
  await resourceDiscard.getByRole("button", { name: "继续编辑" }).click();
  await expect(resourceForm.getByLabel("名称", { exact: true })).toHaveValue(
    "drawer draft",
  );
});
