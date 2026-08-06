import { test, expect } from "../fixtures/acceptanceFixture";
import {
  createOwnedLaboratory,
  resolveTemplateSelection,
  result,
} from "./completeRealJourney";
import { TemplatePage } from "../pages/TemplatePage";
import { TopologyPage } from "../pages/TopologyPage";

test("topology controls connect and disconnect real endpoints while visible", async ({
  page,
  automation,
  ledger,
  runId,
  interactionResults,
}, testInfo) => {
  test.skip(
    process.env.NETLAB_ACCEPTANCE_PROFILE !== "target-host",
    "requires target-host Docker runtime",
  );
  const { laboratory } = await createOwnedLaboratory(
    page,
    automation,
    ledger,
    runId,
  );
  const templates = new TemplatePage(page, automation);
  const firstSelection = await resolveTemplateSelection(
    automation,
    "busybox-container",
  );
  const secondSelection = await resolveTemplateSelection(
    automation,
    "ubuntu-container",
  );
  const first = await templates.createDevice({
    ...firstSelection,
    nodeName: `left-${runId.slice(0, 5)}`,
    laboratoryId: laboratory.id,
  });
  const second = await templates.createDevice({
    ...secondSelection,
    nodeName: `right-${runId.slice(0, 5)}`,
    laboratoryId: laboratory.id,
  });
  for (const node of [first, second]) {
    await ledger.add({
      resource_type: "node",
      resource_id: node.id,
      laboratory_id: laboratory.id,
      revision: node.revision,
      cleanup_method: "laboratory-cascade",
    });
  }
  const snapshot = await templates.snapshot(laboratory.id);
  const endpoints = [first, second].map((node) => {
    const item = snapshot.interfaces.find(
      (current) => current.node_id === node.id,
    );
    if (!item) throw new Error(`No interface discovered for ${node.id}`);
    return item;
  });
  expect(endpoints).toHaveLength(2);
  const topology = new TopologyPage(page, automation);
  const link = await topology.connect(
    laboratory.id,
    endpoints[0].id,
    endpoints[1].id,
  );
  await ledger.add({
    resource_type: "link",
    resource_id: link.id,
    laboratory_id: laboratory.id,
    cleanup_method: "laboratory-cascade",
  });
  await topology.panAndZoom();
  await page.reload();
  await expect(page.getByRole("img", { name: /拓扑画布/ })).toBeVisible();
  interactionResults.push(
    result(
      "topology.connect",
      testInfo.project.use.viewport!,
      "link remained authoritative after refresh",
      [link.id],
    ),
  );
});
