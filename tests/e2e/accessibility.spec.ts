import { expect, test } from "./fixtures/acceptanceFixture";
import { createOwnedLaboratory } from "./journeys/completeRealJourney";

test("topology remains keyboard accessible and responsive", async ({
  page,
  automation,
  ledger,
  runId,
}) => {
  await createOwnedLaboratory(page, automation, ledger, runId);
  await expect(page.getByRole("heading", { name: "NetLab" })).toBeVisible();
  await page.keyboard.press("Tab");
  await expect(page.locator(":focus")).toBeVisible();
  await expect(page.getByRole("tab", { name: "任务" })).toBeVisible();
  await expect(
    page.getByRole("button", { name: "展开或收起检查器" }),
  ).toBeVisible();
  await page.setViewportSize({ width: 375, height: 812 });
  await expect(page.getByRole("heading", { name: "NetLab" })).toBeVisible();
  await expect(
    page.getByRole("button", { name: "切换设备面板" }),
  ).toBeVisible();
  await expect(page.getByLabel(/拓扑画布键盘操作区/)).toBeVisible();
});
