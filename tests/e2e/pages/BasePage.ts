import { expect, type Locator, type Page } from "@playwright/test";

export class BasePage {
  constructor(protected readonly page: Page) {}

  byRole(role: Parameters<Page["getByRole"]>[0], name?: string | RegExp) {
    return this.page.getByRole(role, name === undefined ? undefined : { name });
  }

  byTestId(testId: string) {
    return this.page.getByTestId(testId);
  }

  async activate(locator: Locator, method: "pointer" | "keyboard" = "pointer") {
    const started = Date.now();
    await expect(locator).toBeVisible();
    await expect(locator).toBeEnabled();
    if (method === "keyboard") {
      await locator.focus();
      await this.page.keyboard.press("Enter");
    } else {
      await locator.click();
    }
    return Math.max(1, Date.now() - started);
  }

  async confirmDialog(name?: string | RegExp) {
    const dialog = this.page.getByRole("dialog");
    await expect(dialog).toBeVisible();
    const button = name
      ? dialog.getByRole("button", { name })
      : dialog.getByRole("button", { name: /confirm|delete|continue/i });
    await button.click();
  }

  async refreshAndExpect(locator: Locator) {
    await this.page.reload();
    await expect(locator).toBeVisible();
  }
}
