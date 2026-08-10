import { expect, test } from "../fixtures/acceptanceFixture";
import { waitForCondition } from "../fixtures/waiters";
import {
  createOwnedLaboratory,
  createOwnedLightweightPair,
  result,
} from "./completeRealJourney";

test("unified plus and keyboard connection match direct drag results", async ({
  page,
  automation,
  ledger,
  runId,
  interactionResults,
}, testInfo) => {
  test.skip(
    process.env.NETLAB_ACCEPTANCE_PROFILE !== "target-host",
    "requires target-host topology runtime",
  );
  const { laboratory } = await createOwnedLaboratory(
    page,
    automation,
    ledger,
    runId,
  );
  const { first, second } = await createOwnedLightweightPair(
    page,
    automation,
    ledger,
    laboratory.id,
  );
  const canvas = page.getByLabel(/拓扑画布键盘操作区/);
  await canvas.focus();
  await canvas.press("ArrowRight");
  const connector = page.locator("[data-topology-connector]");
  await expect(connector).toBeVisible();
  const sourceId = await connector.getAttribute("data-connector-resource-id");
  await connector.press("Enter");
  const canvasStatus = page.locator('p[role="status"].absolute');
  await expect(canvasStatus).toContainText("请选择兼容目标");
  const targetId = sourceId === first.id ? second.id : first.id;
  await canvas.focus();
  for (let attempt = 0; attempt < 4; attempt += 1) {
    await canvas.press("ArrowRight");
    if ((await canvas.getByRole("status").textContent())?.includes(targetId))
      break;
  }
  await expect(canvas.getByRole("status")).toContainText(targetId);
  await canvas.press("Enter");
  const chooser = page.getByRole("dialog", { name: /选择目标端点/ });
  if (await chooser.isVisible())
    await chooser.getByRole("option").first().press("Enter");
  const snapshot = await waitForCondition(
    async () =>
      (
        await automation.get(`/api/v1/labs/${laboratory.id}/connections`)
      ).json(),
    (value: { connections?: Array<{ backing_kind?: string }> }) =>
      (value.connections || []).length === 1,
    "plus-created unified connection",
  );
  expect(snapshot.connections[0].backing_kind).toBe("network_object_link");
  await canvas.press("Escape");
  await expect(page.locator("[data-connection-preview]")).toHaveCount(0);
  interactionResults.push(
    result(
      "topology.connection.unified-plus",
      testInfo.project.use.viewport!,
      "plus and keyboard entry reused the unified endpoint chooser and command",
      [laboratory.id],
      "keyboard",
    ),
  );
});
