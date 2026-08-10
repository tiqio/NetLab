import { expect, type Page } from "@playwright/test";
import { BasePage } from "./BasePage";

export class NodeOperationsPage extends BasePage {
  constructor(page: Page) {
    super(page);
  }

  async setLifecycle(action: "Start" | "Stop") {
    const button = this.page.getByRole("button", {
      name: action === "Start" ? /^(Start|启动)$/ : /^(Stop|停止)$/,
    });
    await expect(button).toBeVisible();
    await button.click();
    const lifecycleSection = this.page.locator("section").filter({
      has: this.page.getByRole("heading", { name: "运行控制" }),
    });
    await expect(lifecycleSection.getByRole("status")).toContainText(
      action === "Start" ? /运行中/ : /已停止/,
      { timeout: 120_000 },
    );
  }

  async applyResources(cpu: number, quota: number, memory: number) {
    await this.page.getByLabel("vCPUs").fill(String(cpu));
    await this.page.getByLabel("CPU quota µs").fill(String(quota));
    await this.page.getByLabel("Memory MiB").fill(String(memory));
    await this.page.getByRole("button", { name: "Apply limits" }).click();
  }

  async runGuestCommand(command: string) {
    await this.page.getByLabel("Bounded command").fill(command);
    await this.page
      .getByRole("button", { name: "Run through QEMU guest agent" })
      .click();
  }

  async openTaskCenter() {
    const tab = this.page.getByRole("tab", { name: /^(Tasks|任务)/ });
    if (!(await tab.isVisible().catch(() => false))) {
      await this.page
        .getByRole("button", { name: "展开或收起操作区" })
        .click();
    }
    await expect(tab).toBeVisible();
    await tab.click();
  }
}
