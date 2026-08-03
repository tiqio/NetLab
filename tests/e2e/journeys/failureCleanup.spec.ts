import { expect, test } from "../fixtures/acceptanceFixture";
import { cleanupOwnedResources } from "../fixtures/cleanupCoordinator";
import { injectControlledFailure } from "../fixtures/failureInjection";
import { createOwnedLaboratory } from "./completeRealJourney";

test("controlled failure still restores the clean baseline", async ({
  page,
  automation,
  ledger,
  environment,
  runId,
}) => {
  await createOwnedLaboratory(page, automation, ledger, runId);
  const previous = process.env.NETLAB_ACCEPTANCE_FAILURE_INJECTION;
  if (previous === "after-runtime-create") {
    injectControlledFailure("after-runtime-create");
  }
  process.env.NETLAB_ACCEPTANCE_FAILURE_INJECTION = "after-runtime-create";
  expect(() => injectControlledFailure("after-runtime-create")).toThrow(
    /Controlled acceptance failure/,
  );
  process.env.NETLAB_ACCEPTANCE_FAILURE_INJECTION = previous;
  const cleanup = await cleanupOwnedResources(
    automation,
    ledger,
    environment.baseline_laboratory_ids,
    "failure",
    {},
    environment.baseline_runtime_ownership,
  );
  expect(cleanup.remaining_count).toBe(0);
  expect(cleanup.baseline_restored).toBe(true);
});
