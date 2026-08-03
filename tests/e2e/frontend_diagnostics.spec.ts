import { expect, test } from "./fixtures/acceptanceFixture";
import { createOwnedLaboratory } from "./journeys/completeRealJourney";

test("diagnostic tabs expose real empty-scope outcomes", async ({
  page,
  automation,
  ledger,
  runId,
}) => {
  await createOwnedLaboratory(page, automation, ledger, runId);
  await expect(page.getByRole("tab", { name: "Console" })).toBeVisible();
  await page.getByRole("tab", { name: "Capture" }).click();
  await expect(page.getByText(/Select a node interface or link/)).toBeVisible();
  await page.getByRole("tab", { name: "Traffic Filter" }).click();
  await expect(
    page.getByText(/No matching packets observed yet/),
  ).toBeVisible();
});
