import { expect, test } from "../fixtures/acceptanceFixture";
import {
  createOwnedLaboratory,
  resolveTemplateSelection,
} from "./completeRealJourney";
import { TemplatePage } from "../pages/TemplatePage";

test("Ubuntu QEMU receives cloud-init credentials and stable interface overlays", async ({
  page,
  automation,
  ledger,
  runId,
}) => {
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
  const initialPassword = dialog.getByLabel("Initial password");
  await expect(initialPassword).toHaveValue(/.{12,}/);
  const nodeName = `ubuntu-bootstrap-${crypto.randomUUID().slice(0, 6)}`;
  await dialog.getByLabel("Name", { exact: true }).fill(nodeName);
  await dialog.getByLabel("Device template").selectOption(selection.templateId);
  await dialog.getByLabel("Template version").selectOption(selection.versionId);
  if (selection.imageId) {
    await dialog.getByLabel("Image version").selectOption(selection.imageId);
  }
  await dialog.getByRole("button", { name: "Add to topology" }).click();
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
  const canvas = page.getByLabel(/Topology canvas keyboard area/);
  await canvas.focus();
  await canvas.press("ArrowRight");
  const overlay = page.locator(`[data-interface-id="${nodeInterface!.id}"]`);
  await expect(overlay).toBeVisible();
  const before = await overlay.boundingBox();
  const chart = page.getByRole("img", { name: /Topology canvas/ });
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
