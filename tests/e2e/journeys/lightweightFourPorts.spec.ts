import { expect, test } from "../fixtures/acceptanceFixture";
import { waitForCondition } from "../fixtures/waiters";
import { TopologyPage } from "../pages/TopologyPage";
import { createOwnedLaboratory, result } from "./completeRealJourney";

test("SPA HTTP and MCP create four-port lightweight switches without expanding explicit legacy ports", async ({
  page,
  automation,
  ledger,
  runId,
  interactionResults,
}, testInfo) => {
  test.skip(
    process.env.NETLAB_ACCEPTANCE_PROFILE !== "target-host",
    "requires target-host lightweight network runtime",
  );
  const { laboratory } = await createOwnedLaboratory(
    page,
    automation,
    ledger,
    runId,
  );
  const topology = new TopologyPage(page, automation);
  const drawer = await topology.openResourceDrawer();
  const form = await topology.chooseDrawerResource("Lightweight L2 Switch");
  const spaName = `spa-l2-${runId.slice(0, 6)}`;
  await form.getByLabel("名称", { exact: true }).fill(spaName);
  await form.getByRole("button", { name: "添加到拓扑" }).click();
  await expect(drawer).toBeHidden();

  const spaObject = await networkObjectByName(
    automation,
    laboratory.id,
    spaName,
  );
  expect(portNames(spaObject)).toEqual(["eth0", "eth1", "eth2", "eth3"]);

  let snapshot = await (
    await automation.get(`/api/v1/labs/${laboratory.id}`)
  ).json();
  const httpResponse = await automation.post(
    `/api/v1/labs/${laboratory.id}/network-objects`,
    {
      headers: {
        "If-Match": String(snapshot.laboratory.revision),
        "Idempotency-Key": crypto.randomUUID(),
      },
      data: { name: `http-l3-${runId.slice(0, 6)}`, kind: "switch_l3" },
    },
  );
  expect(httpResponse.status()).toBe(202);
  const httpEnvelope = await httpResponse.json();
  expect(portNames(httpEnvelope.network_object)).toEqual([
    "eth0",
    "eth1",
    "eth2",
    "eth3",
  ]);

  snapshot = await (
    await automation.get(`/api/v1/labs/${laboratory.id}`)
  ).json();
  const mcpResponse = await automation.post("/mcp", {
    headers: { Accept: "application/json" },
    data: {
      jsonrpc: "2.0",
      id: 1,
      method: "tools/call",
      params: {
        name: "netlab.network_objects.create",
        arguments: {
          lab_id: laboratory.id,
          expected_revision: snapshot.laboratory.revision,
          idempotency_key: crypto.randomUUID(),
          name: `mcp-l2-${runId.slice(0, 6)}`,
          kind: "switch_l2",
        },
      },
    },
  });
  expect(mcpResponse.ok()).toBeTruthy();
  expect(JSON.stringify(await mcpResponse.json())).toContain('"name":"eth3"');

  snapshot = await (
    await automation.get(`/api/v1/labs/${laboratory.id}`)
  ).json();
  const legacyResponse = await automation.post(
    `/api/v1/labs/${laboratory.id}/network-objects`,
    {
      headers: {
        "If-Match": String(snapshot.laboratory.revision),
        "Idempotency-Key": crypto.randomUUID(),
      },
      data: {
        name: `legacy-l2-${runId.slice(0, 6)}`,
        kind: "switch_l2",
        config: {
          vlan_filtering: true,
          ports: [{ name: "lan0", pvid: 1, tagged: [] }],
        },
      },
    },
  );
  expect(legacyResponse.status()).toBe(202);
  const legacy = (await legacyResponse.json()).network_object;
  expect(portNames(legacy)).toEqual(["lan0"]);
  await page.reload();
  expect(
    portNames(
      await networkObjectByName(automation, laboratory.id, legacy.name),
    ),
  ).toEqual(["lan0"]);

  const endpoints = await (
    await automation.get(`/api/v1/labs/${laboratory.id}/connections`)
  ).json();
  const spaPorts = endpoints.endpoints.filter(
    (endpoint: { resource_id: string }) =>
      endpoint.resource_id === spaObject.id,
  );
  expect(
    spaPorts.map((endpoint: { port_name: string }) => endpoint.port_name),
  ).toEqual(["eth0", "eth1", "eth2", "eth3"]);

  for (const object of [spaObject, httpEnvelope.network_object, legacy])
    await ledger.add({
      resource_type: "network_object",
      resource_id: object.id,
      laboratory_id: laboratory.id,
      revision: object.revision,
      cleanup_method: "laboratory-cascade",
    });
  interactionResults.push(
    result(
      "topology.lightweight-four-ports",
      testInfo.project.use.viewport!,
      "SPA, HTTP, and MCP defaults exposed four independent ports while explicit legacy config stayed unchanged",
      [spaObject.id, httpEnvelope.network_object.id, legacy.id],
    ),
  );
});

async function networkObjectByName(
  request: Parameters<typeof createOwnedLaboratory>[1],
  laboratoryId: string,
  name: string,
) {
  return waitForCondition(
    async () => {
      const snapshot = await (
        await request.get(`/api/v1/labs/${laboratoryId}`)
      ).json();
      return snapshot.network_objects.find(
        (item: { name: string }) => item.name === name,
      );
    },
    (
      value,
    ): value is {
      id: string;
      name: string;
      revision: number;
      config: Record<string, unknown>;
    } => Boolean(value),
    `network object ${name}`,
  );
}

function portNames(object: { config: Record<string, unknown> }) {
  const values = (object.config.ports ||
    object.config.interfaces ||
    []) as Array<{
    name: string;
  }>;
  return values.map((item) => item.name);
}
