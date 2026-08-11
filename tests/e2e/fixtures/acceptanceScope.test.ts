import { describe, expect, it } from "vitest";
import { requiresCompleteAcceptanceCoverage } from "./acceptanceScope";

describe("acceptance scope", () => {
  it("requires complete coverage for full target-host acceptance", () => {
    expect(requiresCompleteAcceptanceCoverage("remote-privileged", "full")).toBe(
      true,
    );
    expect(
      requiresCompleteAcceptanceCoverage("remote-privileged", undefined),
    ).toBe(true);
  });

  it("limits coverage gates for topology-only target acceptance", () => {
    expect(
      requiresCompleteAcceptanceCoverage(
        "remote-privileged",
        "topology-unification",
      ),
    ).toBe(false);
  });

  it("does not require remote coverage for local acceptance", () => {
    expect(requiresCompleteAcceptanceCoverage("local", "full")).toBe(false);
  });
});
