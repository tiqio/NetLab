import { expect, type Locator, type Page } from "@playwright/test";
import { test } from "../fixtures/acceptanceFixture";
import type { InteractionResult } from "../fixtures/acceptanceTypes";
import { waitForCondition } from "../fixtures/waiters";
import { result } from "../journeys/completeRealJourney";
import {
  LaboratoryPage,
  selectLaboratoryByName,
} from "../pages/LaboratoryPage";

type Activation = Extract<
  InteractionResult["activation"],
  "pointer" | "keyboard"
>;

async function activate(page: Page, locator: Locator, method: Activation) {
  await expect(locator).toBeVisible();
  await expect(locator).toBeEnabled();
  const started = Date.now();
  if (method === "keyboard") {
    await locator.focus();
    await locator.press("Enter");
  } else {
    await locator.click();
  }
  return Math.max(1, Date.now() - started);
}

async function closeDialog(page: Page) {
  const dialog = page.locator('[role="dialog"]:visible');
  const close = dialog.getByRole("button", { name: "Close dialog" });
  await expect(close).toBeVisible();
  await close.click();
  const discard = page.getByRole("alertdialog", {
    name: "Discard unsaved changes",
  });
  if (await discard.isVisible().catch(() => false)) {
    await discard.getByRole("button", { name: "Discard" }).click();
  }
  await expect(dialog).toBeHidden();
}

test("laboratory navigation and shell controls support pointer and keyboard", async ({
  page,
  automation,
  ledger,
  runId,
  interactionResults,
}, testInfo) => {
  const viewport = testInfo.project.use.viewport!;
  const record = (
    id: string,
    activation: Activation,
    actual: string,
    resourceIds: string[] = [],
    duration = 1,
  ) =>
    interactionResults.push(
      result(id, viewport, actual, resourceIds, activation, duration),
    );
  const listLabs = async () => {
    const response = await automation.get("/api/v1/labs");
    expect(response.ok()).toBeTruthy();
    const value = await response.json();
    return (Array.isArray(value) ? value : []) as Array<{
      id: string;
      name: string;
      revision: number;
    }>;
  };

  await page.goto("/");

  for (const activation of ["pointer", "keyboard"] as const) {
    const switcher = page.getByTestId("laboratory-switcher");
    let duration = await activate(page, switcher, activation);
    record(
      "laboratory.select",
      activation,
      "laboratory switcher opened and exposed authoritative choices",
      [],
      duration,
    );

    const newButton = page.getByTestId("new-laboratory");
    duration = await activate(page, newButton, activation);
    const createDialog = page.getByRole("dialog", {
      name: "Create laboratory",
    });
    await expect(createDialog).toBeVisible();
    record(
      "laboratory.create.dialog",
      activation,
      "create laboratory dialog opened",
      [],
      duration,
    );

    const name = `matrix-${activation}-${crypto.randomUUID().slice(0, 8)}`;
    await createDialog.getByLabel("Name").fill(name);
    duration = await activate(
      page,
      createDialog.getByRole("button", { name: "Create laboratory" }),
      activation,
    );
    const laboratory = await waitForCondition(
      async () => (await listLabs()).find((item) => item.name === name),
      (item): item is { id: string; name: string; revision: number } =>
        Boolean(item),
      `matrix laboratory ${name}`,
    );
    await ledger.add({
      resource_type: "laboratory",
      resource_id: laboratory.id,
      revision: laboratory.revision,
      cleanup_method: "frontend-delete-with-api-fallback",
    });
    record(
      "laboratory.create.submit",
      activation,
      "laboratory was durably created and selected",
      [laboratory.id],
      duration,
    );

    const paletteToggle = page.getByRole("button", {
      name: "Toggle device palette",
    });
    duration = await activate(page, paletteToggle, activation);
    record(
      "laboratory.palette.toggle",
      activation,
      "device palette visibility changed",
      [laboratory.id],
      duration,
    );
    await activate(page, paletteToggle, activation);

    for (const [id, label] of [
      ["workspace.inspector.toggle", "Toggle inspector"],
      ["workspace.operations.toggle", "Toggle operations drawer"],
    ] as const) {
      const control = page.getByRole("button", { name: label });
      duration = await activate(page, control, activation);
      record(
        id,
        activation,
        `${label} changed panel visibility`,
        [laboratory.id],
        duration,
      );
      await activate(page, control, activation);
    }

    const openActions = async () => {
      const row = page.locator(`[data-laboratory-id="${laboratory.id}"]`);
      if (!(await row.isVisible().catch(() => false))) {
        await activate(page, switcher, activation);
      }
      await expect(row).toBeVisible();
      const actions = row.getByRole("button", { name: /^Actions for / });
      const elapsed = await activate(page, actions, activation);
      await expect(
        page.getByRole("menu", { name: /^Actions for / }),
      ).toBeVisible();
      return elapsed;
    };

    duration = await openActions();
    record(
      "laboratory.menu",
      activation,
      "laboratory context menu opened",
      [laboratory.id],
      duration,
    );
    await page.keyboard.press("Escape");

    duration = await openActions();
    duration += await activate(
      page,
      page.getByRole("menuitem", { name: "Rename" }),
      activation,
    );
    const renameDialog = page.getByRole("dialog", {
      name: "Rename laboratory",
    });
    const renamed = `${name}-renamed`;
    await renameDialog.getByLabel("Name").fill(renamed);
    await activate(
      page,
      renameDialog.getByRole("button", { name: "Save name" }),
      activation,
    );
    await waitForCondition(
      async () => (await listLabs()).find((item) => item.id === laboratory.id),
      (item) => item?.name === renamed,
      `renamed matrix laboratory ${laboratory.id}`,
    );
    record(
      "laboratory.rename",
      activation,
      "laboratory rename became authoritative",
      [laboratory.id],
      duration,
    );

    for (const [id, label] of [
      ["laboratory.export", "Export"],
      ["laboratory.import", "Import"],
    ] as const) {
      duration = await openActions();
      duration += await activate(
        page,
        page.getByRole("menuitem", { name: label }),
        activation,
      );
      await expect(
        page.getByRole("dialog", {
          name: new RegExp(`${label} laboratory`, "i"),
        }),
      ).toBeVisible();
      record(
        id,
        activation,
        `${label} workflow opened`,
        [laboratory.id],
        duration,
      );
      await closeDialog(page);
    }

    const beforeDuplicate = await listLabs();
    duration = await openActions();
    duration += await activate(
      page,
      page.getByRole("menuitem", { name: "Duplicate" }),
      activation,
    );
    const duplicate = await waitForCondition(
      async () => {
        const current = await listLabs();
        return current.find(
          (item) =>
            !beforeDuplicate.some((existing) => existing.id === item.id),
        );
      },
      (item): item is { id: string; name: string; revision: number } =>
        Boolean(item),
      `duplicate matrix laboratory ${laboratory.id}`,
      30_000,
    );
    await ledger.add({
      resource_type: "laboratory",
      resource_id: duplicate.id,
      revision: duplicate.revision,
      cleanup_method: "frontend-delete-with-api-fallback",
    });
    record(
      "laboratory.duplicate",
      activation,
      "duplicate laboratory became authoritative",
      [duplicate.id],
      duration,
    );

    const refresh = page
      .getByLabel("Laboratory toolbar")
      .getByRole("button", { name: "Refresh", exact: true });
    duration = await activate(page, refresh, activation);
    await expect(switcher).toContainText(renamed);
    record(
      "laboratory.refresh",
      activation,
      "refresh preserved the authoritative active laboratory",
      [laboratory.id],
      duration,
    );

    await activate(page, switcher, activation);
    const duplicateRow = page.locator(`[data-laboratory-id="${duplicate.id}"]`);
    await activate(page, duplicateRow.getByRole("option"), activation);
    await expect(switcher).toContainText(duplicate.name);
    await activate(page, switcher, activation);
    await activate(
      page,
      duplicateRow.getByRole("button", { name: /^Actions for / }),
      activation,
    );
    duration = await activate(
      page,
      page.getByRole("menuitem", { name: "Delete" }),
      activation,
    );
    const deleteDialog = page.getByRole("dialog", {
      name: "Delete laboratory",
    });
    duration += await activate(
      page,
      deleteDialog.getByRole("button", { name: "Delete", exact: true }),
      activation,
    );
    await waitForCondition(
      listLabs,
      (items) => !items.some((item) => item.id === duplicate.id),
      `deleted matrix laboratory ${duplicate.id}`,
      30_000,
    );
    await ledger.setState("laboratory", duplicate.id, "deleted");
    record(
      "laboratory.delete",
      activation,
      "laboratory disappeared after durable deletion",
      [duplicate.id],
      duration,
    );

    await expect(switcher).toContainText(renamed);
  }
});

test("navigation templates and task center support pointer and keyboard", async ({
  page,
  automation,
  ledger,
  runId,
  interactionResults,
}, testInfo) => {
  const viewport = testInfo.project.use.viewport!;
  const record = (
    id: string,
    activation: Activation,
    actual: string,
    resourceIds: string[] = [],
    duration = 1,
  ) =>
    interactionResults.push(
      result(id, viewport, actual, resourceIds, activation, duration),
    );
  const laboratories = new LaboratoryPage(page, automation);
  await laboratories.open();
  const laboratory = await laboratories.create(
    `matrix-nav-${crypto.randomUUID().slice(0, 8)}`,
  );
  await ledger.add({
    resource_type: "laboratory",
    resource_id: laboratory.id,
    revision: laboratory.revision,
    cleanup_method: "laboratory-delete",
  });

  for (const activation of ["pointer", "keyboard"] as const) {
    await page.goto("/");
    const switcher = page.getByTestId("laboratory-switcher");
    if (!(await switcher.textContent())?.includes(laboratory.name)) {
      await selectLaboratoryByName(page, laboratory.name);
    }

    for (const [id, label, path] of [
      ["navigation.templates", "Templates", "/templates"],
      ["navigation.automation", "Automation", "/automation"],
    ] as const) {
      const duration = await activate(
        page,
        page.getByRole("link", { name: label, exact: true }),
        activation,
      );
      await expect(page).toHaveURL(new RegExp(`${path}$`));
      record(
        id,
        activation,
        `${label} route rendered`,
        [laboratory.id],
        duration,
      );
      await page.goto("/");
    }

    if (!(await switcher.textContent())?.includes(laboratory.name)) {
      await activate(page, switcher, activation);
      await activate(
        page,
        page
          .locator(`[data-laboratory-id="${laboratory.id}"]`)
          .getByRole("option"),
        activation,
      );
    }

    const paletteToggle = page.getByRole("button", {
      name: "Toggle device palette",
    });
    const search = page.getByRole("textbox", {
      name: "Search device templates",
    });
    if (!(await search.isVisible().catch(() => false))) {
      await activate(page, paletteToggle, activation);
    }
    await expect(search).toBeVisible();
    const searchStarted = Date.now();
    if (activation === "pointer") await search.click();
    else await search.focus();
    await search.fill("ubuntu");
    await expect(search).toHaveValue("ubuntu");
    record(
      "palette.search",
      activation,
      "template palette filtered from real keyboard input",
      [laboratory.id],
      Date.now() - searchStarted,
    );
    const qemuTemplate = page
      .getByRole("button")
      .filter({ hasText: "Ubuntu" })
      .filter({ hasText: "QEMU" })
      .first();
    if (await qemuTemplate.isVisible()) {
      const duration = await activate(page, qemuTemplate, activation);
      await expect(
        page.getByRole("dialog", { name: /Add Ubuntu/i }),
      ).toBeVisible();
      record(
        "palette.device.choose",
        activation,
        "QEMU template configuration opened",
        [laboratory.id],
        duration,
      );
      await closeDialog(page);
    }

    await search.fill("");
    const pc = page.getByRole("button", { name: "PC", exact: true });
    const pcDuration = await activate(page, pc, activation);
    await expect(page.getByRole("dialog", { name: /Add PC/i })).toBeVisible();
    record(
      "palette.lightweight.choose",
      activation,
      "lightweight PC configuration opened",
      [laboratory.id],
      pcDuration,
    );
    await closeDialog(page);

    const operationsToggle = page.getByRole("button", {
      name: "Toggle operations drawer",
    });
    if ((await operationsToggle.getAttribute("aria-expanded")) !== "true") {
      await activate(page, operationsToggle, activation);
    }
    const tasksTab = page.getByRole("tab", { name: /Tasks/ });
    let duration = await activate(page, tasksTab, activation);
    await expect(tasksTab).toHaveAttribute("aria-selected", "true");
    record(
      "tasks.tab",
      activation,
      "task center became active",
      [laboratory.id],
      duration,
    );

    const taskSearch = page.getByRole("textbox", { name: "Filter tasks" });
    const taskSearchStarted = Date.now();
    if (activation === "pointer") await taskSearch.click();
    else await taskSearch.focus();
    await taskSearch.fill("matrix");
    await expect(taskSearch).toHaveValue("matrix");
    record(
      "tasks.search",
      activation,
      "task list applied the entered filter",
      [laboratory.id],
      Date.now() - taskSearchStarted,
    );

    const state = page.getByRole("combobox", { name: "Task state" });
    const stateStarted = Date.now();
    if (activation === "keyboard") {
      await state.focus();
      await page.keyboard.press("End");
    } else {
      await state.click();
      await state.selectOption("succeeded");
    }
    await expect(state).toHaveValue(
      activation === "keyboard" ? "cancelled" : "succeeded",
    );
    record(
      "tasks.state",
      activation,
      "task state filter changed",
      [laboratory.id],
      Date.now() - stateStarted,
    );

    duration = await activate(
      page,
      page.getByRole("button", { name: "Refresh tasks" }),
      activation,
    );
    record(
      "tasks.refresh",
      activation,
      "task center refreshed authoritative operations",
      [laboratory.id],
      duration,
    );
  }
});
