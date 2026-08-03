import { describe, expect, it } from "vitest";
import { ClientObserver } from "./clientObserver";
import { requiredVersionPlan } from "./versionCoverage";

describe("regression framework", () => {
  it("requires ordered, converged client observations", () => {
    const observer = new ClientObserver();
    observer.record({
      client_id: "browser-a",
      mutation_id: "m1",
      event_sequence: 1,
      resource_revision: 2,
      observed_at: new Date().toISOString(),
      convergence_ms: 10,
    });
    observer.record({
      client_id: "browser-b",
      mutation_id: "m1",
      event_sequence: 2,
      resource_revision: 2,
      observed_at: new Date().toISOString(),
      convergence_ms: 20,
    });
    expect(
      observer.assertConverged("m1", ["browser-a", "browser-b"]),
    ).toHaveLength(2);
  });

  it("selects one full journey and remaining lifecycle coverage", () => {
    const plan = requiredVersionPlan({
      base_url: "http://localhost:8080",
      target_kind: "local-disposable",
      capabilities: {},
      capability_decisions: [],
      baseline_clean: true,
      baseline_laboratory_ids: [],
      baseline_runtime_ownership: [],
      templates: [
        {
          template_id: "t",
          device_family: "ubuntu",
          runtime: "qemu",
          versions: [
            { version_id: "a", available: true },
            { version_id: "b", available: true },
          ],
        },
      ],
    });
    expect(plan.map((item) => item.coverage_level)).toEqual([
      "full-journey",
      "lifecycle-connectivity",
    ]);
  });
});
