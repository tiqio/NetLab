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
      this.page.getByRole("navigation", {
        name: /^(Console sessions|终端会话)$/,
      }),
    ).toBeVisible();
  }

  async switchTo(label: RegExp) {
    await this.activate(this.page.getByRole("button", { name: label }));
  }

  async reconnect() {
    await this.activate(
      this.page.getByRole("button", {
        name: /^(Reconnect|重新连接)$/,
        exact: true,
      }),
    );
    await expect(
      this.page
        .getByRole("region", { name: /^(Diagnostics|诊断)$/ })
        .getByRole("status"),
    ).toContainText(
      /connecting|connected|reconnecting|正在连接|已连接|正在重新连接/,
    );
  }

  async add(mode: "Telnet" | "Serial" | "VNC") {
    await this.activate(
      this.page.getByRole("button", {
        name:
          mode === "Telnet"
            ? /^(Add terminal session|新增终端会话)$/
            : mode === "Serial"
              ? /^(Add serial console|新增串口终端)$/
              : /^(Add VNC session|新增 VNC 会话)$/,
      }),
    );
  }

  async close(label: string) {
    await this.activate(
      this.page.getByRole("button", {
        name: new RegExp(`^(?:Close|关闭) ${label}`),
      }),
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
