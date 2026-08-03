import { describe, expect, it } from "vitest";
import type { AcceptanceEvidence } from "./acceptanceTypes";
import { finalizeEvidence } from "./evidenceReporter";
import { compareRuns } from "./runComparator";
import { renderRunSummary } from "./runSummary";

function evidence(): AcceptanceEvidence {
  const started = "2026-07-27T00:00:00.000Z";
  return {
    schema_version: "1.0.0",
    run_id: "accept-0001",
    status: "passed",
    started_at: started,
    finished_at: "2026-07-27T00:00:01.000Z",
    environment: {
      base_url: "http://127.0.0.1:8080",
      target_kind: "local-disposable",
      capabilities: {},
      capability_decisions: [],
      templates: [],
      baseline_clean: true,
      baseline_laboratory_ids: [],
      baseline_runtime_ownership: [],
    },
    viewports: [
      { width: 1920, height: 1080 },
      { width: 1024, height: 768 },
    ],
    interaction_results: [
      {
        interaction_id: "lab.create",
        status: "passed",
        viewport: { width: 1920, height: 1080 },
        activation: "pointer",
        precondition: "clean",
        action: "create",
        expected: "created",
        actual: "created",
        duration_ms: 10,
        cleanup_status: "deleted",
      },
    ],
    version_coverage: [],
    cleanup: {
      started_at: started,
      finished_at: "2026-07-27T00:00:01.000Z",
      trigger: "success",
      attempted: true,
      resources: [],
      baseline_restored: true,
      remaining_count: 0,
      remediation: [],
    },
  };
}

describe("evidence reporter", () => {
  it("validates sanitized terminal evidence and renders a summary", async () => {
    const value = await finalizeEvidence(evidence());
    expect(value.status).toBe("passed");
    expect(renderRunSummary(value)).toContain("0 remaining");
  });

  it("forces failure when cleanup invariants fail", async () => {
    const value = evidence();
    value.cleanup.remaining_count = 1;
    value.cleanup.baseline_restored = false;
    expect((await finalizeEvidence(value)).status).toBe("failed");
  });

  it("compares deterministic results without timestamps", () => {
    const first = evidence();
    const second = evidence();
    second.run_id = "accept-0002";
    expect(compareRuns(first, second).equal).toBe(true);
  });
});
