import { expect, type Page } from "@playwright/test";
import { BasePage } from "./BasePage";

export class CapturePage extends BasePage {
  constructor(page: Page) {
    super(page);
  }

  private status() {
    return this.page
      .getByRole("region", { name: /^(Diagnostics|诊断)$/ })
      .getByRole("status");
  }

  async start(filter = "") {
    if (filter)
      await this.page
        .getByLabel(/^(Capture filter|抓包过滤表达式)$/)
        .fill(filter);
    await this.activate(
      this.page.getByRole("button", {
        name: /^(Start capture|Start packet capture|开始抓取数据包)$/,
      }),
    );
    await expect(this.status()).toContainText(
      /Capture queued|抓包任务已进入队列/,
    );
  }

  async refresh() {
    await this.activate(
      this.page
        .getByRole("region", { name: /^(Diagnostics|诊断)$/ })
        .getByRole("button", { name: /^(Refresh|刷新)$/, exact: true }),
    );
    await expect(this.status()).toContainText(
      /refreshed|queued|Stop|已刷新|已进入队列|停止任务/,
    );
  }

  async stop() {
    await this.activate(
      this.page
        .getByRole("region", { name: /^(Diagnostics|诊断)$/ })
        .getByRole("button", { name: /^(Stop|停止)$/, exact: true }),
    );
    await expect(this.status()).toContainText(
      /Stop queued|refreshed|停止任务已进入队列|已刷新/,
    );
  }

  async openWireshark() {
    await this.activate(
      this.page.getByRole("button", {
        name: /^(Open Wireshark|使用 Wireshark 打开)$/,
      }),
    );
    await expect(this.status()).toContainText(
      /Wireshark opened|helper|connection refused|Failed to fetch|已使用 Wireshark|辅助程序|无法连接/,
    );
  }

  async expectMetadata() {
    await expect(
      this.page.getByText(/^(Packets|数据包)$/, { exact: true }),
    ).toBeVisible();
    await expect(
      this.page.getByText(/^(Retention|保留方式)$/, { exact: true }),
    ).toBeVisible();
    await expect(
      this.page.getByText(/^(Truncated|是否截断)$/, { exact: true }),
    ).toBeVisible();
    await expect(
      this.page.getByRole("link", { name: /^(Stream|实时流)$/ }),
    ).toHaveAttribute("href", /captures\/[^/]+\/stream/);
  }
}
