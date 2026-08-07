import { promisify } from "node:util";
import { exec as execCallback } from "node:child_process";
import { expect, test } from "../fixtures/acceptanceFixture";
import { createOwnedLaboratory } from "./completeRealJourney";
import { selectLaboratoryByName } from "../pages/LaboratoryPage";

const exec = promisify(execCallback);
const restartCommand = process.env.NETLAB_ACCEPTANCE_RESTART_COMMAND;

test("authoritative placements and connection visuals survive service restart and cleanup", async ({
  page,
  automation,
  ledger,
  runId,
}) => {
  test.skip(!restartCommand, "NETLAB_ACCEPTANCE_RESTART_COMMAND is required");
  const { laboratory } = await createOwnedLaboratory(
    page,
    automation,
    ledger,
    runId,
  );
  let revision = laboratory.revision;
  const objects = [];
  for (const [index, name] of ["recovery-a", "recovery-b"].entries()) {
    const response = await automation.post(
      `/api/v1/labs/${laboratory.id}/network-objects`,
      {
        headers: {
          "If-Match": String(revision),
          "Idempotency-Key": `${runId}-recovery-object-${index}`,
        },
        data: {
          name,
          kind: "switch_l2",
          config: {
            vlan_filtering: true,
            ports: [{ name: "eth0", mode: "access", access_vlan: 1 }],
          },
          placement_intent: {
            preferred_x: index * 320,
            preferred_y: 160,
            footprint_class: "network-object-standard",
          },
        },
      },
    );
    expect(response.ok()).toBeTruthy();
    const body = await response.json();
    objects.push(body.network_object);
    revision = body.laboratory_revision;
  }
  const linkResponse = await automation.post(
    `/api/v1/labs/${laboratory.id}/network-object-links`,
    {
      headers: { "Idempotency-Key": `${runId}-recovery-link` },
      data: {
        object_a_id: objects[0].id,
        port_a_name: "eth0",
        object_b_id: objects[1].id,
        port_b_name: "eth0",
      },
    },
  );
  expect(linkResponse.ok()).toBeTruthy();
  const link = (await linkResponse.json()).network_object_link;

  const beforeResponse = await automation.get(`/api/v1/labs/${laboratory.id}`);
  const before = await beforeResponse.json();
  const placementSummary = Object.fromEntries(
    before.placements.map(
      (item: {
        resource_id: string;
        x: number;
        y: number;
        revision: number;
      }) => [item.resource_id, [item.x, item.y, item.revision]],
    ),
  );
  const linkSummary = before.network_object_links.map(
    (item: {
      id: string;
      object_a_id: string;
      port_a_name: string;
      object_b_id: string;
      port_b_name: string;
      desired_state: string;
    }) => ({
      id: item.id,
      object_a_id: item.object_a_id,
      port_a_name: item.port_a_name,
      object_b_id: item.object_b_id,
      port_b_name: item.port_b_name,
      desired_state: item.desired_state,
    }),
  );

  await exec(restartCommand!);
  await expect
    .poll(async () => (await automation.get("/healthz")).ok(), {
      timeout: 30_000,
    })
    .toBe(true);
  const afterResponse = await automation.get(`/api/v1/labs/${laboratory.id}`);
  expect(afterResponse.ok()).toBeTruthy();
  const after = await afterResponse.json();
  expect(
    Object.fromEntries(
      after.placements.map(
        (item: {
          resource_id: string;
          x: number;
          y: number;
          revision: number;
        }) => [item.resource_id, [item.x, item.y, item.revision]],
      ),
    ),
  ).toEqual(placementSummary);
  expect(
    after.network_object_links.map(
      (item: {
        id: string;
        object_a_id: string;
        port_a_name: string;
        object_b_id: string;
        port_b_name: string;
        desired_state: string;
      }) => ({
        id: item.id,
        object_a_id: item.object_a_id,
        port_a_name: item.port_a_name,
        object_b_id: item.object_b_id,
        port_b_name: item.port_b_name,
        desired_state: item.desired_state,
      }),
    ),
  ).toEqual(linkSummary);

  await page.goto("/");
  await selectLaboratoryByName(page, laboratory.name);
  await expect(page.getByTestId("topology-a11y-summary")).toContainText(
    /recovery-a:eth0 ↔ recovery-b:eth0|recovery-b:eth0 ↔ recovery-a:eth0/,
  );

  const currentResponse = await automation.get(`/api/v1/labs/${laboratory.id}`);
  const current = await currentResponse.json();
  const deletion = await automation.delete(`/api/v1/labs/${laboratory.id}`, {
    headers: {
      "If-Match": String(current.laboratory.revision),
      "Idempotency-Key": `${runId}-recovery-delete`,
    },
  });
  expect(deletion.ok()).toBeTruthy();
  const taskId = (await deletion.json()).task.id;
  await expect
    .poll(
      async () => {
        const taskResponse = await automation.get(`/api/v1/tasks/${taskId}`);
        return (await taskResponse.json()).state;
      },
      { timeout: 30_000 },
    )
    .toBe("succeeded");
  await expect
    .poll(
      async () =>
        (await automation.get(`/api/v1/labs/${laboratory.id}`)).status(),
      {
        timeout: 30_000,
      },
    )
    .toBe(404);
  expect(link.id).toBeTruthy();
});
