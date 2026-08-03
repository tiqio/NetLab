import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";
import type { AcceptanceEvidence } from "./acceptanceTypes";
import { assertNoSensitiveEvidence, sanitizeEvidence } from "./redaction";
import { validateEvidence } from "./schemaValidation";

export async function finalizeEvidence(
  input: AcceptanceEvidence,
): Promise<AcceptanceEvidence> {
  const evidence = sanitizeEvidence(input);
  assertNoSensitiveEvidence(evidence);
  if (Date.parse(evidence.finished_at) < Date.parse(evidence.started_at))
    throw new Error("Evidence finished_at precedes started_at");
  if (
    evidence.cleanup.remaining_count !== 0 ||
    !evidence.cleanup.baseline_restored
  )
    evidence.status = "failed";
  for (const result of evidence.interaction_results) {
    if (!result.actual || !result.expected || !result.cleanup_status)
      throw new Error(`Incomplete terminal result ${result.interaction_id}`);
  }
  return validateEvidence(evidence);
}

export async function writeEvidence(
  input: AcceptanceEvidence,
  outputDirectory: string,
  filename = "evidence.json",
) {
  const evidence = await finalizeEvidence(input);
  await mkdir(outputDirectory, { recursive: true });
  const target = path.join(outputDirectory, filename);
  await writeFile(target, `${JSON.stringify(evidence, null, 2)}\n`, {
    mode: 0o600,
  });
  return target;
}
