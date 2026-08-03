import { defineConfig } from "@playwright/test";

const acceptanceOutput =
  process.env.NETLAB_ACCEPTANCE_OUTPUT_DIR || "test-results/acceptance/local";

export default defineConfig({
  globalSetup: "../tests/e2e/fixtures/globalSetup.ts",
  globalTeardown: "../tests/e2e/fixtures/globalTeardown.ts",
  fullyParallel: false,
  workers: 1,
  timeout: 120_000,
  testDir: "../tests/e2e",
  testMatch: "**/*.spec.ts",
  outputDir: `${acceptanceOutput}/playwright`,
  reporter: [["list"]],
  use: {
    baseURL: "http://127.0.0.1:8080",
    trace: "off",
    screenshot: "off",
    video: "off",
  },
  projects: [
    { name: "desktop", use: { viewport: { width: 1920, height: 1080 } } },
    { name: "minimum", use: { viewport: { width: 1024, height: 768 } } },
  ],
  webServer: {
    command:
      "cd .. && rm -rf /tmp/netlab-e2e && ./bin/netlabd -config deploy/config/netlab.test.yaml",
    url: "http://127.0.0.1:8080/healthz",
    reuseExistingServer: process.env.NETLAB_ACCEPTANCE_REUSE_SERVER === "1",
    timeout: 30_000,
  },
});
