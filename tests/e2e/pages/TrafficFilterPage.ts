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
    await this.page.getByLabel("Traffic characteristic").fill(expression);
    await this.page.getByLabel("Max observations").fill(String(maximum));
    await this.activate(
      this.diagnostics().getByRole("button", { name: "Start", exact: true }),
    );
    await expect(this.status()).toContainText(/Traffic Filter queued/);
  }

  async refresh() {
    await this.activate(
      this.diagnostics().getByRole("button", {
        name: /^(Refresh|刷新)$/,
        exact: true,
      }),
    );
    await expect(this.status()).toContainText(/refreshed|queued/);
  }

  async stop() {
    await this.activate(
      this.diagnostics().getByRole("button", { name: "Stop", exact: true }),
    );
    await expect(this.status()).toContainText(/stop queued|refreshed/i);
  }

  async expectPath() {
    await expect(
      this.page.getByLabel("Traffic Filter observed packet path"),
    ).toBeVisible();
    await expect(
      this.diagnostics().getByText(
        /(?:running|stopped|failed) · \d+\/\d+ observations/,
      ),
    ).toBeVisible();
  }
}
