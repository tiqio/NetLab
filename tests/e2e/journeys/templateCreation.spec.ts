import { test, expect } from "../fixtures/acceptanceFixture";
import {
  createOwnedLaboratory,
  resolveTemplateSelection,
  result,
} from "./completeRealJourney";
import { TemplatePage } from "../pages/TemplatePage";

test("template selection creates exactly one real node", async ({
  page,
  automation,
  ledger,
  runId,
  interactionResults,
}, testInfo) => {
  test.skip(
    process.env.NETLAB_ACCEPTANCE_PROFILE !== "target-host",
    "requires target-host Docker runtime and operator image",
  );
  const { laboratory } = await createOwnedLaboratory(
    page,
    automation,
    ledger,
    runId,
  );
  const selection = await resolveTemplateSelection(
    automation,
    "busybox-container",
  );
  const templates = new TemplatePage(page, automation);
  const node = await templates.createDevice({
    ...selection,
    nodeName: `busybox-${runId.slice(0, 5)}`,
    interfaces: 2,
    laboratoryId: laboratory.id,
  });
  await ledger.add({
    resource_type: "node",
    resource_id: node.id,
    laboratory_id: laboratory.id,
    revision: node.revision,
    cleanup_method: "laboratory-cascade",
  });
  const snapshot = await templates.snapshot(laboratory.id);
  expect(snapshot.nodes.filter((item) => item.name === node.name)).toHaveLength(
    1,
  );
  interactionResults.push(
    result(
      "palette.device.choose",
      testInfo.project.use.viewport!,
      "one authoritative node created",
      [node.id],
    ),
  );
});
