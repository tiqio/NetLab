import { expect, test } from "../fixtures/acceptanceFixture";
import type { Page } from "@playwright/test";
import { TemplatePage } from "../pages/TemplatePage";
import { createOwnedLaboratory, result } from "./completeRealJourney";

function mockedNode(
  id: string,
  name: string,
  kind: "qemu" | "docker",
  desiredState: "running" | "stopped",
  observedState: "running" | "stopped",
) {
  return {
    id,
    laboratory_id: "mock-lab",
    name,
    kind,
    revision: 3,
    desired_state: desiredState,
    observed_state: observedState,
    cpu_count: 1,
    cpu_quota_micros: 100000,
    memory_mib: 256,
    storage_gib: 2,
    interface_limit: 2,
    process_limit: 64,
  };
}

async function installVisualSnapshot(page: Page) {
  const nodes = Array.from({ length: 80 }, (_, index) =>
    mockedNode(
      `node-${index + 1}`,
      index === 0
        ? "QEMU router"
        : index === 1
          ? "Docker host"
          : `Node ${index + 1}`,
      index === 1 ? "docker" : "qemu",
      index === 0 ? "running" : "stopped",
      index === 0 ? "stopped" : index === 1 ? "running" : "stopped",
    ),
  );
  const placements = nodes.map((node, index) => ({
    laboratory_id: "mock-lab",
    resource_id: node.id,
    resource_type: "node",
    x: (index % 10) * 120,
    y: Math.floor(index / 10) * 100,
    revision: 1,
  }));
  await page.route("**/api/v1/**", async (route) => {
    const path = new URL(route.request().url()).pathname;
    if (path === "/api/v1/labs")
      return route.fulfill({
        json: [
          {
            id: "mock-lab",
            name: "Visual mock",
            description: "",
            revision: 9,
            recovery_policy: "auto_restore",
            lifecycle_state: "active",
          },
        ],
      });
    if (path === "/api/v1/labs/mock-lab")
      return route.fulfill({
        json: {
          laboratory: {
            id: "mock-lab",
            name: "Visual mock",
            description: "",
            revision: 9,
            recovery_policy: "auto_restore",
            lifecycle_state: "active",
          },
          nodes,
          interfaces: [
            {
              id: "if-qemu",
              node_id: "node-1",
              slot: 0,
              name: "eth0",
              driver: "virtio-net-pci",
              mac_address: "02:00:00:00:00:01",
              operational_state: "down",
              revision: 1,
            },
            {
              id: "if-docker",
              node_id: "node-2",
              slot: 0,
              name: "eth1",
              driver: "virtio-net-pci",
              mac_address: "02:00:00:00:00:02",
              desired_link_id: "link-1",
              operational_state: "up",
              revision: 1,
            },
          ],
          links: [],
          network_objects: [],
          placements,
          event_sequence: 0,
        },
      });
    if (["/api/v1/tasks", "/api/v1/templates", "/api/v1/images"].includes(path))
      return route.fulfill({ json: [] });
    return route.continue();
  });
}

test("local topology symbols and labels identify every lightweight kind", async ({
  page,
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
  for (const kind of [
    "PC",
    "Bridge",
    "NAT bridge",
    "Layer-2 switch",
    "Layer-3 switch",
  ] as const) {
    const resource = await templates.createLightweight(
      laboratory.id,
      kind,
      `${kind.toLowerCase().replaceAll(" ", "-")}-${runId.slice(0, 4)}`,
    );
    await ledger.add({
      resource_type: "network_object",
      resource_id: resource.id,
      laboratory_id: laboratory.id,
      revision: resource.revision,
      cleanup_method: "laboratory-cascade",
    });
  }
  const summary = page.getByTestId("topology-a11y-summary");
  for (const label of [
    "PC 端点",
    "二层网桥",
    "NAT 网桥",
    "二层交换机",
    "三层交换机",
  ]) {
    await expect(summary).toContainText(label);
  }
  await page.getByRole("button", { name: "切换拓扑标签密度" }).click();
  await page.getByRole("button", { name: "切换拓扑标签密度" }).click();
  await expect(page.getByRole("img", { name: /拓扑画布/ })).toBeVisible();
  interactionResults.push(
    result(
      "topology.canvas.keyboard",
      testInfo.project.use.viewport!,
      "resource kinds remained recognizable at minimal label density",
      [],
      "keyboard",
    ),
  );
});

test("browser recognizes dense QEMU and Docker lifecycle labels and hovered ports", async ({
  page,
  interactionResults,
}, testInfo) => {
  await installVisualSnapshot(page);
  await page.goto("/");
  const canvas = page.getByLabel(/拓扑画布键盘操作区/);
  await expect(canvas).toHaveAttribute("data-dense-topology", "true");
  await expect(canvas).toHaveAttribute("data-label-density", "compact");
  const summary = page.getByTestId("topology-a11y-summary");
  await expect(summary).toContainText(
    "QEMU router: QEMU 虚拟机 · 期望 运行中 · 实际 已停止",
  );
  await expect(summary).toContainText(
    "Docker host: Docker 容器 · 期望 已停止 · 实际 运行中",
  );

  const chart = page.getByRole("img", { name: /拓扑画布/ });
  const box = await chart.boundingBox();
  if (!box) throw new Error("topology canvas has no bounding box");
  const hoverDetails = page.getByTestId("topology-hover-details");
  let hovered = false;
  for (let row = 1; row < 10 && !hovered; row += 1) {
    for (let column = 1; column < 16 && !hovered; column += 1) {
      await page.mouse.move(
        box.x + (box.width * column) / 16,
        box.y + (box.height * row) / 10,
      );
      hovered = /eth[01]/.test((await hoverDetails.textContent()) || "");
    }
  }
  expect(hovered).toBe(true);
  await expect(hoverDetails).toContainText(/可用|已连接/);

  interactionResults.push(
    result(
      "topology.canvas.keyboard",
      testInfo.project.use.viewport!,
      "dense QEMU and Docker desired/actual labels remained accessible and hover disclosed port state",
      ["node-1", "node-2"],
      "pointer",
    ),
  );
});
