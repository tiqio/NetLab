import { expect, test } from "./fixtures/acceptanceFixture";
import {
  createOwnedLaboratory,
  createOwnedLightweightPair,
} from "./journeys/completeRealJourney";

test("creates lightweight nodes through the real service", async ({
  page,
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
  const { first, second } = await createOwnedLightweightPair(
    page,
    automation,
    ledger,
    laboratory.id,
  );
  const response = await automation.get(`/api/v1/labs/${laboratory.id}`);
  expect(response.ok()).toBeTruthy();
  const snapshot = await response.json();
  expect(
    snapshot.network_objects.map((item: { id: string }) => item.id),
  ).toEqual(expect.arrayContaining([first.id, second.id]));
});
