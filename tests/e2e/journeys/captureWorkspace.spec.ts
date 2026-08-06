import { expect, test } from "../fixtures/acceptanceFixture";
import { CapturePage } from "../pages/CapturePage";
import { createRunningDiagnosticNode } from "./diagnosticJourney";

test("real interface capture exposes stream metadata quota and cleanup", async ({
  page,
  automation,
  ledger,
  runId,
}) => {
  test.skip(
    process.env.NETLAB_ACCEPTANCE_PROFILE !== "target-host",
    "requires target-host capture runtime",
  );
  const { laboratory } = await createRunningDiagnosticNode({
    page,
    automation,
    ledger,
    runId,
    templateKey: "busybox-container",
  });
  await page.getByRole("tab", { name: /^(Capture|抓包)$/ }).click();
  const capturePage = new CapturePage(page);
  await capturePage.start();
  await capturePage.refresh();
  await capturePage.expectMetadata();
  const response = await automation.get(
    `/api/v1/captures?laboratory_id=${laboratory.id}`,
  );
  const captures = await response.json();
  const capture = captures.at(-1);
  await ledger.add({
    resource_type: "capture",
    resource_id: capture.id,
    laboratory_id: laboratory.id,
    cleanup_method: "capture-delete",
  });
  expect(capture.max_bytes).toBeGreaterThan(0);
  expect(capture.bytes_written).toBeGreaterThanOrEqual(0);
  await capturePage.stop();
});
