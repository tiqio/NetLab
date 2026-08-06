import { describe, expect, it } from "vitest";
import { summarizeVisualAudit, severityFor } from "./visualAudit";

describe("visual audit evidence", () => {
  it("classifies severe findings and summarizes results", () => {
    expect(severityFor("hit-target-conflict")).toBe("blocking");
    expect(severityFor("overlap")).toBe("serious");
    expect(summarizeVisualAudit([{ scenario_id: "a", surface: "topology", theme: "dark", viewport: { width: 1024, height: 768 }, display_scale: 1, status: "failed", blocking_findings: 1, serious_findings: 2, untranslated_text_count: 0, page_horizontal_overflow: false }])).toMatchObject({ total: 1, failed: 1, blocking_findings: 1, serious_findings: 2 });
  });
});
