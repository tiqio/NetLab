import { expect, test } from "../fixtures/acceptanceFixture";
import { waitForCondition, waitForTask } from "../fixtures/waiters";
import { selectLaboratoryByName } from "../pages/LaboratoryPage";
import { createOwnedLaboratory, result } from "./completeRealJourney";

test("live object-link deletion converges across clients and releases ports", async ({
  page,
  secondPage,
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
  const objectA = await createObject(
    automation,
    laboratory.id,
    `delete-a-${suffix}`,
  );
  const objectB = await createObject(
    automation,
    laboratory.id,
    `delete-b-${suffix}`,
  );
  for (const object of [objectA, objectB]) {
    expect((await waitForTask(automation, object.taskId)).terminal_state).toBe(
      "succeeded",
    );
    await ledger.add({
      resource_type: "network_object",
      resource_id: object.id,
      laboratory_id: laboratory.id,
      revision: object.revision,
      cleanup_method: "laboratory-cascade",
    });
  }
  const link = await createLink(
    automation,
    laboratory.id,
    objectA.id,
    objectB.id,
  );
  expect((await waitForTask(automation, link.taskId)).terminal_state).toBe(
    "succeeded",
  );
  await secondPage.goto("/");
  await selectLaboratoryByName(secondPage, laboratory.name);
  const label = `${objectA.name}:swp1 ↔ ${objectB.name}:swp1`;
  await expect(page.getByTestId("topology-a11y-summary")).toContainText(label);
  await expect(secondPage.getByTestId("topology-a11y-summary")).toContainText(
    label,
  );

  const captureResponse = await automation.post("/api/v1/captures", {
    headers: { "Idempotency-Key": crypto.randomUUID() },
    data: {
      laboratory_id: laboratory.id,
      source_type: "network_object_link",
      source_id: link.id,
      format: "pcap",
      retain: true,
      max_bytes: 1048576,
    },
  });
  expect(captureResponse.ok()).toBeTruthy();
  const captureEnvelope = await captureResponse.json();
  await ledger.add({
    resource_type: "capture",
    resource_id: captureEnvelope.capture.id,
    laboratory_id: laboratory.id,
    cleanup_method: "capture-delete",
  });

  const deletion = await automation.delete(
    `/api/v1/network-object-links/${link.id}`,
    {
      headers: {
        "If-Match": String(link.revision),
        "Idempotency-Key": crypto.randomUUID(),
      },
    },
  );
  expect(deletion.status()).toBe(202);
  const deletionEnvelope = await deletion.json();
  expect(
    (await waitForTask(automation, deletionEnvelope.task.id)).terminal_state,
  ).toBe("succeeded");
  await expect(page.getByTestId("topology-a11y-summary")).not.toContainText(
    label,
  );
  await expect(
    secondPage.getByTestId("topology-a11y-summary"),
  ).not.toContainText(label);
  await Promise.all([page.reload(), secondPage.reload()]);
  await expect(page.getByTestId("topology-a11y-summary")).not.toContainText(
    label,
  );
  await expect(
    secondPage.getByTestId("topology-a11y-summary"),
  ).not.toContainText(label);

  const capture = await waitForCondition(
    async () =>
      (
        await automation.get(`/api/v1/captures/${captureEnvelope.capture.id}`)
      ).json(),
    (value: { completion_reason?: string }) =>
      value.completion_reason === "link_deleted",
    "capture completed by object-link deletion",
    30_000,
  );
  expect(capture.completion_reason).toBe("link_deleted");

  const replacement = await createLink(
    automation,
    laboratory.id,
    objectA.id,
    objectB.id,
  );
  expect(
    (await waitForTask(automation, replacement.taskId)).terminal_state,
  ).toBe("succeeded");
  await waitForCondition(
    async () =>
      (
        await automation.get(
          `/api/v1/labs/${laboratory.id}/network-object-links`,
        )
      ).json(),
    (values: Array<{ id: string }>) =>
      values.some((value) => value.id === replacement.id),
    "released object ports reused",
  );

  const objectDeletion = await automation.delete(
    `/api/v1/network-objects/${objectA.id}`,
    {
      headers: {
        "If-Match": String(objectA.revision),
        "Idempotency-Key": crypto.randomUUID(),
      },
    },
  );
  expect(objectDeletion.status()).toBe(202);
  const objectDeletionEnvelope = await objectDeletion.json();
  expect(
    (await waitForTask(automation, objectDeletionEnvelope.task.id))
      .terminal_state,
  ).toBe("succeeded");
  await waitForCondition(
    async () =>
      (
        await automation.get(
          `/api/v1/labs/${laboratory.id}/network-object-links`,
        )
      ).json(),
    (values: Array<{ id: string }> | null) =>
      !(values || []).some((value) => value.id === replacement.id),
    "network-object cascade removed replacement link",
  );

  interactionResults.push(
    result(
      "network-object-links.live-delete",
      testInfo.project.use.viewport!,
      "both browsers removed the live link, capture completed with link_deleted, ports were reused, and object cascade stayed absent after refresh",
      [link.id, replacement.id, captureEnvelope.capture.id],
      "pointer",
    ),
  );
});

async function createObject(
  request: Parameters<typeof createOwnedLaboratory>[1],
  laboratoryId: string,
  name: string,
) {
  const response = await request.post(
    `/api/v1/labs/${laboratoryId}/network-objects`,
    {
      headers: { "Idempotency-Key": crypto.randomUUID() },
      data: {
        name,
        kind: "switch_l2",
        config: { vlan_filtering: false, ports: [{ name: "swp1" }] },
      },
    },
  );
  expect(response.ok()).toBeTruthy();
  const body = await response.json();
  return { ...body.network_object, taskId: body.task.id } as {
    id: string;
    name: string;
    revision: number;
    taskId: string;
  };
}

async function createLink(
  request: Parameters<typeof createOwnedLaboratory>[1],
  laboratoryId: string,
  objectAId: string,
  objectBId: string,
) {
  const response = await request.post(
    `/api/v1/labs/${laboratoryId}/network-object-links`,
    {
      headers: { "Idempotency-Key": crypto.randomUUID() },
      data: {
        object_a_id: objectAId,
        port_a_name: "swp1",
        object_b_id: objectBId,
        port_b_name: "swp1",
      },
    },
  );
  expect(response.ok()).toBeTruthy();
  const body = await response.json();
  return { ...body.network_object_link, taskId: body.task.id } as {
    id: string;
    revision: number;
    taskId: string;
  };
}
