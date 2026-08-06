import type { Locator, Page } from "@playwright/test";

export interface Rectangle {
  x: number;
  y: number;
  width: number;
  height: number;
}

export function overlapArea(first: Rectangle, second: Rectangle) {
  const width = Math.max(
    0,
    Math.min(first.x + first.width, second.x + second.width) -
      Math.max(first.x, second.x),
  );
  const height = Math.max(
    0,
    Math.min(first.y + first.height, second.y + second.height) -
      Math.max(first.y, second.y),
  );
  return width * height;
}

export function isInsideViewport(
  rectangle: Rectangle,
  viewport: { width: number; height: number },
  tolerance = 1,
) {
  return (
    rectangle.x >= -tolerance &&
    rectangle.y >= -tolerance &&
    rectangle.x + rectangle.width <= viewport.width + tolerance &&
    rectangle.y + rectangle.height <= viewport.height + tolerance
  );
}

export async function expectNoOverlap(
  first: Locator,
  second: Locator,
  toleranceArea = 1,
) {
  const [firstBox, secondBox] = await Promise.all([
    first.boundingBox(),
    second.boundingBox(),
  ]);
  if (!firstBox) throw new Error("first region must be visible");
  if (!secondBox) throw new Error("second region must be visible");
  const area = overlapArea(firstBox, secondBox);
  if (area > toleranceArea) {
    throw new Error(`layout regions overlap by ${area}px²`);
  }
}

export async function expectInsideViewport(locator: Locator, page: Page) {
  const [box, viewport] = await Promise.all([
    locator.boundingBox(),
    page.evaluate(() => ({
      width: document.documentElement.clientWidth,
      height: document.documentElement.clientHeight,
    })),
  ]);
  if (!box) throw new Error("region must be visible");
  if (!isInsideViewport(box, viewport)) {
    throw new Error("layout region is outside the viewport");
  }
}

export async function expectNoPageHorizontalOverflow(page: Page) {
  const overflow = await page.evaluate(
    () =>
      document.documentElement.scrollWidth -
      document.documentElement.clientWidth,
  );
  if (overflow > 1) {
    throw new Error(`page has ${overflow}px of horizontal overflow`);
  }
}
