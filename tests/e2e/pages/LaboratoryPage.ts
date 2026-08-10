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
    const toolbar = this.page.getByLabel(/^(Laboratory toolbar|实验室工具栏)$/);
    let lastError: unknown;
    for (let attempt = 0; attempt < 3; attempt += 1) {
      try {
        await this.page.goto("/", { waitUntil: "domcontentloaded" });
        await expect(toolbar).toBeVisible({ timeout: 5_000 });
        return;
      } catch (error) {
        lastError = error;
        await this.page.waitForTimeout(250);
      }
    }
    throw lastError;
  }

  async list(): Promise<LaboratoryRecord[]> {
    const response = await this.request.get("/api/v1/labs");
    expect(response.ok()).toBeTruthy();
    const value = await response.json();
    return Array.isArray(value) ? value : [];
  }

  async create(name: string): Promise<LaboratoryRecord> {
    await this.openCreateDialog();
    const dialog = this.page.getByRole("dialog", {
      name: /^(Create laboratory|创建实验室)$/,
    });
    await expect(dialog).toBeVisible();
    await dialog.locator('[data-field="name"] input').fill(name);
    await dialog
      .getByRole("button", { name: /^(Create laboratory|创建实验室)$/ })
      .click();
    const laboratory = await waitForCondition(
      async () => (await this.list()).find((item) => item.name === name),
      (item): item is LaboratoryRecord => Boolean(item),
      `laboratory ${name}`,
    );
    const switcher = this.page.getByTestId("laboratory-switcher");
    if (!(await switcher.textContent())?.includes(laboratory.name)) {
      await this.select(laboratory.id);
    }
    await expect(switcher).toContainText(laboratory.name);
    return laboratory;
  }

  async select(id: string) {
    const laboratory = (await this.list()).find((item) => item.id === id);
    if (!laboratory) throw new Error(`Unknown laboratory ${id}`);
    const switcher = this.page.getByTestId("laboratory-switcher");
    const row = this.page.locator(`[data-laboratory-id="${id}"]`);
    for (let attempt = 0; attempt < 5; attempt += 1) {
      if (!(await row.isVisible().catch(() => false))) await switcher.click();
      try {
        await expect(row).toBeVisible({ timeout: 5_000 });
        await row.getByRole("option").click({ timeout: 5_000 });
        await expect(switcher).toContainText(laboratory.name);
        return;
      } catch {
        await this.page.waitForTimeout(250);
      }
    }
    throw new Error(`Unable to select laboratory ${id}`);
  }

  async openCreateDialog() {
    const switcher = this.page.getByTestId("laboratory-switcher");
    const createButton = this.page.getByTestId("new-laboratory");
    const dialog = this.page.getByRole("dialog", {
      name: /^(Create laboratory|创建实验室)$/,
    });
    for (let attempt = 0; attempt < 5; attempt += 1) {
      try {
        if (!(await createButton.isVisible().catch(() => false)))
          await switcher.click({ timeout: 5_000 });
        await expect(createButton).toBeVisible({ timeout: 5_000 });
        await createButton.click({ timeout: 5_000 });
        await expect(dialog).toBeVisible({ timeout: 5_000 });
        return;
      } catch {
        await this.page.waitForTimeout(250);
      }
    }
    throw new Error("Unable to open create laboratory dialog");
  }

  async openActiveActions() {
    await this.page.getByTestId("laboratory-switcher").click();
    const activeRow = this.page.locator('[data-laboratory-row="true"]').filter({
      has: this.page.locator('[role="option"][aria-selected="true"]'),
    });
    const actions = activeRow.getByRole("button", { name: / 的操作$/ });
    await expect(actions).toBeVisible();
    await actions.click();
    await expect(
      this.page.getByRole("menu", { name: / 的操作$/ }),
    ).toBeVisible();
  }

  async rename(current: LaboratoryRecord, name: string) {
    await this.openActiveActions();
    await this.page.getByRole("menuitem", { name: "重命名" }).click();
    const dialog = this.page.getByRole("dialog", { name: "重命名实验室" });
    await dialog.getByLabel(/^(Name|名称)$/).fill(name);
    await dialog.getByRole("button", { name: "保存名称" }).click();
    return waitForCondition(
      async () => (await this.list()).find((item) => item.id === current.id),
      (item) => item?.name === name,
      `renamed laboratory ${current.id}`,
    );
  }

  async duplicate(current: LaboratoryRecord) {
    const before = await this.list();
    await this.openActiveActions();
    await this.page.getByRole("menuitem", { name: "复制" }).click();
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
    const translatedMode = mode === "Export" ? "导出" : "导入";
    await this.page.getByRole("menuitem", { name: translatedMode }).click();
    await expect(
      this.page.getByRole("dialog", {
        name: `${translatedMode}实验室`,
      }),
    ).toBeVisible();
  }

  async closeDialog() {
    await this.page.getByRole("button", { name: "关闭对话框" }).click();
  }

  async cancelDelete() {
    await this.openActiveActions();
    await this.page.getByRole("menuitem", { name: "删除" }).click();
    const dialog = this.page.getByRole("dialog", { name: "删除实验室" });
    await dialog.getByRole("button", { name: "取消" }).click();
    await expect(dialog).toBeHidden();
  }

  async delete(laboratory: LaboratoryRecord) {
    await this.select(laboratory.id);
    await this.openActiveActions();
    await this.page.getByRole("menuitem", { name: "删除" }).click();
    const dialog = this.page.getByRole("dialog", { name: "删除实验室" });
    await dialog.getByRole("button", { name: "删除" }).click();
    await waitForCondition(
      async () => (await this.list()).some((item) => item.id === laboratory.id),
      (present) => !present,
      `laboratory deletion ${laboratory.id}`,
      60_000,
    );
  }

  async refresh() {
    await this.page.getByRole("button", { name: "刷新", exact: true }).click();
  }
}
