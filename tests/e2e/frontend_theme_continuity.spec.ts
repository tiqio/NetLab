import { expect, test } from "./fixtures/acceptanceFixture";

test("主题切换即时生效并在刷新与路由切换后保持", async ({ page }) => {
  await page.goto("/");
  const selector = page.getByRole("combobox", { name: "外观主题" });
  await selector.selectOption("light");
  await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
  await page.reload();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
  await page.goto("/templates");
  await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
  await page.getByRole("combobox", { name: "外观主题" }).selectOption("dark");
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
});

test("两个浏览器上下文的主题偏好相互隔离", async ({ browser }) => {
  const lightContext = await browser.newContext();
  const darkContext = await browser.newContext();
  const lightPage = await lightContext.newPage();
  const darkPage = await darkContext.newPage();
  await Promise.all([lightPage.goto("/"), darkPage.goto("/")]);
  await lightPage
    .getByRole("combobox", { name: "外观主题" })
    .selectOption("light");
  await darkPage
    .getByRole("combobox", { name: "外观主题" })
    .selectOption("dark");
  await expect(lightPage.locator("html")).toHaveAttribute(
    "data-theme",
    "light",
  );
  await expect(darkPage.locator("html")).toHaveAttribute("data-theme", "dark");
  await lightContext.close();
  await darkContext.close();
});
