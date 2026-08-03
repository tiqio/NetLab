import {
  test as base,
  expect,
  type APIRequestContext,
  type Page,
} from "@playwright/test";
import { mkdir, writeFile } from "node:fs/promises";
import { basename, resolve } from "node:path";
import type {
  AcceptanceEvidence,
  EnvironmentSnapshot,
  InteractionResult,
  VersionCoverage,
  ClientObservation,
} from "./acceptanceTypes";
import { cleanupOwnedResources } from "./cleanupCoordinator";
import { discoverEnvironment } from "./preflight";
import { ResourceLedger } from "./resourceLedger";
import { writeEvidence } from "./evidenceReporter";
import { renderRunSummary } from "./runSummary";

interface AcceptanceFixtures {
  runId: string;
  environment: EnvironmentSnapshot;
  ledger: ResourceLedger;
  interactionResults: InteractionResult[];
  versionCoverage: VersionCoverage[];
  secondPage: Page;
  automation: APIRequestContext;
  outputDirectory: string;
  runOutputDirectory: string;
  evidence: Partial<AcceptanceEvidence>;
  clientObservations: ClientObservation[];
}

export const test = base.extend<AcceptanceFixtures>({
  runId: async ({}, use) => {
    await use(process.env.NETLAB_ACCEPTANCE_RUN_ID || crypto.randomUUID());
  },
  runOutputDirectory: async ({ runId }, use) => {
    const directory = resolve(
      process.cwd(),
      process.env.NETLAB_ACCEPTANCE_OUTPUT_DIR ||
        `test-results/acceptance/${runId}`,
    );
    await mkdir(directory, { recursive: true });
    await use(directory);
  },
  outputDirectory: async ({ runOutputDirectory }, use, testInfo) => {
    const slug = `${testInfo.project.name}-${testInfo.testId.replace(/[^a-zA-Z0-9_-]+/g, "-")}`;
    const directory = resolve(runOutputDirectory, "tests", slug);
    await mkdir(directory, { recursive: true });
    await use(directory);
  },
  ledger: async ({ runId, runOutputDirectory }, use) => {
    await use(
      new ResourceLedger(
        runId,
        resolve(runOutputDirectory, "resources.ledger.json"),
      ),
    );
  },
  interactionResults: async ({}, use) => use([]),
  versionCoverage: async ({}, use) => use([]),
  clientObservations: async ({}, use) => use([]),
  automation: async ({ playwright, baseURL }, use) => {
    const context = await playwright.request.newContext({ baseURL });
    await use(context);
    await context.dispose();
  },
  environment: async ({ automation, baseURL }, use) => {
    const targetKind =
      process.env.NETLAB_ACCEPTANCE_PROFILE === "target-host"
        ? "remote-privileged"
        : "local-disposable";
    const environment = await discoverEnvironment(
      automation,
      baseURL || "http://127.0.0.1:8080",
      targetKind,
    );
    if (targetKind === "remote-privileged" && !environment.baseline_clean) {
      throw new Error(
        "Target-host acceptance requires a clean laboratory baseline",
      );
    }
    await use(environment);
  },
  secondPage: async ({ browser }, use) => {
    const context = await browser.newContext();
    const page = await context.newPage();
    await use(page);
    await context.close();
  },
  evidence: [
    async (
      {
        runId,
        environment,
        interactionResults,
        versionCoverage,
        ledger,
        automation,
        outputDirectory,
        clientObservations,
      },
      use,
      testInfo,
    ) => {
      const evidence: Partial<AcceptanceEvidence> = {
        schema_version: "1.0.0",
        run_id: runId,
        started_at: new Date().toISOString(),
        environment,
        viewports: [
          { width: 1920, height: 1080 },
          { width: 1024, height: 768 },
        ],
        interaction_results: interactionResults,
        version_coverage: versionCoverage,
        client_observations: clientObservations,
      };
      await use(evidence);
      const eligibleSkip =
        testInfo.status === "skipped" &&
        environment.target_kind === "local-disposable";
      const successfulOutcome = testInfo.status === "passed" || eligibleSkip;
      const trigger = successfulOutcome ? "success" : "failure";
      evidence.cleanup = await cleanupOwnedResources(
        automation,
        ledger,
        environment.baseline_laboratory_ids,
        trigger,
        {},
        environment.baseline_runtime_ownership,
      );
      evidence.finished_at = new Date().toISOString();
      evidence.status =
        successfulOutcome &&
        evidence.cleanup.remaining_count === 0 &&
        evidence.cleanup.baseline_restored
          ? "passed"
          : "failed";
      if (interactionResults.length === 0) {
        const skipReason = testInfo.annotations.find(
          (annotation) => annotation.type === "skip",
        )?.description;
        const stableTestId = [
          basename(testInfo.file).replace(/\.spec\.ts$/, ""),
          testInfo.title.replace(/[^a-zA-Z0-9]+/g, "-").replace(/^-|-$/g, ""),
        ]
          .join(".")
          .toLowerCase();
        interactionResults.push({
          interaction_id: `test.${stableTestId}`,
          status: eligibleSkip
            ? "skipped"
            : testInfo.status === "passed"
              ? "passed"
              : "failed",
          viewport: testInfo.project.use.viewport || { width: 1, height: 1 },
          activation: "pointer",
          precondition: "test fixture initialized",
          action: testInfo.title,
          expected: "test reaches its declared terminal outcome",
          actual: testInfo.status || "unknown",
          duration_ms: testInfo.duration,
          cleanup_status: evidence.cleanup.baseline_restored
            ? "baseline restored"
            : "cleanup incomplete",
          ...(eligibleSkip
            ? {
                skip: {
                  capability: "target-host-runtime",
                  class: "environment-optional" as const,
                  reason:
                    skipReason ||
                    "privileged runtime validation is not available locally",
                  evidence: environment.target_kind,
                },
              }
            : {}),
        });
      }
      const finalized = evidence as AcceptanceEvidence;
      await writeEvidence(finalized, outputDirectory);
      await writeFile(
        resolve(outputDirectory, "summary.txt"),
        `${renderRunSummary(finalized)}\n`,
        { mode: 0o600 },
      );
    },
    { auto: true },
  ],
});

export { expect };
