import { expect, test } from "../fixtures/acceptanceFixture";
import type { Locator, Page } from "@playwright/test";
import { waitForCondition } from "../fixtures/waiters";
import { selectLaboratoryByName } from "../pages/LaboratoryPage";
import { TemplatePage } from "../pages/TemplatePage";
import { createOwnedLaboratory, result } from "./completeRealJourney";

test("three lightweight objects form a shared browser-created path", async ({
  page,
  secondPage,
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
  const templates = new TemplatePage(page, automation);
  const names = ["edge-a", "transit-b", "edge-c"].map(
    (name) => `${name}-${runId.slice(0, 5)}`,
  );
  const first = await templates.createLightweight(
    laboratory.id,
    "Layer-2 switch",
    names[0],
  );
  const middleDialog = await templates.chooseLightweight("Layer-2 switch");
  await middleDialog.getByLabel("名称", { exact: true }).fill(names[1]);
  await middleDialog.getByRole("button", { name: "添加二层端口" }).click();
  await middleDialog.getByRole("button", { name: "添加到拓扑" }).click();
  const middle = await waitForCondition(
    async () =>
      (await templates.snapshot(laboratory.id)).network_objects.find(
        (item) => item.name === names[1],
      ),
    (
      item,
    ): item is Record<string, unknown> & { id: string; revision: number } =>
      Boolean(item),
    "middle lightweight switch",
    30_000,
  );
  await expect(middleDialog).toBeHidden();
  const third = await templates.createLightweight(
    laboratory.id,
    "Layer-2 switch",
    names[2],
  );
  for (const object of [first, middle, third]) {
    await ledger.add({
      resource_type: "network_object",
      resource_id: object.id,
      laboratory_id: laboratory.id,
      revision: object.revision,
      cleanup_method: "laboratory-cascade",
    });
  }

  await secondPage.goto("/");
  await selectLaboratoryByName(secondPage, laboratory.name);

  const canvas = page.getByLabel(/拓扑画布键盘操作区/);
  await createKeyboardObjectLink(
    page,
    canvas,
    templates,
    laboratory.id,
    first.id,
    "eth0",
    middle.id,
    "eth0",
    1,
  );
  await expect(page.getByTestId("topology-a11y-summary")).toContainText(
    undirectedLinkPattern(names[0], "eth0", names[1], "eth0"),
  );

  const links = await createKeyboardObjectLink(
    page,
    canvas,
    templates,
    laboratory.id,
    middle.id,
    "eth1",
    third.id,
    "eth0",
    2,
  );
  for (const link of links) {
    await ledger.add({
      resource_type: "network_object_link",
      resource_id: link.id,
      laboratory_id: laboratory.id,
      cleanup_method: "laboratory-cascade",
    });
  }

  const secondSummary = secondPage.getByTestId("topology-a11y-summary");
  await expect(secondSummary).toContainText(
    undirectedLinkPattern(names[0], "eth0", names[1], "eth0"),
  );
  await expect(secondSummary).toContainText(
    undirectedLinkPattern(names[1], "eth1", names[2], "eth0"),
  );
  await secondPage.reload();
  await expect(secondSummary).toContainText(
    undirectedLinkPattern(names[1], "eth1", names[2], "eth0"),
  );

  interactionResults.push(
    result(
      "network-object-links.shared-browser-path",
      testInfo.project.use.viewport!,
      "created a three-object path from canvas ports and observed both links in a second browser before and after refresh",
      links.map((link) => link.id),
      "keyboard",
    ),
  );
});

function undirectedLinkPattern(
  objectA: string,
  portA: string,
  objectB: string,
  portB: string,
) {
  const left = `${objectA}:${portA}`;
  const right = `${objectB}:${portB}`;
  return new RegExp(`(?:${left} ↔ ${right}|${right} ↔ ${left})`);
}

async function focusTopologyResource(canvas: Locator, resourceId: string) {
  const announcement = canvas.getByRole("status");
  for (let attempt = 0; attempt < 12; attempt += 1) {
    await canvas.press("ArrowRight");
    if ((await announcement.textContent())?.includes(resourceId)) return;
  }
  throw new Error(`Unable to focus topology resource ${resourceId}`);
}

async function createKeyboardObjectLink(
  page: Page,
  canvas: Locator,
  templates: TemplatePage,
  laboratoryId: string,
  sourceId: string,
  sourcePort: string,
  targetId: string,
  targetPort: string,
  expectedCount: number,
) {
  for (let attempt = 0; attempt < 2; attempt += 1) {
    await page.getByRole("button", { name: "刷新", exact: true }).click();
    await canvas.focus();
    await focusTopologyResource(canvas, sourceId);
    await focusTopologyPort(canvas, sourcePort);
    await canvas.press("Enter");
    await canvas.focus();
    await focusTopologyResource(canvas, targetId);
    await focusTopologyPort(canvas, targetPort);
    await canvas.press("Enter");
    try {
      return await waitForCondition(
        async () => (await templates.snapshot(laboratoryId)).network_object_links,
        (items) => items.length === expectedCount,
        `${expectedCount} shared network object links`,
        10_000,
      );
    } catch (error) {
      if (attempt > 0) {
        const statuses = await page.locator('[role="status"]').allTextContents();
        throw new Error(`${String(error)}; statuses=${JSON.stringify(statuses)}`);
      }
    }
  }
  throw new Error("unreachable connection retry state");
}

async function focusTopologyPort(canvas: Locator, portName: string) {
  const announcement = canvas.getByRole("status");
  for (let attempt = 0; attempt < 8; attempt += 1) {
    await canvas.press(attempt === 0 ? "p" : "ArrowRight");
    if ((await announcement.textContent())?.includes(`接口 ${portName}`)) return;
  }
  throw new Error(`Unable to focus topology port ${portName}`);
}
