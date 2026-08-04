import { expect, type Page } from "@playwright/test";
import { BasePage } from "./BasePage";

export class TrafficFilterPage extends BasePage {
  constructor(page: Page) {
    super(page);
  }

  private diagnostics() {
    return this.page.getByRole("region", { name: "Diagnostics" });
  }

  private status() {
    return this.diagnostics().getByRole("status");
  }

  async start(expression: string, maximum = 100) {
    await this.page.getByLabel("pcap 过滤表达式").fill(expression);
    await this.page.getByLabel("最大记录数").fill(String(maximum));
    await this.activate(
      this.diagnostics().getByRole("button", {
        name: /^(Start|启动)$/,
        exact: true,
      }),
    );
    await expect(this.status()).toContainText(/正在启动|已运行/);
  }

  async refresh() {
    await this.activate(
      this.diagnostics().getByRole("button", {
        name: /^(Refresh|刷新)$/,
        exact: true,
      }),
    );
    await expect(this.status()).toContainText(/已刷新|正在启动|已运行/);
  }

  async stop() {
    await this.activate(
      this.diagnostics().getByRole("button", {
        name: /^(Stop|停止)$/,
        exact: true,
      }),
    );
    await expect(this.status()).toContainText(/正在停止|已停止/);
  }

  async expectPath() {
    await expect(
      this.diagnostics().getByRole("heading", { name: "拓扑流量高亮" }),
    ).toBeVisible();
    await expect(
      this.diagnostics().getByText(
        /(?:running|stopped|failed) · \d+\/\d+ 条记录/,
      ),
    ).toBeVisible();
  }
}
