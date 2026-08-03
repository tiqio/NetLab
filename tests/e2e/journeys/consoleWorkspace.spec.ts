import { expect, test } from "../fixtures/acceptanceFixture";
import { ConsolePage } from "../pages/ConsolePage";
import { TopologyPage } from "../pages/TopologyPage";
import { createRunningDiagnosticNode } from "./diagnosticJourney";

test("real Telnet and VNC sessions switch reconnect and close", async ({
  page,
  automation,
  ledger,
  runId,
}) => {
  test.skip(
    process.env.NETLAB_ACCEPTANCE_PROFILE !== "target-host",
    "requires target-host QEMU consoles",
  );
  await createRunningDiagnosticNode({
    page,
    automation,
    ledger,
    runId,
    templateKey: "ubuntu-qemu",
  });
  await new TopologyPage(page, automation).openSelectedTerminal();
  const consolePage = new ConsolePage(page);
  await expect(
    page.getByRole("button", { name: "TELNET 1", exact: true }),
  ).toBeVisible();
  await consolePage.add("VNC");
  await expect(
    page.getByRole("button", { name: "VNC 2", exact: true }),
  ).toBeVisible();
  await consolePage.reconnect();
  await page.setViewportSize({ width: 1024, height: 768 });
  await expect(
    page.getByRole("navigation", { name: "Console sessions" }),
  ).toBeVisible();
  const close = page.getByRole("button", { name: /^Close / }).first();
  if (await close.count()) await close.click();
});
