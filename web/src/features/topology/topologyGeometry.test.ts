import { describe, expect, it } from "vitest";
import {
  exceedsDragThreshold,
  nearestPoint,
  quadraticRoute,
  fitViewport,
  screenToWorld,
  worldToScreen,
  zoomAroundPoint,
  deterministicPortTrack,
  topologyLabelPriority,
  resolveResourceBodyHit,
  nearestPortHit,
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

describe("connection target geometry", () => {
  it("resolves the nearest port with a screen-stable hit radius", () => {
    expect(
      nearestPortHit(
        { x: 104, y: 98 },
        [
          { id: "far", x: 140, y: 100 },
          { id: "near", x: 100, y: 100 },
        ],
        14,
      )?.id,
    ).toBe("near");
  });

  it("resolves resource bodies using their declared footprint", () => {
    expect(
      resolveResourceBodyHit({ x: 52, y: 25 }, [
        { id: "node", center: { x: 0, y: 0 }, halfWidth: 32, halfHeight: 32 },
        {
          id: "bridge",
          center: { x: 60, y: 30 },
          halfWidth: 44,
          halfHeight: 36,
        },
      ])?.id,
    ).toBe("bridge");
  });
});

describe("deterministicPortTrack", () => {
  it.each([1, 2, 4, 8, 16])(
    "lays out %i ports on stable non-overlapping tracks",
    (count) => {
      const first = deterministicPortTrack(count, { x: 100, y: 80 });
      const second = deterministicPortTrack(count, { x: 100, y: 80 });
      expect(second).toEqual(first);
      expect(first).toHaveLength(count);
      expect(new Set(first.map((port) => `${port.x}:${port.y}`)).size).toBe(
        count,
      );
      for (const port of first) {
        expect(
          Math.hypot(port.labelX - port.x, port.labelY - port.y),
        ).toBeGreaterThanOrEqual(10);
        if (port.side === "left") expect(port.textAnchor).toBe("end");
        if (port.side === "right") expect(port.textAnchor).toBe("start");
      }
    },
  );

  it("preserves internal coordinates when the owner moves", () => {
    const original = deterministicPortTrack(8, { x: 100, y: 80 });
    const moved = deterministicPortTrack(8, { x: 145, y: 110 });
    moved.forEach((port, index) => {
      expect(port.x - original[index].x).toBeCloseTo(45);
      expect(port.y - original[index].y).toBeCloseTo(30);
    });
  });
});

describe("topologyLabelPriority", () => {
  it("reduces secondary labels before hiding identity", () => {
    expect(topologyLabelPriority(1, "comfortable")).toBe("full");
    expect(topologyLabelPriority(0.7, "comfortable")).toBe("identity-state");
    expect(topologyLabelPriority(0.4, "comfortable")).toBe("identity");
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
