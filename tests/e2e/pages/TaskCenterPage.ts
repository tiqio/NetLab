import { expect, type Page } from "@playwright/test";
import { BasePage } from "./BasePage";

export class TaskCenterPage extends BasePage {
  constructor(page: Page) {
    super(page);
  }

  async filter(text: string) {
    await this.page.getByRole("textbox", { name: "Filter tasks" }).fill(text);
  }

  async selectState(state: string) {
    await this.page
      .getByRole("combobox", { name: "Task state" })
      .selectOption(state);
  }

  async refresh() {
    await this.page.getByRole("button", { name: "Refresh tasks" }).click();
  }

  async expectTask(taskId: string) {
    await expect(
      this.page.locator("article").filter({ hasText: taskId }),
    ).toBeVisible();
  }
}
