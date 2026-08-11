import { readFile, readdir, writeFile } from "node:fs/promises";
import { resolve } from "node:path";
import { request, type FullConfig } from "@playwright/test";
import type {
  AcceptanceEvidence,
  EnvironmentSnapshot,
} from "./acceptanceTypes";
import { enforceAcceptanceArtifactPolicy } from "./artifactPolicy";
import { requiresCompleteAcceptanceCoverage } from "./acceptanceScope";
import { cleanupOwnedResources } from "./cleanupCoordinator";
import { writeEvidence } from "./evidenceReporter";
import { assertCompleteInteractionCoverage } from "./interactionCoverage";
import { ResourceLedger } from "./resourceLedger";
import { renderRunSummary } from "./runSummary";
import { assertCompleteVersionCoverage } from "./versionCoverage";
import inventoryDocument from "../matrices/interaction-inventory.json";

async function evidenceFiles(root: string) {
  const values: string[] = [];
  const aggregatePath = resolve(root, "evidence.json");
  async function visit(directory: string) {
    for (const entry of await readdir(directory, { withFileTypes: true }).catch(
      () => [],
    )) {
      const target = resolve(directory, entry.name);
      if (entry.isDirectory()) await visit(target);
      else if (entry.name === "evidence.json" && target !== aggregatePath)
        values.push(target);
    }
  }
  await visit(root);
  return values;
}

export default async function globalTeardown(config: FullConfig) {
  const output = process.env.NETLAB_ACCEPTANCE_OUTPUT_DIR;
  if (!output) return;
  const runId = process.env.NETLAB_ACCEPTANCE_RUN_ID || "unknown-run";
  const environment = JSON.parse(
    await readFile(resolve(output, "environment.json"), "utf8"),
  ) as EnvironmentSnapshot;
  const ledgerPath = resolve(output, "resources.ledger.json");
  const context = await request.newContext({
    baseURL: String(config.projects[0]?.use.baseURL || environment.base_url),
  });
  try {
    const ledger = await ResourceLedger.load(runId, ledgerPath).catch(
      () => new ResourceLedger(runId, ledgerPath),
    );
    const files = await evidenceFiles(output);
    const evidence = await Promise.all(
      files.map(
        async (file) =>
          JSON.parse(await readFile(file, "utf8")) as AcceptanceEvidence,
      ),
    );
    const failed = evidence.some((item) => item.status === "failed");
    const cleanup = await cleanupOwnedResources(
      context,
      ledger,
      environment.baseline_laboratory_ids,
      failed ? "failure" : "success",
      {},
      environment.baseline_runtime_ownership,
    );
    const aggregate: AcceptanceEvidence = {
      schema_version: "1.0.0",
      run_id: runId,
      status:
        !failed && cleanup.baseline_restored && cleanup.remaining_count === 0
          ? "passed"
          : "failed",
      started_at:
        evidence.map((item) => item.started_at).sort()[0] ||
        new Date().toISOString(),
      finished_at: new Date().toISOString(),
      environment,
      viewports: [
        { width: 1920, height: 1080 },
        { width: 1024, height: 768 },
      ],
      interaction_results: evidence.flatMap((item) => item.interaction_results),
      version_coverage: evidence.flatMap((item) => item.version_coverage),
      cleanup,
      client_observations: evidence.flatMap(
        (item) => item.client_observations || [],
      ),
      visual_audit_results: evidence.flatMap(
        (item) => item.visual_audit_results || [],
      ),
    };
    const gateErrors: string[] = [];
    const completeCoverage = requiresCompleteAcceptanceCoverage(
      environment.target_kind,
      process.env.NETLAB_ACCEPTANCE_SCOPE,
    );
    try {
      assertCompleteInteractionCoverage(
        inventoryDocument.interactions,
        aggregate,
        completeCoverage,
      );
    } catch (error) {
      gateErrors.push(error instanceof Error ? error.message : String(error));
    }
    if (completeCoverage) {
      try {
        assertCompleteVersionCoverage(environment, aggregate.version_coverage);
      } catch (error) {
        gateErrors.push(error instanceof Error ? error.message : String(error));
      }
    }
    if (gateErrors.length) {
      aggregate.status = "failed";
      aggregate.cleanup.remediation.push(...gateErrors);
    }
    await writeEvidence(aggregate, output);
    await writeFile(
      resolve(output, "summary.txt"),
      `${renderRunSummary(aggregate)}\n`,
      { mode: 0o600 },
    );
    const summary = {
      run_id: runId,
      status: aggregate.status,
      evidence: "evidence.json",
      summary: "summary.txt",
      test_evidence: files.map((file) => file.replace(`${output}/`, "")),
      cleanup,
      removed_artifacts: [] as string[],
    };
    await writeFile(
      resolve(output, "run-summary.json"),
      `${JSON.stringify(summary, null, 2)}\n`,
      { mode: 0o600 },
    );
    summary.removed_artifacts = await enforceAcceptanceArtifactPolicy(output);
    await writeFile(
      resolve(output, "run-summary.json"),
      `${JSON.stringify(summary, null, 2)}\n`,
      { mode: 0o600 },
    );
    await enforceAcceptanceArtifactPolicy(output);
    if (gateErrors.length) {
      throw new Error(
        `Acceptance coverage gate failed: ${gateErrors.join("; ")}`,
      );
    }
  } finally {
    await context.dispose();
  }
}
