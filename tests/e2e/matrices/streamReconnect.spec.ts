import { expect, test } from "../fixtures/acceptanceFixture";
import { createOwnedLaboratory } from "../journeys/completeRealJourney";

test("page refresh restores authoritative task and diagnostic workspaces", async ({
  page,
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
  await page.getByRole("tab", { name: "Capture" }).click();
  await expect(page.getByText(/Select a node interface or link/)).toBeVisible();
  await page.reload();
  await expect(page.getByTestId("laboratory-switcher")).toContainText(
    laboratory.name,
  );
  await page.getByRole("tab", { name: /Tasks/ }).click();
  await expect(
    page.getByRole("button", { name: "Refresh tasks" }),
  ).toBeVisible();
});
