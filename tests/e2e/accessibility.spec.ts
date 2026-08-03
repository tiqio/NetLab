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
  await expect(page.getByRole("tab", { name: /Tasks/ })).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Toggle inspector" }),
  ).toBeVisible();
  await page.setViewportSize({ width: 375, height: 812 });
  await expect(page.getByRole("heading", { name: "NetLab" })).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Toggle device palette" }),
  ).toBeVisible();
  await expect(page.getByLabel(/Topology canvas keyboard area/)).toBeVisible();
});
