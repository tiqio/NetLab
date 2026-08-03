import { access, mkdir, writeFile } from "node:fs/promises";
import { constants } from "node:fs";
import { resolve } from "node:path";
import { request, type FullConfig } from "@playwright/test";
import { discoverEnvironment } from "./preflight";
import { timeoutScale } from "./timeoutScale";

export default async function globalSetup(config: FullConfig) {
  timeoutScale();
  const baseURL = String(config.projects[0]?.use.baseURL || "");
  const parsed = new URL(baseURL);
  if (parsed.username || parsed.password) {
    throw new Error("Acceptance base URL must not contain credentials");
  }
  const runId = process.env.NETLAB_ACCEPTANCE_RUN_ID || crypto.randomUUID();
  process.env.NETLAB_ACCEPTANCE_RUN_ID = runId;
  const output = resolve(
    process.cwd(),
    process.env.NETLAB_ACCEPTANCE_OUTPUT_DIR ||
      `test-results/acceptance/${runId}`,
  );
  process.env.NETLAB_ACCEPTANCE_OUTPUT_DIR = output;
  await mkdir(output, { recursive: true, mode: 0o700 });
  const probe = resolve(output, ".write-probe");
  await writeFile(probe, "ok", { mode: 0o600 });
  const launcher = process.env.NETLAB_ACCEPTANCE_WIRESHARK_LAUNCHER;
  if (launcher) await access(launcher, constants.X_OK);
  const context = await request.newContext({ baseURL });
  try {
    const environment = await discoverEnvironment(
      context,
      baseURL,
      process.env.NETLAB_ACCEPTANCE_PROFILE === "target-host"
        ? "remote-privileged"
        : "local-disposable",
    );
    if (
      environment.target_kind === "remote-privileged" &&
      !environment.baseline_clean
    ) {
      throw new Error("Target-host acceptance requires a clean baseline");
    }
    await writeFile(
      resolve(output, "environment.json"),
      `${JSON.stringify(environment, null, 2)}\n`,
      { mode: 0o600 },
    );
  } finally {
    await context.dispose();
  }
}
