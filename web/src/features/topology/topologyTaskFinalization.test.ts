import { describe, expect, it, vi } from "vitest";
import type { OperationTask } from "@/api";
import { waitForTopologyTaskFinal } from "./topologyTaskFinalization";

const task = (state: OperationTask["state"]): OperationTask => ({
  id: "task-1",
  kind: "topology_connection.create",
  resource_type: "topology_connection",
  resource_id: "connection-1",
  state,
  progress_current: state === "cancelled" ? 1 : 0,
  progress_total: 2,
  created_at: "2026-08-10T00:00:00Z",
});

describe("waitForTopologyTaskFinal", () => {
  it("queries until the authoritative task is terminal", async () => {
    const getTask = vi
      .fn()
      .mockResolvedValueOnce(task("cancelling"))
      .mockResolvedValueOnce(task("running"))
      .mockResolvedValueOnce(task("cancelled"));
    const finalTask = await waitForTopologyTaskFinal(getTask, "task-1", {
      intervalMs: 0,
      delay: async () => undefined,
    });
    expect(finalTask.state).toBe("cancelled");
    expect(getTask).toHaveBeenCalledTimes(3);
  });

  it("returns the latest authority when the polling budget expires", async () => {
    const getTask = vi.fn().mockResolvedValue(task("running"));
    const finalTask = await waitForTopologyTaskFinal(getTask, "task-1", {
      attempts: 2,
      intervalMs: 0,
      delay: async () => undefined,
    });
    expect(finalTask.state).toBe("running");
    expect(getTask).toHaveBeenCalledTimes(2);
  });
});
