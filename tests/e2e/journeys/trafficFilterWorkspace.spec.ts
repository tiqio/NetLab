import { expect, test } from "../fixtures/acceptanceFixture";
import { TrafficFilterPage } from "../pages/TrafficFilterPage";
import { createRunningDiagnosticNode } from "./diagnosticJourney";

test("real Traffic Filter maps input and survives refresh and stop", async ({
  page,
  automation,
  ledger,
  runId,
}) => {
  test.skip(
    process.env.NETLAB_ACCEPTANCE_PROFILE !== "target-host",
    "requires target-host Traffic Filter runtime",
  );
  const { laboratory } = await createRunningDiagnosticNode({
    page,
    automation,
    ledger,
    runId,
    templateKey: "busybox-container",
  });
  await page.getByRole("tab", { name: /^(Traffic Filter|流量过滤)$/ }).click();
  const filterPage = new TrafficFilterPage(page);
  await filterPage.start("tcp port 443", 20);
  await filterPage.refresh();
  await filterPage.expectPath();
  const response = await automation.get(
    `/api/v1/traffic-filters?laboratory_id=${laboratory.id}`,
  );
  const entries = await response.json();
  const current = entries.at(-1)?.traffic_filter;
  expect(current).toMatchObject({
    laboratory_id: laboratory.id,
    expression: "(tcp and src port 443) or (tcp and dst port 443)",
    max_observations: 20,
  });
  await ledger.add({
    resource_type: "traffic_filter",
    resource_id: current.id,
    laboratory_id: laboratory.id,
    cleanup_method: "traffic-filter-delete",
  });
  await filterPage.stop();
});
