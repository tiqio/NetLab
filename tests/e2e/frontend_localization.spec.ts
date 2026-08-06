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
  await expect(page.getByText("REST 与 MCP 能力一致性")).toBeVisible();
  await expect(page.getByText("不存在仅界面可用的变更操作")).toBeVisible();
});

test("中文长文本在最小视口保持可达且不产生页面级横向滚动", async ({ page }) => {
  await page.setViewportSize({ width: 1024, height: 768 });
  for (const route of ["/", "/templates", "/automation"]) {
    await page.goto(route);
    const dimensions = await page.evaluate(() => ({
      viewport: document.documentElement.clientWidth,
      page: document.documentElement.scrollWidth,
    }));
    expect(dimensions.page - dimensions.viewport).toBeLessThanOrEqual(1);
  }
});
