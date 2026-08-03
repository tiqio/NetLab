import { describe, expect, it } from "vitest";
import {
  defaultWorkspacePreferences,
  parseWorkspacePreferences,
  pruneLinkRoutes,
  removeWorkspacePreferences,
} from "./useWorkspacePreferences";

describe("workspace preferences", () => {
  it("falls back for invalid or cross-laboratory values", () => {
    expect(
      parseWorkspacePreferences({ schemaVersion: 2 }, "lab"),
    ).toMatchObject({
      ...defaultWorkspacePreferences("lab"),
      updatedAt: expect.any(String),
    });
    expect(
      parseWorkspacePreferences(
        { schemaVersion: 1, laboratoryId: "other" },
        "lab",
      ).laboratoryId,
    ).toBe("lab");
  });
  it("clamps sizes and zoom and removes legacy local placements", () => {
    const value = parseWorkspacePreferences(
      {
        schemaVersion: 1,
        laboratoryId: "lab",
        panels: {
          devicePalette: { size: 2 },
          inspector: { size: 9000 },
          bottomDrawer: { size: 1 },
        },
        viewport: { zoom: 99 },
        placements: { good: { x: 1, y: 2 }, bad: { x: "secret", y: 2 } },
        activeBottomTab: "console",
        labelDensity: "minimal",
        reducedMotion: true,
      },
      "lab",
    );
    expect(value.panels.devicePalette.size).toBe(180);
    expect(value.panels.inspector.size).toBe(640);
    expect(value.viewport.zoom).toBe(8);
    expect("placements" in value).toBe(false);
    expect(value.labelDensity).toBe("minimal");
    expect(value.reducedMotion).toBe(true);
  });

  it("removes routes missing from the authoritative link snapshot", () => {
    const preferences = defaultWorkspacePreferences("lab");
    preferences.linkRoutes = {
      current: [{ x: 10, y: 20 }],
      deleted: [{ x: 30, y: 40 }],
    };

    expect(pruneLinkRoutes(preferences, ["current"])).toBe(true);
    expect(preferences.linkRoutes).toEqual({
      current: [{ x: 10, y: 20 }],
    });
    expect(pruneLinkRoutes(preferences, ["current"])).toBe(false);
  });

  it("removes all browser-local state for a deleted laboratory", () => {
    const removed: string[] = [];
    removeWorkspacePreferences("lab", {
      removeItem: (key) => removed.push(key),
    });
    expect(removed).toEqual(["netlab.workspace.v1.lab"]);
  });
});
