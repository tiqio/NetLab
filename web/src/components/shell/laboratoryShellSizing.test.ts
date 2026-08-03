import { describe, expect, it } from "vitest";
import {
  BOTTOM_DRAWER_MIN_HEIGHT,
  CANVAS_MIN_HEIGHT,
  RESIZE_HANDLE_HEIGHT,
  clampBottomDrawerSize,
} from "./laboratoryShellSizing";

describe("laboratory shell sizing", () => {
  it("always leaves enough room for the topology canvas and resize handle", () => {
    const availableHeight = 700;
    expect(clampBottomDrawerSize(900, availableHeight)).toBe(
      availableHeight - CANVAS_MIN_HEIGHT - RESIZE_HANDLE_HEIGHT,
    );
  });

  it("keeps the operations drawer large enough to remain usable", () => {
    expect(clampBottomDrawerSize(20, 700)).toBe(BOTTOM_DRAWER_MIN_HEIGHT);
  });

  it("handles unusually short windows without producing a negative size", () => {
    expect(clampBottomDrawerSize(400, 240)).toBe(BOTTOM_DRAWER_MIN_HEIGHT);
  });
});
