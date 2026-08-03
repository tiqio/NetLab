import { readFile, readdir, rm, stat } from "node:fs/promises";
import path from "node:path";
import { assertNoSensitiveEvidence } from "./redaction";

const retainedNames = new Set([
  "environment.json",
  "resources.ledger.json",
  "evidence.json",
  "run-summary.json",
  "summary.txt",
]);

function mayRetain(relative: string) {
  const normalized = relative.split(path.sep).join("/");
  if (normalized.startsWith("playwright/")) return false;
  if (retainedNames.has(normalized)) return true;
  return /^tests\/[^/]+\/(evidence\.json|summary\.txt)$/.test(normalized);
}

async function assertSafeFile(target: string) {
  const body = await readFile(target, "utf8");
  if (target.endsWith(".json")) {
    assertNoSensitiveEvidence(JSON.parse(body));
    return;
  }
  assertNoSensitiveEvidence(body);
}

export async function enforceAcceptanceArtifactPolicy(root: string) {
  const removed: string[] = [];
  async function visit(directory: string) {
    for (const name of await readdir(directory).catch(() => [])) {
      const target = path.join(directory, name);
      const metadata = await stat(target);
      if (metadata.isDirectory()) {
        await visit(target);
      } else if (!mayRetain(path.relative(root, target))) {
        await rm(target, { force: true });
        removed.push(path.relative(root, target));
      } else {
        await assertSafeFile(target);
      }
    }
  }
  await visit(root);
  return removed;
}
