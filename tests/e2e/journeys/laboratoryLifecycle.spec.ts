import { test, expect } from "../fixtures/acceptanceFixture";
import { result } from "./completeRealJourney";
import { LaboratoryPage } from "../pages/LaboratoryPage";

test("laboratory controls complete a real shared lifecycle", async ({
  page,
  automation,
  ledger,
  runId,
  interactionResults,
}, testInfo) => {
  const laboratories = new LaboratoryPage(page, automation);
  await laboratories.open();
  const created = await laboratories.create(`lab-${runId.slice(0, 8)}`);
  await ledger.add({
    resource_type: "laboratory",
    resource_id: created.id,
    revision: created.revision,
    cleanup_method: "frontend-delete-with-api-fallback",
  });
  interactionResults.push(
    result(
      "laboratory.create.submit",
      testInfo.project.use.viewport!,
      "laboratory visible after create",
      [created.id],
    ),
  );

  const renamed = await laboratories.rename(created, `${created.name}-renamed`);
  await ledger.add({
    resource_type: "laboratory",
    resource_id: renamed.id,
    revision: renamed.revision,
    cleanup_method: "frontend-delete-with-api-fallback",
  });
  await laboratories.refresh();
  await expect(page.getByTestId("laboratory-switcher")).toContainText(
    created.name,
  );
  interactionResults.push(
    result(
      "laboratory.rename",
      testInfo.project.use.viewport!,
      "renamed laboratory remained selected after refresh",
      [renamed.id],
    ),
    result(
      "laboratory.refresh",
      testInfo.project.use.viewport!,
      "authoritative laboratory selection was preserved",
      [renamed.id],
    ),
  );

  await laboratories.cancelDelete();
  expect(
    (await laboratories.list()).some((item) => item.id === created.id),
  ).toBeTruthy();

  await laboratories.openTransfer("Export");
  interactionResults.push(
    result(
      "laboratory.export",
      testInfo.project.use.viewport!,
      "export workflow opened",
      [renamed.id],
    ),
  );
  await laboratories.closeDialog();
  await laboratories.openTransfer("Import");
  interactionResults.push(
    result(
      "laboratory.import",
      testInfo.project.use.viewport!,
      "import workflow opened",
      [renamed.id],
    ),
  );
  await laboratories.closeDialog();

  const duplicate = await laboratories.duplicate(renamed);
  await ledger.add({
    resource_type: "laboratory",
    resource_id: duplicate.id,
    revision: duplicate.revision,
    cleanup_method: "frontend-delete-with-api-fallback",
  });
  interactionResults.push(
    result(
      "laboratory.duplicate",
      testInfo.project.use.viewport!,
      "duplicate laboratory became authoritative",
      [duplicate.id],
    ),
  );
  await laboratories.delete(duplicate);
  await ledger.setState("laboratory", duplicate.id, "deleted");
  await laboratories.delete(renamed);
  await ledger.setState("laboratory", renamed.id, "deleted");

  interactionResults.push(
    result(
      "laboratory.delete",
      testInfo.project.use.viewport!,
      "laboratories absent after durable deletion",
      [created.id, duplicate.id],
    ),
  );
});
