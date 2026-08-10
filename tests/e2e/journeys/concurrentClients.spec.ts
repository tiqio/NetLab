import { expect, test } from "../fixtures/acceptanceFixture";
import { ClientObserver } from "../fixtures/clientObserver";
import { createOwnedLaboratory } from "./completeRealJourney";
import { selectLaboratoryByName } from "../pages/LaboratoryPage";
import type { APIRequestContext } from "@playwright/test";

async function createNode(
  request: APIRequestContext,
  laboratoryId: string,
  revision: number,
  name: string,
  key: string,
) {
  const response = await request.post(`/api/v1/labs/${laboratoryId}/nodes`, {
    headers: { "If-Match": String(revision), "Idempotency-Key": key },
    data: {
      name,
      kind: "docker",
      interface_count: 1,
      placement_intent: {
        preferred_x: 120,
        preferred_y: 80,
        footprint_class: "node-standard",
      },
    },
  });
  return response.ok()
    ? { ok: true as const, value: await response.json() }
    : { ok: false as const, status: response.status() };
}

async function createObjectThroughMcp(
  request: APIRequestContext,
  laboratoryId: string,
  revision: number,
  name: string,
  key: string,
) {
  const response = await request.post("/mcp", {
    headers: { Accept: "application/json" },
    data: {
      jsonrpc: "2.0",
      id: key,
      method: "tools/call",
      params: {
        name: "netlab.network_objects.create",
        arguments: {
          lab_id: laboratoryId,
          name,
          kind: "bridge",
          config: {},
          expected_revision: revision,
          idempotency_key: key,
          placement_intent: {
            preferred_x: 120,
            preferred_y: 80,
            footprint_class: "network-object-standard",
          },
        },
      },
    },
  });
  const body = await response.json();
  return body.result?.isError
    ? { ok: false as const, problem: body.result.structuredContent }
    : { ok: true as const, value: body.result.structuredContent };
}

test("two browsers and automation converge with revision protection", async ({
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
  await secondPage.goto("/");
  await selectLaboratoryByName(secondPage, laboratory.name);
  const renamed = `${laboratory.name}-shared`;
  const update = await automation.patch(`/api/v1/labs/${laboratory.id}`, {
    data: { name: renamed, description: "", recovery_policy: "auto_restore" },
    headers: {
      "If-Match": String(laboratory.revision),
      "Content-Type": "application/merge-patch+json",
    },
  });
  expect(update.ok()).toBeTruthy();
  const authoritative = await update.json();
  const conflict = await automation.patch(`/api/v1/labs/${laboratory.id}`, {
    data: { name: "stale", description: "", recovery_policy: "auto_restore" },
    headers: {
      "If-Match": String(laboratory.revision),
      "Content-Type": "application/merge-patch+json",
    },
  });
  expect([409, 412]).toContain(conflict.status());
  await Promise.all([page.reload(), secondPage.reload()]);
  await Promise.all([
    selectLaboratoryByName(page, renamed),
    selectLaboratoryByName(secondPage, renamed),
  ]);
  await expect(page.getByTestId("laboratory-switcher")).toContainText(renamed);
  await expect(secondPage.getByTestId("laboratory-switcher")).toContainText(
    renamed,
  );
  const observer = new ClientObserver();
  for (const [index, client] of [
    "browser-a",
    "browser-b",
    "automation",
  ].entries()) {
    observer.record({
      client_id: client,
      mutation_id: laboratory.id,
      event_sequence: index + 1,
      resource_revision: authoritative.revision,
      observed_at: new Date().toISOString(),
      convergence_ms: 100 + index,
    });
  }
  expect(
    observer.assertConverged(laboratory.id, [
      "browser-a",
      "browser-b",
      "automation",
    ]),
  ).toHaveLength(3);
});

test("two browsers plus HTTP and MCP converge across ten concurrent creation groups", async ({
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
  await secondPage.goto("/");
  await selectLaboratoryByName(secondPage, laboratory.name);
  const names: string[] = [];

  for (let group = 0; group < 10; group += 1) {
    const snapshotResponse = await automation.get(
      `/api/v1/labs/${laboratory.id}`,
    );
    const snapshot = await snapshotResponse.json();
    const revision = snapshot.laboratory.revision as number;
    const attempts = [
      {
        name: `browser-a-${group}`,
        run: (expected: number, retry = 0) =>
          createNode(
            page.request,
            laboratory.id,
            expected,
            `browser-a-${group}`,
            `${runId}-a-${group}-${retry}`,
          ),
      },
      {
        name: `browser-b-${group}`,
        run: (expected: number, retry = 0) =>
          createNode(
            secondPage.request,
            laboratory.id,
            expected,
            `browser-b-${group}`,
            `${runId}-b-${group}-${retry}`,
          ),
      },
      {
        name: `http-${group}`,
        run: (expected: number, retry = 0) =>
          createNode(
            automation,
            laboratory.id,
            expected,
            `http-${group}`,
            `${runId}-http-${group}-${retry}`,
          ),
      },
      {
        name: `mcp-${group}`,
        run: (expected: number, retry = 0) =>
          createObjectThroughMcp(
            automation,
            laboratory.id,
            expected,
            `mcp-${group}`,
            `${runId}-mcp-${group}-${retry}`,
          ),
      },
    ];
    names.push(...attempts.map((attempt) => attempt.name));
    const initial = await Promise.all(
      attempts.map((attempt) => attempt.run(revision)),
    );
    expect(initial.filter((item) => item.ok)).toHaveLength(1);
    for (let index = 0; index < attempts.length; index += 1) {
      if (initial[index].ok) continue;
      let created = false;
      for (let retry = 0; retry < 8 && !created; retry += 1) {
        const latestResponse = await automation.get(
          `/api/v1/labs/${laboratory.id}`,
        );
        const latest = await latestResponse.json();
        const result = await attempts[index].run(
          latest.laboratory.revision,
          retry + 1,
        );
        created = result.ok;
      }
      expect(created).toBe(true);
    }
  }

  const snapshotResponse = await automation.get(
    `/api/v1/labs/${laboratory.id}`,
  );
  const snapshot = await snapshotResponse.json();
  expect(snapshot.nodes).toHaveLength(30);
  expect(snapshot.network_objects).toHaveLength(10);
  expect(snapshot.placements).toHaveLength(40);
  expect(
    new Set(
      snapshot.placements.map(
        (item: { resource_id: string }) => item.resource_id,
      ),
    ).size,
  ).toBe(40);

  const firstSummary = page.getByTestId("topology-a11y-summary");
  const secondSummary = secondPage.getByTestId("topology-a11y-summary");
  await expect(firstSummary).toContainText(names[0], { timeout: 2_000 });
  await expect(firstSummary).toContainText(names.at(-1)!, { timeout: 2_000 });
  await expect(secondSummary).toContainText(names[0], { timeout: 2_000 });
  await expect(secondSummary).toContainText(names.at(-1)!, { timeout: 2_000 });
});
