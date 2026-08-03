import type { AcceptanceEvidence } from "./acceptanceTypes";

export function renderRunSummary(evidence: AcceptanceEvidence) {
  const passed = evidence.interaction_results.filter(
    (item) => item.status === "passed",
  ).length;
  const failed = evidence.interaction_results.filter(
    (item) => item.status === "failed",
  ).length;
  const skipped = evidence.interaction_results.filter(
    (item) => item.status === "skipped",
  ).length;
  const duration =
    Date.parse(evidence.finished_at) - Date.parse(evidence.started_at);
  return [
    `Status: ${evidence.status}`,
    `Interactions: ${passed} passed, ${failed} failed, ${skipped} skipped`,
    `Duration: ${duration} ms`,
    `Version coverage: ${evidence.version_coverage.length}`,
    `Cleanup: ${evidence.cleanup.remaining_count} remaining; baseline restored=${evidence.cleanup.baseline_restored}`,
  ].join("\n");
}
