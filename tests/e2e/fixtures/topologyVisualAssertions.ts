import { expect, type Locator, type Page } from "@playwright/test";

export interface WorldPlacementRectangle {
  resourceId: string;
  x: number;
  y: number;
  width: number;
  height: number;
  clearanceX?: number;
  clearanceY?: number;
}

export function placementRectanglesOverlap(
  first: WorldPlacementRectangle,
  second: WorldPlacementRectangle,
) {
  const firstClearanceX = first.clearanceX ?? 0;
  const firstClearanceY = first.clearanceY ?? 0;
  const secondClearanceX = second.clearanceX ?? 0;
  const secondClearanceY = second.clearanceY ?? 0;
  return !(
    first.x + first.width / 2 + firstClearanceX <=
      second.x - second.width / 2 - secondClearanceX ||
    second.x + second.width / 2 + secondClearanceX <=
      first.x - first.width / 2 - firstClearanceX ||
    first.y + first.height / 2 + firstClearanceY <=
      second.y - second.height / 2 - secondClearanceY ||
    second.y + second.height / 2 + secondClearanceY <=
      first.y - first.height / 2 - firstClearanceY
  );
}

export function expectNoPlacementOverlap(
  rectangles: WorldPlacementRectangle[],
) {
  for (let firstIndex = 0; firstIndex < rectangles.length; firstIndex += 1) {
    for (
      let secondIndex = firstIndex + 1;
      secondIndex < rectangles.length;
      secondIndex += 1
    ) {
      const first = rectangles[firstIndex];
      const second = rectangles[secondIndex];
      expect(
        placementRectanglesOverlap(first, second),
        `${first.resourceId} overlaps ${second.resourceId}`,
      ).toBe(false);
    }
  }
}

export function topologyConnection(page: Page, connectionId: string) {
  return page.locator(`[data-connection-id="${connectionId}"]`);
}

export async function expectConnectionState(
  connection: Locator,
  state: string,
) {
  await expect(connection).toHaveAttribute("data-connection-state", state);
  await expect(connection).toHaveAttribute("aria-label", /.+/);
}

export async function expectIndependentConnectionTargets(
  connections: Locator[],
) {
  const boxes = await Promise.all(
    connections.map(async (connection) => {
      await expect(connection).toBeVisible();
      return connection.boundingBox();
    }),
  );
  expect(boxes.every(Boolean)).toBe(true);
  const centers = boxes.map((box) =>
    box ? `${Math.round(box.x + box.width / 2)}:${Math.round(box.y + box.height / 2)}` : "",
  );
  expect(new Set(centers).size).toBe(centers.length);
}

export async function expectSemanticLegendItem(
  page: Page,
  marker: string,
  count: number,
) {
  const item = page.locator(`[data-semantic-marker="${marker}"]`);
  await expect(item).toBeVisible();
  await expect(item).toHaveAttribute("data-connection-count", String(count));
}
