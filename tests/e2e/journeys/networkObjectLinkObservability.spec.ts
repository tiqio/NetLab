import { expect, test } from "../fixtures/acceptanceFixture";
import type { Locator } from "@playwright/test";
import { waitForCondition, waitForTask } from "../fixtures/waiters";
import { createOwnedLaboratory, result } from "./completeRealJourney";

test("capture and Traffic Filter stay isolated across parallel object links", async ({
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
  const suffix = runId.slice(0, 6);
  const objectA = await createL2Object(
    automation,
    laboratory.id,
    `observe-a-${suffix}`,
    ["a0", "a1"],
  );
  const objectB = await createL2Object(
    automation,
    laboratory.id,
    `observe-b-${suffix}`,
    ["b0", "b1"],
  );
  expect((await waitForTask(automation, objectA.taskId)).terminal_state).toBe(
    "succeeded",
  );
  expect((await waitForTask(automation, objectB.taskId)).terminal_state).toBe(
    "succeeded",
  );
  for (const object of [objectA, objectB]) {
    await ledger.add({
      resource_type: "network_object",
      resource_id: object.id,
      laboratory_id: laboratory.id,
      revision: object.revision,
      cleanup_method: "laboratory-cascade",
    });
  }
  const primary = await createObjectLink(
    automation,
    laboratory.id,
    objectA.id,
    "a0",
    objectB.id,
    "b0",
    crypto.randomUUID(),
  );
  const parallel = await createObjectLink(
    automation,
    laboratory.id,
    objectA.id,
    "a1",
    objectB.id,
    "b1",
    crypto.randomUUID(),
  );
  expect((await waitForTask(automation, primary.taskId)).terminal_state).toBe(
    "succeeded",
  );
  expect((await waitForTask(automation, parallel.taskId)).terminal_state).toBe(
    "succeeded",
  );
  for (const link of [primary, parallel]) {
    await ledger.add({
      resource_type: "network_object_link",
      resource_id: link.id,
      laboratory_id: laboratory.id,
      cleanup_method: "laboratory-cascade",
    });
  }
  await waitForCondition(
    async () =>
      (
        await automation.get(
          `/api/v1/labs/${laboratory.id}/network-object-links`,
        )
      ).json(),
    (links: Array<{ id: string; observed_state: string }>) =>
      links.length === 2 &&
      links.every((link) => link.observed_state === "connected"),
    "parallel object links connected",
    30_000,
  );
  await page.reload();
  const canvas = page.getByLabel(/拓扑画布键盘操作区/);
  await canvas.focus();
  await focusTopologyResource(canvas, primary.id);
  await canvas.press("Enter");

  const diagnostics = page.getByRole("region", { name: "诊断" });
  await page.getByRole("tab", { name: "抓包" }).click();
  await expect(
    diagnostics.getByRole("button", {
      name: `${objectA.name}:a0 ↔ ${objectB.name}:b0`,
    }),
  ).toBeVisible();
  await diagnostics.getByRole("button", { name: "开始抓包" }).click();
  const captures = await waitForCondition(
    async () =>
      (
        await automation.get(`/api/v1/captures?laboratory_id=${laboratory.id}`)
      ).json(),
    (
      items: Array<{
        id: string;
        source_type: string;
        source_id: string;
        state: string;
      }>,
    ) =>
      items.some(
        (item) =>
          item.source_type === "network_object_link" &&
          item.source_id === primary.id,
      ),
    "selected object-link capture",
    30_000,
  );
  const capture = captures.find(
    (item) =>
      item.source_type === "network_object_link" &&
      item.source_id === primary.id,
  )!;
  expect(capture.source_id).toBe(primary.id);
  expect(capture.source_id).not.toBe(parallel.id);
  await ledger.add({
    resource_type: "capture",
    resource_id: capture.id,
    laboratory_id: laboratory.id,
    cleanup_method: "capture-delete",
  });
  await diagnostics.getByRole("button", { name: "停止", exact: true }).click();
  await waitForCondition(
    async () => (await automation.get(`/api/v1/captures/${capture.id}`)).json(),
    (item: { state: string }) =>
      !["starting", "running", "stopping"].includes(item.state),
    "capture stopped",
    30_000,
  );

  await page.getByRole("tab", { name: "流量过滤" }).click();
  await page.getByLabel("pcap 过滤表达式").fill("icmp");
  await page.getByLabel("最大记录数").fill("20");
  await diagnostics.getByRole("button", { name: "启动", exact: true }).click();
  const filterEntries = await waitForCondition(
    async () =>
      (
        await automation.get(
          `/api/v1/traffic-filters?laboratory_id=${laboratory.id}`,
        )
      ).json(),
    (
      items: Array<{
        traffic_filter: { id: string; network_object_link_ids?: string[] };
      }>,
    ) =>
      items
        .map((item) => item.traffic_filter)
        .some((item) => item.network_object_link_ids?.includes(primary.id)),
    "selected object-link Traffic Filter",
    30_000,
  );
  const filter = filterEntries
    .map((item) => item.traffic_filter)
    .find((item) => item.network_object_link_ids?.includes(primary.id))!;
  expect(filter.network_object_link_ids).toEqual([primary.id]);
  expect(filter.network_object_link_ids).not.toContain(parallel.id);
  await ledger.add({
    resource_type: "traffic_filter",
    resource_id: filter.id,
    laboratory_id: laboratory.id,
    cleanup_method: "traffic-filter-delete",
  });
  await diagnostics.getByRole("button", { name: "停止", exact: true }).click();

  interactionResults.push(
    result(
      "network-object-links.observability-isolation",
      testInfo.project.use.viewport!,
      "capture and Traffic Filter retained the selected stable object-link identity across two parallel links",
      [primary.id, parallel.id, capture.id, filter.id],
      "keyboard",
    ),
  );
});

async function createL2Object(
  request: Parameters<typeof createOwnedLaboratory>[1],
  laboratoryId: string,
  name: string,
  ports: string[],
) {
  const laboratory = await (
    await request.get(`/api/v1/labs/${laboratoryId}`)
  ).json();
  const response = await request.post(
    `/api/v1/labs/${laboratoryId}/network-objects`,
    {
      headers: {
        "If-Match": String(laboratory.laboratory.revision),
        "Idempotency-Key": crypto.randomUUID(),
      },
      data: {
        name,
        kind: "switch_l2",
        config: {
          vlan_filtering: false,
          ports: ports.map((portName) => ({ name: portName })),
        },
      },
    },
  );
  expect(response.ok()).toBeTruthy();
  const body = await response.json();
  return {
    ...body.network_object,
    taskId: body.task.id,
  } as {
    id: string;
    name: string;
    revision: number;
    taskId: string;
  };
}

async function createObjectLink(
  request: Parameters<typeof createOwnedLaboratory>[1],
  laboratoryId: string,
  objectAId: string,
  portAName: string,
  objectBId: string,
  portBName: string,
  idempotencyKey: string,
) {
  const response = await request.post(
    `/api/v1/labs/${laboratoryId}/network-object-links`,
    {
      headers: { "Idempotency-Key": idempotencyKey },
      data: {
        object_a_id: objectAId,
        port_a_name: portAName,
        object_b_id: objectBId,
        port_b_name: portBName,
      },
    },
  );
  expect(response.ok()).toBeTruthy();
  const body = await response.json();
  return { ...body.network_object_link, taskId: body.task.id } as {
    id: string;
    taskId: string;
  };
}

async function focusTopologyResource(canvas: Locator, resourceId: string) {
  const announcement = canvas.getByRole("status");
  for (let attempt = 0; attempt < 20; attempt += 1) {
    await canvas.press("ArrowRight");
    if ((await announcement.textContent())?.includes(resourceId)) return;
  }
  throw new Error(`Unable to focus topology resource ${resourceId}`);
}
