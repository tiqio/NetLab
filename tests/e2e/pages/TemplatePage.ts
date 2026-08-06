import { expect, type APIRequestContext, type Page } from "@playwright/test";
import { BasePage } from "./BasePage";
import { waitForCondition } from "../fixtures/waiters";

export class TemplatePage extends BasePage {
  constructor(
    page: Page,
    private readonly request: APIRequestContext,
  ) {
    super(page);
  }

  async openPalette() {
    const search = this.page.getByRole("textbox", {
      name: /^(Search device templates|搜索设备模板)$/,
    });
    if (!(await search.isVisible().catch(() => false))) {
      await this.page
        .getByRole("button", { name: /^(Toggle device palette|切换设备面板)$/ })
        .click();
    }
    await expect(search).toBeVisible();
  }

  async chooseDevice(displayName: string, runtime: "QEMU" | "DOCKER") {
    await this.openPalette();
    const button = this.page
      .getByRole("button")
      .filter({ hasText: displayName })
      .filter({ hasText: runtime })
      .first();
    await expect(button).toBeVisible();
    await button.click();
    return this.page.getByRole("dialog", {
      name: new RegExp(`(?:Add|添加) ${displayName}`, "i"),
    });
  }

  async chooseLightweight(name: string) {
    const paletteName =
      name === "Layer-2 switch"
        ? "Lightweight L2 Switch"
        : name === "Layer-3 switch"
          ? "Lightweight L3 Switch"
          : name;
    await this.openPalette();
    await this.page
      .getByRole("button", {
        name: new RegExp(
          `^${paletteName.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}\\b`,
          "i",
        ),
      })
      .click();
    return this.page.getByRole("dialog", {
      name: new RegExp(`(?:Add|添加) ${paletteName}`, "i"),
    });
  }

  async createDevice(options: {
    displayName: string;
    runtime: "QEMU" | "DOCKER";
    nodeName: string;
    templateId: string;
    versionId: string;
    imageId?: string;
    interfaces?: number;
    laboratoryId: string;
  }) {
    const dialog = await this.chooseDevice(
      options.displayName,
      options.runtime,
    );
    await dialog.locator('[data-field="name"] input').fill(options.nodeName);
    const selectors = dialog.locator("select");
    await selectors.nth(0).selectOption(options.templateId);
    await selectors.nth(1).selectOption(options.versionId);
    if (options.imageId) {
      await selectors.nth(2).selectOption(options.imageId);
    }
    if (options.interfaces) {
      await dialog
        .getByLabel(/^(Interfaces \(count\)|接口数量)$/)
        .fill(String(options.interfaces));
    }
    const before = await this.snapshot(options.laboratoryId);
    const submit = dialog.getByRole("button", {
      name: /^(Add to topology|添加到拓扑)$/,
    });
    await expect(submit).toBeEnabled();
    await submit.click();
    const node = await waitForCondition(
      async () => {
        const current = await this.snapshot(options.laboratoryId);
        return current.nodes.find(
          (node) =>
            node.name === options.nodeName &&
            !before.nodes.some((existing) => existing.id === node.id),
        );
      },
      (
        node,
      ): node is Record<string, unknown> & { id: string; revision: number } =>
        Boolean(node),
      `node ${options.nodeName}`,
      30_000,
    );
    await expect(dialog).toBeHidden();
    await expect(this.page.getByTestId("topology-a11y-summary")).toContainText(
      options.nodeName,
    );
    return node;
  }

  async createLightweight(
    laboratoryId: string,
    kind: "PC" | "Layer-2 switch" | "Layer-3 switch" | "Bridge" | "NAT bridge",
    name: string,
  ) {
    const dialog = await this.chooseLightweight(kind);
    await dialog.getByLabel(/^(Name|名称)$/, { exact: true }).fill(name);
    await dialog
      .getByRole("button", { name: /^(Add to topology|添加到拓扑)$/ })
      .click();
    const resource = await waitForCondition(
      async () => {
        const current = await this.snapshot(laboratoryId);
        return current.network_objects.find((item) => item.name === name);
      },
      (
        item,
      ): item is Record<string, unknown> & { id: string; revision: number } =>
        Boolean(item),
      `lightweight resource ${name}`,
      30_000,
    );
    await expect(dialog).toBeHidden();
    await expect(this.page.getByTestId("topology-a11y-summary")).toContainText(
      name,
    );
    return resource;
  }

  async snapshot(laboratoryId: string) {
    const response = await this.request.get(`/api/v1/labs/${laboratoryId}`);
    expect(response.ok()).toBeTruthy();
    const snapshot = (await response.json()) as {
      nodes: Array<
        Record<string, unknown> & { id: string; name: string; revision: number }
      >;
      interfaces: Array<
        Record<string, unknown> & { id: string; node_id: string }
      >;
      links: Array<Record<string, unknown> & { id: string }>;
      network_objects: Array<
        Record<string, unknown> & { id: string; name: string; revision: number }
      >;
      network_object_links: Array<
        Record<string, unknown> & {
          id: string;
          object_a_id: string;
          port_a_name: string;
          object_b_id: string;
          port_b_name: string;
        }
      >;
    };
    return {
      ...snapshot,
      nodes: snapshot.nodes || [],
      interfaces: snapshot.interfaces || [],
      links: snapshot.links || [],
      network_objects: snapshot.network_objects || [],
      network_object_links: snapshot.network_object_links || [],
    };
  }
}
