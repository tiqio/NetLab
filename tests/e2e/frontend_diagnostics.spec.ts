import { expect, test } from "./fixtures/acceptanceFixture";
import { createOwnedLaboratory } from "./journeys/completeRealJourney";

test("diagnostic tabs expose real empty-scope outcomes", async ({
  page,
  automation,
  ledger,
  runId,
}) => {
  await createOwnedLaboratory(page, automation, ledger, runId);
  await expect(page.getByRole("tab", { name: "终端" })).toBeVisible();
  await page.getByRole("tab", { name: "抓包" }).click();
  await expect(page.getByText(/请在上方选择节点和接口/)).toBeVisible();
  await page.getByRole("tab", { name: "流量过滤" }).click();
  await expect(
    page.getByRole("heading", { name: "拓扑流量高亮" }),
  ).toBeVisible();
  await expect(page.getByText("匹配包数")).toBeVisible();
});
