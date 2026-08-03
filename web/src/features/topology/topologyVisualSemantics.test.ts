import { describe, expect, it } from "vitest";
import {
  normalizeTopologyKind,
  resourceVisualSemantic,
} from "./topologyVisualSemantics";

describe("topology visual semantics", () => {
  it.each([
    ["qemu", "QEMU virtual machine"],
    ["docker", "Docker container"],
    ["pc", "PC endpoint"],
    ["bridge", "Layer 2 bridge"],
    ["nat_bridge", "NAT bridge"],
    ["switch_l2", "Layer 2 switch"],
    ["switch_l3", "Layer 3 switch"],
  ])("labels %s without relying on color", (kind, label) => {
    expect(resourceVisualSemantic(kind, "running").label).toContain(label);
  });

  it.each([
    ["running", "Running", "solid"],
    ["stopped", "Stopped or unknown", "solid"],
    ["starting", "Transitioning", "dashed"],
    ["failed", "Failed", "dotted"],
  ] as const)("maps %s state", (state, label, pattern) => {
    expect(resourceVisualSemantic("qemu", state)).toMatchObject({
      stateLabel: label,
      pattern,
    });
  });

  it("uses a stable fallback and explicit selection/traffic borders", () => {
    expect(normalizeTopologyKind("vendor-unknown")).toBe("fallback");
    expect(
      resourceVisualSemantic("vendor-unknown", "failed", true, true),
    ).toMatchObject({
      kind: "fallback",
      selected: true,
      traffic: true,
      borderColor: "#f8fafc",
    });
  });
  it("labels desired and actual state independently", () => {
    expect(
      resourceVisualSemantic("qemu", "stopped", false, false, "running").label,
    ).toContain("Desired Running · Actual Stopped or unknown");
  });
});
