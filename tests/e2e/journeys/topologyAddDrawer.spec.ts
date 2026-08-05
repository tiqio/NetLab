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
  const canvas = page.getByRole("img", { name: /Topology canvas/ });
  const canvasBefore = await canvas.boundingBox();
  const drawer = await topology.openResourceDrawer();
  const drawerBox = await drawer.boundingBox();
  expect(drawerBox).not.toBeNull();
  expect(drawerBox!.x).toBeGreaterThan(
    testInfo.project.use.viewport!.width / 2,
  );
  expect(drawerBox!.height).toBeGreaterThan(
    testInfo.project.use.viewport!.height * 0.9,
  );

  const form = await topology.chooseDrawerResource("PC");
  const name = `drawer-pc-${runId.slice(0, 6)}`;
  await form.getByLabel("Name", { exact: true }).fill(name);
  await form.getByRole("button", { name: "Add to topology" }).click();

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
      "topology.add-drawer.lightweight",
      testInfo.project.use.viewport!,
      "opened the right drawer and created an authoritative PC without moving the canvas",
      [resource.id],
    ),
  );
});
