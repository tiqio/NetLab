import { test, expect } from "../fixtures/acceptanceFixture";
import { cleanupOwnedResources } from "../fixtures/cleanupCoordinator";
import { createOwnedLaboratory } from "./completeRealJourney";

test("an interrupted journey can clean every owned resource", async ({
  page,
  automation,
  ledger,
  runId,
  environment,
}) => {
  await createOwnedLaboratory(page, automation, ledger, runId);
  const cleanup = await cleanupOwnedResources(
    automation,
    ledger,
    environment.baseline_laboratory_ids,
    "interrupt",
    {},
    environment.baseline_runtime_ownership,
  );
  expect(cleanup.remaining_count).toBe(0);
  expect(cleanup.baseline_restored).toBeTruthy();
});
