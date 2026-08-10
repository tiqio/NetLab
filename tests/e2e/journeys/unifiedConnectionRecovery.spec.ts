import { promisify } from "node:util";
import { exec as execCallback } from "node:child_process";
import { expect, test } from "../fixtures/acceptanceFixture";
import { waitForCondition } from "../fixtures/waiters";
import {
  createOwnedLaboratory,
  createOwnedLightweightPair,
  result,
} from "./completeRealJourney";
const exec = promisify(execCallback);
const restartCommand = process.env.NETLAB_ACCEPTANCE_RESTART_COMMAND;

test("unified connection tasks recover across cancellation restart and laboratory deletion", async ({
  page,
  automation,
  ledger,
  runId,
  interactionResults,
}, testInfo) => {
  test.skip(
    process.env.NETLAB_ACCEPTANCE_PROFILE !== "target-host",
    "requires target-host connection runtime",
  );
  test.skip(!restartCommand, "NETLAB_ACCEPTANCE_RESTART_COMMAND is required");
  const { laboratory } = await createOwnedLaboratory(
    page,
    automation,
    ledger,
    runId,
  );
  const { first, second } = await createOwnedLightweightPair(
    page,
    automation,
    ledger,
    laboratory.id,
  );
  let snapshot = await (
    await automation.get(`/api/v1/labs/${laboratory.id}/connections`)
  ).json();
  const source = snapshot.endpoints.find(
    (item: { resource_id: string; port_name?: string }) =>
      item.resource_id === first.id && item.port_name === "eth0",
  );
  const target = snapshot.endpoints.find(
    (item: { resource_id: string; port_name?: string }) =>
      item.resource_id === second.id && item.port_name === "eth0",
  );
  const revision = (
    await (await automation.get(`/api/v1/labs/${laboratory.id}`)).json()
  ).laboratory.revision;
  const create = await automation.post(
    `/api/v1/labs/${laboratory.id}/connections`,
    {
      headers: {
        "If-Match": String(revision),
        "Idempotency-Key": `${runId}-recovery-create`,
      },
      data: { source, target },
    },
  );
  expect(create.status()).toBe(202);
  const envelope = await create.json();
  const cancellation = await automation.post(
    `/api/v1/tasks/${envelope.task.id}/cancel`,
  );
  expect([200, 409]).toContain(cancellation.status());
  await waitForCondition(
    async () =>
      (await automation.get(`/api/v1/tasks/${envelope.task.id}`)).json(),
    (task: { state?: string }) =>
      ["succeeded", "failed", "cancelled"].includes(task.state || ""),
    "connection task terminal state",
  );
  await exec(restartCommand!);
  await waitForCondition(
    async () => automation.get("/api/v1/capabilities"),
    (response) => response.ok(),
    "service restart readiness",
  );
  snapshot = await (
    await automation.get(`/api/v1/labs/${laboratory.id}/connections`)
  ).json();
  expect(snapshot.connections.length).toBeLessThanOrEqual(1);
  const lab = await (
    await automation.get(`/api/v1/labs/${laboratory.id}`)
  ).json();
  const deletion = await automation.delete(`/api/v1/labs/${laboratory.id}`, {
    headers: {
      "If-Match": String(lab.laboratory.revision),
      "Idempotency-Key": `${runId}-recovery-lab-delete`,
    },
  });
  expect(deletion.status()).toBe(202);
  await waitForCondition(
    async () => automation.get(`/api/v1/labs/${laboratory.id}`),
    (response) => response.status() === 404,
    "laboratory cleanup",
  );
  const ownership = (await (
    await automation.get("/api/v1/runtime-ownership")
  ).json()) as Array<{ resource_id: string }>;
  const ownedResourceIds = new Set([
    first.id,
    second.id,
    envelope.connection.id,
  ]);
  expect(
    ownership.filter((item) => ownedResourceIds.has(item.resource_id)),
  ).toHaveLength(0);
  interactionResults.push(
    result(
      "topology.connection.recovery-cleanup",
      testInfo.project.use.viewport!,
      "task reached an authoritative terminal state, restart recovered connection truth, and laboratory deletion removed visible resources",
      [laboratory.id],
    ),
  );
});
