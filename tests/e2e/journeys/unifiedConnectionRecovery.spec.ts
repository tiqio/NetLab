import { promisify } from "node:util";
import { exec as execCallback } from "node:child_process";
import { expect, test } from "../fixtures/acceptanceFixture";
import { waitForCondition } from "../fixtures/waiters";
import {
  createOwnedLaboratory,
  createOwnedLightweightPair,
  resolveTemplateSelection,
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

test("all unified connection backings preserve identity reservations and ownership across restart", async ({
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
  const { templates, first, second } = await createOwnedLightweightPair(
    page,
    automation,
    ledger,
    laboratory.id,
  );
  const selection = await resolveTemplateSelection(
    automation,
    "busybox-container",
  );
  const nodes = [];
  for (const suffix of ["link-a", "link-b", "attachment"] as const) {
    const node = await templates.createDevice({
      displayName: selection.displayName,
      runtime: selection.runtime,
      nodeName: `${runId.slice(0, 6)}-${suffix}`,
      templateId: selection.templateId,
      versionId: selection.versionId,
      imageId: selection.imageId,
      interfaces: 1,
      laboratoryId: laboratory.id,
    });
    nodes.push(node);
    await ledger.add({
      resource_type: "node",
      resource_id: node.id,
      laboratory_id: laboratory.id,
      revision: node.revision,
      cleanup_method: "laboratory-cascade",
    });
  }
  for (const node of nodes) {
    const start = await automation.put(`/api/v1/nodes/${node.id}/state`, {
      headers: {
        "If-Match": String(node.revision),
        "Idempotency-Key": `${runId}-start-${node.id}`,
      },
      data: { desired_state: "running" },
    });
    expect(start.status()).toBe(202);
    await waitForCondition(
      async () => (await automation.get(`/api/v1/nodes/${node.id}`)).json(),
      (value: { observed_state?: string }) =>
        value.observed_state === "running",
      `node ${node.id} running`,
      60_000,
    );
  }
  const topology = await (
    await automation.get(`/api/v1/labs/${laboratory.id}`)
  ).json();
  const interfaces = new Map(
    topology.interfaces.map((item: { node_id: string; id: string }) => [
      item.node_id,
      item.id,
    ]),
  );
  const createConnection = async (
    key: string,
    sourceMatcher: (endpoint: Record<string, unknown>) => boolean,
    targetMatcher: (endpoint: Record<string, unknown>) => boolean,
  ) => {
    const current = await (
      await automation.get(`/api/v1/labs/${laboratory.id}/connections`)
    ).json();
    const source = current.endpoints.find(sourceMatcher);
    const target = current.endpoints.find(targetMatcher);
    expect(source, `${key} source endpoint`).toBeTruthy();
    expect(target, `${key} target endpoint`).toBeTruthy();
    const response = await automation.post(
      `/api/v1/labs/${laboratory.id}/connections`,
      {
        headers: {
          "If-Match": String(current.laboratory_revision),
          "Idempotency-Key": `${runId}-${key}`,
        },
        data: { source, target },
      },
    );
    expect(response.status()).toBe(202);
    const envelope = await response.json();
    return waitForCondition(
      async () =>
        (
          await automation.get(`/api/v1/connections/${envelope.connection.id}`)
        ).json(),
      (value: { observed_state?: string }) =>
        value.observed_state === "connected",
      `${key} connected`,
      60_000,
    );
  };
  const nodeLink = await createConnection(
    "link",
    (endpoint) => endpoint.port_id === interfaces.get(nodes[0].id),
    (endpoint) => endpoint.port_id === interfaces.get(nodes[1].id),
  );
  const attachment = await createConnection(
    "attachment",
    (endpoint) => endpoint.port_id === interfaces.get(nodes[2].id),
    (endpoint) =>
      endpoint.resource_id === first.id && endpoint.port_name === "eth1",
  );
  const objectLink = await createConnection(
    "object-link",
    (endpoint) =>
      endpoint.resource_id === first.id && endpoint.port_name === "eth0",
    (endpoint) =>
      endpoint.resource_id === second.id && endpoint.port_name === "eth0",
  );
  const beforeRestart = [nodeLink, attachment, objectLink] as Array<{
    id: string;
    backing_kind: string;
    observed_state: string;
  }>;
  expect(new Set(beforeRestart.map((item) => item.backing_kind))).toEqual(
    new Set(["link", "network_attachment", "network_object_link"]),
  );

  await exec(restartCommand!);
  await waitForCondition(
    async () => automation.get("/api/v1/capabilities"),
    (response) => response.ok(),
    "service restart readiness",
  );
  const recovered = await waitForCondition(
    async () =>
      (
        await automation.get(`/api/v1/labs/${laboratory.id}/connections`)
      ).json(),
    (value: { connections?: unknown[] }) => value.connections?.length === 3,
    "three backing recovery",
    60_000,
  );
  const recoveredByID = new Map(
    recovered.connections.map((item: { id: string }) => [item.id, item]),
  );
  for (const expected of beforeRestart) {
    const value = recoveredByID.get(expected.id) as
      { backing_kind: string; observed_state: string } | undefined;
    expect(value, `recovered ${expected.id}`).toBeTruthy();
    expect(value?.backing_kind).toBe(expected.backing_kind);
    expect(value?.observed_state).toBe("connected");
  }
  const occupiedEndpointCount = recovered.endpoints.filter(
    (endpoint: { availability?: string }) =>
      endpoint.availability === "occupied",
  ).length;
  expect(occupiedEndpointCount).toBeGreaterThanOrEqual(6);
  const recoveredSnapshot = await (
    await automation.get(`/api/v1/labs/${laboratory.id}`)
  ).json();
  expect(
    recoveredSnapshot.links.some((item: { id: string }) =>
      beforeRestart.some(
        (connection) =>
          connection.id === item.id && connection.backing_kind === "link",
      ),
    ),
  ).toBe(true);
  expect(
    recoveredSnapshot.network_attachments.some((item: { id: string }) =>
      beforeRestart.some(
        (connection) =>
          connection.id === item.id &&
          connection.backing_kind === "network_attachment",
      ),
    ),
  ).toBe(true);
  expect(
    recoveredSnapshot.network_object_links.some((item: { id: string }) =>
      beforeRestart.some(
        (connection) =>
          connection.id === item.id &&
          connection.backing_kind === "network_object_link",
      ),
    ),
  ).toBe(true);
  const ownership = (await (
    await automation.get("/api/v1/runtime-ownership")
  ).json()) as Array<{ resource_id: string }>;
  for (const connection of beforeRestart) {
    expect(
      ownership.some((record) => record.resource_id === connection.id),
      `runtime ownership for ${connection.id}`,
    ).toBe(true);
  }

  const latestLab = await (
    await automation.get(`/api/v1/labs/${laboratory.id}`)
  ).json();
  const deletion = await automation.delete(`/api/v1/labs/${laboratory.id}`, {
    headers: {
      "If-Match": String(latestLab.laboratory.revision),
      "Idempotency-Key": `${runId}-strict-recovery-delete`,
    },
  });
  expect(deletion.status()).toBe(202);
  await waitForCondition(
    async () => automation.get(`/api/v1/labs/${laboratory.id}`),
    (response) => response.status() === 404,
    "strict recovery laboratory cleanup",
    60_000,
  );
  const ownershipAfterDelete = (await (
    await automation.get("/api/v1/runtime-ownership")
  ).json()) as Array<{ resource_id: string }>;
  const ownedIDs = new Set([
    ...nodes.map((node) => node.id),
    first.id,
    second.id,
    ...beforeRestart.map((connection) => connection.id),
  ]);
  expect(
    ownershipAfterDelete.filter((record) => ownedIDs.has(record.resource_id)),
  ).toHaveLength(0);
  interactionResults.push(
    result(
      "topology.connection.three-backing-recovery-cleanup",
      testInfo.project.use.viewport!,
      "link, attachment, and object-link identities, occupied endpoints, backing records, and runtime ownership recovered before laboratory deletion removed all ownership",
      [laboratory.id, ...beforeRestart.map((connection) => connection.id)],
    ),
  );
});
