import { describe, expect, it } from "vitest";
import { resourceVisualSemantic } from "./topologyVisualSemantics";

describe("network object runtime state", () => {
  it("renders active as healthy instead of stopped or unknown", () => {
    expect(resourceVisualSemantic("nat_bridge", "active")).toMatchObject({
      stateLabel: "Running",
      color: "#22c55e",
    });
  });
});
