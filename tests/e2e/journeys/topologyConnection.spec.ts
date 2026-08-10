import { expect, test } from "../fixtures/acceptanceFixture";
import { waitForCondition } from "../fixtures/waiters";
import { TemplatePage } from "../pages/TemplatePage";
import {
  createOwnedLaboratory,
  resolveTemplateSelection,
  result,
} from "./completeRealJourney";

test("keyboard connection chooser reconnect and disconnect preserve authoritative links", async ({
  page,
  automation,
  ledger,
  runId,
  interactionResults,
}, testInfo) => {
  test.skip(
    process.env.NETLAB_ACCEPTANCE_PROFILE !== "target-host",
    "requires target-host node and data-plane runtime",
  );
  const { laboratory } = await createOwnedLaboratory(
    page,
    automation,
    ledger,
    runId,
  );
  const templates = new TemplatePage(page, automation);
  const selection = await resolveTemplateSelection(
    automation,
    "busybox-container",
  );
  const nodes = [];
  for (const name of ["connect-a", "connect-b"]) {
    const node = await templates.createDevice({
      ...selection,
      nodeName: `${name}-${runId.slice(0, 4)}`,
      interfaces: 2,
      laboratoryId: laboratory.id,
    });
    nodes.push(node);
    await ledger.add({
      resource_type: "node",
      resource_id: node.id,
      laboratory_id: laboratory.id,
      revision: node.revision,
      cleanup_method: "laboratory-cascade",
    });
  }
  const canvas = page.getByLabel(/拓扑画布键盘操作区/);
  await canvas.focus();
  await canvas.press("ArrowRight");
  await canvas.press("c");
  const sourceChooser = page.getByRole("dialog", {
    name: /^(Choose source interface|选择源接口)$/,
  });
  await expect(sourceChooser).toBeVisible();
  await sourceChooser.getByRole("option").first().click();
  await canvas.focus();
  await canvas.press("ArrowRight");
  await canvas.press("Enter");
  const targetChooser = page.getByRole("dialog", {
    name: /^(Choose target interface|选择目标接口)$/,
  });
  await expect(targetChooser).toBeVisible();
  await targetChooser.getByRole("option").first().click();
  const link = await waitForCondition(
    async () => {
      const response = await automation.get(`/api/v1/labs/${laboratory.id}`);
      const snapshot = await response.json();
      return snapshot.links?.[0];
    },
    (value): value is { id: string } => Boolean(value),
    "keyboard-created link",
  );
  await ledger.add({
    resource_type: "link",
    resource_id: link.id,
    laboratory_id: laboratory.id,
    cleanup_method: "laboratory-cascade",
  });
  await canvas.focus();
  await canvas.press("ArrowRight");
  await page.getByRole("button", { name: "链路操作" }).click();
  await page.getByRole("button", { name: "重新连接端点" }).click();
  const reconnectChooser = page.getByRole("dialog", {
    name: /^(Choose replacement interface|选择替换接口)$/,
  });
  await reconnectChooser.getByRole("option").first().click();
  await expect(page.getByTestId("reconnect-task-feedback")).toBeVisible();
  await canvas.press("Delete");
  interactionResults.push(
    result(
      "topology.connection.keyboard",
      testInfo.project.use.viewport!,
      "keyboard connection and atomic reconnect submitted without replacing the original link early",
      [link.id],
      "keyboard",
    ),
    result(
      "topology.reconnect",
      testInfo.project.use.viewport!,
      "reconnect task feedback remained visible",
      [link.id],
    ),
  );
});
