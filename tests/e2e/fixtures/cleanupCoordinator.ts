import type { APIRequestContext } from "@playwright/test";
import type {
  CleanupRecord,
  OwnedResource,
  RuntimeOwnershipObservation,
} from "./acceptanceTypes";
import { ResourceLedger } from "./resourceLedger";
import { scaledTimeout } from "./timeoutScale";

export interface CleanupOptions {
  timeoutMs?: number;
}

async function deleteOwnedResource(
  request: APIRequestContext,
  resource: OwnedResource,
): Promise<"deleted" | "cascade"> {
  if (resource.resource_type === "laboratory") {
    const current = await request.get(`/api/v1/labs/${resource.resource_id}`);
    if (current.status() === 404) return "deleted";
    let revision = resource.revision || 1;
    if (current.ok()) {
      const snapshot = (await current.json()) as {
        laboratory?: { revision?: number };
        revision?: number;
      };
      revision = snapshot.laboratory?.revision ?? snapshot.revision ?? revision;
    }
    let response = await request.delete(
      `/api/v1/labs/${resource.resource_id}`,
      {
        headers: {
          "If-Match": String(revision),
          "Idempotency-Key": crypto.randomUUID(),
        },
      },
    );
    if ([409, 412].includes(response.status())) {
      const refreshed = await request.get(
        `/api/v1/labs/${resource.resource_id}`,
      );
      if (refreshed.ok()) {
        const snapshot = (await refreshed.json()) as {
          laboratory?: { revision?: number };
          revision?: number;
        };
        const revision = snapshot.laboratory?.revision ?? snapshot.revision;
        if (revision) {
          response = await request.delete(
            `/api/v1/labs/${resource.resource_id}`,
            {
              headers: {
                "If-Match": String(revision),
                "Idempotency-Key": crypto.randomUUID(),
              },
            },
          );
        }
      }
    }
    if (![200, 202, 204, 404].includes(response.status())) {
      throw new Error(`laboratory cleanup returned ${response.status()}`);
    }
    return "deleted";
  }
  if (resource.resource_type === "capture") {
    const response = await request.delete(
      `/api/v1/captures/${resource.resource_id}`,
      { headers: { "Idempotency-Key": crypto.randomUUID() } },
    );
    if (![200, 202, 204, 404, 409].includes(response.status())) {
      throw new Error(`capture cleanup returned ${response.status()}`);
    }
    return "deleted";
  }
  if (resource.resource_type === "traffic_filter") {
    const terminal = async () => {
      const current = await request.get(
        `/api/v1/traffic-filters/${resource.resource_id}`,
      );
      if (current.status() === 404) return true;
      if (!current.ok()) {
        const problem = (await current.json().catch(() => ({}))) as {
          code?: string;
          message?: string;
        };
        return (
          current.status() === 400 &&
          problem.code === "invalid_request" &&
          /traffic filter not found/i.test(problem.message || "")
        );
      }
      const body = (await current.json()) as {
        traffic_filter?: { state?: string };
      };
      return ["stopped", "cancelled", "failed"].includes(
        body.traffic_filter?.state || "",
      );
    };
    if (await terminal()) return "deleted";
    const response = await request.delete(
      `/api/v1/traffic-filters/${resource.resource_id}`,
      { headers: { "Idempotency-Key": crypto.randomUUID() } },
    );
    if (response.status() === 400 && (await terminal())) return "deleted";
    if (![200, 202, 204, 404, 409].includes(response.status())) {
      throw new Error(`filter cleanup returned ${response.status()}`);
    }
    return "deleted";
  }
  if (resource.cleanup_method === "laboratory-cascade") return "cascade";
  throw new Error(
    `unsupported cleanup method ${resource.cleanup_method} for ${resource.resource_type}`,
  );
}

function ownershipKey(record: RuntimeOwnershipObservation) {
  return [
    record.resource_type,
    record.resource_id,
    record.object_kind,
    record.cleanup_state,
    record.ownership_class,
  ].join(":");
}

function ownershipKeys(records: RuntimeOwnershipObservation[]) {
  return [...new Set(enforcedOwnership(records).map(ownershipKey))].sort();
}

function enforcedOwnership(records: RuntimeOwnershipObservation[]) {
  return records.filter(
    (record) =>
      record.ownership_class !== "foreign_observed" &&
      !(record.resource_type === "unknown" && !record.ownership_class),
  );
}

export async function cleanupOwnedResources(
  request: APIRequestContext,
  ledger: ResourceLedger,
  baselineLaboratoryIds: string[],
  trigger: CleanupRecord["trigger"],
  options: CleanupOptions = {},
  baselineRuntimeOwnership: RuntimeOwnershipObservation[] = [],
): Promise<CleanupRecord> {
  const startedAt = new Date().toISOString();
  const deadline = Date.now() + scaledTimeout(options.timeoutMs || 60_000);
  const remediation: string[] = [];
  const resources = ledger
    .active()
    .sort(
      (left, right) =>
        Number(left.resource_type === "laboratory") -
        Number(right.resource_type === "laboratory"),
    );

  for (const resource of resources) {
    try {
      await ledger.setState(
        resource.resource_type,
        resource.resource_id,
        "deleting",
      );
      const disposition = await deleteOwnedResource(request, resource);
      if (disposition === "deleted") {
        await ledger.setState(
          resource.resource_type,
          resource.resource_id,
          "deleted",
        );
      }
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      const action = `${resource.resource_type}:${resource.resource_id}: ${message}`;
      remediation.push(action);
      await ledger.setState(
        resource.resource_type,
        resource.resource_id,
        "leaked",
        action,
      );
    }
    if (Date.now() > deadline) break;
  }

  let baselineRestored = false;
  let ownershipRestored = false;
  const baselineOwnership = ownershipKeys(baselineRuntimeOwnership);
  try {
    while (Date.now() <= deadline) {
      const [labsResponse, ownershipResponse] = await Promise.all([
        request.get("/api/v1/labs"),
        request.get("/api/v1/runtime-ownership"),
      ]);
      if (labsResponse.ok() && ownershipResponse.ok()) {
        const raw = await labsResponse.json();
        const laboratories = (Array.isArray(raw) ? raw : []) as Array<{
          id: string;
        }>;
        const current = laboratories.map((laboratory) => laboratory.id).sort();
        baselineRestored =
          JSON.stringify(current) ===
          JSON.stringify([...baselineLaboratoryIds].sort());
        const rawOwnership = await ownershipResponse.json();
        const currentOwnership = (Array.isArray(rawOwnership)
          ? rawOwnership
          : []) as RuntimeOwnershipObservation[];
        ownershipRestored =
          JSON.stringify(
            ownershipKeys(currentOwnership),
          ) ===
          JSON.stringify(baselineOwnership);
        if (baselineRestored && ownershipRestored) break;
      }
      await new Promise((resolve) => setTimeout(resolve, 250));
    }
  } catch (error) {
    remediation.push(error instanceof Error ? error.message : String(error));
  }

  if (baselineRestored && ownershipRestored) {
    for (const resource of ledger.active()) {
      if (resource.cleanup_method !== "laboratory-cascade") continue;
      await ledger.setState(
        resource.resource_type,
        resource.resource_id,
        "deleted",
      );
    }
  } else if (!ownershipRestored) {
    remediation.push("runtime ownership baseline was not restored");
  }

  const remaining = ledger.active();
  return {
    started_at: startedAt,
    finished_at: new Date().toISOString(),
    trigger,
    attempted: true,
    resources: ledger.list(),
    baseline_restored: baselineRestored && ownershipRestored,
    remaining_count: remaining.length,
    remediation,
  };
}
