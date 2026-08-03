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
  await expect(
    page.getByText(/Select a node and interface above/),
  ).toBeVisible();
  await page.getByRole("tab", { name: "Traffic Filter" }).click();
  await expect(
    page.getByRole("heading", { name: "拓扑流量高亮" }),
  ).toBeVisible();
  await expect(page.getByText("匹配包数")).toBeVisible();
});
