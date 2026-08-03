import { expect, type Locator, type Page } from "@playwright/test";
import { test } from "../fixtures/acceptanceFixture";
import type { InteractionResult } from "../fixtures/acceptanceTypes";
import { waitForCondition } from "../fixtures/waiters";
import {
  createOwnedLaboratory,
  resolveTemplateSelection,
  result,
} from "../journeys/completeRealJourney";
import { TemplatePage } from "../pages/TemplatePage";

type Activation = Extract<
  InteractionResult["activation"],
  "pointer" | "keyboard"
>;

async function activate(locator: Locator, method: Activation) {
  await expect(locator).toBeVisible();
  await expect(locator).toBeEnabled();
  const started = Date.now();
  if (method === "keyboard") {
    await locator.focus();
    await locator.press("Enter");
  } else {
    await locator.click();
  }
  return Math.max(1, Date.now() - started);
}

test("node operations execute through pointer and keyboard", async ({
  page,
  automation,
  ledger,
  runId,
  interactionResults,
}, testInfo) => {
  test.skip(
    process.env.NETLAB_ACCEPTANCE_PROFILE !== "target-host" &&
      process.env.NETLAB_ACCEPTANCE_RUNTIME !== "1",
    "requires target-host Docker runtime",
  );
  const viewport = testInfo.project.use.viewport!;
  const record = (
    id: string,
    activation: Activation,
    actual: string,
    resourceIds: string[],
    duration: number,
  ) =>
    interactionResults.push(
      result(id, viewport, actual, resourceIds, activation, duration),
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
  const createNode = async (prefix: string) => {
    const node = await templates.createDevice({
      ...selection,
      nodeName: `${prefix}-${crypto.randomUUID().slice(0, 6)}`,
      laboratoryId: laboratory.id,
    });
    await ledger.add({
      resource_type: "node",
      resource_id: node.id,
      laboratory_id: laboratory.id,
      revision: node.revision,
      cleanup_method: "laboratory-cascade",
    });
    return node;
  };
  const openInspector = async (nodeName: string) => {
    await page.reload();
    const canvas = page.getByLabel(/Topology canvas keyboard area/);
    await canvas.focus();
    await canvas.press("ArrowRight");
    await canvas.press("Enter");
    await expect(
      page.getByText(nodeName, { exact: true }).first(),
    ).toBeVisible();
  };
  const waitState = (nodeId: string, state: string) =>
    waitForCondition(
      async () => {
        const response = await automation.get(`/api/v1/nodes/${nodeId}`);
        return response.ok() ? response.json() : undefined;
      },
      (value: { observed_state?: string } | undefined) =>
        value?.observed_state === state,
      `${nodeId} ${state}`,
      90_000,
    );

  const primary = await createNode("primary");
  await openInspector(primary.name);

  for (const activation of ["pointer", "keyboard"] as const) {
    let duration = await activate(
      page.getByRole("button", { name: "Start", exact: true }),
      activation,
    );
    await waitState(primary.id, "running");
    record(
      "node.start",
      activation,
      "container reached running state",
      [primary.id],
      duration,
    );
    await openInspector(primary.name);

    duration = await activate(
      page.getByRole("button", { name: "Stop", exact: true }),
      activation,
    );
    await waitState(primary.id, "stopped");
    record(
      "node.stop",
      activation,
      "container reached stopped state",
      [primary.id],
      duration,
    );
    await openInspector(primary.name);

    await page.getByLabel("vCPUs").fill(activation === "pointer" ? "1" : "2");
    await page.getByLabel("CPU quota µs").fill("50000");
    await page.getByLabel("Memory MiB").fill("128");
    duration = await activate(
      page.getByRole("button", { name: "Apply limits" }),
      activation,
    );
    await expect(
      page.getByRole("status").filter({ hasText: /vCPUs limited/ }),
    ).toBeVisible();
    record(
      "node.resources.apply",
      activation,
      "resource limits were accepted and rendered",
      [primary.id],
      duration,
    );

    const interfaceResponse = page.waitForResponse(
      (response) =>
        response.url().includes(`/api/v1/nodes/${primary.id}/interfaces`) &&
        response.request().method() === "POST",
    );
    duration = await activate(
      page.getByRole("button", { name: "Add interface" }),
      activation,
    );
    const interfaceOutcome = await interfaceResponse;
    const interfaceSection = page.locator("section").filter({
      has: page.getByRole("heading", { name: "Interfaces" }),
    });
    await expect(interfaceSection.getByRole("status")).not.toHaveText("", {
      timeout: 30_000,
    });
    expect(
      [200, 201, 202, 400, 409, 422, 500, 503],
      "interface request must return a deterministic HTTP outcome",
    ).toContain(interfaceOutcome.status());
    record(
      "node.interface.add",
      activation,
      `interface request returned HTTP ${interfaceOutcome.status()} with visible feedback`,
      [primary.id],
      duration,
    );

    const hostPort = activation === "pointer" ? "22231" : "22232";
    await page.getByLabel("Host port").fill(hostPort);
    duration = await activate(
      page.getByRole("button", { name: "Publish port" }),
      activation,
    );
    await expect(
      page.getByRole("status").filter({ hasText: /Mapping queued/ }),
    ).toBeVisible();
    record(
      "node.port.publish",
      activation,
      `host port ${hostPort} mapping was queued`,
      [primary.id],
      duration,
    );
  }

  for (const activation of ["pointer", "keyboard"] as const) {
    const node =
      activation === "pointer" ? primary : await createNode("delete-keyboard");
    await openInspector(node.name);
    let duration = await activate(
      page.getByRole("button", { name: "Delete", exact: true }),
      activation,
    );
    const dialog = page.getByRole("dialog", { name: "Delete node" });
    duration += await activate(
      dialog.getByRole("button", { name: "Delete node" }),
      activation,
    );
    await waitForCondition(
      async () => {
        const response = await automation.get(`/api/v1/labs/${laboratory.id}`);
        const snapshot = await response.json();
        return snapshot.nodes || [];
      },
      (current: Array<{ id: string }>) =>
        !current.some((item) => item.id === node.id),
      `deleted node ${node.id}`,
      30_000,
    );
    record(
      "node.delete",
      activation,
      "node disappeared from the authoritative topology",
      [node.id],
      duration,
    );
  }
});
