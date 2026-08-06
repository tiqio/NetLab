import { expect, test } from "./fixtures/acceptanceFixture";

test("主题切换即时生效并在刷新与路由切换后保持", async ({ page }) => {
  await page.goto("/");
  const selector = page.getByRole("combobox", { name: "外观主题" });
  await selector.selectOption("light");
  await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
  const lightSelectColors = await selector.evaluate((element) => {
    const style = getComputedStyle(element);
    return { background: style.backgroundColor, foreground: style.color };
  });
  expect(lightSelectColors.background).not.toBe(lightSelectColors.foreground);
  expect(lightSelectColors.background).toBe("rgb(255, 255, 255)");
  expect(lightSelectColors.foreground).toBe("rgb(20, 32, 43)");
  await page.reload();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
  await page.goto("/templates");
  await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
  await page.getByRole("combobox", { name: "外观主题" }).selectOption("dark");
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
  await page.goto("/");
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
  await expect(page.getByRole("combobox", { name: "外观主题" })).toHaveValue(
    "dark",
  );

  for (let iteration = 0; iteration < 20; iteration += 1) {
    const theme = iteration % 2 === 0 ? "light" : "dark";
    const route =
      iteration % 3 === 0
        ? "/templates"
        : iteration % 3 === 1
          ? "/automation"
          : "/";
    await page.getByRole("combobox", { name: "外观主题" }).selectOption(theme);
    await page.goto(route);
    await expect(page.locator("html")).toHaveAttribute("data-theme", theme);
    const overflow = await page.evaluate(
      () =>
        document.documentElement.scrollWidth -
        document.documentElement.clientWidth,
    );
    expect(overflow).toBeLessThanOrEqual(1);
  }
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
  await Promise.all([lightPage.reload(), darkPage.reload()]);
  await expect(lightPage.locator("html")).toHaveAttribute(
    "data-theme",
    "light",
  );
  await expect(darkPage.locator("html")).toHaveAttribute("data-theme", "dark");
  await lightContext.close();
  await darkContext.close();
});
