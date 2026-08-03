import { expect, type Locator, type Page } from "@playwright/test";

export async function assertKeyboardReachable(page: Page, locator: Locator) {
  await locator.focus();
  await expect(locator).toBeFocused();
  await expect(locator).toBeInViewport();
  const outline = await locator.evaluate((element) => {
    const style = getComputedStyle(element);
    return `${style.outlineStyle}:${style.outlineWidth}:${style.boxShadow}`;
  });
  expect(outline).not.toBe("none:0px:none");
}

export async function assertDialogReachable(page: Page) {
  const dialog = page.getByRole("dialog").last();
  await expect(dialog).toBeVisible();
  await expect(dialog).toBeInViewport();
  await expect(dialog.getByRole("button").last()).toBeInViewport();
}

export async function activateByKeyboard(page: Page, locator: Locator) {
  await assertKeyboardReachable(page, locator);
  await page.keyboard.press("Enter");
}
