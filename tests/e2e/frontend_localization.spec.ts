import { expect, test } from "./fixtures/acceptanceFixture";

test("主要页面提供中文产品界面并保留技术名称", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByRole("combobox", { name: "外观主题" })).toBeVisible();
  await expect(page.getByText("NetLab", { exact: true })).toBeVisible();
  await page.goto("/templates");
  await expect(page.getByText("模板与镜像")).toBeVisible();
  await expect(page.locator('option[value="qemu"]').first()).toHaveText("QEMU");
  await page.goto("/automation");
  await expect(page.getByText("自动化与共享控制")).toBeVisible();
});
