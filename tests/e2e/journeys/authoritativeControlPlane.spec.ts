import { expect, test } from "../fixtures/acceptanceFixture";
import { createOwnedLaboratory } from "./completeRealJourney";
import { selectLaboratoryByName } from "../pages/LaboratoryPage";

test("browser and HTTP automation use the authoritative release and shared state", async ({
  page,
  secondPage,
  automation,
  ledger,
  runId,
}) => {
  const capabilities = await automation.get("/api/v1/capabilities");
  expect(capabilities.ok()).toBeTruthy();
  const capabilityBody = await capabilities.json();
  expect(capabilityBody.release.candidate_id).toBeTruthy();
  expect(capabilityBody.release.contract_digest).toMatch(
    /^sha256:[a-f0-9]{64}$/,
  );

  const mcpTools = await automation.post("/mcp", {
    data: { jsonrpc: "2.0", id: 1, method: "tools/list", params: {} },
    headers: { Accept: "application/json", "Content-Type": "application/json" },
  });
  expect(mcpTools.ok()).toBeTruthy();

  const { laboratory } = await createOwnedLaboratory(
    page,
    automation,
    ledger,
    runId,
  );
  await secondPage.goto("/");
  await selectLaboratoryByName(secondPage, laboratory.name);
});
