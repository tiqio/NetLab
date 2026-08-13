import { expect, test } from "../fixtures/acceptanceFixture";
import type { Page } from "@playwright/test";
import { waitForCondition } from "../fixtures/waiters";
import { TemplatePage } from "../pages/TemplatePage";
import { createOwnedLaboratory, result } from "./completeRealJourney";
import { selectLaboratoryByName } from "../pages/LaboratoryPage";

async function installRouteSnapshot(page: Page) {
  await page.addInitScript(() => {
    localStorage.setItem(
      "netlab.workspace.v1.route-lab",
      JSON.stringify({
        schemaVersion: 1,
        laboratoryId: "route-lab",
        updatedAt: new Date().toISOString(),
        panels: {
          devicePalette: { collapsed: false, size: 260 },
          inspector: { collapsed: false, size: 340 },
          bottomDrawer: { collapsed: false, size: 250 },
        },
        viewport: { centerX: 0, centerY: 0, zoom: 1 },
        groups: [],
        linkRoutes: {
          "link-current": [{ x: 0, y: 60 }],
          "link-deleted": [{ x: 1, y: 1 }],
        },
        labelDensity: "comfortable",
        reducedMotion: false,
        activeBottomTab: "tasks",
      }),
    );
  });
  await page.route("**/api/v1/**", async (route) => {
    const path = new URL(route.request().url()).pathname;
    const laboratory = {
      id: "route-lab",
      name: "Route mock",
      description: "",
      revision: 4,
      recovery_policy: "auto_restore",
      lifecycle_state: "active",
    };
    if (path === "/api/v1/labs") return route.fulfill({ json: [laboratory] });
    if (path === "/api/v1/labs/route-lab")
      return route.fulfill({
        json: {
          laboratory,
          nodes: [
            {
              id: "route-a",
              laboratory_id: "route-lab",
              name: "Route A",
              kind: "qemu",
              revision: 1,
              desired_state: "stopped",
              observed_state: "stopped",
              cpu_count: 1,
              cpu_quota_micros: 100000,
              memory_mib: 256,
              storage_gib: 2,
              interface_limit: 1,
              process_limit: 64,
            },
            {
              id: "route-b",
              laboratory_id: "route-lab",
              name: "Route B",
              kind: "docker",
              revision: 1,
              desired_state: "stopped",
              observed_state: "stopped",
              cpu_count: 1,
              cpu_quota_micros: 100000,
              memory_mib: 256,
              storage_gib: 2,
              interface_limit: 1,
              process_limit: 64,
            },
          ],
          interfaces: [
            {
              id: "route-if-a",
              node_id: "route-a",
              slot: 0,
              name: "eth0",
              driver: "virtio-net-pci",
              mac_address: "02:00:00:00:00:11",
              desired_link_id: "link-current",
              operational_state: "up",
              revision: 1,
            },
            {
              id: "route-if-b",
              node_id: "route-b",
              slot: 0,
              name: "eth0",
              driver: "virtio-net-pci",
              mac_address: "02:00:00:00:00:12",
              desired_link_id: "link-current",
              operational_state: "up",
              revision: 1,
            },
          ],
          links: [
            {
              id: "link-current",
              laboratory_id: "route-lab",
              endpoint_a_id: "route-if-a",
              endpoint_b_id: "route-if-b",
              revision: 2,
              desired_state: "connected",
              observed_state: "connected",
            },
          ],
          network_objects: [],
          placements: [
            {
              laboratory_id: "route-lab",
              resource_id: "route-a",
              resource_type: "node",
              x: -120,
              y: 0,
              revision: 1,
            },
            {
              laboratory_id: "route-lab",
              resource_id: "route-b",
              resource_type: "node",
              x: 120,
              y: 0,
              revision: 1,
            },
          ],
          event_sequence: 0,
        },
      });
    if (["/api/v1/tasks", "/api/v1/templates", "/api/v1/images"].includes(path))
      return route.fulfill({ json: [] });
    return route.continue();
  });
}

test("topology navigation persists shared coordinates across browser contexts", async ({
  page,
  context,
  automation,
  ledger,
  runId,
  interactionResults,
}, testInfo) => {
  const { laboratory } = await createOwnedLaboratory(
    page,
    automation,
    ledger,
    runId,
  );
  const templates = new TemplatePage(page, automation);
  const first = await templates.createLightweight(
    laboratory.id,
    "PC",
    `nav-a-${runId.slice(0, 4)}`,
  );
  const second = await templates.createLightweight(
    laboratory.id,
    "PC",
    `nav-b-${runId.slice(0, 4)}`,
  );
  for (const resource of [first, second]) {
    await ledger.add({
      resource_type: "network_object",
      resource_id: resource.id,
      laboratory_id: laboratory.id,
      revision: resource.revision,
      cleanup_method: "laboratory-cascade",
    });
  }

  const canvas = page.getByLabel(/拓扑画布键盘操作区/);
  await canvas.focus();
  await canvas.press("ArrowRight");
  const announcement = await canvas.getByRole("status").textContent();
  const selectedId = [first.id, second.id].find((id) =>
    announcement?.includes(id),
  );
  expect(
    selectedId,
    `unexpected keyboard announcement: ${announcement}`,
  ).toBeTruthy();
  await canvas.press("+");
  await expect(canvas.getByRole("status")).toContainText("已放大");
  await canvas.press("Alt+ArrowRight");
  const placement = await waitForCondition(
    async () => {
      const response = await automation.get(`/api/v1/labs/${laboratory.id}`);
      const snapshot = await response.json();
      return snapshot.placements?.find(
        (item: { resource_id: string }) => item.resource_id === selectedId,
      );
    },
    (value): value is { x: number; y: number; revision: number } =>
      Boolean(value),
    "keyboard placement",
  );

  const chart = page.getByRole("img", { name: /拓扑画布/ });
  await chart.hover();
  await page.mouse.wheel(0, -240);
  const box = await chart.boundingBox();
  if (!box) throw new Error("topology canvas has no bounding box");
  await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2);
  await page.mouse.down();
  await page.mouse.move(
    box.x + box.width / 2 + 30,
    box.y + box.height / 2 + 20,
  );
  await page.mouse.up();
  const persistedPlacement = await waitForCondition(
    async () => {
      const response = await automation.get(`/api/v1/labs/${laboratory.id}`);
      const snapshot = await response.json();
      return snapshot.placements?.find(
        (item: { resource_id: string }) => item.resource_id === selectedId,
      );
    },
    (value): value is { x: number; y: number; revision: number } =>
      Boolean(value && value.revision > placement.revision),
    "pointer placement",
  );
  const fitAll = page.getByTestId("fit-all");
  const fitSelection = page.getByTestId("fit-selection");
  const reset = page.getByTestId("reset-view");
  await fitAll.click();
  const organizedPlacement = await waitForCondition(
    async () => {
      const response = await automation.get(`/api/v1/labs/${laboratory.id}`);
      const snapshot = await response.json();
      return snapshot.placements?.find(
        (item: { resource_id: string }) => item.resource_id === selectedId,
      );
    },
    (value): value is { x: number; y: number; revision: number } =>
      Boolean(value && value.revision > persistedPlacement.revision),
    "organized placement",
  );
  interactionResults.push(
    result(
      "topology.viewport.fit-all",
      testInfo.project.use.viewport!,
      "pointer activation persisted a layered layout and fitted the viewport",
      [selectedId!],
      "pointer",
    ),
  );
  await fitAll.focus();
  await fitAll.press("Enter");
  const keyboardOrganizedPlacement = await waitForCondition(
    async () => {
      const response = await automation.get(`/api/v1/labs/${laboratory.id}`);
      const snapshot = await response.json();
      return snapshot.placements?.find(
        (item: { resource_id: string }) => item.resource_id === selectedId,
      );
    },
    (value): value is { x: number; y: number; revision: number } =>
      Boolean(value && value.revision > organizedPlacement.revision),
    "keyboard organized placement",
  );
  interactionResults.push(
    result(
      "topology.viewport.fit-all",
      testInfo.project.use.viewport!,
      "keyboard activation persisted a layered layout and fitted the viewport",
      [selectedId!],
      "keyboard",
    ),
  );
  for (const [interactionId, control] of [
    ["topology.viewport.fit-selection", fitSelection],
    ["topology.viewport.reset", reset],
  ] as const) {
    await control.click();
    interactionResults.push(
      result(
        interactionId,
        testInfo.project.use.viewport!,
        "pointer activation updated the browser-local viewport",
        [selectedId!],
        "pointer",
      ),
    );
    await control.focus();
    await control.press("Enter");
    interactionResults.push(
      result(
        interactionId,
        testInfo.project.use.viewport!,
        "keyboard activation updated the browser-local viewport",
        [selectedId!],
        "keyboard",
      ),
    );
  }

  await page.reload();
  await expect(page.getByRole("img", { name: /拓扑画布/ })).toBeVisible();
  const secondPage = await context.newPage();
  await secondPage.goto("/");
  await selectLaboratoryByName(secondPage, laboratory.name);
  await expect(secondPage.getByRole("img", { name: /拓扑画布/ })).toBeVisible();
  const response = await automation.get(`/api/v1/labs/${laboratory.id}`);
  const snapshot = await response.json();
  expect(
    snapshot.placements.find(
      (item: { resource_id: string }) => item.resource_id === selectedId,
    ),
  ).toMatchObject({
    x: keyboardOrganizedPlacement.x,
    y: keyboardOrganizedPlacement.y,
  });
  await secondPage.close();

  for (const interactionId of [
    "topology.viewport.wheel",
    "topology.viewport.zoom-keyboard",
    "topology.viewport.pan",
    "topology.selection.move-keyboard",
  ]) {
    interactionResults.push(
      result(
        interactionId,
        testInfo.project.use.viewport!,
        "viewport remained local and shared placement converged",
        [selectedId!],
        interactionId.includes("keyboard") ? "keyboard" : "pointer",
      ),
    );
  }
});

test("browser-local routes support cleanup cancel reset save and refresh", async ({
  page,
  interactionResults,
}, testInfo) => {
  await installRouteSnapshot(page);
  await page.goto("/");
  const readRoutes = () =>
    page.evaluate(() =>
      JSON.parse(localStorage.getItem("netlab.workspace.v1.route-lab") || "{}"),
    );
  await expect
    .poll(async () => Object.keys((await readRoutes()).linkRoutes || {}))
    .toEqual(["link-current"]);

  const canvas = page.getByLabel(/拓扑画布键盘操作区/);
  await canvas.focus();
  await canvas.press("ArrowRight");
  await canvas.press("ArrowRight");
  await canvas.press("ArrowRight");
  await page.getByRole("button", { name: "链路操作" }).click();
  await page.getByRole("button", { name: "编辑本地路由" }).click();
  await page.getByRole("button", { name: "取消路径编辑" }).click();
  expect((await readRoutes()).linkRoutes["link-current"]).toEqual([
    { x: 0, y: 60 },
  ]);

  await page.getByRole("button", { name: "链路操作" }).click();
  await page.getByRole("button", { name: "编辑本地路由" }).click();
  await page.getByRole("button", { name: "恢复自动布线" }).click();
  await expect
    .poll(async () => (await readRoutes()).linkRoutes["link-current"])
    .toBeUndefined();

  await page.getByRole("button", { name: "链路操作" }).click();
  await page.getByRole("button", { name: "编辑本地路由" }).click();
  await page.getByRole("button", { name: "保存路径" }).click();
  await expect
    .poll(async () => (await readRoutes()).linkRoutes["link-current"]?.length)
    .toBe(1);
  await page.reload();
  await expect
    .poll(async () => (await readRoutes()).linkRoutes["link-current"]?.length)
    .toBe(1);

  interactionResults.push(
    result(
      "topology.route",
      testInfo.project.use.viewport!,
      "browser-local route cleanup, cancellation, reset, save, and refresh persistence completed",
      ["link-current"],
      "pointer",
    ),
  );
});
