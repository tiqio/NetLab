import { expect, test } from "./fixtures/acceptanceFixture";
import { createOwnedLaboratory } from "./journeys/completeRealJourney";
import { selectLaboratoryByName } from "./pages/LaboratoryPage";

test("two clients observe the same laboratory identity", async ({
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
  await expect(secondPage.getByTestId("laboratory-switcher")).toContainText(
    laboratory.name,
  );
});
