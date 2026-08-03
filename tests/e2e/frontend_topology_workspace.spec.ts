import { expect, test } from "./fixtures/acceptanceFixture";
import { createOwnedLaboratory } from "./journeys/completeRealJourney";
import { selectLaboratoryByName } from "./pages/LaboratoryPage";

test("workspace shares topology while keeping preferences browser-local", async ({
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
  await expect(
    secondPage.getByLabel(/Topology canvas keyboard area/),
  ).toBeVisible();
  await page.evaluate(
    (id) =>
      localStorage.setItem(
        `netlab.workspace.v1.${id}`,
        JSON.stringify({ local: true }),
      ),
    laboratory.id,
  );
  expect(
    await secondPage.evaluate(
      (id) => localStorage.getItem(`netlab.workspace.v1.${id}`),
      laboratory.id,
    ),
  ).toBeNull();
});
