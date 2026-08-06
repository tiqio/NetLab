import { describe, expect, it, vi } from "vitest";
import type { APIRequestContext, APIResponse } from "@playwright/test";
import { cleanupOwnedResources } from "./cleanupCoordinator";
import { ResourceLedger } from "./resourceLedger";

function response(body: unknown, status = 200) {
  return {
    ok: () => status >= 200 && status < 300,
    status: () => status,
    json: async () => body,
  } as APIResponse;
}

describe("cleanup coordinator", () => {
  it("ignores foreign observations while enforcing managed ownership", async () => {
    const ledger = new ResourceLedger("run-foreign");
    const request = {
      get: vi.fn(async (path: string) =>
        path === "/api/v1/labs"
          ? response([])
          : response([
              {
                resource_type: "unknown",
                resource_id: "foreign-2",
                object_kind: "linux_link",
                object_name: "eth999",
                cleanup_state: "unknown_observed",
                ownership_class: "foreign_observed",
              },
            ]),
      ),
    } as unknown as APIRequestContext;

    const cleanup = await cleanupOwnedResources(
      request,
      ledger,
      [],
      "success",
      { timeoutMs: 1_000 },
      [
        {
          resource_type: "unknown",
          resource_id: "foreign-1",
          object_kind: "linux_link",
          object_name: "eth998",
          cleanup_state: "unknown_observed",
          ownership_class: "foreign_observed",
        },
      ],
    );

    expect(cleanup.baseline_restored).toBe(true);
  });

  it("does not silently delete an unsupported resource", async () => {
    const ledger = new ResourceLedger("run-unsupported");
    await ledger.add({
      resource_type: "temporary-file",
      resource_id: "tmp-1",
      cleanup_method: "unlink",
    });
    const request = {
      get: vi
        .fn()
        .mockResolvedValueOnce(response([]))
        .mockResolvedValueOnce(response([])),
    } as unknown as APIRequestContext;

    const cleanup = await cleanupOwnedResources(
      request,
      ledger,
      [],
      "failure",
      { timeoutMs: 1 },
    );

    expect(cleanup.baseline_restored).toBe(true);
    expect(cleanup.remaining_count).toBe(1);
    expect(cleanup.resources[0]?.cleanup_state).toBe("leaked");
    expect(cleanup.baseline_restored).toBe(true);
    expect(cleanup.remediation).toContain(
      "temporary-file:tmp-1: unsupported cleanup method unlink for temporary-file",
    );
  });

  it("keeps cascade resources active while runtime ownership remains", async () => {
    const ledger = new ResourceLedger("run-owned");
    await ledger.add({
      resource_type: "qemu-process",
      resource_id: "node-1",
      cleanup_method: "laboratory-cascade",
    });
    const request = {
      get: vi.fn(async (path: string) =>
        path === "/api/v1/labs"
          ? response([])
          : response([
              {
                resource_type: "node",
                resource_id: "node-1",
                object_kind: "process",
                object_name: "qemu-node-1",
                cleanup_state: "active",
              },
            ]),
      ),
    } as unknown as APIRequestContext;

    const cleanup = await cleanupOwnedResources(
      request,
      ledger,
      [],
      "failure",
      { timeoutMs: 1 },
    );

    expect(cleanup.baseline_restored).toBe(false);
    expect(cleanup.remaining_count).toBe(1);
    expect(cleanup.resources[0]?.cleanup_state).toBe("deleting");
    expect(cleanup.remediation).toContain(
      "runtime ownership baseline was not restored",
    );
  });
});
