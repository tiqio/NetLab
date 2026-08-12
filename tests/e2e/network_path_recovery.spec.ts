import { expect, test } from "./fixtures/acceptanceFixture";
import { createOwnedLaboratory } from "./journeys/completeRealJourney";

test("temporary laboratory covers recovery VLAN roles workload counters and cleanup", async ({
  page,
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
  let snapshot = await (
    await automation.get(`/api/v1/labs/${laboratory.id}`)
  ).json();
  const l3 = await automation.post(
    `/api/v1/labs/${laboratory.id}/network-objects`,
    {
      headers: {
        "If-Match": String(snapshot.laboratory.revision),
        "Idempotency-Key": crypto.randomUUID(),
      },
      data: {
        name: "journey-l3",
        kind: "switch_l3",
        config: {
          forward_ipv4: true,
          forward_ipv6: true,
          interfaces: [
            { name: "eth0", addresses: ["10.10.10.1/24", "fd10::1/64"] },
          ],
        },
      },
    },
  );
  expect(l3.status()).toBe(202);
  const l3Object = (await l3.json()).network_object;
  const reconcileResponse = await automation.post(
    `/api/v1/network-objects/${l3Object.id}/reconcile`,
    {
      headers: {
        "If-Match": String(l3Object.revision),
        "Idempotency-Key": crypto.randomUUID(),
      },
    },
  );
  expect([202, 503]).toContain(reconcileResponse.status());
  snapshot = await (
    await automation.get(`/api/v1/labs/${laboratory.id}`)
  ).json();
  const l2 = await automation.post(
    `/api/v1/labs/${laboratory.id}/network-objects`,
    {
      headers: {
        "If-Match": String(snapshot.laboratory.revision),
        "Idempotency-Key": crypto.randomUUID(),
      },
      data: {
        name: "journey-l2",
        kind: "switch_l2",
        config: {
          vlan_filtering: true,
          ports: [
            { name: "eth0", pvid: 10, tagged: [20] },
            { name: "eth1", pvid: 20, tagged: [10] },
          ],
        },
      },
    },
  );
  expect(l2.status()).toBe(202);
  snapshot = await (
    await automation.get(`/api/v1/labs/${laboratory.id}`)
  ).json();
  const node = await automation.post(`/api/v1/labs/${laboratory.id}/nodes`, {
    headers: {
      "If-Match": String(snapshot.laboratory.revision),
      "Idempotency-Key": crypto.randomUUID(),
    },
    data: {
      name: "journey-docker",
      kind: "docker",
      interface_count: 2,
      config: {
        forward_ipv4: true,
        forward_ipv6: true,
        device_roles: [
          { interface_id: "eth0", role: "wan" },
          { interface_id: "eth1", role: "lan" },
        ],
      },
    },
  });
  expect([201, 202]).toContain(node.status());
  const nodeBody = await node.json();
  const nodeValue = nodeBody.node || nodeBody;
  const readiness = await automation.get(
    `/api/v1/nodes/${nodeValue.id}/device-readiness`,
  );
  expect(readiness.ok()).toBeTruthy();
  const source = l3Object;
  const workloadResponse = await automation.post("/api/v1/traffic-workloads", {
    data: {
      laboratory_id: laboratory.id,
      name: "journey ping",
      source: { kind: "network_object", resource_id: source.id },
      protocol: "icmp",
      address_family: "ipv4",
      destination: { address: "192.0.2.1" },
      interval_seconds: 5,
      timeout_seconds: 2,
    },
  });
  expect(workloadResponse.status()).toBe(202);
  await page.goto("/");
  await page.getByRole("tab", { name: "稳定流量" }).click();
  await expect(page.getByText("journey ping")).toBeVisible();
  const exported = await automation.post(
    `/api/v1/labs/${laboratory.id}/exports`,
  );
  expect(exported.status()).toBe(202);
});
