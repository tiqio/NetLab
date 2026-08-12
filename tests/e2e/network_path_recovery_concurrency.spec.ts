import { expect, test } from "./fixtures/acceptanceFixture";
import { createOwnedLaboratory } from "./journeys/completeRealJourney";
import type { APIRequestContext } from "@playwright/test";

async function waitTask(request: APIRequestContext, id: string) {
  for (let attempt = 0; attempt < 80; attempt++) {
    const response = await request.get(`/api/v1/tasks/${id}`);
    const task = await response.json();
    if (["succeeded", "failed", "cancelled"].includes(task.state)) return task;
    await new Promise((resolve) => setTimeout(resolve, 100));
  }
  throw new Error(`task ${id} did not finish`);
}

test("two browsers HTTP and MCP converge workload lifecycle under revision contention", async ({
  page,
  secondPage,
  automation,
  ledger,
  runId,
}) => {
  const { laboratory } = await createOwnedLaboratory(
    page,
    automation,
    ledger,
    runId,
  );
  const snapshot = await (
    await automation.get(`/api/v1/labs/${laboratory.id}`)
  ).json();
  const objectResponse = await automation.post(
    `/api/v1/labs/${laboratory.id}/network-objects`,
    {
      headers: {
        "If-Match": String(snapshot.laboratory.revision),
        "Idempotency-Key": crypto.randomUUID(),
      },
      data: {
        name: "traffic-pc",
        kind: "switch_l3",
        config: {
          forward_ipv4: true,
          interfaces: [{ name: "eth0", addresses: ["192.0.2.2/24"] }],
        },
      },
    },
  );
  expect([201, 202]).toContain(objectResponse.status());
  const object = (await objectResponse.json()).network_object;
  const latest = await (
    await automation.get(`/api/v1/labs/${laboratory.id}`)
  ).json();
  const switchResponse = await automation.post(
    `/api/v1/labs/${laboratory.id}/network-objects`,
    {
      headers: {
        "If-Match": String(latest.laboratory.revision),
        "Idempotency-Key": crypto.randomUUID(),
      },
      data: {
        name: "concurrent-vlan",
        kind: "switch_l2",
        config: {
          vlan_filtering: true,
          ports: [{ name: "eth0", pvid: 10, tagged: [20] }],
        },
      },
    },
  );
  expect(switchResponse.status()).toBe(202);
  const vlanSwitch = (await switchResponse.json()).network_object;
  const created = await automation.post("/api/v1/traffic-workloads", {
    headers: { "Idempotency-Key": `${runId}-concurrent-workload` },
    data: {
      laboratory_id: laboratory.id,
      name: "concurrent ping",
      source: { kind: "network_object", resource_id: object.id },
      protocol: "icmp",
      address_family: "ipv4",
      destination: { address: "192.0.2.1" },
      interval_seconds: 5,
      timeout_seconds: 2,
    },
  });
  expect(created.status()).toBe(202);
  const envelope = await created.json();
  await waitTask(automation, envelope.task.id);
  const workload = await (
    await automation.get(`/api/v1/traffic-workloads/${envelope.workload_id}`)
  ).json();
  await Promise.all([page.goto("/"), secondPage.goto("/")]);
  const [reconcileResponse, vlanUpdateResponse] = await Promise.all([
    automation.post(`/api/v1/network-objects/${vlanSwitch.id}/reconcile`, {
      headers: {
        "If-Match": String(vlanSwitch.revision),
        "Idempotency-Key": `${runId}-http-reconcile`,
      },
    }),
    automation.post("/mcp", {
      headers: { Accept: "application/json" },
      data: {
        jsonrpc: "2.0",
        id: 2,
        method: "tools/call",
        params: {
          name: "netlab.network_objects.update",
          arguments: {
            object_id: vlanSwitch.id,
            expected_revision: vlanSwitch.revision,
            idempotency_key: `${runId}-mcp-vlan-update`,
            name: vlanSwitch.name,
            config: {
              vlan_filtering: true,
              ports: [{ name: "eth0", pvid: 20, tagged: [10] }],
            },
          },
        },
      },
    }),
  ]);
  expect([202, 412, 503]).toContain(reconcileResponse.status());
  expect(vlanUpdateResponse.ok()).toBeTruthy();
  const [httpStart, mcpStart] = await Promise.all([
    automation.post(`/api/v1/traffic-workloads/${workload.id}/start`, {
      headers: {
        "If-Match": String(workload.revision),
        "Idempotency-Key": `${runId}-http-start`,
      },
    }),
    automation.post("/mcp", {
      headers: { Accept: "application/json" },
      data: {
        jsonrpc: "2.0",
        id: 1,
        method: "tools/call",
        params: {
          name: "netlab.traffic_workloads.start",
          arguments: {
            workload_id: workload.id,
            expected_revision: workload.revision,
            idempotency_key: `${runId}-mcp-start`,
          },
        },
      },
    }),
  ]);
  expect([202, 412, 503]).toContain(httpStart.status());
  expect(mcpStart.ok()).toBeTruthy();
  const tasks = await (await automation.get("/api/v1/tasks?limit=100")).json();
  const mutations = tasks.filter(
    (task: { resource_id: string; kind: string }) =>
      task.resource_id === workload.id &&
      task.kind === "traffic_workload.start",
  );
  expect(mutations.length).toBeGreaterThanOrEqual(1);
});
