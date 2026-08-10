import { expect, test } from "./fixtures/acceptanceFixture";
import { result } from "./journeys/completeRealJourney";

test("主题切换即时生效并在刷新与路由切换后保持", async ({
  page,
  interactionResults,
}, testInfo) => {
  await page.goto("/");
  const selector = page.getByRole("combobox", { name: "外观主题" });
  await expect(selector).toBeVisible();
  await selector.selectOption("light");
  await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
  await expect(selector).toHaveCSS("background-color", "rgb(251, 249, 244)");
  await expect(selector).toHaveCSS("color", "rgb(51, 47, 41)");
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

  await page.goto("/");
  const themeSelector = page.getByRole("combobox", { name: "外观主题" });
  for (const [theme, interactionId] of [
    ["system", "appearance.theme.system"],
    ["light", "appearance.theme.light"],
    ["dark", "appearance.theme.dark"],
  ] as const) {
    await themeSelector.selectOption(theme);
    await expect(themeSelector).toHaveValue(theme);
    interactionResults.push(
      result(
        interactionId,
        testInfo.project.use.viewport!,
        `pointer-equivalent selection applied ${theme} theme preference`,
      ),
    );
  }

  await themeSelector.focus();
  for (const [key, theme, interactionId] of [
    ["Home", "system", "appearance.theme.system"],
    ["ArrowDown", "light", "appearance.theme.light"],
    ["ArrowDown", "dark", "appearance.theme.dark"],
  ] as const) {
    await page.keyboard.press(key);
    await expect(themeSelector).toHaveValue(theme);
    interactionResults.push(
      result(
        interactionId,
        testInfo.project.use.viewport!,
        `keyboard selection applied ${theme} theme preference`,
        [],
        "keyboard",
      ),
    );
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
