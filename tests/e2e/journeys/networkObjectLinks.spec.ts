import { expect, test } from "../fixtures/acceptanceFixture";
import type { Locator } from "@playwright/test";
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
  await middleDialog.getByLabel("Name", { exact: true }).fill(names[1]);
  await middleDialog.getByRole("button", { name: "添加二层端口" }).click();
  await middleDialog.getByRole("button", { name: "Add to topology" }).click();
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

  const canvas = page.getByLabel(/Topology canvas keyboard area/);
  await canvas.focus();
  await focusTopologyResource(canvas, first.id);
  await canvas.press("p");
  await canvas.press("Enter");
  await focusTopologyResource(canvas, middle.id);
  await canvas.press("p");
  await canvas.press("Enter");

  await waitForCondition(
    async () => (await templates.snapshot(laboratory.id)).network_object_links,
    (items) => items.length === 1,
    "first shared network object link",
    30_000,
  );
  await expect(page.getByTestId("topology-a11y-summary")).toContainText(
    undirectedLinkPattern(names[0], "eth0", names[1], "eth0"),
  );

  await focusTopologyResource(canvas, middle.id);
  await canvas.press("p");
  await canvas.press("Enter");
  await focusTopologyResource(canvas, third.id);
  await canvas.press("p");
  await canvas.press("Enter");

  const links = await waitForCondition(
    async () => (await templates.snapshot(laboratory.id)).network_object_links,
    (items) => items.length === 2,
    "two shared network object links",
    30_000,
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
