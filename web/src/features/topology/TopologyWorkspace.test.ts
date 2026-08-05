import { describe, expect, it, vi } from "vitest";
import type { NetworkObjectLink, NetworkObjectLinkTaskEnvelope } from "@/api";
import { runObjectLinkDeletion } from "./objectLinkDeletion";
import { openTopologyCreateDrawer } from "./topologyCreateDrawerState";

describe("TopologyWorkspace create drawer state", () => {
  it("opens one selecting drawer from the toolbar", () => {
    expect(openTopologyCreateDrawer()).toEqual({
      open: true,
      selection: undefined,
    });
  });

  it("opens the same drawer with a palette preselection", () => {
    expect(
      openTopologyCreateDrawer({
        kind: "pc",
        name: "PC",
        networkObjectKind: "pc",
      }),
    ).toMatchObject({
      open: true,
      selection: { networkObjectKind: "pc" },
    });
  });
});

const link: NetworkObjectLink = {
  id: "object-link-1",
  laboratory_id: "lab",
  object_a_id: "switch-a",
  port_a_name: "swp1",
  object_b_id: "switch-b",
  port_b_name: "swp1",
  revision: 3,
  desired_state: "connected",
  observed_state: "connected",
};

const envelope = {
  network_object_link: { ...link, observed_state: "disconnecting" },
  task: {
    id: "task-1",
    kind: "network_object_link.delete",
    resource_type: "network_object_link",
    resource_id: link.id,
    state: "queued",
    progress_current: 0,
    progress_total: 1,
    created_at: "2026-08-03T00:00:00Z",
  },
} satisfies NetworkObjectLinkTaskEnvelope;

describe("TopologyWorkspace object-link deletion", () => {
  it("hides and clears before submitting, then records the durable task", async () => {
    const calls: string[] = [];
    await runObjectLinkDeletion(link, {
      hide: () => calls.push("hide"),
      clearSelection: () => calls.push("clear"),
      submit: vi.fn(async () => {
        calls.push("submit");
        return envelope;
      }),
      recordTask: () => calls.push("task"),
      unhide: vi.fn(),
      reload: vi.fn(),
    });
    expect(calls).toEqual(["hide", "clear", "submit", "task"]);
  });

  it("restores and reloads a link when submission fails", async () => {
    const unhide = vi.fn();
    const reload = vi.fn();
    await expect(
      runObjectLinkDeletion(link, {
        hide: vi.fn(),
        clearSelection: vi.fn(),
        submit: vi.fn(async () => {
          throw new Error("revision conflict");
        }),
        recordTask: vi.fn(),
        unhide,
        reload,
      }),
    ).rejects.toThrow("revision conflict");
    expect(unhide).toHaveBeenCalledWith(link.id);
    expect(reload).toHaveBeenCalledOnce();
  });
});
