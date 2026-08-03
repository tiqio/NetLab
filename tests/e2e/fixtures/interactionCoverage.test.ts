import { describe, expect, it } from "vitest";
import { assertCompleteInteractionCoverage } from "./interactionCoverage";
import type { InteractionDefinition, InteractionResult } from "./acceptanceTypes";

const definition: InteractionDefinition = {
  id: "laboratory.refresh",
  area: "laboratory",
  label: "Refresh",
  locator: { role: "button", name: "Refresh" },
  applicable_states: ["always"],
  activation: ["pointer", "keyboard"],
  outcome_class: "mutation",
  cleanup_effect: "none",
  sensitive_evidence: [],
};

function result(
  activation: "pointer" | "keyboard",
  width: number,
): InteractionResult {
  return {
    interaction_id: definition.id,
    status: "passed",
    viewport: { width, height: width === 1920 ? 1080 : 768 },
    activation,
    precondition: "ready",
    action: "refresh",
    expected: "authoritative refresh",
    actual: "refreshed",
    duration_ms: 10,
    cleanup_status: "none",
  };
}

describe("interaction coverage", () => {
  it("requires every activation at every viewport", () => {
    const complete = [
      result("pointer", 1920),
      result("keyboard", 1920),
      result("pointer", 1024),
      result("keyboard", 1024),
    ];
    expect(() =>
      assertCompleteInteractionCoverage([definition], {
        interaction_results: complete,
        viewports: [
          { width: 1920, height: 1080 },
          { width: 1024, height: 768 },
        ],
      }),
    ).not.toThrow();
    expect(() =>
      assertCompleteInteractionCoverage([definition], {
        interaction_results: complete.slice(0, 3),
        viewports: [
          { width: 1920, height: 1080 },
          { width: 1024, height: 768 },
        ],
      }),
    ).toThrow(/laboratory.refresh\/keyboard\/1024x768/);
  });

  it("still rejects unknown evidence in partial local runs", () => {
    expect(() =>
      assertCompleteInteractionCoverage(
        [definition],
        {
          interaction_results: [
            { ...result("pointer", 1920), interaction_id: "unknown.control" },
          ],
          viewports: [{ width: 1920, height: 1080 }],
        },
        false,
      ),
    ).toThrow(/unknown ids/);
    expect(() =>
      assertCompleteInteractionCoverage(
        [definition],
        {
          interaction_results: [],
          viewports: [{ width: 1920, height: 1080 }],
        },
        false,
      ),
    ).not.toThrow();
  });
});
