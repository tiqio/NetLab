import { defineConfig } from "@playwright/test";
import { scaledTimeout } from "../tests/e2e/fixtures/timeoutScale";

const baseURL = process.env.NETLAB_ACCEPTANCE_BASE_URL;
const acceptanceOutput =
  process.env.NETLAB_ACCEPTANCE_OUTPUT_DIR ||
  "test-results/acceptance/target-host";

if (!baseURL) {
  throw new Error(
    "NETLAB_ACCEPTANCE_BASE_URL is required for target-host acceptance",
  );
}

export default defineConfig({
  globalSetup: "../tests/e2e/fixtures/globalSetup.ts",
  globalTeardown: "../tests/e2e/fixtures/globalTeardown.ts",
  testDir: "../tests/e2e",
  testMatch: "**/*.spec.ts",
  outputDir: `${acceptanceOutput}/playwright`,
  fullyParallel: false,
  workers: 1,
  timeout: scaledTimeout(120_000),
  reporter: [["list"]],
  use: {
    baseURL,
    trace: "off",
    screenshot: "off",
    video: "off",
  },
  projects: [
    { name: "desktop", use: { viewport: { width: 1920, height: 1080 } } },
    { name: "standard", use: { viewport: { width: 1366, height: 768 } } },
    { name: "minimum", use: { viewport: { width: 1024, height: 768 } } },
  ],
});
