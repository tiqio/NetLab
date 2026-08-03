import { describe, expect, it } from "vitest";
import { mkdtemp, mkdir, readFile, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { enforceAcceptanceArtifactPolicy } from "./artifactPolicy";

describe("acceptance artifact policy", () => {
  it("retains only sanitized evidence and removes browser artifacts", async () => {
    const root = await mkdtemp(join(tmpdir(), "netlab-artifacts-"));
    await mkdir(join(root, "playwright", "failure"), { recursive: true });
    await mkdir(join(root, "tests", "case"), { recursive: true });
    await writeFile(join(root, "environment.json"), "{}\n");
    await writeFile(
      join(root, "tests", "case", "evidence.json"),
      '{"status":"passed"}\n',
    );
    await writeFile(
      join(root, "playwright", "failure", "error-context.md"),
      "console payload must not survive",
    );
    const removed = await enforceAcceptanceArtifactPolicy(root);
    expect(removed).toContain("playwright/failure/error-context.md");
    await expect(
      readFile(join(root, "tests", "case", "evidence.json"), "utf8"),
    ).resolves.toContain("passed");
  });
});
