import { expect, type Page } from "@playwright/test";
import { BasePage } from "./BasePage";

export class CapturePage extends BasePage {
  constructor(page: Page) {
    super(page);
  }

  private status() {
    return this.page
      .getByRole("region", { name: "Diagnostics" })
      .getByRole("status");
  }

  async start(filter = "") {
    if (filter) await this.page.getByLabel("Capture filter").fill(filter);
    await this.activate(
      this.page.getByRole("button", { name: "Start capture" }),
    );
    await expect(this.status()).toContainText(/Capture queued/);
  }

  async refresh() {
    await this.activate(
      this.page
        .getByRole("region", { name: "Diagnostics" })
        .getByRole("button", { name: "Refresh", exact: true }),
    );
    await expect(this.status()).toContainText(/refreshed|queued|Stop/);
  }

  async stop() {
    await this.activate(
      this.page
        .getByRole("region", { name: "Diagnostics" })
        .getByRole("button", { name: "Stop", exact: true }),
    );
    await expect(this.status()).toContainText(/Stop queued|refreshed/);
  }

  async openWireshark() {
    await this.activate(
      this.page.getByRole("button", { name: "Open Wireshark" }),
    );
    await expect(this.status()).toContainText(
      /Wireshark opened|helper|connection refused|Failed to fetch/,
    );
  }

  async expectMetadata() {
    await expect(this.page.getByText("Packets", { exact: true })).toBeVisible();
    await expect(
      this.page.getByText("Retention", { exact: true }),
    ).toBeVisible();
    await expect(
      this.page.getByText("Truncated", { exact: true }),
    ).toBeVisible();
    await expect(
      this.page.getByRole("link", { name: "Stream" }),
    ).toHaveAttribute("href", /captures\/[^/]+\/stream/);
  }
}
