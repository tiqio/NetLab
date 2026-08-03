import { describe, expect, it } from "vitest";
import {
  exceedsDragThreshold,
  nearestPoint,
  quadraticRoute,
  fitViewport,
  screenToWorld,
  worldToScreen,
  zoomAroundPoint,
} from "./topologyGeometry";

describe("topology geometry", () => {
  it("round-trips coordinates and keeps the pointer anchor while zooming", () => {
    const viewport = { centerX: 100, centerY: 80, zoom: 2 };
    const point = { x: 160, y: 120 };
    expect(worldToScreen(screenToWorld(point, viewport), viewport)).toEqual(
      point,
    );
    const zoomed = zoomAroundPoint(viewport, point, 1.5);
    expect(worldToScreen(screenToWorld(point, viewport), zoomed)).toEqual(
      point,
    );
  });

  it("distinguishes click jitter from drag and finds the nearest port", () => {
    expect(exceedsDragThreshold({ x: 0, y: 0 }, { x: 3, y: 3 })).toBe(false);
    expect(exceedsDragThreshold({ x: 0, y: 0 }, { x: 6, y: 0 })).toBe(true);
    expect(
      nearestPoint(
        { x: 8, y: 0 },
        [
          { id: "a", x: 0, y: 0 },
          { id: "b", x: 10, y: 0 },
        ],
        5,
      )?.id,
    ).toBe("b");
  });

  it("creates a finite curved route for overlapping and distant endpoints", () => {
    for (const route of [
      quadraticRoute({ x: 0, y: 0 }, { x: 0, y: 0 }, 20),
      quadraticRoute({ x: -100, y: 20 }, { x: 200, y: 40 }, 30),
    ]) {
      expect(
        route.flatMap((point) => [point.x, point.y]).every(Number.isFinite),
      ).toBe(true);
    }
  });
});

describe("fitViewport", () => {
  it("fits points inside the padded viewport and handles empty input", () => {
    const viewport = fitViewport(
      [
        { x: -100, y: -50 },
        { x: 100, y: 50 },
      ],
      1000,
      600,
      100,
    );
    expect(viewport).toEqual({ centerX: 0, centerY: 0, zoom: 4 });
    expect(fitViewport([], 800, 600)).toEqual({
      centerX: 0,
      centerY: 0,
      zoom: 1,
    });
  });
});
