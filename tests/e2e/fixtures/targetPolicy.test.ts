import { describe, expect, it } from "vitest";
import { validateTargetAcceptance } from "./targetPolicy";

const environment = {
  base_url: "http://10.72.1.7:18082",
  target_kind: "remote-privileged" as const,
  service_version: "v1",
  release: {
    version: "0.5.1-test",
    candidate_id: "candidate-test",
    binary_digest: `sha256:${"1".repeat(64)}`,
    contract_digest: `sha256:${"2".repeat(64)}`,
    built_at: "2026-08-04T00:00:00Z",
  },
  capabilities: {},
  capability_decisions: [],
  templates: [],
  baseline_clean: false,
  baseline_laboratory_ids: ["existing-lab"],
  baseline_runtime_ownership: [],
};

describe("target acceptance policy", () => {
  it("requires preserve mode for an existing production baseline", () => {
    expect(() =>
      validateTargetAcceptance(environment, {
        profile: "target-host",
        baselineMode: "clean",
        expectedHost: "10.72.1.7",
      }),
    ).toThrow(/preserve/);
    expect(() =>
      validateTargetAcceptance(environment, {
        profile: "target-host",
        baselineMode: "preserve",
        expectedHost: "10.72.1.7",
      }),
    ).not.toThrow();
  });

  it("rejects local-disposable classification and placeholder releases", () => {
    expect(() =>
      validateTargetAcceptance(
        { ...environment, target_kind: "local-disposable" },
        {
          profile: "target-host",
          baselineMode: "preserve",
          expectedHost: "10.72.1.7",
        },
      ),
    ).toThrow(/remote-privileged/);
    expect(() =>
      validateTargetAcceptance(
        {
          ...environment,
          release: { ...environment.release, candidate_id: "dev" },
        },
        {
          profile: "target-host",
          baselineMode: "preserve",
          expectedHost: "10.72.1.7",
        },
      ),
    ).toThrow(/immutable release/);
  });
});
