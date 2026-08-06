import { expect, test } from "../fixtures/acceptanceFixture";
import { waitForCondition } from "../fixtures/waiters";
import { TopologyPage } from "../pages/TopologyPage";
import { createOwnedLaboratory, result } from "./completeRealJourney";

test("the right-side add drawer creates an authoritative lightweight resource", async ({
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
  const canvas = page.getByRole("img", { name: /拓扑画布/ });
  const canvasBefore = await canvas.boundingBox();
  const drawer = await topology.openResourceDrawer();
  const drawerBox = await drawer.boundingBox();
  expect(drawerBox).not.toBeNull();
  expect(
    Math.abs(
      drawerBox!.x + drawerBox!.width - testInfo.project.use.viewport!.width,
    ),
  ).toBeLessThanOrEqual(1);
  expect(drawerBox!.height).toBeGreaterThan(
    testInfo.project.use.viewport!.height * 0.9,
  );

  const form = await topology.chooseDrawerResource("PC");
  const name = `drawer-pc-${runId.slice(0, 6)}`;
  await form.getByLabel("名称", { exact: true }).fill(name);
  await form.getByRole("button", { name: "添加到拓扑" }).click();

  const resource = await waitForCondition(
    async () => {
      const response = await automation.get(`/api/v1/labs/${laboratory.id}`);
      const snapshot = await response.json();
      return snapshot.network_objects.find(
        (item: { name: string }) => item.name === name,
      );
    },
    (value): value is { id: string; revision: number } => Boolean(value),
    "PC created from add drawer",
    30_000,
  );
  await ledger.add({
    resource_type: "network_object",
    resource_id: resource.id,
    laboratory_id: laboratory.id,
    revision: resource.revision,
    cleanup_method: "laboratory-cascade",
  });
  await expect(form).toBeHidden();
  await expect(page.getByTestId("topology-a11y-summary")).toContainText(name);

  const canvasAfter = await canvas.boundingBox();
  expect(canvasAfter).toEqual(canvasBefore);
  interactionResults.push(
    result(
      "topology.add-drawer.submit",
      testInfo.project.use.viewport!,
      "opened the right drawer and created an authoritative PC without moving the canvas",
      [resource.id],
    ),
  );
});

test("the add drawer keeps long-form state and confirms dirty close", async ({
  page,
  automation,
  ledger,
  runId,
}) => {
  await createOwnedLaboratory(page, automation, ledger, runId);
  const topology = new TopologyPage(page, automation);
  await topology.openResourceDrawer();
  const form = await topology.chooseDrawerResource("PC");
  const name = form.getByLabel("名称", { exact: true });
  await name.fill(`draft-${runId.slice(0, 6)}`);
  const pageScroll = await page.evaluate(() => window.scrollY);
  const body = page.locator("[data-sheet-body]");
  await body.evaluate((element) => {
    element.scrollTop = element.scrollHeight;
  });
  await expect(name).toHaveValue(`draft-${runId.slice(0, 6)}`);
  expect(await page.evaluate(() => window.scrollY)).toBe(pageScroll);
  await form.getByRole("button", { name: "取消" }).click();
  const discard = page.getByRole("alertdialog", { name: "放弃未保存的更改" });
  await expect(discard).toBeVisible();
  await discard.getByRole("button", { name: "继续编辑" }).click();
  await expect(name).toHaveValue(`draft-${runId.slice(0, 6)}`);
});

test("the add drawer creates every lightweight resource kind through the real API", async ({
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
  const topology = new TopologyPage(page, automation);
  for (const [label, kind] of [
    ["PC", "pc"],
    ["Bridge", "bridge"],
    ["NAT bridge", "nat_bridge"],
    ["Lightweight L2 Switch", "switch_l2"],
    ["Lightweight L3 Switch", "switch_l3"],
  ] as const) {
    await topology.openResourceDrawer();
    const form = await topology.chooseDrawerResource(label);
    const name = `drawer-${kind}-${runId.slice(0, 6)}`;
    await form.getByLabel("名称", { exact: true }).fill(name);
    await form.getByRole("button", { name: "添加到拓扑" }).click();
    const resource = await waitForCondition(
      async () => {
        const response = await automation.get(`/api/v1/labs/${laboratory.id}`);
        const snapshot = await response.json();
        return snapshot.network_objects.find(
          (item: { name: string; kind: string }) =>
            item.name === name && item.kind === kind,
        );
      },
      (value): value is { id: string; revision: number } => Boolean(value),
      `${kind} created from add drawer`,
      30_000,
    );
    await ledger.add({
      resource_type: "network_object",
      resource_id: resource.id,
      laboratory_id: laboratory.id,
      revision: resource.revision,
      cleanup_method: "laboratory-cascade",
    });
    await expect(form).toBeHidden();
  }
});
