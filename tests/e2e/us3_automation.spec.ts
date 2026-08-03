import { expect, test } from "./fixtures/acceptanceFixture";
import { selectLaboratoryByName } from "./pages/LaboratoryPage";

test("browser converges on a laboratory created by automation", async ({
  page,
  automation,
  ledger,
  runId,
}) => {
  const name = `automation-${runId.slice(0, 8)}`;
  const created = await automation.post("/api/v1/labs", {
    data: { name, recovery_policy: "remain_stopped" },
    headers: { "Idempotency-Key": crypto.randomUUID() },
  });
  expect(created.ok()).toBeTruthy();
  const laboratory = await created.json();
  await ledger.add({
    resource_type: "laboratory",
    resource_id: laboratory.id,
    revision: laboratory.revision,
    cleanup_method: "api-delete",
  });
  await page.goto("/");
  await selectLaboratoryByName(page, laboratory.name);
});
