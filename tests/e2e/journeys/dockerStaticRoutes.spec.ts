import type { APIRequestContext } from "@playwright/test";
import { test, expect } from "../fixtures/acceptanceFixture";
import { waitForCondition } from "../fixtures/waiters";
import {
  createOwnedLaboratory,
  resolveTemplateSelection,
  result,
} from "./completeRealJourney";
import { TemplatePage } from "../pages/TemplatePage";
import { TopologyPage } from "../pages/TopologyPage";

test("Docker routes are created, edited, and read back through the frontend", async ({
  page,
  automation,
  environment,
  ledger,
  runId,
  interactionResults,
}, testInfo) => {
  const busyboxAvailable = environment.templates.some(
    (template) =>
      template.device_family === "busybox-container" &&
      template.versions.some((version) => version.available),
  );
  test.skip(
    !environment.capabilities.docker || !busyboxAvailable,
    "requires Docker runtime and an available BusyBox image",
  );
  const { laboratory } = await createOwnedLaboratory(
    page,
    automation,
    ledger,
    runId,
  );
  const selection = await resolveTemplateSelection(
    automation,
    "busybox-container",
  );
  const templates = new TemplatePage(page, automation);
  const dialog = await templates.chooseDevice(
    selection.displayName,
    selection.runtime,
  );
  const nodeName = `routes-${runId.slice(0, 6)}`;
  await dialog.getByLabel("名称", { exact: true }).fill(nodeName);
  await dialog
    .locator('[data-field="template"] select')
    .selectOption(selection.templateId);
  await dialog
    .locator('[data-field="version"] select')
    .selectOption(selection.versionId);
  await dialog
    .locator('[data-field="image"] select')
    .selectOption(selection.imageId);
  await dialog.getByTestId("docker-ipv4-mode").selectOption("static");
  await dialog.getByTestId("docker-ipv4-address").fill("192.0.2.10/24");
  await dialog.getByRole("button", { name: "添加 IPv4 路由" }).click();
  await dialog
    .getByTestId("docker-route-0-destination")
    .fill("198.51.100.99/24");
  await dialog.getByTestId("docker-route-0-gateway").fill("192.0.2.1");
  await dialog.getByTestId("docker-route-0-metric").fill("20");
  await dialog.getByTestId("docker-ipv6-mode").selectOption("static");
  await dialog.getByTestId("docker-ipv6-address").fill("2001:db8:1::10/64");
  await dialog.getByRole("button", { name: "添加 IPv6 路由" }).click();
  await dialog
    .getByTestId("docker-route-1-destination")
    .fill("2001:db8:2::99/64");
  await dialog.getByTestId("docker-route-1-gateway").fill("2001:db8:1::1");
  await dialog.getByRole("button", { name: "添加到拓扑" }).click();

  const node = await waitForCondition(
    async () => {
      const snapshot = await templates.snapshot(laboratory.id);
      return snapshot.nodes.find((item) => item.name === nodeName);
    },
    (
      value,
    ): value is Record<string, unknown> & { id: string; revision: number } =>
      Boolean(value),
    "Docker node with routes",
    30_000,
  );
  await ledger.add({
    resource_type: "node",
    resource_id: node.id,
    laboratory_id: laboratory.id,
    revision: node.revision,
    cleanup_method: "laboratory-cascade",
  });
  await expect(dialog).toBeHidden();

  const created = await readNode(automation, node.id);
  expect(routeDestinations(created)).toEqual([
    "198.51.100.0/24",
    "2001:db8:2::/64",
  ]);

  const topology = new TopologyPage(page, automation);
  await topology.selectResourceByKeyboard(0);
  await topology.openSelectedInspector();
  await expect(
    page.getByTestId("inspector-docker-route-readiness"),
  ).toContainText("将在下次启动时应用");
  await page.getByLabel("eth0 路由 1 目标").fill("203.0.113.77/24");
  await page.getByLabel("删除 eth0 路由 2").click();
  await page.getByRole("button", { name: "保存配置" }).click();

  const updated = await waitForCondition(
    () => readNode(automation, node.id),
    (value) =>
      JSON.stringify(routeDestinations(value)) ===
      JSON.stringify(["203.0.113.0/24"]),
    "updated Docker route readback",
    30_000,
  );
  expect(routeDestinations(updated)).toEqual(["203.0.113.0/24"]);
  await expect(
    page.getByTestId("inspector-docker-route-readiness"),
  ).toContainText("203.0.113.0/24");

  interactionResults.push(
    result(
      "docker.routes.frontend",
      testInfo.project.use.viewport!,
      "created dual-stack routes and replaced them with the exact stopped-node route set",
      [node.id],
    ),
  );
});

async function readNode(
  request: APIRequestContext,
  nodeId: string,
): Promise<Record<string, unknown>> {
  const response = await request.get(`/api/v1/nodes/${nodeId}`);
  expect(response.ok()).toBeTruthy();
  return response.json();
}

function routeDestinations(node: Record<string, unknown>) {
  const config = (node.config || {}) as Record<string, unknown>;
  const interfaces = Array.isArray(config.network_interfaces)
    ? (config.network_interfaces as Array<Record<string, unknown>>)
    : [];
  return interfaces.flatMap((item) =>
    Array.isArray(item.routes)
      ? (item.routes as Array<Record<string, unknown>>).map((route) =>
          String(route.destination),
        )
      : [],
  );
}
