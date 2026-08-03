import { expect, type Page } from "@playwright/test";
import { BasePage } from "./BasePage";

export class ConsolePage extends BasePage {
  constructor(page: Page) {
    super(page);
  }

  async open(mode: "Telnet" | "VNC") {
    await this.activate(
      this.page.getByRole("button", { name: `Open ${mode}` }),
    );
    await expect(
      this.page.getByRole("navigation", { name: "Console sessions" }),
    ).toBeVisible();
  }

  async switchTo(label: RegExp) {
    await this.activate(this.page.getByRole("button", { name: label }));
  }

  async reconnect() {
    await this.activate(
      this.page.getByRole("button", { name: "Reconnect", exact: true }),
    );
    await expect(
      this.page
        .getByRole("region", { name: "Diagnostics" })
        .getByRole("status"),
    ).toContainText(/connecting|connected|reconnecting/);
  }

  async add(mode: "Telnet" | "VNC") {
    await this.activate(
      this.page.getByRole("button", {
        name: mode === "Telnet" ? "Add terminal session" : "Add VNC session",
      }),
    );
  }

  async close(label: string) {
    await this.activate(
      this.page.getByRole("button", { name: `Close ${label}` }),
    );
  }

  async expectUnsupported(mode: "Telnet" | "VNC") {
    const button = this.page.getByRole("button", { name: `Open ${mode}` });
    await expect(button).toBeDisabled();
    await expect(button).toHaveAttribute(
      "title",
      new RegExp("not supported", "i"),
    );
  }
}
