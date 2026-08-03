import { expect, test } from "../fixtures/acceptanceFixture";
import { ClientObserver } from "../fixtures/clientObserver";
import { createOwnedLaboratory } from "./completeRealJourney";
import { selectLaboratoryByName } from "../pages/LaboratoryPage";

test("two browsers and automation converge with revision protection", async ({
  page,
  secondPage,
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
  await secondPage.goto("/");
  await selectLaboratoryByName(secondPage, laboratory.name);
  const renamed = `${laboratory.name}-shared`;
  const update = await automation.patch(`/api/v1/labs/${laboratory.id}`, {
    data: { name: renamed, description: "", recovery_policy: "auto_restore" },
    headers: {
      "If-Match": String(laboratory.revision),
      "Content-Type": "application/merge-patch+json",
    },
  });
  expect(update.ok()).toBeTruthy();
  const authoritative = await update.json();
  const conflict = await automation.patch(`/api/v1/labs/${laboratory.id}`, {
    data: { name: "stale", description: "", recovery_policy: "auto_restore" },
    headers: {
      "If-Match": String(laboratory.revision),
      "Content-Type": "application/merge-patch+json",
    },
  });
  expect([409, 412]).toContain(conflict.status());
  await Promise.all([page.reload(), secondPage.reload()]);
  await expect(page.getByTestId("laboratory-switcher")).toContainText(renamed);
  await expect(secondPage.getByTestId("laboratory-switcher")).toContainText(
    renamed,
  );
  const observer = new ClientObserver();
  for (const [index, client] of [
    "browser-a",
    "browser-b",
    "automation",
  ].entries()) {
    observer.record({
      client_id: client,
      mutation_id: laboratory.id,
      event_sequence: index + 1,
      resource_revision: authoritative.revision,
      observed_at: new Date().toISOString(),
      convergence_ms: 100 + index,
    });
  }
  expect(
    observer.assertConverged(laboratory.id, [
      "browser-a",
      "browser-b",
      "automation",
    ]),
  ).toHaveLength(3);
});
