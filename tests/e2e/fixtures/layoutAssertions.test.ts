import { describe, expect, it } from "vitest";
import { isInsideViewport, overlapArea } from "./layoutAssertions";

describe("visual layout assertions", () => {
  it("calculates only the intersecting area", () => {
    expect(
      overlapArea(
        { x: 0, y: 0, width: 20, height: 10 },
        { x: 15, y: 4, width: 20, height: 10 },
      ),
    ).toBe(30);
    expect(
      overlapArea(
        { x: 0, y: 0, width: 10, height: 10 },
        { x: 20, y: 20, width: 10, height: 10 },
      ),
    ).toBe(0);
  });

  it("allows one pixel of viewport rounding", () => {
    expect(
      isInsideViewport(
        { x: -0.5, y: 0, width: 1001, height: 768 },
        { width: 1000, height: 768 },
      ),
    ).toBe(true);
    expect(
      isInsideViewport(
        { x: -2, y: 0, width: 1000, height: 768 },
        { width: 1000, height: 768 },
      ),
    ).toBe(false);
  });
});
