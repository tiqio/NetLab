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
      name: "Search device templates",
    });
    if (!(await search.isVisible().catch(() => false))) {
      await this.page
        .getByRole("button", { name: "Toggle device palette" })
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
      name: new RegExp(`Add ${displayName}`, "i"),
    });
  }

  async chooseLightweight(name: string) {
    await this.openPalette();
    await this.page.getByRole("button", { name, exact: true }).click();
    return this.page.getByRole("dialog", {
      name: new RegExp(`Add ${name}`, "i"),
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
    await dialog.getByLabel("Name", { exact: true }).fill(options.nodeName);
    await dialog.getByLabel("Device template").selectOption(options.templateId);
    await dialog.getByLabel("Template version").selectOption(options.versionId);
    if (options.imageId) {
      await dialog.getByLabel("Image version").selectOption(options.imageId);
    }
    if (options.interfaces) {
      await dialog
        .getByLabel("Interfaces (count)")
        .fill(String(options.interfaces));
    }
    const before = await this.snapshot(options.laboratoryId);
    const submit = dialog.getByRole("button", { name: "Add to topology" });
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
    await dialog.getByLabel("Name", { exact: true }).fill(name);
    await dialog.getByRole("button", { name: "Add to topology" }).click();
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
    };
    return {
      ...snapshot,
      nodes: snapshot.nodes || [],
      interfaces: snapshot.interfaces || [],
      links: snapshot.links || [],
      network_objects: snapshot.network_objects || [],
    };
  }
}
