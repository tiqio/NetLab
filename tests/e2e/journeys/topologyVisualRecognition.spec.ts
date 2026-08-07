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

async function installVisualSnapshot(
  page: Page,
  options: { semanticConnections?: boolean; traffic?: boolean } = {},
) {
  const semanticConnections = options.semanticConnections !== false;
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
  placements.push(
    {
      laboratory_id: "mock-lab",
      resource_id: "nat-1",
      resource_type: "network_object",
      x: -180,
      y: 80,
      revision: 1,
    },
    {
      laboratory_id: "mock-lab",
      resource_id: "bridge-1",
      resource_type: "network_object",
      x: 360,
      y: 80,
      revision: 1,
    },
  );
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
            {
              id: "if-qemu-2",
              node_id: "node-1",
              slot: 1,
              name: "eth1",
              driver: "virtio-net-pci",
              mac_address: "02:00:00:00:00:03",
              desired_link_id: "link-parallel-a",
              operational_state: "up",
              revision: 1,
            },
            {
              id: "if-docker-2",
              node_id: "node-2",
              slot: 1,
              name: "eth2",
              driver: "veth",
              mac_address: "02:00:00:00:00:04",
              desired_link_id: "link-parallel-a",
              operational_state: "up",
              revision: 1,
            },
            {
              id: "if-qemu-3",
              node_id: "node-1",
              slot: 2,
              name: "eth2",
              driver: "virtio-net-pci",
              mac_address: "02:00:00:00:00:05",
              desired_link_id: "link-parallel-b",
              operational_state: "up",
              revision: 1,
            },
            {
              id: "if-docker-3",
              node_id: "node-2",
              slot: 2,
              name: "eth3",
              driver: "veth",
              mac_address: "02:00:00:00:00:06",
              desired_link_id: "link-parallel-b",
              operational_state: "up",
              revision: 1,
            },
            {
              id: "if-qemu-4",
              node_id: "node-1",
              slot: 3,
              name: "eth3",
              driver: "virtio-net-pci",
              mac_address: "02:00:00:00:00:07",
              desired_link_id: "link-parallel-c",
              operational_state: "up",
              revision: 1,
            },
            {
              id: "if-docker-4",
              node_id: "node-2",
              slot: 3,
              name: "eth4",
              driver: "veth",
              mac_address: "02:00:00:00:00:08",
              desired_link_id: "link-parallel-c",
              operational_state: "up",
              revision: 1,
            },
          ],
          links: [
            {
              id: "link-parallel-a",
              laboratory_id: "mock-lab",
              endpoint_a_id: "if-qemu-2",
              endpoint_b_id: "if-docker-2",
              revision: 1,
              desired_state: "connected",
              observed_state: "connected",
            },
            {
              id: "link-parallel-b",
              laboratory_id: "mock-lab",
              endpoint_a_id: "if-qemu-3",
              endpoint_b_id: "if-docker-3",
              revision: 1,
              desired_state: "connected",
              observed_state: "connected",
            },
            {
              id: "link-parallel-c",
              laboratory_id: "mock-lab",
              endpoint_a_id: "if-qemu-4",
              endpoint_b_id: "if-docker-4",
              revision: 1,
              desired_state: "connected",
              observed_state: "connected",
            },
          ],
          network_objects: [
            {
              id: "nat-1",
              laboratory_id: "mock-lab",
              name: "Internet NAT",
              kind: "nat_bridge",
              revision: 1,
              desired_state: "active",
              observed_state: "active",
              config: {},
            },
            {
              id: "bridge-1",
              laboratory_id: "mock-lab",
              name: "Shared LAN",
              kind: "bridge",
              revision: 1,
              desired_state: "active",
              observed_state: "active",
              config: {},
            },
          ],
          network_attachments: semanticConnections
            ? [
                {
                  id: "attachment-nat",
                  network_object_id: "nat-1",
                  interface_id: "if-qemu",
                  port_name: "nat0",
                  observed_state: "failed",
                },
                {
                  id: "attachment-bridge",
                  network_object_id: "bridge-1",
                  interface_id: "if-docker",
                  port_name: "lan0",
                  observed_state: "provisioning",
                },
              ]
            : [],
          network_object_links: [],
          placements,
          event_sequence: 0,
        },
      });
    if (path === "/api/v1/traffic-filters")
      return route.fulfill({
        json: options.traffic
          ? [
              {
                ambiguous: false,
                traffic_filter: {
                  id: "filter-1",
                  laboratory_id: "mock-lab",
                  expression: "icmp",
                  color: "#22c55e",
                  state: "running",
                  max_observations: 100,
                  link_ids: ["link-parallel-a"],
                  observations: [
                    {
                      fingerprint: "icmp-one",
                      resource_type: "link",
                      resource_id: "link-parallel-a",
                      interface_id: "if-qemu-2",
                      link_id: "link-parallel-a",
                      direction: "a_to_b",
                      first_seen: "2026-08-07T08:00:00Z",
                      last_seen: "2026-08-07T08:00:00Z",
                      count: 1,
                      bytes: 84,
                    },
                  ],
                  created_at: "2026-08-07T08:00:00Z",
                },
              },
            ]
          : [],
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

test("mixed connection states and semantic legend remain recognizable across themes", async ({
  page,
}) => {
  await installVisualSnapshot(page);
  await page.goto("/");
  const summary = page.getByTestId("topology-a11y-summary");
  await expect(summary).toContainText("QEMU router:eth0 ↔ Internet NAT:nat0");
  await expect(summary).toContainText("失败");
  await expect(summary).toContainText("Docker host:eth1 ↔ Shared LAN:lan0");
  await expect(summary).toContainText("状态转换中");
  const legend = page.getByTestId("topology-connection-legend");
  await expect(legend).toContainText("NAT 管理上联");
  await expect(legend).toContainText("共享广播域");
  await legend.getByRole("button", { name: /收起连接语义图例/ }).click();
  await expect(legend).toHaveAttribute("data-collapsed", "true");
  await legend.getByRole("button", { name: /展开连接语义图例/ }).press("Enter");
  await expect(legend).toHaveAttribute("data-collapsed", "false");
  const theme = page.locator('select[aria-label="外观主题"]');
  await theme.selectOption("dark");
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
  await theme.selectOption("light");
  await expect(page.locator("html")).toHaveAttribute("data-theme", "light");

  await page.unroute("**/api/v1/**");
  await installVisualSnapshot(page, { semanticConnections: false });
  await page.reload();
  await expect(legend).toHaveCount(0);
});

test("parallel links remain keyboard selectable at 50, 100, and 200 percent zoom", async ({
  page,
}) => {
  await installVisualSnapshot(page);
  await page.goto("/");
  for (const zoom of [0.5, 1, 2]) {
    await page.evaluate((value) => {
      localStorage.setItem(
        "netlab.workspace.v1.mock-lab",
        JSON.stringify({
          schemaVersion: 1,
          laboratoryId: "mock-lab",
          updatedAt: new Date().toISOString(),
          panels: {
            devicePalette: { collapsed: false, size: 260 },
            inspector: { collapsed: false, size: 340 },
            bottomDrawer: { collapsed: false, size: 250 },
          },
          viewport: { centerX: 0, centerY: 0, zoom: value },
          groups: [],
          linkRoutes: {},
          labelDensity: "comfortable",
          reducedMotion: false,
          activeBottomTab: "tasks",
        }),
      );
    }, zoom);
    await page.reload();
    const canvas = page.getByLabel(/拓扑画布键盘操作区/);
    await expect(canvas).toHaveAttribute("data-viewport-zoom", String(zoom));
    await canvas.focus();
    const found = new Set<string>();
    for (let index = 0; index < 90 && found.size < 3; index += 1) {
      await canvas.press("ArrowRight");
      const announcement =
        (await canvas.getByRole("status").textContent()) || "";
      for (const id of [
        "link-parallel-a",
        "link-parallel-b",
        "link-parallel-c",
      ])
        if (announcement.includes(id)) found.add(id);
    }
    expect([...found].sort()).toEqual([
      "link-parallel-a",
      "link-parallel-b",
      "link-parallel-c",
    ]);
  }
});

test("traffic particles expire before the lingering direction guide", async ({
  page,
}) => {
  await installVisualSnapshot(page, { traffic: true });
  await page.goto("/");
  await page.getByRole("tab", { name: "流量过滤" }).click();
  const canvas = page.getByLabel(/拓扑画布键盘操作区/);
  await expect(canvas).toHaveAttribute("data-traffic-recent", "1");
  await expect(canvas).toHaveAttribute("data-traffic-lingering", "1");
  await expect(canvas).toHaveAttribute("data-traffic-recent", "0", {
    timeout: 2_000,
  });
  await expect(canvas).toHaveAttribute("data-traffic-lingering", "1");
  await expect(canvas).toHaveAttribute("data-traffic-lingering", "0", {
    timeout: 6_000,
  });
});
