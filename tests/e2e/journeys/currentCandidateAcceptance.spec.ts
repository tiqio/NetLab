import type { APIRequestContext } from "@playwright/test";
import { expect, test } from "../fixtures/acceptanceFixture";
import { selectLaboratoryByName } from "../pages/LaboratoryPage";
import { TemplatePage } from "../pages/TemplatePage";
import { TopologyPage } from "../pages/TopologyPage";
import { createRunningDiagnosticNode } from "./diagnosticJourney";
import {
  createOwnedLaboratory,
  resolveTemplateSelection,
} from "./completeRealJourney";
import { waitForCondition } from "../fixtures/waiters";

async function waitTask(automation: APIRequestContext, taskId: string) {
  return waitForCondition(
    async () => (await automation.get(`/api/v1/tasks/${taskId}`)).json(),
    (task: { state?: string }) =>
      task.state === "succeeded" ||
      task.state === "failed" ||
      task.state === "cancelled",
    `task ${taskId}`,
    120_000,
  );
}

async function startNode(
  automation: APIRequestContext,
  node: { id: string; revision: number },
) {
  const response = await automation.put(`/api/v1/nodes/${node.id}/state`, {
    headers: {
      "Content-Type": "application/json",
      "Idempotency-Key": crypto.randomUUID(),
      "If-Match": String(node.revision),
    },
    data: { desired_state: "running" },
  });
  expect(response.ok()).toBeTruthy();
  const body = await response.json();
  const task = await waitTask(automation, body.task.id);
  expect(task.state).toBe("succeeded");
  await waitForCondition(
    async () => (await automation.get(`/api/v1/nodes/${node.id}`)).json(),
    (value: { observed_state?: string }) => value.observed_state === "running",
    `${node.id} running`,
    120_000,
  );
}

async function createLink(
  automation: APIRequestContext,
  laboratoryId: string,
  endpointA: string,
  endpointB: string,
) {
  const response = await automation.post(`/api/v1/labs/${laboratoryId}/links`, {
    headers: {
      "Content-Type": "application/json",
      "Idempotency-Key": crypto.randomUUID(),
    },
    data: { endpoint_a_id: endpointA, endpoint_b_id: endpointB },
  });
  expect(response.ok()).toBeTruthy();
  const body = await response.json();
  if (body.task?.id)
    expect((await waitTask(automation, body.task.id)).state).toBe("succeeded");
  return body.link as { id: string; revision: number };
}

async function consoleCommand(
  page: import("@playwright/test").Page,
  nodeId: string,
  command: string,
) {
  await page.evaluate(
    async ({ nodeId, command }) => {
      const descriptors = await fetch(`/api/v1/nodes/${nodeId}/consoles`).then(
        (response) => response.json(),
      );
      const descriptor = descriptors.find(
        (item: { mode?: string }) => item.mode === "telnet",
      );
      if (!descriptor?.stream_url)
        throw new Error(`No Telnet console for ${nodeId}`);
      const url = new URL(descriptor.stream_url, window.location.origin);
      url.protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
      await new Promise<void>((resolve, reject) => {
        const socket = new WebSocket(url);
        const timeout = window.setTimeout(() => {
          socket.close();
          reject(new Error(`Console command timed out for ${nodeId}`));
        }, 5000);
        socket.onopen = () => {
          socket.send(new TextEncoder().encode(`${command}\n`));
          window.setTimeout(() => {
            window.clearTimeout(timeout);
            socket.close();
            resolve();
          }, 600);
        };
        socket.onerror = () => {
          window.clearTimeout(timeout);
          reject(new Error(`Console WebSocket failed for ${nodeId}`));
        };
      });
    },
    { nodeId, command },
  );
}

test("current candidate shares browser HTTP and MCP state without stale resurrection", async ({
  page,
  secondPage,
  automation,
  ledger,
  runId,
}) => {
  const capabilities = await automation.get("/api/v1/capabilities");
  expect(capabilities.ok()).toBeTruthy();
  const identity = await capabilities.json();
  expect(identity.release.candidate_id).toBeTruthy();
  expect(identity.release.contract_digest).toMatch(/^sha256:[a-f0-9]{64}$/);

  const { laboratory } = await createOwnedLaboratory(
    page,
    automation,
    ledger,
    runId,
  );
  const mcp = await automation.post("/mcp", {
    headers: { "Content-Type": "application/json", Accept: "application/json" },
    data: {
      jsonrpc: "2.0",
      id: 1,
      method: "tools/call",
      params: { name: "netlab.labs.get", arguments: { lab_id: laboratory.id } },
    },
  });
  expect(mcp.ok()).toBeTruthy();
  expect(JSON.stringify(await mcp.json())).toContain(laboratory.id);

  await secondPage.goto("/");
  await selectLaboratoryByName(secondPage, laboratory.name);
  await page.getByRole("tab", { name: "Console" }).click();
  await page.getByRole("tab", { name: "Capture" }).click();
  await page.getByRole("tab", { name: "Console" }).click();
  await expect(page.getByRole("tab", { name: "Console" })).toHaveAttribute(
    "aria-selected",
    "true",
  );
});

test("target candidate validates console capture filter and live rewire", async ({
  page,
  secondPage,
  automation,
  ledger,
  runId,
}) => {
  test.skip(
    process.env.NETLAB_ACCEPTANCE_PROFILE !== "target-host",
    "requires target-host Docker, capture, and live-link runtimes",
  );

  const first = await createRunningDiagnosticNode({
    page,
    automation,
    ledger,
    runId,
    templateKey: "busybox-container",
  });
  const selection = await resolveTemplateSelection(
    automation,
    "busybox-container",
  );
  const templates = new TemplatePage(page, automation);
  const second = await templates.createDevice({
    ...selection,
    nodeName: `candidate-b-${runId.slice(0, 5)}`,
    laboratoryId: first.laboratory.id,
  });
  const third = await templates.createDevice({
    ...selection,
    nodeName: `candidate-c-${runId.slice(0, 5)}`,
    laboratoryId: first.laboratory.id,
  });
  for (const node of [second, third]) {
    await ledger.add({
      resource_type: "node",
      resource_id: node.id,
      laboratory_id: first.laboratory.id,
      revision: node.revision,
      cleanup_method: "laboratory-cascade",
    });
    await startNode(automation, node);
  }

  const snapshot = await waitForCondition(
    async () =>
      (await automation.get(`/api/v1/labs/${first.laboratory.id}`)).json(),
    (value: { interfaces?: Array<{ node_id: string }> }) =>
      (value.interfaces || []).filter((item) =>
        [second.id, third.id].includes(item.node_id),
      ).length >= 2,
    "candidate interfaces",
    30_000,
  );
  const secondInterface = snapshot.interfaces.find(
    (item: { node_id: string }) => item.node_id === second.id,
  ).id as string;
  const thirdInterface = snapshot.interfaces.find(
    (item: { node_id: string }) => item.node_id === third.id,
  ).id as string;
  const link = await createLink(
    automation,
    first.laboratory.id,
    first.interface.id,
    secondInterface,
  );
  await ledger.add({
    resource_type: "link",
    resource_id: link.id,
    laboratory_id: first.laboratory.id,
    revision: link.revision,
    cleanup_method: "laboratory-cascade",
  });

  await new TopologyPage(page, automation).openSelectedTerminal();
  await expect(
    page.getByRole("button", { name: "TELNET 1", exact: true }),
  ).toBeVisible();

  const captureResponse = await automation.post("/api/v1/captures", {
    headers: {
      "Content-Type": "application/json",
      "Idempotency-Key": crypto.randomUUID(),
    },
    data: {
      laboratory_id: first.laboratory.id,
      source_type: "interface",
      source_id: first.interface.id,
      format: "pcap",
      retain: true,
      duration_seconds: 10,
      max_bytes: 1048576,
    },
  });
  expect(captureResponse.ok()).toBeTruthy();
  const capture = await captureResponse.json();
  await ledger.add({
    resource_type: "capture",
    resource_id: capture.capture.id,
    laboratory_id: first.laboratory.id,
    cleanup_method: "capture-delete",
  });
  expect(capture.stream_url).toContain(capture.capture.id);
  expect(capture.wireshark.media_type).toBe("application/vnd.tcpdump.pcap");

  const filterResponse = await automation.post("/api/v1/traffic-filters", {
    headers: {
      "Content-Type": "application/json",
      "Idempotency-Key": crypto.randomUUID(),
    },
    data: {
      laboratory_id: first.laboratory.id,
      match: {},
      max_observations: 100,
      interface_ids: [first.interface.id, secondInterface, thirdInterface],
      link_ids: [link.id],
    },
  });
  expect(filterResponse.ok()).toBeTruthy();
  const filter = await filterResponse.json();
  await ledger.add({
    resource_type: "traffic_filter",
    resource_id: filter.traffic_filter.id,
    laboratory_id: first.laboratory.id,
    cleanup_method: "traffic-filter-delete",
  });

  const secondInterfaceRecord = snapshot.interfaces.find(
    (item: { node_id: string }) => item.node_id === second.id,
  ) as { id: string; name: string };
  await consoleCommand(
    page,
    first.node.id,
    `ip addr replace 10.77.0.1/30 dev ${first.interface.name}; ip link set ${first.interface.name} up`,
  );
  await consoleCommand(
    page,
    second.id,
    `ip addr replace 10.77.0.2/30 dev ${secondInterfaceRecord.name}; ip link set ${secondInterfaceRecord.name} up`,
  );
  await page.getByRole("tab", { name: "Traffic Filter" }).click();
  await page
    .getByRole("region", { name: "Diagnostics" })
    .getByRole("button", { name: "Refresh", exact: true })
    .click();
  await consoleCommand(page, first.node.id, "ping -c 1 -W 1 10.77.0.2");
  await consoleCommand(page, second.id, "nc -l -p 19001 >/dev/null 2>&1 &");
  await consoleCommand(
    page,
    first.node.id,
    "printf tcp | nc -w 1 10.77.0.2 19001",
  );
  await consoleCommand(page, second.id, "nc -u -l -p 19002 >/dev/null 2>&1 &");
  await consoleCommand(
    page,
    first.node.id,
    "printf udp | nc -u -w 1 10.77.0.2 19002",
  );
  const canvas = page.getByLabel("Topology canvas keyboard area");
  await expect(canvas).toHaveAttribute("data-traffic-recent", /[1-9]\d*/, {
    timeout: 500,
  });
  await expect(
    page.locator(`[data-traffic-path-id="traffic:${link.id}"]`),
  ).toHaveAttribute("data-traffic-direction", /single|bidirectional/);
  await expect(canvas).toHaveAttribute("data-traffic-recent", "0", {
    timeout: 2000,
  });
  await expect(canvas).toHaveAttribute("data-traffic-lingering", "0", {
    timeout: 6000,
  });

  const reconnect = await automation.post(
    `/api/v1/links/${link.id}/reconnect`,
    {
      headers: {
        "Content-Type": "application/json",
        "Idempotency-Key": crypto.randomUUID(),
        "If-Match": String(link.revision),
      },
      data: {
        retained_endpoint_id: first.interface.id,
        replacement_endpoint_id: thirdInterface,
      },
    },
  );
  expect(reconnect.ok()).toBeTruthy();
  expect(
    (await waitTask(automation, (await reconnect.json()).task.id)).state,
  ).toBe("succeeded");

  await secondPage.goto("/");
  await selectLaboratoryByName(secondPage, first.laboratory.name);
  await expect(secondPage.getByTestId("topology-a11y-summary")).toContainText(
    third.name,
    { timeout: 15_000 },
  );
  const finalSnapshot = await automation.get(
    `/api/v1/labs/${first.laboratory.id}`,
  );
  const finalBody = await finalSnapshot.json();
  expect(
    finalBody.links.some(
      (value: { id: string; endpoint_b_id: string }) =>
        value.id === link.id && value.endpoint_b_id === thirdInterface,
    ),
  ).toBeTruthy();
});
