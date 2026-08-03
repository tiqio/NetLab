import type { APIRequestContext, Page } from "@playwright/test";
import type { ResourceLedger } from "../fixtures/resourceLedger";
import { waitForCondition } from "../fixtures/waiters";
import { NodeOperationsPage } from "../pages/NodeOperationsPage";
import { TemplatePage } from "../pages/TemplatePage";
import { TopologyPage } from "../pages/TopologyPage";
import {
  createOwnedLaboratory,
  resolveTemplateSelection,
} from "./completeRealJourney";

export async function createRunningDiagnosticNode(options: {
  page: Page;
  automation: APIRequestContext;
  ledger: ResourceLedger;
  runId: string;
  templateKey: string;
}) {
  const { page, automation, ledger, runId, templateKey } = options;
  const { laboratory } = await createOwnedLaboratory(
    page,
    automation,
    ledger,
    runId,
  );
  const selection = await resolveTemplateSelection(automation, templateKey);
  const node = await new TemplatePage(page, automation).createDevice({
    ...selection,
    nodeName: `diag-${runId.slice(0, 6)}`,
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
  await new NodeOperationsPage(page).setLifecycle("Start");
  const running = await waitForCondition(
    async () => (await automation.get(`/api/v1/nodes/${node.id}`)).json(),
    (value: { observed_state?: string }) => value.observed_state === "running",
    `${templateKey} running`,
    180_000,
  );
  const snapshot = await waitForCondition(
    async () => (await automation.get(`/api/v1/labs/${laboratory.id}`)).json(),
    (value: { interfaces?: unknown[] }) => Boolean(value.interfaces?.length),
    "diagnostic interface discovery",
    30_000,
  );
  return {
    laboratory,
    node: running as typeof node,
    interface: snapshot.interfaces[0] as { id: string; name: string },
  };
}
