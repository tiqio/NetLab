import { expect, test } from "../fixtures/acceptanceFixture";
import {
  dragTopologyPort,
  expectNoGhostConnection,
  expectViewportStable,
} from "../fixtures/topologyConnectionAssertions";
import { waitForCondition } from "../fixtures/waiters";
import {
  createOwnedLaboratory,
  createOwnedLightweightPair,
  result,
} from "./completeRealJourney";

test("mixed resource ports drag with stable preview, cancellation, and zoom", async ({
  page,
  automation,
  ledger,
  runId,
  interactionResults,
}, testInfo) => {
  test.skip(
    process.env.NETLAB_ACCEPTANCE_PROFILE !== "target-host",
    "requires target-host topology runtime",
  );
  const { laboratory } = await createOwnedLaboratory(
    page,
    automation,
    ledger,
    runId,
  );
  await createOwnedLightweightPair(page, automation, ledger, laboratory.id);
  const surface = page.getByLabel(/拓扑画布键盘操作区/);
  await surface.focus();
  await surface.press("ArrowRight");
  const source = page.locator('[data-interface-id$=":eth0"]').first();
  await expect(source).toBeVisible();
  await source.dispatchEvent("pointerdown", {
    pointerId: 40,
    button: 0,
    clientX: 0,
    clientY: 0,
  });
  await source.dispatchEvent("pointercancel", { pointerId: 40 });
  await expectNoGhostConnection(page);

  await surface.evaluate((element) => {
    const current = Number(element.getAttribute("data-viewport-zoom") || 1);
    element.setAttribute("data-test-original-zoom", String(current));
  });
  const ports = page.locator('[data-interface-id$=":eth0"]');
  await dragTopologyPort(page, ports.first(), ports.nth(1));
  await waitForCondition(
    async () =>
      (
        await automation.get(`/api/v1/labs/${laboratory.id}/connections`)
      ).json(),
    (snapshot: { connections?: unknown[] }) =>
      (snapshot.connections || []).length === 1,
    "unified dragged connection",
  );
  await expectViewportStable(surface, 1);
  await expectNoGhostConnection(page);
  interactionResults.push(
    result(
      "topology.connection.port-drag",
      testInfo.project.use.viewport!,
      "source-anchored preview and compatible target converged through the unified command",
      [laboratory.id],
      "pointer",
    ),
  );
});
