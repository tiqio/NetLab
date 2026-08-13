import { describe, expect, it } from "vitest";
import {
  boxSelect,
  cleanSelection,
  rangeSelect,
  selectAll,
  selectOne,
  toggleSingleSelected,
  toggleSelected,
  captureSelection,
  restoreSelection,
} from "./topologySelection";

const bounds = [
  { id: "a", left: 0, top: 0, right: 20, bottom: 20 },
  { id: "b", left: 30, top: 0, right: 50, bottom: 20 },
  { id: "c", left: 60, top: 0, right: 80, bottom: 20 },
];

describe("topology selection", () => {
  it("selects every unique applicable resource", () => {
    expect(selectAll(["a", "b", "a", ""])).toEqual(["a", "b"]);
  });
  it("supports immutable single and toggle selection", () => {
    const original = ["a", "b"];
    expect(selectOne("c")).toEqual(["c"]);
    expect(toggleSelected(original, "b")).toEqual(["a"]);
    expect(toggleSelected(original, "c")).toEqual(["a", "b", "c"]);
    expect(original).toEqual(["a", "b"]);
  });
  it("toggles a normal single click while preserving other selected items", () => {
    expect(toggleSingleSelected(["a"], "a")).toEqual([]);
    expect(toggleSingleSelected(["a", "b"], "a")).toEqual(["b"]);
    expect(toggleSingleSelected(["a", "b"], "c")).toEqual(["c"]);
  });

  it("selects ordered ranges and intersecting boxes", () => {
    expect(rangeSelect(["a", "b", "c"], "a", "c")).toEqual(["a", "b", "c"]);
    expect(
      boxSelect({ left: 10, top: -5, right: 45, bottom: 25 }, bounds),
    ).toEqual(["a", "b"]);
    expect(
      boxSelect(
        { left: 25, top: -5, right: 75, bottom: 25 },
        bounds,
        ["a"],
        true,
      ),
    ).toEqual(["a", "b", "c"]);
  });

  it("removes deleted resources without mutating selection", () => {
    const selected = ["a", "missing", "c"];
    expect(cleanSelection(selected, new Set(["a", "b", "c"]))).toEqual([
      "a",
      "c",
    ]);
    expect(selected).toEqual(["a", "missing", "c"]);
  });
});

describe("connection selection restoration", () => {
  it("captures immutable selection and restores a fresh copy", () => {
    const selected = ["node-a"];
    const snapshot = captureSelection(selected, "node", "node-a");
    selected.push("node-b");
    const restored = restoreSelection(snapshot);
    expect(restored).toEqual({
      ids: ["node-a"],
      type: "node",
      focusedResourceId: "node-a",
    });
    expect(restored.ids).not.toBe(snapshot.ids);
  });
});
