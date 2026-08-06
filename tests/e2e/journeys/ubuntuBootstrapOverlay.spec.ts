import { expect, test } from "../fixtures/acceptanceFixture";
import {
  createOwnedLaboratory,
  resolveTemplateSelection,
} from "./completeRealJourney";
import { TemplatePage } from "../pages/TemplatePage";

test("Ubuntu QEMU receives cloud-init credentials and stable interface overlays", async ({
  page,
  automation,
  environment,
  ledger,
  runId,
}) => {
  const ubuntuAvailable = environment.templates.some(
    (template) =>
      template.device_family === "ubuntu-qemu" &&
      template.versions.some((version) => version.available),
  );
  test.skip(!ubuntuAvailable, "requires an available Ubuntu QEMU image");
  const { laboratory } = await createOwnedLaboratory(
    page,
    automation,
    ledger,
    runId,
  );
  const selection = await resolveTemplateSelection(automation, "ubuntu-qemu");
  const templates = new TemplatePage(page, automation);
  const dialog = await templates.chooseDevice(
    selection.displayName,
    selection.runtime,
  );
  const initialPassword = dialog.getByLabel("初始密码");
  await expect(initialPassword).toHaveValue(/.{12,}/);
  const nodeName = `ubuntu-bootstrap-${crypto.randomUUID().slice(0, 6)}`;
  await dialog.getByLabel("名称", { exact: true }).fill(nodeName);
  await dialog.getByLabel("设备模板").selectOption(selection.templateId);
  await dialog.getByLabel("模板版本").selectOption(selection.versionId);
  if (selection.imageId) {
    await dialog.getByLabel("镜像版本").selectOption(selection.imageId);
  }
  await dialog.getByRole("button", { name: "添加到拓扑" }).click();
  await expect(dialog).toBeHidden();

  const snapshot = await templates.snapshot(laboratory.id);
  const node = snapshot.nodes.find((item) => item.name === nodeName);
  expect(node).toBeTruthy();
  expect(node?.config).toEqual(
    expect.objectContaining({ seed_iso: expect.stringMatching(/seed\.iso$/) }),
  );
  await ledger.add({
    resource_type: "node",
    resource_id: node!.id,
    laboratory_id: laboratory.id,
    revision: node!.revision,
    cleanup_method: "laboratory-cascade",
  });

  const nodeInterface = snapshot.interfaces.find(
    (item) => item.node_id === node!.id,
  );
  expect(nodeInterface).toBeTruthy();
  const canvas = page.getByLabel(/拓扑画布键盘操作区/);
  await canvas.focus();
  await canvas.press("ArrowRight");
  const overlay = page.locator(`[data-interface-id="${nodeInterface!.id}"]`);
  await expect(overlay).toBeVisible();
  const before = await overlay.boundingBox();
  const chart = page.getByRole("img", { name: /拓扑画布/ });
  const chartBox = await chart.boundingBox();
  expect(before).toBeTruthy();
  expect(chartBox).toBeTruthy();
  await page.mouse.move(chartBox!.x + 5, chartBox!.y + 5);
  await page.waitForTimeout(100);
  const after = await overlay.boundingBox();
  expect(after).toBeTruthy();
  expect(Math.abs(after!.x - before!.x)).toBeLessThanOrEqual(2);
  expect(Math.abs(after!.y - before!.y)).toBeLessThanOrEqual(2);
});
