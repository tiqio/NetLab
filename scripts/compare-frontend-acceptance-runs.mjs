#!/usr/bin/env node

import { readFile } from "node:fs/promises";

const [leftPath, rightPath] = process.argv.slice(2);
if (!leftPath || !rightPath) {
  console.error("usage: compare-frontend-acceptance-runs.mjs <first-evidence> <second-evidence>");
  process.exit(2);
}

function normalize(evidence) {
  return {
    status: evidence.status,
    capabilities: evidence.environment?.capabilities,
    templates: evidence.environment?.templates,
    viewports: evidence.viewports,
    interactions: [...(evidence.interaction_results || [])]
      .map((item) => ({
        interaction_id: item.interaction_id,
        status: item.status,
        viewport: item.viewport,
        activation: item.activation,
        actual: item.actual,
        cleanup_status: item.cleanup_status,
        skip: item.skip,
      }))
      .sort((left, right) =>
        JSON.stringify(left).localeCompare(JSON.stringify(right)),
      ),
    versions: [...(evidence.version_coverage || [])]
      .map((item) => ({
        runtime: item.runtime,
        device_family: item.device_family,
        version_id: item.version_id,
        image_id: item.image_id,
        coverage_level: item.coverage_level,
        result: item.result,
        skip: item.skip,
      }))
      .sort((left, right) =>
        JSON.stringify(left).localeCompare(JSON.stringify(right)),
      ),
    cleanup: {
      baseline_restored: evidence.cleanup?.baseline_restored,
      remaining_count: evidence.cleanup?.remaining_count,
    },
  };
}

const first = normalize(JSON.parse(await readFile(leftPath, "utf8")));
const second = normalize(JSON.parse(await readFile(rightPath, "utf8")));
if (JSON.stringify(first) !== JSON.stringify(second)) {
  console.error("consecutive acceptance runs differ after normalization");
  console.error(JSON.stringify({ first, second }, null, 2));
  process.exit(1);
}
console.log("consecutive acceptance runs are equivalent");
