import { expect, test } from "../fixtures/acceptanceFixture";
import { waitForCondition } from "../fixtures/waiters";
import {
  createOwnedLaboratory,
  createOwnedLightweightPair,
  result,
} from "./completeRealJourney";

test("two browsers HTTP and MCP serialize ten endpoint contention rounds", async ({
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
  const peer = await page.context().newPage();
  await peer.goto(`/laboratories/${laboratory.id}`);
  const convergenceSamples: Array<Record<string, number>> = [];
  for (let round = 0; round < 10; round += 1) {
    const snapshot = await (
      await automation.get(`/api/v1/labs/${laboratory.id}/connections`)
    ).json();
    const source = snapshot.endpoints.find(
      (item: {
        resource_id: string;
        port_name?: string;
        availability?: string;
      }) =>
        item.resource_id === first.id &&
        item.port_name === "eth0" &&
        item.availability === "free",
    );
    const target = snapshot.endpoints.find(
      (item: {
        resource_id: string;
        port_name?: string;
        availability?: string;
      }) =>
        item.resource_id === second.id &&
        item.port_name === "eth0" &&
        item.availability === "free",
    );
    expect(source).toBeTruthy();
    expect(target).toBeTruthy();
    const revision = (
      await (await automation.get(`/api/v1/labs/${laboratory.id}`)).json()
    ).laboratory.revision;
    const body = { source, target };
    const browserCreate = (browserPage: typeof page, key: string) =>
      browserPage.evaluate(
        async ({
          laboratoryId,
          revisionValue,
          requestBody,
          idempotencyKey,
        }) => {
          const response = await fetch(
            `/api/v1/labs/${laboratoryId}/connections`,
            {
              method: "POST",
              headers: {
                "Content-Type": "application/json",
                "If-Match": String(revisionValue),
                "Idempotency-Key": idempotencyKey,
              },
              body: JSON.stringify(requestBody),
            },
          );
          return { status: response.status, body: await response.json() };
        },
        {
          laboratoryId: laboratory.id,
          revisionValue: revision,
          requestBody: body,
          idempotencyKey: `${runId}-browser-${round}-${key}`,
        },
      );
    const contentionStartedAt = Date.now();
    const [browserA, browserB, http, mcp] = await Promise.all([
      browserCreate(page, "a"),
      browserCreate(peer, "b"),
      automation
        .post(`/api/v1/labs/${laboratory.id}/connections`, {
          headers: {
            "If-Match": String(revision),
            "Idempotency-Key": `${runId}-http-${round}`,
          },
          data: body,
        })
        .then(async (response) => ({
          status: response.status(),
          body: await response.json(),
        })),
      automation
        .post("/mcp", {
          headers: { Accept: "application/json" },
          data: {
            jsonrpc: "2.0",
            id: round + 1,
            method: "tools/call",
            params: {
              name: "netlab.topology_connections.create",
              arguments: {
                laboratory_id: laboratory.id,
                expected_revision: revision,
                idempotency_key: `${runId}-mcp-${round}`,
                source,
                target,
              },
            },
          },
        })
        .then(async (response) => ({
          status: response.status(),
          body: await response.json(),
        })),
    ]);
    const outcomes = [browserA, browserB, http, mcp];
    const observeBrowser = async (browserPage: typeof page) =>
      browserPage.evaluate(async (laboratoryId) => {
        const response = await fetch(
          `/api/v1/labs/${laboratoryId}/connections`,
        );
        return response.json();
      }, laboratory.id);
    const observeMCP = async () => {
      const response = await automation.post("/mcp", {
        headers: { Accept: "application/json" },
        data: {
          jsonrpc: "2.0",
          id: `observe-${round}`,
          method: "tools/call",
          params: {
            name: "netlab.topology_connections.list",
            arguments: { laboratory_id: laboratory.id },
          },
        },
      });
      return response.json();
    };
    const convergence = async (
      name: string,
      observe: () => Promise<unknown>,
    ) => {
      const value = await waitForCondition(
        observe,
        (current: unknown) =>
          JSON.stringify(current).match(/\"backing_kind\"/g)?.length === 1,
        `${name} contention round ${round}`,
      );
      return { value, elapsed: Date.now() - contentionStartedAt };
    };
    const [browserAState, browserBState, httpState, mcpState] =
      await Promise.all([
        convergence("browser A", () => observeBrowser(page)),
        convergence("browser B", () => observeBrowser(peer)),
        convergence("HTTP", async () =>
          (
            await automation.get(`/api/v1/labs/${laboratory.id}/connections`)
          ).json(),
        ),
        convergence("MCP", observeMCP),
      ]);
    const roundConvergence = {
      browser_a: browserAState.elapsed,
      browser_b: browserBState.elapsed,
      http: httpState.elapsed,
      mcp: mcpState.elapsed,
    };
    convergenceSamples.push(roundConvergence);
    for (const [client, elapsed] of Object.entries(roundConvergence)) {
      expect(
        elapsed,
        `${client} convergence round ${round}`,
      ).toBeLessThanOrEqual(2_000);
    }
    const authoritative = httpState.value as {
      connections: Array<{ id: string; revision: number }>;
    };
    expect(authoritative.connections).toHaveLength(1);
    expect(outcomes.some((item) => item.status < 300)).toBe(true);
    let deletionStatus = 0;
    for (let attempt = 0; attempt < 5; attempt += 1) {
      const current = await (
        await automation.get(`/api/v1/labs/${laboratory.id}/connections`)
      ).json();
      const winner = current.connections[0];
      const deletion = await automation.delete(
        `/api/v1/connections/${winner.id}`,
        {
          headers: {
            "If-Match": String(winner.revision),
            "Idempotency-Key": `${runId}-delete-${round}-${attempt}`,
          },
        },
      );
      deletionStatus = deletion.status();
      if (deletionStatus === 202) break;
      expect(deletionStatus).toBe(409);
      await page.waitForTimeout(100);
    }
    expect(deletionStatus).toBe(202);
    await waitForCondition(
      async () =>
        (
          await automation.get(`/api/v1/labs/${laboratory.id}/connections`)
        ).json(),
      (value: { connections?: unknown[] }) =>
        (value.connections || []).length === 0,
      `contention cleanup ${round}`,
    );
  }
  await peer.close();
  interactionResults.push(
    result(
      "topology.connection.concurrent-control-planes",
      testInfo.project.use.viewport!,
      `ten rounds converged to one winner within 2s per client and released both endpoint reservations; samples=${JSON.stringify(convergenceSamples)}`,
      [laboratory.id],
    ),
  );
});
