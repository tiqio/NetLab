import { expect, type Locator, type Page } from "@playwright/test";

export async function dragTopologyPort(
  page: Page,
  source: Locator,
  target: Locator,
) {
  const sourceBox = await source.boundingBox();
  if (!sourceBox) throw new Error("source topology port is not visible");
  await source.dispatchEvent("pointerdown", {
    pointerId: 41,
    button: 0,
    clientX: sourceBox.x + sourceBox.width / 2,
    clientY: sourceBox.y + sourceBox.height / 2,
  });
  await expect(page.locator("[data-connection-preview]")).toHaveAttribute(
    "data-source-anchored",
    "true",
  );
  const targetBox = await target.boundingBox();
  if (!targetBox) throw new Error("target topology port is not visible");
  const point = {
    pointerId: 41,
    button: 0,
    clientX: targetBox.x + targetBox.width / 2,
    clientY: targetBox.y + targetBox.height / 2,
  };
  await source.dispatchEvent("pointermove", point);
  await expect(target).toHaveAttribute(
    "data-connection-target-state",
    "compatible",
  );
  await source.dispatchEvent("pointerup", point);
}

export async function expectNoGhostConnection(page: Page) {
  await expect(page.locator("[data-connection-preview]")).toHaveCount(0);
}

export async function expectViewportStable(
  surface: Locator,
  expectedZoom: number,
) {
  await expect(surface).toHaveAttribute(
    "data-viewport-zoom",
    String(expectedZoom),
  );
}
