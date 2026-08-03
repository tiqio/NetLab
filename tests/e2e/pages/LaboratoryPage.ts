import { expect, type APIRequestContext, type Page } from "@playwright/test";
import { BasePage } from "./BasePage";
import { waitForCondition } from "../fixtures/waiters";

export interface LaboratoryRecord {
  id: string;
  name: string;
  revision: number;
}

export async function selectLaboratoryByName(page: Page, name: string) {
  const switcher = page.getByTestId("laboratory-switcher");
  for (let attempt = 0; attempt < 5; attempt += 1) {
    if ((await switcher.getAttribute("aria-expanded")) !== "true")
      await switcher.click();
    try {
      await page
        .getByRole("option", { name: new RegExp(name) })
        .click({ timeout: 5_000 });
      await expect(switcher).toContainText(name);
      return;
    } catch {
      await page.waitForTimeout(250);
    }
  }
  throw new Error(`Unable to select laboratory ${name}`);
}

export class LaboratoryPage extends BasePage {
  constructor(
    page: Page,
    private readonly request: APIRequestContext,
  ) {
    super(page);
  }

  async open() {
    await this.page.goto("/");
    await expect(this.page.getByLabel("Laboratory toolbar")).toBeVisible();
  }

  async list(): Promise<LaboratoryRecord[]> {
    const response = await this.request.get("/api/v1/labs");
    expect(response.ok()).toBeTruthy();
    const value = await response.json();
    return Array.isArray(value) ? value : [];
  }

  async create(name: string): Promise<LaboratoryRecord> {
    await this.openCreateDialog();
    const dialog = this.page.getByRole("dialog", { name: "Create laboratory" });
    await expect(dialog).toBeVisible();
    await dialog.getByLabel("Name").fill(name);
    await dialog.getByRole("button", { name: "Create laboratory" }).click();
    const laboratory = await waitForCondition(
      async () => (await this.list()).find((item) => item.name === name),
      (item): item is LaboratoryRecord => Boolean(item),
      `laboratory ${name}`,
    );
    await expect(this.page.getByTestId("laboratory-switcher")).toContainText(
      laboratory.name,
    );
    return laboratory;
  }

  async select(id: string) {
    const laboratory = (await this.list()).find((item) => item.id === id);
    if (!laboratory) throw new Error(`Unknown laboratory ${id}`);
    await this.page.getByTestId("laboratory-switcher").click();
    const row = this.page.locator(`[data-laboratory-id="${id}"]`);
    await expect(row).toBeVisible({ timeout: 15_000 });
    await row.getByRole("option").click();
    await expect(this.page.getByTestId("laboratory-switcher")).toContainText(
      laboratory.name,
    );
  }

  async openCreateDialog() {
    const switcher = this.page.getByTestId("laboratory-switcher");
    for (let attempt = 0; attempt < 5; attempt += 1) {
      if ((await switcher.getAttribute("aria-expanded")) !== "true")
        await switcher.click();
      try {
        await this.activate(this.page.getByTestId("new-laboratory"));
        return;
      } catch {
        await this.page.waitForTimeout(250);
      }
    }
    throw new Error("Unable to open the laboratory creation dialog");
  }

  async openActiveActions() {
    await this.page.getByTestId("laboratory-switcher").click();
    const activeRow = this.page.locator('[data-laboratory-row="true"]').filter({
      has: this.page.locator('[role="option"][aria-selected="true"]'),
    });
    const actions = activeRow.getByRole("button", { name: /^Actions for / });
    await expect(actions).toBeVisible();
    await actions.click();
    await expect(
      this.page.getByRole("menu", { name: /^Actions for / }),
    ).toBeVisible();
  }

  async rename(current: LaboratoryRecord, name: string) {
    await this.openActiveActions();
    await this.page.getByRole("menuitem", { name: "Rename" }).click();
    const dialog = this.page.getByRole("dialog", { name: "Rename laboratory" });
    await dialog.getByLabel("Name").fill(name);
    await dialog.getByRole("button", { name: "Save name" }).click();
    return waitForCondition(
      async () => (await this.list()).find((item) => item.id === current.id),
      (item) => item?.name === name,
      `renamed laboratory ${current.id}`,
    );
  }

  async duplicate(current: LaboratoryRecord) {
    const before = await this.list();
    await this.openActiveActions();
    await this.page.getByRole("menuitem", { name: "Duplicate" }).click();
    const duplicate = await waitForCondition(
      async () => {
        const currentLabs = await this.list();
        return currentLabs.find(
          (item) => !before.some((existing) => existing.id === item.id),
        );
      },
      (item): item is LaboratoryRecord => Boolean(item),
      `duplicate of ${current.id}`,
      30_000,
    );
    await this.refresh();
    return duplicate;
  }

  async openTransfer(mode: "Export" | "Import") {
    await this.openActiveActions();
    await this.page.getByRole("menuitem", { name: mode }).click();
    await expect(
      this.page.getByRole("dialog", {
        name: new RegExp(`${mode} laboratory`, "i"),
      }),
    ).toBeVisible();
  }

  async closeDialog() {
    await this.page.getByRole("button", { name: "Close dialog" }).click();
  }

  async cancelDelete() {
    await this.openActiveActions();
    await this.page.getByRole("menuitem", { name: "Delete" }).click();
    const dialog = this.page.getByRole("dialog", { name: "Delete laboratory" });
    await dialog.getByRole("button", { name: "Cancel" }).click();
    await expect(dialog).toBeHidden();
  }

  async delete(laboratory: LaboratoryRecord) {
    await this.select(laboratory.id);
    await this.openActiveActions();
    await this.page.getByRole("menuitem", { name: "Delete" }).click();
    const dialog = this.page.getByRole("dialog", { name: "Delete laboratory" });
    await dialog.getByRole("button", { name: "Delete" }).click();
    await waitForCondition(
      async () => (await this.list()).some((item) => item.id === laboratory.id),
      (present) => !present,
      `laboratory deletion ${laboratory.id}`,
      60_000,
    );
  }

  async refresh() {
    await this.page
      .getByRole("button", { name: "Refresh", exact: true })
      .click();
  }
}
