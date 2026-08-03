import { test, expect } from "../fixtures/acceptanceFixture";
import {
  createOwnedLaboratory,
  resolveTemplateSelection,
  result,
} from "./completeRealJourney";
import { TemplatePage } from "../pages/TemplatePage";
import { TopologyPage } from "../pages/TopologyPage";
import { NodeOperationsPage } from "../pages/NodeOperationsPage";
import { waitForCondition } from "../fixtures/waiters";

test("node controls expose lifecycle tasks and authoritative state", async ({
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
  const selection = await resolveTemplateSelection(
    automation,
    "busybox-container",
  );
  const templates = new TemplatePage(page, automation);
  const node = await templates.createDevice({
    ...selection,
    nodeName: `ops-${runId.slice(0, 5)}`,
    laboratoryId: laboratory.id,
  });
  await ledger.add({
    resource_type: "node",
    resource_id: node.id,
    laboratory_id: laboratory.id,
    revision: node.revision,
    cleanup_method: "laboratory-cascade",
  });
  const topology = new TopologyPage(page, automation);
  await topology.selectResourceByKeyboard(0);
  await topology.openSelectedInspector();
  const operations = new NodeOperationsPage(page);
  await operations.setLifecycle("Start");
  await waitForCondition(
    async () => {
      const response = await automation.get(`/api/v1/nodes/${node.id}`);
      return response.json();
    },
    (value: { observed_state?: string }) => value.observed_state === "running",
    "node running",
    90_000,
  );
  await operations.openTaskCenter();
  await expect(
    page.getByRole("combobox", { name: "Task state" }),
  ).toBeVisible();
  interactionResults.push(
    result(
      "node.start",
      testInfo.project.use.viewport!,
      "node reached running and task center remained usable",
      [node.id],
    ),
  );
});
