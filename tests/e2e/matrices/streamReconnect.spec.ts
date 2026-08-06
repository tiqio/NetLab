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
  await page.getByRole("tab", { name: "抓包" }).click();
  await expect(page.getByText(/请在上方选择节点和接口/)).toBeVisible();
  await page.reload();
  await expect(page.getByTestId("laboratory-switcher")).toContainText(
    laboratory.name,
  );
  await page.getByRole("tab", { name: "任务" }).click();
  await expect(page.getByRole("button", { name: "刷新任务" })).toBeVisible();
});
