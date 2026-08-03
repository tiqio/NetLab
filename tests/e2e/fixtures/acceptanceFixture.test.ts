import { describe, expect, it } from "vitest";
import { mkdtemp, readFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { optionalEnvironmentDecision } from "./preflight";
import { assertNoSensitiveEvidence, sanitizeEvidence } from "./redaction";
import { ResourceLedger } from "./resourceLedger";

describe("acceptance foundations", () => {
  it("permits skips only for environment-optional capabilities", () => {
    expect(
      optionalEnvironmentDecision("desktop_wireshark", false, "headless"),
    ).toMatchObject({
      class: "environment-optional",
      decision: "skip",
    });
  });

  it("redacts sensitive keys and credential-looking values", () => {
    const value = sanitizeEvidence({
      password: "not-retained",
      message: "password=not-retained",
      safe: "task-1",
    });
    expect(value).toEqual({ message: "[REDACTED]", safe: "task-1" });
    expect(() => assertNoSensitiveEvidence(value)).not.toThrow();
  });

  it("persists an owned resource ledger without changing ownership", async () => {
    const directory = await mkdtemp(join(tmpdir(), "netlab-ledger-"));
    const path = join(directory, "ledger.json");
    const ledger = new ResourceLedger("run-12345678", path);
    await ledger.add({
      resource_type: "laboratory",
      resource_id: "lab-1",
      revision: 1,
      cleanup_method: "frontend-delete",
    });
    await ledger.setState("laboratory", "lab-1", "deleted");
    const persisted = JSON.parse(await readFile(path, "utf8"));
    expect(persisted[0]).toMatchObject({
      owner_run_id: "run-12345678",
      cleanup_state: "deleted",
    });
  });
});
