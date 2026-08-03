import { mkdir, readFile, writeFile } from "node:fs/promises";
import { dirname } from "node:path";
import type { OwnedResource, ResourceCleanupState } from "./acceptanceTypes";
import { sanitizeEvidence } from "./redaction";

export class ResourceLedger {
  private resources = new Map<string, OwnedResource>();

  constructor(
    readonly runId: string,
    readonly path?: string,
  ) {}

  private key(resourceType: string, resourceId: string) {
    return `${resourceType}:${resourceId}`;
  }

  async add(
    resource: Omit<
      OwnedResource,
      "owner_run_id" | "created_at" | "cleanup_state"
    > &
      Partial<Pick<OwnedResource, "created_at" | "cleanup_state">>,
  ) {
    const value: OwnedResource = {
      ...resource,
      owner_run_id: this.runId,
      created_at: resource.created_at || new Date().toISOString(),
      cleanup_state: resource.cleanup_state || "owned",
    };
    this.resources.set(this.key(value.resource_type, value.resource_id), value);
    await this.persist();
    return value;
  }

  async setState(
    resourceType: string,
    resourceId: string,
    cleanupState: ResourceCleanupState,
    remediation?: string,
  ) {
    const value = this.resources.get(this.key(resourceType, resourceId));
    if (!value) {
      throw new Error(`Unknown owned resource ${resourceType}:${resourceId}`);
    }
    value.cleanup_state = cleanupState;
    value.remediation = remediation;
    await this.persist();
  }

  list() {
    return [...this.resources.values()].map((resource) => ({ ...resource }));
  }

  active() {
    return this.list().filter(
      (resource) => !["deleted", "missing"].includes(resource.cleanup_state),
    );
  }

  async persist() {
    if (!this.path) return;
    await mkdir(dirname(this.path), { recursive: true });
    const existing = await readFile(this.path, "utf8")
      .then((value) => JSON.parse(value) as OwnedResource[])
      .catch(() => []);
    for (const resource of existing) {
      const key = this.key(resource.resource_type, resource.resource_id);
      if (!this.resources.has(key)) this.resources.set(key, resource);
    }
    await writeFile(
      this.path,
      JSON.stringify(sanitizeEvidence(this.list()), null, 2),
      { mode: 0o600 },
    );
  }

  static async load(runId: string, path: string) {
    const ledger = new ResourceLedger(runId, path);
    const values = JSON.parse(await readFile(path, "utf8")) as OwnedResource[];
    for (const value of values) {
      if (value.owner_run_id !== runId) {
        throw new Error(`Ledger owner mismatch for ${value.resource_id}`);
      }
      ledger.resources.set(
        ledger.key(value.resource_type, value.resource_id),
        value,
      );
    }
    return ledger;
  }
}
