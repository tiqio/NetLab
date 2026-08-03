import type { AcceptanceEvidence } from "./acceptanceTypes";

export function normalizeRun(evidence: AcceptanceEvidence) {
  return {
    status: evidence.status,
    interactions: evidence.interaction_results
      .map(({ interaction_id, status, viewport, activation }) => ({
        interaction_id,
        status,
        viewport,
        activation,
      }))
      .sort((a, b) => JSON.stringify(a).localeCompare(JSON.stringify(b))),
    versions: evidence.version_coverage
      .map(
        ({ runtime, device_family, version_id, coverage_level, result }) => ({
          runtime,
          device_family,
          version_id,
          coverage_level,
          result,
        }),
      )
      .sort((a, b) => JSON.stringify(a).localeCompare(JSON.stringify(b))),
    cleanup: {
      baseline_restored: evidence.cleanup.baseline_restored,
      remaining_count: evidence.cleanup.remaining_count,
    },
  };
}

export function compareRuns(
  first: AcceptanceEvidence,
  second: AcceptanceEvidence,
) {
  const left = normalizeRun(first);
  const right = normalizeRun(second);
  return {
    equal: JSON.stringify(left) === JSON.stringify(right),
    first: left,
    second: right,
  };
}
