import Ajv2020 from "ajv/dist/2020.js";
import addFormats from "ajv-formats";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

const schema = JSON.parse(
  readFileSync(
    fileURLToPath(
      new URL(
        "../../../specs/008-ui-overlap-remediation/contracts/acceptance-evidence.schema.json",
        import.meta.url,
      ),
    ),
    "utf8",
  ),
);

function validator() {
  const ajv = new Ajv2020({ strict: true });
  addFormats(ajv);
  return ajv.compile(schema);
}

function validEvidence() {
  return {
    schema_version: "1.0",
    run_id: "ui-audit-1",
    candidate: {
      candidate_id: "candidate-1",
      commit_sha: "1234567",
      artifact_digest: "sha256:example",
    },
    matrix: {
      themes: ["light", "dark"],
      viewports: [
        { width: 1024, height: 768 },
        { width: 1366, height: 768 },
        { width: 1920, height: 1080 },
      ],
      display_scales: [1, 1.25],
    },
    scenarios: [
      {
        id: "topology-ports",
        surface: "topology",
        theme: "dark",
        viewport: { width: 1024, height: 768 },
        display_scale: 1,
        result: "passed",
        checks: [{ kind: "overlap", result: "passed", message: "none" }],
        artifacts: [],
      },
    ],
    summary: {
      result: "passed",
      blocking_findings: 0,
      serious_findings: 0,
      unapproved_english_count: 0,
      page_horizontal_overflow_count: 0,
    },
    cleanup: {
      baseline_digest: "before",
      final_digest: "after",
      owned_residual_count: 0,
    },
    created_at: "2026-08-06T00:00:00Z",
  };
}

describe("UI audit acceptance evidence schema", () => {
  it("accepts a complete passing result", () => {
    expect(validator()(validEvidence())).toBe(true);
  });

  it("requires a waiver reason", () => {
    const evidence = validEvidence();
    evidence.scenarios[0].result = "waived";
    const validate = validator();
    expect(validate(evidence)).toBe(false);
    expect(validate.errors?.some((error) => error.keyword === "required")).toBe(
      true,
    );
  });
});
