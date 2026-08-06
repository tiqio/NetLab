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
    page.getByRole("button", { name: /^(SSH|SERIAL) \d+$/ }).first(),
  ).toBeVisible();
  const serialSession = page.getByRole("button", { name: /^SERIAL \d+$/ });
  if (!(await serialSession.count())) await consolePage.add("Serial");
  await expect(serialSession.first()).toBeVisible();
  await consolePage.add("VNC");
  await expect(page.getByRole("button", { name: /^VNC \d+$/ })).toBeVisible();
  await consolePage.reconnect();
  await page.setViewportSize({ width: 1024, height: 768 });
  await expect(
    page.getByRole("navigation", { name: /^(Console sessions|终端会话)$/ }),
  ).toBeVisible();
  const close = page.getByRole("button", { name: /^(Close|关闭) / }).first();
  if (await close.count()) await close.click();
});
